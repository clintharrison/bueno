package ace

//#cgo CFLAGS: -Iinclude
//#cgo LDFLAGS: -lace_bt -lace_osal
//#include "ace.go.h"
import "C"
import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"runtime"
	"slices"
	"sync"
	"time"
	"unsafe"

	"github.com/clintharrison/bueno/ace/address"
	"github.com/clintharrison/bueno/core/withlock"
	"github.com/google/uuid"
)

// Unfortunately these have to be globals because the C callbacks need to access them.
var (
	adapter            *aceAdapter
	adapterOnce        sync.Once
	sessionHandle      C.aceBT_sessionHandle
	gGattService       []C.aceBT_bleGattsService_t
	scanMu             sync.Mutex
	scanInstanceHandle C.aceBT_scanInstanceHandle
	scanResultFunc     func(adapter Adapter, device ScanResult)
	shutdownFuncs      []func()
	// this is used repeatedly for any notify messages
	notifyCh chan []byte

	// these largely signal completion of async operations
	initCh              chan struct{}
	bleRegisterCh       chan struct{}
	beaconRegisterCh    chan struct{}
	connectCh           chan ConnHandle
	pairCh              chan struct{}
	gattcSvcDiscoveryCh chan struct{}
	gattcDbCh           chan struct{}
	gattDisconnectCh    chan struct{}
	charsWriteCh        chan struct{}
	bleWriteDescCh      chan struct{}
)

type ConnHandle struct {
	conn C.aceBT_bleConnHandle
}

type GattServiceType int

const (
	PrimaryService   GattServiceType = C.ACEBT_BLE_GATT_SERVICE_TYPE_PRIMARY
	SecondaryService GattServiceType = C.ACEBT_BLE_GATT_SERVICE_TYPE_SECONDARY
	IncludedService  GattServiceType = C.ACEBT_BLE_GATT_SERVICE_TYPE_INCLUDED
)

type DeviceService struct {
	UUID   uuid.UUID
	Handle uint16
	Type   GattServiceType
	svc    *C.aceBT_bleGattsService_t
}

type BLEWriteType int

const (
	_ BLEWriteType = iota
	BLEWriteType_NoResponse
	BLEWriteType_Default
	BLEWriteType_Signed
)

func (s *DeviceService) Characteristics() iter.Seq[DeviceCharacteristic] {
	return func(yield func(chr DeviceCharacteristic) bool) {
		for head := s.svc.charsList.stqh_first; head != nil; head = head.link.stqe_next {
			char_val := head.value
			var record C.aceBT_bleGattRecord_t
			C.cgo_getRecordFromChar(&char_val, &record)
			var desc C.aceBT_bleGattDescriptor_t
			C.cgo_getDescriptorFromChar(&char_val, &desc)
			// TODO: Follow linked list on characteristic for additional descriptors
			// slog.Debug("Characteristic",
			// 	"uuid", UUIDFromGATTCharRecord(&char_val),
			// 	"handle", record.handle,
			// 	"is_notify", desc.is_notify,
			// 	"is_set", desc.is_set,
			// 	"write_type", desc.write_type)
			writeType := BLEWriteType(desc.write_type)
			char := DeviceCharacteristic{
				UUID:      UUIDFromACEUUID_LE(record.uuid),
				Service:   s,
				IsNotify:  bool(desc.is_notify),
				isSet:     bool(desc.is_set),
				WriteType: writeType,
				aceChar:   &head.value,
			}
			if !yield(char) {
				return
			}
		}
	}
}

func UUIDFromGATTCharRecord(char_rec *C.aceBT_bleGattCharacteristicsValue_t) uuid.UUID {
	ret := make([]byte, 16)
	C.cgo_getUUIDFromGATTCharRecord(char_rec, (*C.uint8_t)(unsafe.SliceData(ret)))
	slices.Reverse(ret)
	return uuid.Must(uuid.FromBytes(ret))
}

