package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/clintharrison/bueno/ace"
	"github.com/clintharrison/bueno/ace/address"
	"github.com/clintharrison/bueno/kindle-keymap/config"
)

func runSelfAsPairingProcess(ctx context.Context) error {
	selfPath, err := os.Readlink("/proc/self/exe")
	if err != nil {
		slog.Error("os.Readlink()", "error", err)
		return err
	}
	cmd := exec.CommandContext(ctx, selfPath)
	cmd.Env = append(os.Environ(), "KINDLE_KEYMAP_RUN_BLUETOOTH_PAIR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		slog.Error("cmd.Start()", "error", err)
		return err
	}
	processDone := make(chan error, 1)

	go func() {
		err := cmd.Wait()
		processDone <- err
	}()

	select {
	case <-ctx.Done():
		slog.Info("pairing process context cancelled, killing pairing process", "pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			slog.Error("failed to kill pairing process", "error", err)
		}
	case err := <-processDone:
		if err != nil {
			slog.Error("pairing process exited with error", "error", err)
			return err
		}
		slog.Info("pairing process exited successfully")
	}
	return nil
}

func runPairProcessInner(ctx context.Context, cfg *config.Config) error {
	slog.Debug("starting Bluetooth pairing loop", "devices", len(cfg.Devices))

	ace.DropPrivileges()
	adapter, err := ace.Enable()
	if err != nil {
		slog.Error("ace.Enable()", "error", err)
		return err
	}
	defer adapter.Close()

	toBePaired := make(map[address.Address]struct{})
	for _, device := range cfg.Devices {
		toBePaired[device.Address()] = struct{}{}
	}

	deviceFoundChan := make(chan struct {
		address.Address
		error
	})

	// Try to connect to every device in the config.
	// This way, subsequent runs will auto-connect and pick up the device without needing
	// a fiddly connection here.
	// Only one pairing attempt can occur at a time, so this is done serially.
	go func() {
		for _, device := range cfg.Devices {
			addr := device.Address()
			addrStr := addr.ToString()
			slog.Info("trying to connect", "device", addrStr)
			err := adapter.PairIfNeeded(addr)
			if err != nil {
				slog.Warn("config.yaml device failed to pair", "error", err, "device", addrStr)
			}
			deviceFoundChan <- struct {
				address.Address
				error
			}{addr, err}
		}
	}()

	// Wait for a device to be found or time out after a while.
	// Since devices are paired one-by-one, we wait depending on the number of devices
	// It seems to take ~5-15 seconds to time out, so this number is kind of a random guess.
	timeout := time.Duration(len(cfg.Devices)*12) * time.Second
	devicesPaired := 0
	for {
		select {
		case pair := <-deviceFoundChan:
			addr, err := pair.Address, pair.error
			devicesPaired++
			if _, ok := toBePaired[addr]; !ok {
				slog.Info("unexpected device paired successfully", "address", addr.ToString())
			} else {
				if err != nil {
					slog.Warn("device failed to pair", "error", err, "address", addr.ToString())
				} else {
					slog.Info("device paired successfully", "address", addr.ToString())
				}
				// whether we succeeded or not, we should stop waiting...
				delete(toBePaired, addr)
			}
			if len(toBePaired) == 0 {
				// we didn't necessarily pair all devices, but we made an attempt at all known ones.
				// exit successfully and the user can retry manually
				return nil
			}
			slog.Info("waiting for remaining devices to pair", "remaining_devices", len(toBePaired))
		case <-time.After(timeout):
			slog.Info("timed out waiting for devices to pair", "paired_devices", devicesPaired)
			return errors.New("timed out waiting for devices to pair")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
