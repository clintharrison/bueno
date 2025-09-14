package ace

//#include "ace.go.h"
import "C"

import (
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/clintharrison/bueno/ace/address"
)

func must[T any](v T, err error) T {
	if err != nil {
		slog.Error("Must failed", "error", err)
		panic(err)
	}
	return v
}

func NewAddressFromAce(addr C.aceBT_bdAddr_t) address.Address {
	bs := addr.address
	return address.Address{Bytes: [6]uint8{uint8(bs[0]), uint8(bs[1]), uint8(bs[2]), uint8(bs[3]), uint8(bs[4]), uint8(bs[5])}}
}

func AddressToAce(a address.Address) *C.aceBT_bdAddr_t {
	addr := &C.aceBT_bdAddr_t{}
	for i := 0; i < 6; i++ {
		addr.address[i] = C.uint8_t(a.Bytes[i])
	}
	return addr
}

type ScanResult struct {
	// The raw record, used by aceBT_scanRecordExtract* functions
	record *C.aceBT_BeaconScanRecord_t
	// Device address
	addr address.Address
	// RSSI of the remote advertisement
	rssi C.int
}

func (sr *ScanResult) Name() string {
	var name C.aceBT_bdName_t
	nameLen := C.aceBT_scanRecordExtractName(sr.record, &name)
	if nameLen > 0 {
		nameBytes := C.GoBytes(unsafe.Pointer(&name.name[0]), C.int(nameLen))
		return string(nameBytes)
	}
	return "<unknown>"
}

func (sr *ScanResult) Address() address.Address {
	return sr.addr
}

func (sr *ScanResult) RSSI() int {
	return int(sr.rssi)
}

func (sr *ScanResult) TxPower() int {
	var txPower C.int
	len := C.aceBT_scanRecordExtractTxPower(sr.record, &txPower)
	if len == 1 {
		return int(txPower)
	}
	return 0
}

func errForStatus(status C.ace_status_t) error {
	switch status {
	case C.ACE_STATUS_OK:
		return nil
	case C.ACEBT_STATUS_NOMEM:
		return errors.New("ACE out of memory")
	case C.ACEBT_STATUS_BUSY:
		return errors.New("ACE is busy connecting another device")
	case C.ACEBT_STATUS_PARM_INVALID:
		return errors.New("ACE request contains invalid parameters")
	case C.ACEBT_STATUS_NOT_READY:
		return errors.New("ACE server not ready")
	case C.ACEBT_STATUS_FAIL:
		return errors.New("ACE failed")
	default:
		return fmt.Errorf("ACE unknown error: %s", StatusFromCode(status))
	}
}