type Adapter interface {
	Scan(f func(adapter Adapter, device ScanResult)) error
	StopScan() error
	RadioState() (AceRadioState, error)
	EnableRadio() error
	GetServices(conn ConnHandle) ([]DeviceService, error)
	Connect(addr address.Address) (ConnHandle, error)
	Disconnect(ConnHandle) error
	Pair(addr address.Address) error
	IsBonded(addr address.Address) (bool, error)
	Close()
	GetCharacteristics(svc *DeviceService) ([]DeviceCharacteristic, error)
}

type DeviceCharacteristic struct {
	UUID      uuid.UUID
	Service   *DeviceService
	Handle    uint16
	IsNotify  bool
	isSet     bool
	WriteType BLEWriteType
	aceChar   *C.aceBT_bleGattCharacteristicsValue_t
}

type AceResponseType int

const (
	// writeresponse not required
	BLEWriteTypeResp_No AceResponseType = iota

	// write response required
	BLEWriteTypeResp_Required
)

func (dc *DeviceCharacteristic) SetNotify(conn ConnHandle) (chan []byte, error) {
	if notifyCh == nil {
		notifyCh = make(chan []byte)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bleWriteDescCh = make(chan struct{})
	slog.Debug("SetNotify()", "conn", conn, "characteristic", dc.UUID.String())
	if err := errForStatus(C.cgo_bleSetNotification(
		sessionHandle, conn.conn, dc.aceChar, true)); err != nil {
		return nil, err
	}
	select {
	case <-bleWriteDescCh:
		// Notification descriptor write completed
		slog.Debug("Notification descriptor write completed")
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout SetNotify(): %w", ctx.Err())
	}
	return notifyCh, nil
}

// Write always does WriteRequest (vs no response writes)
func (dc *DeviceCharacteristic) Write(conn ConnHandle, data []uint8) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	slog.Info("Write()", "conn", conn, "data", fmt.Sprintf("%x", data))

	charsWriteCh = make(chan struct{})
	// warning, the characteristic is mutated in-place
	buffLen := len(data)
	if buffLen > 20 {
		buffLen = 20
	}
	if err := errForStatus(C.cgo_bleWriteCharacteristics(
		/* aceBT_sessionHandle*/ sessionHandle,
		/*aceBT_bleConnHandle*/ conn.conn,
		/* aceBT_bleGattCharacteristicsValue_t* */ dc.aceChar,
		/* aceBT_responseType_t */ C.ACEBT_BLE_WRITE_TYPE_RESP_REQUIRED,
		(*C.uint8_t)(unsafe.SliceData(data)),
		C.size_t(len(data)),
	)); err != nil {
		slog.Error("Failed to write characteristic", "error", err)
		return err
	}

	select {
	case <-charsWriteCh:
		// Characteristic write completed
		slog.Debug("characteristic write finished")
	case <-ctx.Done():
		slog.Error("Timed out waiting for characteristic write", "error", ctx.Err())
		return ctx.Err()
	}
	return nil
}

func (a *aceAdapter) GetCharacteristics(svc *DeviceService) ([]DeviceCharacteristic, error) {
	slog.Debug("Getting characteristics for service", "uuid", svc.UUID.String(), "handle", svc.Handle, "numChars", svc.svc.no_characteristics)
	chars := make([]DeviceCharacteristic, 0, svc.svc.no_characteristics)
	for char := range svc.Characteristics() {
		chars = append(chars, char)
	}
	// C.cgo_dumpChars(svc.svc)
	return chars, nil
}

func registerCleanupFunc(f func()) {
	if f == nil {
		return
	}
	shutdownFuncs = append(shutdownFuncs, f)
}

func (a *aceAdapter) Close() {
	for _, f := range shutdownFuncs {
		f()
	}
	if sessionHandle != nil {
		slog.Info("Closing ACE session", "sessionHandle", fmt.Sprintf("%p", sessionHandle))
		if err := errForStatus(C.aceBT_closeSession(sessionHandle)); err != nil {
			slog.Error("Failed to close ACE session", "sessionHandle", fmt.Sprintf("%p", sessionHandle), "error", err)
			sessionHandle = nil
		} else {
			slog.Info("Closed ACE session", "sessionHandle", fmt.Sprintf("%p", sessionHandle))
			sessionHandle = nil
		}
	}
}

