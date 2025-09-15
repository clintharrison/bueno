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
	"github.com/pilebones/go-udev/netlink"

	"github.com/clintharrison/bueno/core/log"
	"github.com/clintharrison/bueno/kindle-keymap/config"
	"github.com/clintharrison/bueno/kindle-keymap/lipcaction"
	"github.com/clintharrison/bueno/udev"
	"github.com/clintharrison/bueno/xkb"
)

const (
	ActionNextPage       string = "next_page"
	ActionPrevPage       string = "prev_page"
	ActionBrightnessUp   string = "brightness_up"
	ActionBrightnessDown string = "brightness_down"
	ActionWarmthUp       string = "warmth_up"
	ActionWarmthDown     string = "warmth_down"
)

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

	log.ConfigureInteractiveLogger()

	x11, err := xkb.Open()
	if err != nil {
		slog.Error("xkb.Open()", "error", err)
		os.Exit(1)
	}
	defer x11.Close()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config.Load()", "error", err)
		os.Exit(1)
	}

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
		if dev, err := findExistingDevice(cfg.DeviceNamePattern); err == nil {
			if devName, err := dev.Name(); err == nil {
				slog.Info("using existing input device", "devname", devName, "path", dev.Path())
				deviceCh <- dev

			}
		}
	}()

	client, err := lipcaction.NewLipcClient()
	if err != nil {
		slog.Error("lipcaction.NewLipcClient()", "error", err)
		os.Exit(1)
	}
	// defer client.Close()
	brightness := lipcaction.NewBrightnessAction(client)

	w := watcher{x11, cfg, brightness}

	// now we wait for devices to show up, and then watch their events in a separate goroutine
	// (a device may "show up" from the existing devices check above, or from udev)
	for {
		slog.Debug("waiting for device...")
		select {
		case <-ctx.Done():
			slog.Info("bye bye!")
			return
		case dev := <-deviceCh:
			slog.Debug("got new device")
			go w.watch(ctx, dev)
		}
	}
}

type watcher struct {
	x11        *xkb.X11
	cfg        *config.Config
	brightness *lipcaction.BrightnessAction
}

const eventHandlerTimeout = 5 * time.Second

func (w *watcher) watch(ctx context.Context, dev *evdev.InputDevice) {
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
			eventCtx, cancel := context.WithTimeout(ctx, eventHandlerTimeout)
			defer cancel()
			w.handleEvent(eventCtx, ev)
		}
	}
}

func (w *watcher) handleEvent(ctx context.Context, ev *evdev.InputEvent) {
	if ev == nil {
		return
	}
	if ev.Type == evdev.EV_KEY && ev.Value == 1 { // Key press event
		keyName := evdev.CodeName(ev.Type, ev.Code)
		mappedAction := w.cfg.BindingForKey(keyName)
		if mappedAction == "" {
			mappedAction = "<unmapped>"
		}
		slog.Info("key pressed", "code", ev.Code, "name", keyName, "mapped_action", mappedAction)
		switch mappedAction {
		case ActionNextPage:
			w.x11.KeyPress(xkb.XKPageDown)
		case ActionPrevPage:
			w.x11.KeyPress(xkb.XKPageUp)
		case ActionBrightnessUp:
			if err := w.brightness.IncreaseBrightness(ctx); err != nil {
				slog.Error("IncreaseBrightness()", "error", err)
			}
		case ActionBrightnessDown:
			if err := w.brightness.DecreaseBrightness(ctx); err != nil {
				slog.Error("DecreaseBrightness()", "error", err)
			}
		case ActionWarmthUp:
			if err := w.brightness.IncreaseWarmth(ctx); err != nil {
				slog.Error("IncreaseWarmth()", "error", err)
			}
		case ActionWarmthDown:
			if err := w.brightness.DecreaseWarmth(ctx); err != nil {
				slog.Error("DecreaseWarmth()", "error", err)
			}
		default:
			// ignore unmapped keys
		}
	}
}
