// Package watcher is responsible for watching evdev events for new input devices, and performing
// actions when those devices send key presses.
package watcher

import (
	"context"
	"errors"
	"log/slog"
	"syscall"
	"time"

	"github.com/clintharrison/bueno/kindle-keymap/actions"
	"github.com/clintharrison/bueno/kindle-keymap/config"
	"github.com/clintharrison/bueno/kindle-keymap/lipcaction"
	"github.com/clintharrison/bueno/quietly"
	"github.com/clintharrison/bueno/xkb"
	"github.com/holoplot/go-evdev"
)

type Watcher struct {
	x11        *xkb.X11
	brightness *lipcaction.BrightnessAction
	rotation   *lipcaction.RotationAction
}

const eventHandlerTimeout = 5 * time.Second

func New(x11 *xkb.X11, brightness *lipcaction.BrightnessAction, rotation *lipcaction.RotationAction) *Watcher {
	return &Watcher{
		x11:        x11,
		brightness: brightness,
		rotation:   rotation,
	}
}

func (w *Watcher) Watch(ctx context.Context, dev *evdev.InputDevice, cfg *config.Device) {
	defer quietly.Close(dev)
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

var errUnknownEvent = errors.New("unknown event")

func syntheticKeyEventForAbsEvent(ev *evdev.InputEvent, absInfos map[evdev.EvCode]evdev.AbsInfo) (*evdev.InputEvent, error) {
	if ev == nil || ev.Type != evdev.EV_ABS || absInfos == nil {
		return nil, errUnknownEvent
	}
	absInfo, ok := absInfos[ev.Code]
	if !ok {
		return nil, errUnknownEvent
	}
	if absInfo.Minimum == 0 && absInfo.Maximum == 0 {
		// this axis doesn't have a range, so we can't interpret it
		return nil, errUnknownEvent
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
	return nil, errUnknownEvent
}

func (w *Watcher) handleEvent(ctx context.Context, ev *evdev.InputEvent, cfg *config.Device, absInfos map[evdev.EvCode]evdev.AbsInfo) {
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
		case actions.NextPage:
			err := w.x11.KeyPress(xkb.XKPageDown)
			if err != nil {
				slog.Error("KeyPress(XKPageDown)", "error", err)
			}
		case actions.PrevPage:
			err := w.x11.KeyPress(xkb.XKPageUp)
			if err != nil {
				slog.Error("KeyPress(XKPageUp)", "error", err)
			}
		case actions.BrightnessUp:
			err := w.brightness.IncreaseBrightness(ctx)
			if err != nil {
				slog.Error("IncreaseBrightness()", "error", err)
			}
		case actions.BrightnessDown:
			err := w.brightness.DecreaseBrightness(ctx)
			if err != nil {
				slog.Error("DecreaseBrightness()", "error", err)
			}
		case actions.WarmthUp:
			err := w.brightness.IncreaseWarmth(ctx)
			if err != nil {
				slog.Error("IncreaseWarmth()", "error", err)
			}
		case actions.WarmthDown:
			err := w.brightness.DecreaseWarmth(ctx)
			if err != nil {
				slog.Error("DecreaseWarmth()", "error", err)
			}
		case actions.RotateCW:
			err := w.rotation.Rotate(ctx, lipcaction.RotationClockwise)
			if err != nil {
				slog.Error("Rotate()", "error", err)
			}
		case actions.RotateCCW:
			err := w.rotation.Rotate(ctx, lipcaction.RotationCounterclockwise)
			if err != nil {
				slog.Error("Rotate()", "error", err)
			}
		case actions.OrientLockUp:
			err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationPortrait)
			if err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case actions.OrientLockDown:
			err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationPortraitInverted)
			if err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case actions.OrientLockLeft:
			err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationLandscapeLeft)
			if err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		case actions.OrientLockRight:
			err := w.rotation.SetOrientationLock(ctx, lipcaction.OrientationLandscapeRight)
			if err != nil {
				slog.Error("SetOrientationLock()", "error", err)
			}
		default:
			// ignore unmapped keys
		}
	}
}