func (a *aceAdapter) IsBonded(addr address.Address) (bool, error) {
	var deviceList *C.aceBT_deviceList_t
	// must call aceBT_freeDeviceList on deviceList when done
	aceStatus := C.aceBT_getBondedDevices((**C.aceBT_deviceList_t)(unsafe.Pointer(&deviceList)))
	if err := errForStatus(aceStatus); err != nil {
		slog.Error("Failed to get bonded devices", "status", aceStatus, "error", err)
		return false, err
	}
	defer C.aceBT_freeDeviceList((*C.aceBT_deviceList_t)(unsafe.Pointer(deviceList)))

	slog.Debug("Bonded", "num_devices", deviceList.num_devices)

	cgoDeviceList := C.cgo_getDeviceList(deviceList)
	bondedDevices := unsafe.Slice(cgoDeviceList.p_devices, cgoDeviceList.num_devices)

	// Check if the device is in the list of bonded devices
	for _, device := range bondedDevices {
		if NewAddressFromAce(device) == addr {
			slog.Debug("Found bonded device", "address", addr.ToString())
			return true, nil
		} else {
			slog.Debug("Device bonded but not recognized", "address", NewAddressFromAce(device).ToString())
		}
	}
	return false, nil
}

func UUIDFromACEUUID_LE(aceUuid C.aceBT_uuid_t) uuid.UUID {
	ret := make([]byte, 16)
	for i := 0; i < 16; i++ {
		ret[i] = byte(aceUuid.uu[15-i])
	}
	return uuid.Must(uuid.FromBytes(ret))
}

type aceAdapter struct {
}

func (a *aceAdapter) Pair(addr address.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pairCh := make(chan struct{})

	status := C.aceBT_pair(AddressToAce(addr), C.ACEBT_TRANSPORT_AUTO)
	if status == C.ACEBT_STATUS_DONE {
		slog.Info("Already paired", "address", addr.ToString(), "status", StatusFromCode(status))
		return nil
	}
	if err := errForStatus(status); err != nil {
		return fmt.Errorf("failed to pair device %s: %w", addr.ToString(), err)
	}

	select {
	case <-pairCh:
		slog.Info("Paired with device", "address", addr.ToString())
	case <-ctx.Done():
		slog.Error("Timed out waiting for pairing", "address", addr.ToString())
		return fmt.Errorf("timed out waiting for pairing with device %s", addr.ToString())
	}

	return nil
}

var _ Adapter = (*aceAdapter)(nil)

