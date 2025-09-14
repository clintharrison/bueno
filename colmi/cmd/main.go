package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/lmittmann/tint"

	"github.com/clintharrison/bueno/ace"
	"github.com/clintharrison/bueno/ace/address"
	"github.com/clintharrison/bueno/colmi"
	"github.com/clintharrison/bueno/xkb"
)

const (
	BLUETOOTH_UID = 1003
	BLUETOOTH_GID = 1003
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// dropPrivileges sets the process's user and group ID to the Bluetooth user and group
// ACE will not allow the process to run as root.
func dropPrivileges() {
	if os.Geteuid() == 0 {
		err := syscall.Setgid(BLUETOOTH_GID)
		if err != nil {
			slog.Error("Failed to set GID", "error", err)
			os.Exit(1)
		}

		err = syscall.Setuid(BLUETOOTH_UID)
		if err != nil {
			slog.Error("Failed to set UID", "error", err)
			os.Exit(1)
		}
	}

	uid := syscall.Getuid()
	gid := syscall.Getgid()
	slog.Info("running as nonroot user", "uid", uid, "gid", gid)
}

// configureLogger sets up the default structured logger to use tint on stderr
func configureLogger() {
	w := os.Stderr

	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.TimeOnly,
		}),
	))
}

func findColmiDevice(adapter ace.Adapter) (address.Address, error) {
	slog.Info("Starting scan for Colmi R02 devices")

	var deviceAddr address.Address
	devicesSeen := make(map[address.Address]bool)
	deviceFoundChan := make(chan struct{})

	err := adapter.Scan(func(adapter ace.Adapter, device ace.ScanResult) {
		if _, ok := devicesSeen[device.Address()]; ok {
			// quietly ignore devices we've already seen
			return
		}
		devicesSeen[device.Address()] = true

		// TODO: Implement device selection logic based on characteristics
		if strings.HasPrefix(device.Name(), "R02") {
			deviceAddr = device.Address()
			slog.Info("found Colmi R02 device, stopping scan", "name", device.Name(), "address", device.Address().ToString(), "rssi", device.RSSI(), "tx_power", device.TxPower(), "adapter", adapter)
			// Stop the scan after finding the device
			if err := adapter.StopScan(); err != nil {
				slog.Error("failed to stop scan", "error", err)
			}
			close(deviceFoundChan)
		} else {
			slog.Info("found device", "name", device.Name(), "address", device.Address().ToString(), "rssi", device.RSSI(), "tx_power", device.TxPower())
		}
	})
	if err != nil {
		slog.Error("Failed to start scan", "error", err)
		return address.Address{}, err
	}

	// Wait for a device to be found or timeout after 10 seconds
	select {
	case <-deviceFoundChan:
		// channel closed and device found
	case <-time.After(10 * time.Second):
		slog.Info("No device found within 10 seconds, stopping scan")
		if err := adapter.StopScan(); err != nil {
			slog.Error("Failed to stop scan", "error", err)
			return address.Address{}, err
		}
		return address.Address{}, errors.New("no Colmi R02 device found")
	}

	return deviceAddr, nil
}

func main() {
	ctx := context.Background()
	configureLogger()

	x11, err := xkb.Open()
	if err != nil {
		slog.Error("Failed to open XKB", "error", err)
		os.Exit(1)
	}
	defer x11.Close()

	dropPrivileges()

	slog.Info("Go Version", "version", runtime.Version(), "hostname", must(os.Hostname()))

	var deviceAddr address.Address
	if len(os.Args) >= 2 {
		deviceAddr, err = address.NewFromString(os.Args[1])
		if err != nil {
			slog.Error("First arg expected to be addr", "error", err)
			os.Exit(1)
		}
	}

	adapter, err := ace.Enable()
	if err != nil {
		slog.Error("Failed to initialize ACE adapter", "error", err)
		os.Exit(1)
	}
	defer adapter.Close()

	if deviceAddr == (address.Address{}) {
		deviceAddr, err = findColmiDevice(adapter)
		if err != nil {
			slog.Error("Failed to start scan", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Connecting to device", "address", deviceAddr.ToString())
	result, err := connectAndFindCharacteristics(ctx, adapter, deviceAddr)
	if err != nil {
		slog.Error("Failed to connect to device", "error", err)
		os.Exit(1)
	}
	defer adapter.Disconnect(result.conn)

	enableGestures(ctx, result, func(data []byte) {
		isPhotoAction := colmi.IsCameraTakePhotoAction(data)
		slog.Info("Received notification data", "data", data, "is_camera_action", isPhotoAction)
		if isPhotoAction {
			x11.KeyPress(xkb.XKPageDown)
		} else {
			slog.Debug("unrecognized message received", "data", data)
		}
	})
}

type ConnectResult struct {
	conn      ace.ConnHandle
	readChar  *ace.DeviceCharacteristic
	writeChar *ace.DeviceCharacteristic
}

func connectAndFindCharacteristics(_ context.Context, adapter ace.Adapter, addr address.Address) (ConnectResult, error) {
	slog.Info("Connecting to device", "address", addr.ToString())
	conn, err := adapter.Connect(addr)
	if err != nil {
		return ConnectResult{}, err
	}

	services, err := adapter.GetServices(conn)
	if err != nil {
		slog.Error("Failed to get services", "error", err)
		return ConnectResult{}, err
	}
	var commandReadChr *ace.DeviceCharacteristic
	var commandWriteChr *ace.DeviceCharacteristic

	for _, svc := range services {
		chars, _ := adapter.GetCharacteristics(&svc)
		for _, char := range chars {
			switch char.UUID {
			case colmi.R02_COMMANDS_READ_UUID:
				commandReadChr = &char
			case colmi.R02_COMMANDS_WRITE_UUID:
				commandWriteChr = &char
			}
		}
	}

	slog.Debug("found command chars", "read", fmt.Sprintf("%p", commandReadChr), "write", fmt.Sprintf("%p", commandWriteChr))
	return ConnectResult{
		conn:      conn,
		readChar:  commandReadChr,
		writeChar: commandWriteChr,
	}, nil
}

func enableGestures(ctx context.Context, cr ConnectResult, f func(data []byte)) error {
	slog.Info("Setting Notify on the read characteristic")
	notifyCh, err := cr.readChar.SetNotify(cr.conn)
	if err != nil {
		return err
	}
	go func() {
		for data := range notifyCh {
			f(data)
		}
	}()

	// Turn on the "camera" gesture -- this causes a "ACTION_TAKE_PHOTO" notification on a hand shake or something.
	enablePacket, _ := colmi.MakeCameraPacket(colmi.ACTION_ENABLE_CAMERA_GESTURE)
	err = cr.writeChar.Write(cr.conn, enablePacket)
	if err != nil {
		return err
	}

	// Blink LED twice -- this lets the user know we're listening for the gesture
	packet, _ := colmi.MakePacket(0x10, []byte{})
	err = cr.writeChar.Write(cr.conn, packet)
	if err != nil {
		return err
	}

	slog.Info("Waiting for gestures!")

	select {
	case <-ctx.Done():
		slog.Info("Context canceled")
	case <-time.After(10 * time.Second):
		slog.Info("10 seconds passed")
	}

	disablePacket, _ := colmi.MakeCameraPacket(colmi.ACTION_DISABLE_CAMERA_GESTURE)
	err = cr.writeChar.Write(cr.conn, disablePacket)
	if err != nil {
		return err
	}
	return nil
}
