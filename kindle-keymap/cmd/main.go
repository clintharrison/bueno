package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/holoplot/go-evdev"
	"github.com/lmittmann/tint"
	"github.com/pilebones/go-udev/netlink"

	"github.com/clintharrison/bueno/kindle-keymap/config"
	"github.com/clintharrison/bueno/kindle-keymap/udev"
	"github.com/clintharrison/bueno/kindle-keymap/xkb"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// configureLogger sets up the default structured logger to use tint on stderr
func configureLogger() {
	w := os.Stderr

	defaultLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		defaultLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      defaultLevel,
			TimeFormat: time.TimeOnly,
		}),
	))
}

func findExistingDevice(pattern *regexp.Regexp) (*evdev.InputDevice, error) {
	devicePaths, err := evdev.ListDevicePaths()
	if err != nil {
		slog.Warn("evdev.ListDevicePaths()", "error", err)
		return nil, err
	}

	devPath := ""
	for _, d := range devicePaths {
		if pattern.MatchString(d.Name) {
			slog.Info("found device", "name", d.Name, "path", d.Path)
			devPath = d.Path
			break
		}
	}
	if devPath != "" {
		dev, err := evdev.Open(devPath)
		if err != nil {
			return nil, err
		}
		return dev, nil
	}

	return nil, fmt.Errorf("no input device found matching pattern %q", pattern.String())
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	configureLogger()

	x11, err := xkb.Open()
	if err != nil {
		slog.Error("xkb.Open()", "error", err)
		os.Exit(1)
	}
	defer x11.Close()

	cfg := must(config.Load())

	// here we set up a udev watcher to look for _new_ input devices matching the given pattern
	// we also check any existing devices, and start watching them too
	deviceCh := make(chan *evdev.InputDevice)
	idw := &udev.InputDeviceWatcher{
		Pattern: cfg.DeviceNamePattern,
		AddFunc: func(dev *evdev.InputDevice) {
			if devName, err := dev.Name(); err == nil {
				slog.Info("new input device detected", "devname", devName, "path", dev.Path())
				deviceCh <- dev
			}
		},
		// TODO: consider proactively removing watches on devices that are removed?
		// It's likely their goroutines are already in a blocking read though, so there's
		// not a ton we can do there.
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
		if dev, err := findExistingDevice(cfg.DeviceNamePattern); err == nil {
			if devName, err := dev.Name(); err == nil {
				slog.Info("using existing input device", "devname", devName, "path", dev.Path())
				deviceCh <- dev

			}
		}
	}()

	// now we wait for devices to show up, and then watch their events in a separate goroutine
	for {
		slog.Debug("waiting for device...")
		select {
		case <-ctx.Done():
			slog.Info("bye bye!")
			return
		case dev := <-deviceCh:
			slog.Debug("got new device")
			go watchDevice(ctx, dev, x11, cfg)
		}
	}
}

func watchDevice(ctx context.Context, dev *evdev.InputDevice, x11 *xkb.X11, cfg *config.Config) {
	defer dev.Close()
	devName, _ := dev.Name()
	slog.Info("watching device for key events", "devname", devName, "path", dev.Path())

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping watch on device", "devname", devName, "path", dev.Path())
			return
		default:
			ev, err := dev.ReadOne()
			if err != nil {
				// it's normal for bluetooth devices to disconnect - don't log it as an error
				if errors.Is(err, syscall.ENODEV) {
					slog.Info("assuming device is disconnected - stopping watch", "devname", devName, "path", dev.Path(), "error", err)
					return
				}
				slog.Error("unexpected read error on device, stopping watch", "devname", devName, "path", dev.Path(), "error", err)
				return
			}
			handleEvent(ev, x11, cfg)
		}
	}
}

func handleEvent(ev *evdev.InputEvent, x11 *xkb.X11, cfg *config.Config) {
	if ev == nil {
		return
	}
	if ev.Type == evdev.EV_KEY && ev.Value == 1 { // Key press event
		keyName := evdev.CodeName(ev.Type, ev.Code)
		mappedAction := cfg.BindingForKey(keyName)
		if mappedAction == "" {
			mappedAction = "<unmapped>"
		}
		slog.Info("key pressed", "code", ev.Code, "name", keyName, "mapped_action", mappedAction)
		switch mappedAction {
		case "next_page":
			x11.KeyPress(xkb.XKPageDown)
		case "prev_page":
			x11.KeyPress(xkb.XKPageUp)
		default:
			// ignore unmapped keys
		}
	}
}