func Enable() (Adapter, error) {
	var err error
	adapterOnce.Do(func() {
		adapter, err = initAdapter()
		if err != nil {
			slog.Error("Failed to initialize ACE adapter", "error", err)
		}
	})
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

func initAdapter() (*aceAdapter, error) {
	runtime.LockOSThread()
	a := &aceAdapter{}
	if err := errForStatus(C.ace_init()); err != nil {
		return nil, err
	}
	if err := a.OpenSession(); err != nil {
		return nil, err
	}

	state, err := a.RadioState()
	if err != nil {
		slog.Error("Failed to get radio state", "error", err)
		return nil, err
	}
	if state != RadioEnabled {
		slog.Info("Radio is not enabled", "state", state)
		err = a.EnableRadio()
		if err != nil {
			slog.Error("Failed to enable radio", "error", err)
			return nil, err
		}
	}

	if err := a.register(); err != nil {
		slog.Error("Failed to register ACE callbacks", "error", err)
		return nil, err
	}
	return a, nil
}

func (a *aceAdapter) Scan(f func(adapter Adapter, device ScanResult)) error {
	err := withlock.DoErr(&scanMu, func() error {
		if scanInstanceHandle != nil {
			return errors.New("scan already in progress")
		}
		scanResultFunc = f
		return nil
	})

	if err != nil {
		slog.Error("Failed to start scan", "error", err)
		return err
	}
	client_id := (C.aceBT_BeaconClientId)(C.ACE_BEACON_CLIENT_TYPE_MONEYPENNY)
	aceStatus := C.aceBT_startBeaconScanWithDefaultParams(sessionHandle, client_id, &scanInstanceHandle)
	if err := errForStatus(aceStatus); err != nil {
		slog.Error("Failed to start beacon scan", "status", aceStatus, "error", err)
		return err
	}
	return nil
}

func (a *aceAdapter) StopScan() error {
	err := withlock.DoErr(&scanMu, func() error {
		if scanInstanceHandle == nil {
			return errors.New("no scan in progress")
		}
		aceStatus := C.aceBT_stopBeaconScan(scanInstanceHandle)
		if err := errForStatus(aceStatus); err != nil {
			slog.Error("Failed to stop beacon scan", "status", aceStatus, "error", err)
			return err
		}
		slog.Info("Stopped beacon scan")
		scanInstanceHandle = nil
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *aceAdapter) OpenSession() error {
	session_type := (C.aceBT_sessionType_t)(C.ACEBT_SESSION_TYPE_DUAL_MODE)
	status := C.aceBT_openSession(session_type, &C.session_callbacks, &sessionHandle)
	if err := errForStatus(status); err != nil {
		slog.Error("Failed to open ACE session", "status", status, "error", err)
		return err
	}
	slog.Info("Opened ACE session", "sessionHandle", fmt.Sprintf("%p", sessionHandle))
	return nil
}

func (a *aceAdapter) EnableRadio() error {
	const readyWaitDelay = 500 * time.Millisecond
	maxRetries := 10
	if sessionHandle == nil {
		return errors.New("session handle is nil, cannot enable radio")
	}
	slog.Info("Enabling radio", "sessionHandle", fmt.Sprintf("%p", sessionHandle))

	if err := errForStatus(C.aceBT_enableRadio(sessionHandle)); err != nil {
		slog.Error("failed to enable radio", "error", err)
	}

	for i := 0; i < maxRetries; i++ {
		radioState, err := a.RadioState()
		if err != nil {
			slog.Error("failed to get radio state", "error", err)
			return err
		}
		if radioState == RadioEnabled {
			slog.Debug("radio is enabled, quitting retry loop")
			return nil
		}
		time.Sleep(readyWaitDelay)
	}
	return fmt.Errorf("radio did not enable after %d retries", maxRetries)
}

func (a *aceAdapter) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bleRegisterCh = make(chan struct{})
	bleStatus := C.aceBT_bleRegister(sessionHandle, &C.ble_callbacks)
	if err := errForStatus(bleStatus); err != nil {
		slog.Error("Failed to register BLE callbacks", "status", bleStatus, "error", err)
		return err
	}
	select {
	case <-bleRegisterCh:
		// callback closed the channel
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for BLE client registration: %w", ctx.Err())
	}

	bleStatus = C.aceBt_bleRegisterGattClient(
		sessionHandle,
		&C.ble_gatt_client_callbacks,
		// This seems to be what the legacy function called by the CLI does?
		C.ACE_BT_BLE_APPID_GADGETS,
	)
	if err := errForStatus(bleStatus); err != nil {
		slog.Error("Failed to register GATT client", "status", bleStatus, "error", err)
		return err
	}

	beaconRegisterCh = make(chan struct{})
	bleStatus = C.aceBT_RegisterBeaconClient(
		sessionHandle,
		&C.beacon_callbacks,
	)
	if err := errForStatus(bleStatus); err != nil {
		slog.Error("Failed to register beacon client", "status", bleStatus, "error", err)
		return err
	}

	select {
	case <-beaconRegisterCh:
		// callback closed the channel
	case <-ctx.Done():
		slog.Error("Timed out waiting for beacon client registration", "err", ctx.Err())
		return fmt.Errorf("timed out waiting for beacon client registration")
	}

	slog.Debug("Registered ACE callbacks: BLE, beacon, GATTC", "sessionHandle", fmt.Sprintf("%p", sessionHandle))

	return nil
}

func (a *aceAdapter) Disconnect(conn ConnHandle) error {
	gattDisconnectCh = make(chan struct{})
	if err := errForStatus(C.aceBT_bleDisconnect(conn.conn)); err != nil {
		slog.Error("Failed to disconnect from device", "conn_handle", unsafe.Pointer(conn.conn), "error", err)
		return err
	}
	select {
	case <-gattDisconnectCh:
		slog.Info("Disconnected from device (and channel closed by callback)", "conn_handle", unsafe.Pointer(conn.conn))
	case <-time.After(10 * time.Second):
		slog.Error("Timed out waiting for disconnect", "conn_handle", unsafe.Pointer(conn.conn))
		return fmt.Errorf("timed out waiting for disconnect after 10 seconds")
	}
	C.aceBT_bleDeRegisterGattClient(sessionHandle)
	sessionHandle = nil
	return nil
}

func bondIfNeeded(ctx context.Context, adapter Adapter, addr address.Address) error {
	slog.Info("Checking if already bonded", "address", addr.ToString())
	bonded, err := adapter.IsBonded(addr)
	if err != nil {
		slog.Error("Failed to check if device is bonded", "error", err)
		return err
	}

	if !bonded {
		slog.Info("Ensuring paired", "address", addr.ToString())
		err := adapter.Pair(addr)
		if err != nil {
			slog.Error("Failed to pair with device", "error", err)
			return err
		}
		slog.Info("Device paired successfully", "address", addr.ToString())
	}
	return nil
}

func (a *aceAdapter) Connect(addr address.Address) (ConnHandle, error) {
	// TODO: call bondIfNeeded()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var connHandle ConnHandle
	connectCh = make(chan ConnHandle)
	slog.Debug("calling aceBt_bleConnect()", "address", addr.ToString())
	status := C.aceBt_bleConnect(
		/* aceBT_sessionHandle */ sessionHandle,
		/* aceBT_bdAddr_t* */ AddressToAce(addr),
		/* aceBt_bleConnParam_t */ C.ACE_BT_BLE_CONN_PARAM_BALANCED,
		/* aceBT_bleConnRole_t */ C.ACEBT_BLE_GATT_CLIENT_ROLE,
		/* autoconnect */ false,
		/* aceBt_bleConnPriority_t */ C.ACE_BT_BLE_CONN_PRIO_MEDIUM,
	)
	if err := errForStatus(status); err != nil {
		slog.Error("Failed to connect to device", "address", addr.ToString(), "error", err)
		return ConnHandle{}, err
	}
	slog.Debug("called aceBt_bleConnect", "status", status)

	select {
	case <-ctx.Done():
		slog.Error("Connection timed out", "timeout", ctx.Err(), "address", addr.ToString())
		return ConnHandle{}, ctx.Err()
	case connHandle = <-connectCh:
		slog.Info("Connected to device", "address", addr.ToString(), "conn_handle", unsafe.Pointer(connHandle.conn))
	}
	return connHandle, nil
}

func (a *aceAdapter) GetServices(conn ConnHandle) ([]DeviceService, error) {
	discoveryCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Discover the GATT services
	gattcSvcDiscoveryCh = make(chan struct{})
	C.aceBT_bleDiscoverAllServices(sessionHandle, conn.conn)
	select {
	case <-discoveryCtx.Done():
		slog.Error("Timed out waiting for GATT services to be discovered")
		return nil, fmt.Errorf("timed out waiting for GATT services to be discovered")
	case <-gattcSvcDiscoveryCh:
		slog.Info("GATT services discovered successfully")
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	gattcDbCh = make(chan struct{})
	// Actually get the GATT database? Dunno why this is two steps.
	C.aceBT_bleGetService(conn.conn)
	select {
	case <-dbCtx.Done():
		slog.Error("Timed out waiting for GATT DB")
		return nil, fmt.Errorf("timed out waiting for GATT DB")
	case <-gattcDbCh:
		slog.Info("GATT DB populated successfully")
	}

	if len(gGattService) == 0 {
		slog.Error("No GATT services available")
		return nil, errors.New("no GATT services available")
	}
	deviceServices := make([]DeviceService, len(gGattService))
	for i, svc := range gGattService {
		deviceServices[i] = deviceServiceFromAceService(&svc)
	}
	return deviceServices, nil
}

func deviceServiceFromAceService(svc *C.aceBT_bleGattsService_t) DeviceService {
	return DeviceService{
		UUID:   UUIDFromACEUUID_LE(svc.uuid),
		Handle: uint16(svc.handle),
		svc:    svc,
	}
}

func (a *aceAdapter) DumpGATTDB(conn ConnHandle) error {
	// if len(gattService) == 0 {
	// 	slog.Error("No GATT services available")
	// 	return nil
	// }
	return nil
}

//export advChangeCallback
func advChangeCallback(adv_instance C.aceBT_advInstanceHandle, state C.aceBT_beaconAdvState_t, power_mode C.aceBT_beaconPowerMode_t, beacon_mode C.aceBT_beaconAdvMode_t) {
	slog.Info("Beacon advertisement state changed", "adv_instance", adv_instance, "state", state, "power_mode", power_mode, "beacon_mode", beacon_mode)
}

//export onBleRegistered
func onBleRegistered(status C.aceBT_status_t) {
	defer close(bleRegisterCh)
	if err := errForStatus(status); err != nil {
		slog.Error("BLE registration failed", "status", status, "error", err)
		return
	}
	slog.Info("BLE registered successfully", "status", status)
}

//export scanResultCallback
func scanResultCallback(scan_instance C.aceBT_scanInstanceHandle, record *C.aceBT_BeaconScanRecord_t) {
	scanResultFunc(adapter, ScanResult{
		record: record,
		addr:   NewAddressFromAce(record.addr),
		rssi:   record.rssi,
	})
}

//export scanChangeCallback
func scanChangeCallback(scan_instance C.aceBT_scanInstanceHandle, state C.aceBT_beaconScanState_t, interval uint32, window uint32) {
	stateStr := "unknown"
	switch state {
	case C.ACEBT_BEACON_SCAN_FAILED:
		stateStr = "failed"
	case C.ACEBT_BEACON_SCAN_QUEUED:
		stateStr = "queued"
	case C.ACEBT_BEACON_SCAN_STARTED:
		stateStr = "started"
	case C.ACEBT_BEACON_SCAN_PAUSED:
		stateStr = "paused"
	case C.ACEBT_BEACON_SCAN_STOPPED:
		stateStr = "stopped"
	}
	slog.Info("Beacon scan state changed", "state", stateStr, "interval", interval, "window", window)
}

//export onBeaconClientRegistered
func onBeaconClientRegistered(status C.ace_status_t) {
	defer close(beaconRegisterCh)
	if err := errForStatus(status); err != nil {
		slog.Error("Beacon client registration failed", "status", status, "error", err)
		return
	}
	slog.Info("Beacon client registered successfully", "status", status)
}

//export onBleConnectionStateChanged
func onBleConnectionStateChanged(state C.aceBT_bleConnState_t, status C.aceBT_gattStatus_t, conn_handle C.aceBT_bleConnHandle, p_addr *C.aceBT_bdAddr_t) {
	slog.Info("BLE connection state changed",
		"state", state,
		"gatt_status", status,
		"conn_handle", unsafe.Pointer(conn_handle),
		"address", NewAddressFromAce(*p_addr).ToString(),
	)
	if status != C.ACEBT_GATT_STATUS_SUCCESS {
		slog.Error("Failed to connect",
			"address", NewAddressFromAce(*p_addr).ToString(),
			"gatt_status", status,
			"conn_state", state,
			"conn_handle", unsafe.Pointer(conn_handle))
		return
	}
	switch state {
	case C.ACEBT_BLE_STATE_CONNECTED:
		slog.Info("Connected to device", "conn_handle", unsafe.Pointer(conn_handle), "connectCh", connectCh)
		if connectCh != nil {
			connectCh <- ConnHandle{conn_handle}
			close(connectCh)
		} else {
			slog.Warn("connectCh is nil, cannot close channel")
		}
	case C.ACEBT_BLE_STATE_DISCONNECTED:
		slog.Info("Disconnected from device", "conn_handle", unsafe.Pointer(conn_handle), "channel", gattDisconnectCh)
		if gattDisconnectCh != nil {
			close(gattDisconnectCh)
		} else {
			slog.Warn("gattDisconnectCh is nil, cannot close channel")
		}
	}
}

//export onBleGattcServiceDiscovered
func onBleGattcServiceDiscovered(conn_handle C.aceBT_bleConnHandle, status C.ace_status_t) {
	st := StatusFromCode(status)
	defer close(gattcSvcDiscoveryCh)
	slog.Info("GATT service discovered",
		"conn_handle", unsafe.Pointer(conn_handle),
		"status", st,
	)
}

//export onAdapterStateChanged
func onAdapterStateChanged(state C.aceBT_state_t) {
	slog.Info("Adapter state changed",
		"state", state,
	)
}

type BondState int

const (
	BOND_NONE BondState = iota
	BOND_BONDING
	BOND_BONDED
)

//export onBondStateChanged
func onBondStateChanged(status C.aceBT_status_t, p_remote_addr *C.aceBT_bdAddr_t, state C.aceBT_bondState_t) {
	var addrStr string
	if p_remote_addr != nil {
		addrStr = NewAddressFromAce(*p_remote_addr).ToString()
	} else {
		addrStr = "<null>"
	}

	slog.Info("Bond state changed",
		"status", status,
		"remote_address", addrStr,
		"bond_state", state,
	)
	if status == C.ACEBT_STATUS_DONE {
		slog.Info("Already bonded", "address", addrStr, "bond_state", state, "status", status)
	}
	if status != C.ACEBT_STATUS_SUCCESS {
		slog.Error("Non-success status on bond state change?", "address", addrStr, "bond_state", state, "status", status, "status_str", StatusFromCode(status))
		return
	}
	switch state {
	case C.ACEBT_BOND_STATE_BONDED:
		slog.Info("Bonded successfully", "address", addrStr, "bond_state", state, "status", status)
		if pairCh != nil {
			slog.Debug("closing pairCh")
			close(pairCh)
		} else {
			slog.Warn("pairCh is nil, cannot close channel")
		}
	case C.ACEBT_BOND_STATE_NONE:
		slog.Info("Not bonded", "address", addrStr, "bond_state", state, "status", status)
	case C.ACEBT_BOND_STATE_BONDING:
		slog.Info("Bonding in progress", "address", addrStr, "bond_state", state, "status", status)
	default:
		slog.Error("Unknown bond state", "address", addrStr, "bond_state", state, "status", status)
	}
}

// I think this function is unused?
//
//export onBleGattcServiceRegistered
func onBleGattcServiceRegistered(status C.aceBT_status_t) {
	slog.Info("BLE GATT service registered", "status", status)
	if err := errForStatus(status); err != nil {
		slog.Error("Failed to register GATT service", "error", err)
		return
	}
}

//export onBleGattcReadCharacteristics
func onBleGattcReadCharacteristics(conn_handle C.aceBT_bleConnHandle, chars_value C.aceBT_bleGattCharacteristicsValue_t, status C.aceBT_status_t) {
	slog.Info("onBleGattcReadCharacteristics",
		"conn_handle", conn_handle,
		"chars_value", chars_value,
		"status", status,
	)
}

//export onBleGattcWriteCharacteristics
func onBleGattcWriteCharacteristics(conn_handle C.aceBT_bleConnHandle, gatt_characteristics C.aceBT_bleGattCharacteristicsValue_t, status C.aceBT_status_t) {
	defer close(charsWriteCh)
	slog.Info("onBleGattcWriteCharacteristics",
		"conn_handle", unsafe.Pointer(conn_handle),
		"gatt_characteristics", gatt_characteristics,
		"status", status,
	)
	if status == C.ACEBT_STATUS_SUCCESS {
		slog.Debug("characteristic write successful")
	} else {
		slog.Error("characteristic write failed", "status", errForStatus(status))
	}
}

//export onBleGattcNotifyCharacteristics
func onBleGattcNotifyCharacteristics(conn_handle C.aceBT_bleConnHandle, gatt_characteristics C.aceBT_bleGattCharacteristicsValue_t) {
	slog.Info("onBleGattcNotifyCharacteristics",
		"conn_handle", conn_handle,
		"gatt_characteristics", gatt_characteristics,
	)
	rawData := C.getDataFromCharsValue(&gatt_characteristics)
	if rawData.data == nil || rawData.len == 0 {
		slog.Warn("Received notification with no data", "conn_handle", unsafe.Pointer(conn_handle))
		return
	}
	// C.GoBytes makes a copy of the data
	data := C.GoBytes(unsafe.Pointer(rawData.data), C.int(rawData.len))
	go func() {
		notifyCh <- data
	}()
}

//export onBleGattcWriteDescriptor
func onBleGattcWriteDescriptor(conn_handle C.aceBT_bleConnHandle, gatt_characteristics C.aceBT_bleGattCharacteristicsValue_t, status C.aceBT_status_t) {
	defer close(bleWriteDescCh)
	slog.Info("onBleGattcWriteDescriptor",
		"conn_handle", conn_handle,
		"gatt_characteristics", gatt_characteristics,
		"status", status,
	)
}

//export onBleGattcReadDescriptor
func onBleGattcReadDescriptor(conn_handle C.aceBT_bleConnHandle, chars_value C.aceBT_bleGattCharacteristicsValue_t, status C.aceBT_status_t) {
	slog.Info("onBleGattcReadDescriptor",
		"conn_handle", conn_handle,
		"chars_value", chars_value,
		"status", status,
	)
}

//export onBleGattcGetGattDb
func onBleGattcGetGattDb(conn_handle C.aceBT_bleConnHandle, gatt_service *C.aceBT_bleGattsService_t, no_svc C.uint32_t) {
	defer close(gattcDbCh)
	slog.Info("onBleGattcGetGattDb",
		"conn_handle", unsafe.Pointer(conn_handle),
		"gatt_service", unsafe.Pointer(gatt_service),
		"no_svc", int(no_svc),
		"sizeof_svc", unsafe.Sizeof(*gatt_service),
	)

	if gatt_service == nil || no_svc == 0 {
		slog.Error("Received nil GATT service or no services found", "conn_handle", unsafe.Pointer(conn_handle), "no_svc", no_svc)
		return
	}

	var clonedServices *C.aceBT_bleGattsService_t
	if err := errForStatus(C.aceBT_bleCloneGattService(&clonedServices, gatt_service, C.int(no_svc))); err != nil {
		slog.Error("Failed to clone GATT service", "conn_handle", unsafe.Pointer(conn_handle), "error", err)
		return
	}
	gGattService = unsafe.Slice(clonedServices, no_svc)
	for i, svc := range gGattService {
		svcTypeStr := "unknown"
		switch svc.serviceType {
		case C.ACEBT_BLE_GATT_SERVICE_TYPE_PRIMARY:
			svcTypeStr = "primary"
		case C.ACEBT_BLE_GATT_SERVICE_TYPE_SECONDARY:
			svcTypeStr = "secondary"
		case C.ACEBT_BLE_GATT_SERVICE_TYPE_INCLUDED:
			svcTypeStr = "included (characteristic)"
		}
		slog.Info("Gatt Database", "idx", i, "uuid", UUIDFromACEUUID_LE(svc.uuid), "handle", svc.handle, "type", svcTypeStr)
	}

	registerCleanupFunc(func() {
		if clonedServices != nil {
			slog.Info("cleaning up gatt service", "gatt_service", unsafe.Pointer(clonedServices))
			status := C.aceBT_bleCleanupGattService(clonedServices, C.int(no_svc))
			if err := errForStatus(status); err != nil {
				slog.Error("Failed to cleanup GATT service", "conn_handle", unsafe.Pointer(conn_handle), "error", err)
			} else {
				slog.Debug("Cleaned up GATT service", "conn_handle", unsafe.Pointer(conn_handle))
			}
			clonedServices = nil
		}
	})
}

//export onBleGattcExecuteWrite
func onBleGattcExecuteWrite(conn_handle C.aceBT_bleConnHandle, status C.aceBT_status_t) {
	slog.Info("onBleGattcExecuteWrite",
		"conn_handle", conn_handle,
		"status", status,
	)
}

//export onSessionStateChanged
func onSessionStateChanged(session_handle C.aceBT_sessionHandle, state C.aceBT_sessionState_t) {
	defer close(initCh)
	slog.Info("onSessionStateChanged",
		"session_handle", unsafe.Pointer(session_handle),
		"state", state,
	)
}
