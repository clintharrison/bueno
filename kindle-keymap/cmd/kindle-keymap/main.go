package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/holoplot/go-evdev"
	"github.com/pilebones/go-udev/netlink"

	"github.com/clintharrison/bueno/core/log"
	"github.com/clintharrison/bueno/kindle-keymap/config"
	"github.com/clintharrison/bueno/kindle-keymap/install"
	"github.com/clintharrison/bueno/kindle-keymap/lipcaction"
	"github.com/clintharrison/bueno/kindle-keymap/watcher"
	"github.com/clintharrison/bueno/udev"
	"github.com/clintharrison/bueno/xkb"
)

func findExistingDevice(devices []config.Device) (*evdev.InputDevice, *config.Device, error) {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		slog.Warn("evdev.ListDevicePaths()", "error", err)
		return nil, nil, err
	}

	var cfgDev *config.Device
	devPath := ""
	for _, d := range devicePaths {
		for _, cd := range devices {
			if cd.NamePattern().MatchString(d.Name) {
				cfgDev = &cd
				break
			}
		}
		if cfgDev != nil {
			slog.Info("found device", "name", d.Name, "path", d.Path)
			devPath = d.Path
			break
		}
	}
	if devPath != "" {
		dev, err := evdev.Open(devPath)
		if err != nil {
			return nil, cfgDev, err
		}
		return dev, cfgDev, nil
	}

	return nil, nil, fmt.Errorf("no input device found matching given patterns")
}

func startDeviceWatcher(ctx context.Context, cfg *config.Config) chan *evdev.InputDevice {
	// here we set up a udev watcher to look for _new_ input devices matching the given pattern
	// we also check any existing devices, and start watching them too
	deviceCh := make(chan *evdev.InputDevice)
	patterns := make([]*regexp.Regexp, 0, len(cfg.Devices))
	for _, d := range cfg.Devices {
		patterns = append(patterns, d.NamePattern())
	}
	idw := &udev.InputDeviceWatcher{
		Patterns: patterns,
		AddFunc: func(dev *evdev.InputDevice) {
			if devName, err := dev.Name(); err == nil {
				slog.Info("new input device detected", "devname", devName, "path", dev.Path())
				deviceCh <- dev
			}
		},
		// TODO: consider proactively removing watches on devices that are removed?
		// It's likely their goroutines are already in a blocking read though, so there's
		// not a ton we can do there. Once that errors, the goroutine will exit anyway.
		RemoveFunc: func(uevent netlink.UEvent) {
			subsystem := uevent.Env["SUBSYSTEM"]
			devname := uevent.Env["DEVNAME"]
			if subsystem != "input" && subsystem != "hid" {
				return
			}
			if devname != "" {
				slog.Debug("input device removed", "devname", devname, "env", uevent.Env)
			} else {
				slog.Debug("unknown input device removed", "env", uevent.Env)
			}
		},
	}
	go idw.Start(ctx)
	go func() {
	}()
	return deviceCh
}

func runKeymapLoop(ctx context.Context, cfg *config.Config) error {
	err := install.MaybeInstallUdevRule(ctx)
	if err != nil {
		slog.Error("maybeInstallUdev()", "error", err)
		os.Exit(1)
	}

	x11, err := xkb.Open()
	if err != nil {
		slog.Error("xkb.Open()", "error", err)
		return err
	}
	defer x11.Close()

	client, err := lipcaction.NewLipcClient()
	if err != nil {
		slog.Error("lipcaction.NewLipcClient()", "error", err)
		return err
	}
	defer client.Close()
	brightness := lipcaction.NewBrightnessAction(client)
	rotation := lipcaction.NewRotationAction(client)

	w := watcher.New(x11, brightness, rotation)

	// kick off the background device watcher
	deviceCh := startDeviceWatcher(ctx, cfg)
	// "catch up" on any existing devices that match the pattern
	deviceFound := false
	if dev, matchedCfg, err := findExistingDevice(cfg.Devices); err == nil {
		if devName, err := dev.Name(); err == nil {
			slog.Info("using existing input device", "devname", devName, "path", dev.Path(), "pattern", matchedCfg.NamePattern().String())
			deviceFound = true
			go func() { deviceCh <- dev }()
		}
	}

	pairCancelCtx, pairCancel := context.WithCancel(ctx)
	defer pairCancel()
	// if we didn't find any existing devices, kick off the process to run pairing
	if !deviceFound {
		slog.Info("no existing devices found, starting Bluetooth pairing process")
		go runSelfAsPairingProcess(pairCancelCtx)
	}

	// now we wait for devices to show up, and then watch their events in a separate goroutine
	for {
		slog.Debug("waiting for device...")
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return nil
		case dev := <-deviceCh:
			slog.Debug("got new device")
			n, err := dev.Name()
			if err != nil {
				// if we got here, presumably the device's name matched.
				// but it doesn't hurt to check again
				slog.Error("dev.Name()", "error", err)
				dev.Close()
				continue
			}
			cfg := cfg.MergedMatchingDevice(n)
			slog.Debug("using config", "cfg", cfg.Dump())
			if cfg == nil {
				slog.Error("no config found matching device name", "devname", n, "path", dev.Path())
				dev.Close()
				continue
			}
			// now that we found a device, cancel the process looking for devices to pair
			pairCancel()
			go w.Watch(ctx, dev, cfg)
		}
	}
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.ConfigureInteractiveLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config.Load()", "error", err)
		os.Exit(1)
	}

	// If this env var is set, we're in the subprocess expected to scan, pair, and exit.
	if os.Getenv("KINDLE_KEYMAP_RUN_BLUETOOTH_PAIR") == "1" {
		if err := runPairProcessInner(ctx, cfg); err != nil {
			slog.Error("runPairProcessInner()", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := runKeymapLoop(ctx, cfg); err != nil {
		slog.Error("error in one of the main goroutines", "error", err)
		os.Exit(1)
	}
}
