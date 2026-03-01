package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"

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
	cmd := exec.Command(selfPath)
	cmd.Env = append(os.Environ(), "KINDLE_KEYMAP_RUN_BLUETOOTH_PAIR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
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
		if err := cmd.Process.Kill(); err != nil {
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

	deviceFoundChan := make(chan address.Address)

	// Try to connect to every device in the config.
	// This way, subsequent runs will auto-connect and pick up the device without needing
	// a fiddly connection here.
	for _, device := range cfg.Devices {
		go func() {
			addr := device.Address()
			addrStr := addr.ToString()
			slog.Info("trying to connect", "device", addrStr)
			if err := adapter.PairIfNeeded(addr); err != nil {
				slog.Error("failed to pair with device", "error", err, "device", addrStr)
			} else {
				deviceFoundChan <- device.Address()
			}
		}()
	}

	// Wait for a device to be found or timeout after 30 seconds
	devicesPaired := 0
	select {
	case addr := <-deviceFoundChan:
		slog.Info("device paired successfully", "address", addr.ToString())
		devicesPaired++
	case <-ctx.Done():
		slog.Info("pairing process context cancelled, stopping scan", "paired_devices", devicesPaired)
		if err := adapter.StopScan(); err != nil {
			slog.Error("failed to stop scan", "error", err)
		}
		return ctx.Err()
	}
	slog.Info("pairing process completed", "paired_devices", devicesPaired)
	return nil
}
