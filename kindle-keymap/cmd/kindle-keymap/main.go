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
	ActionNextPage        string = "next_page"
	ActionPrevPage        string = "prev_page"
	ActionBrightnessUp    string = "brightness_up"
	ActionBrightnessDown  string = "brightness_down"
	ActionWarmthUp        string = "warmth_up"
	ActionWarmthDown      string = "warmth_down"
	ActionRotateCW        string = "rotate_cw"
	ActionRotateCCW       string = "rotate_ccw"
	ActionOrientLockUp    string = "orientation_lock_up"
	ActionOrientLockDown  string = "orientation_lock_down"
	ActionOrientLockLeft  string = "orientation_lock_left"
	ActionOrientLockRight string = "orientation_lock_right"
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
		if dev, matchedCfg, err := findExistingDevice(cfg.Devices); err == nil {
			if devName, err := dev.Name(); err == nil {
				slog.Info("using existing input device", "devname", devName, "path", dev.Path(), "pattern", matchedCfg.NamePattern().String())
				deviceCh <- dev
			}
		}
	}()

	client, err := lipcaction.NewLipcClient()
	if err != nil {
		slog.Error("lipcaction.NewLipcClient()", "error", err)
		os.Exit(1)
	}
	defer client.Close()
	brightness := lipcaction.NewBrightnessAction(client)
	rotation := lipcaction.NewRotationAction(client)

	w := watcher{x11, brightness, rotation}

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
			n, err := dev.Name()
			if err != nil {
				// if we got here, presumably the device's name matched.
				// but it doesn't hurt to check again
				slog.Error("dev.Name()", "error", err)
				dev.Close()
				continue
			}
			cfg := cfg.MatchingDevice(n)
			if cfg == nil {
				slog.Error("no config found matching device name", "devname", n, "path", dev.Path())
				dev.Close()
				continue
			}
			go w.watch(ctx, dev, cfg)
		}
	}
}

type watcher struct {
	x11        *xkb.X11
	brightness *lipcaction.BrightnessAction
	rotation   *lipcaction.RotationAction
}

const eventHandlerTimeout = 5 * time.Second

func (w *watcher) watch(ctx context.Context, dev *evdev.InputDevice, cfg *config.Device) {
	defer dev.Close()
	devName, _ := dev.Name()
	slog.Info("watching device for key events", "devname", devName, "path", dev.Path())

	absInfos, err := dev.AbsInfos()
	if err != nil {
		slog.Warn("dev.AbsInfos() failed", "devname", devName, "path", dev.Path(), "error", err)
	}

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
			w.handleEvent(eventCtx, ev, cfg, absInfos)
		}
	}
}

func syntheticKeyEventForAbsEvent(ev *evdev.InputEvent, absInfos map[evdev.EvCode]evdev.AbsInfo) (*evdev.InputEvent, error) {
	if ev == nil || ev.Type != evdev.EV_ABS || absInfos == nil {
		return nil, nil
	}
	absInfo, ok := absInfos[ev.Code]
	if !ok {
		return nil, nil
	}
	if absInfo.Minimum == 0 && absInfo.Maximum == 0 {
		// this axis doesn't have a range, so we can't interpret it
		return nil, nil
	}

	newEvent := &evdev.InputEvent{
		Time: ev.Time,
		Type: evdev.EV_KEY,
		// treat it as a key press, since we don't get key release events for ABS
		Value: 1,
		Code:  0,
	}
	// D-pad on 8bitdo shows ABS_X 127 at rest, 0 when left pressed, 255 when right pressed
	// and ABS_Y 127 at rest, 0 when up pressed, 255 when down pressed
	// Since we only get events when the value changes, we
	switch ev.Code {
	case evdev.ABS_X:
		switch ev.Value {
		case absInfo.Minimum:
			newEvent.Code = evdev.BTN_DPAD_LEFT
		case absInfo.Maximum:
			newEvent.Code = evdev.BTN_DPAD_RIGHT
		}
	case evdev.ABS_Y:
		switch ev.Value {
		case absInfo.Minimum:
			newEvent.Code = evdev.BTN_DPAD_UP
		case absInfo.Maximum:
			newEvent.Code = evdev.BTN_DPAD_DOWN
		}
	}

	if newEvent.Code != 0 {
		return newEvent, nil
	}

	// if we didn't remap the event, that's not an error
	return nil, nil
}

func (w *watcher) handleEvent(ctx context.Context, ev *evdev.InputEvent, cfg *config.Device, absInfos map[evdev.EvCode]evdev.AbsInfo) {
	// TODO: handle watcher vs device vs event state better
	if ev == nil {
		return
	}

	// 8bitdo gamepad sends EV_ABS events for the D-pad, so we'll remap those to equivalent KEY_* events
	// we don't keep state, so keys are never released
	syntheticEv, err := syntheticKeyEventForAbsEvent(ev, absInfos)
	if err != nil {
		slog.Error("syntheticKeyEventForAbsEvent()", "error", err)
		return
	} else if syntheticEv != nil {
		ev = syntheticEv
	}

	if ev.Type == evdev.EV_KEY && ev.Value == 1 { // Key press event
		keyName := ev.CodeName()
		mappedAction := cfg.BindingForKey(keyName)
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
		case ActionRotateCW:
			if err := w.rotation.Rotate(ctx, lipcaction.RotationClockwise); err != nil {
				slog.Error("Rotate()", "error", err)
			}
		case ActionRotateCCW:
			if err := w.rotation.Rotate(ctx, lipcaction.RotationCounterclockwise); err != nil {
				slog.Error("Rotate()", "error", err)
			}
		case ActionOrientLockUp:
			if err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationPortrait); err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case ActionOrientLockDown:
			if err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationPortraitInverted); err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case ActionOrientLockLeft:
			if err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationLandscapeLeft); err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case ActionOrientLockRight:
			if err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationLandscapeRight); err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		default:
			// ignore unmapped keys
		}
	}
}
