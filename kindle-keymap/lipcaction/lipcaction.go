// Package lipcaction holds key map actions that use LIPC to run.
package lipcaction

import (
	"context"
	"errors"
	"log/slog"

	"github.com/clintharrison/bueno/lipc"
	"github.com/godbus/dbus/v5"
)

type LipcClient struct {
	conn *dbus.Conn
}

func NewLipcClient() (*LipcClient, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	return &LipcClient{conn: conn}, nil
}

func (c *LipcClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type BrightnessAction struct {
	client       *LipcClient
	maxIntensity int32
}

func NewBrightnessAction(client *LipcClient) *BrightnessAction {
	ba := &BrightnessAction{client: client}
	return ba
}

func (a *BrightnessAction) InitRanges() error {
	val, err := lipc.LipcGetProperty[int32](a.client.conn.Context(), a.client.conn, "com.lab126.powerd", "flMaxIntensity")
	if err != nil {
		return err
	}
	a.maxIntensity = val
	return nil
}

func (a *BrightnessAction) adjust(ctx context.Context, prop string, delta int32) error {
	if a.maxIntensity == 0 {
		if err := a.InitRanges(); err != nil {
			slog.Error("InitRanges()", "error", err)
			return err
		}
	}
	curr, err := lipc.LipcGetProperty[int32](ctx, a.client.conn, "com.lab126.powerd", prop)
	if err != nil {
		return err
	}
	newVal := curr + delta
	if newVal < 0 {
		newVal = 0
	} else if newVal > a.maxIntensity {
		newVal = a.maxIntensity
	}
	slog.Debug("adjust()", "prop", prop, "curr", curr, "delta", delta, "new", newVal, "max", a.maxIntensity)
	return lipc.LipcSetProperty(ctx, a.client.conn, "com.lab126.powerd", prop, newVal)
}

func (a *BrightnessAction) DecreaseBrightness(ctx context.Context) error {
	return a.adjust(ctx, "flIntensity", -1)
}

func (a *BrightnessAction) IncreaseBrightness(ctx context.Context) error {
	return a.adjust(ctx, "flIntensity", 1)
}

func (a *BrightnessAction) DecreaseWarmth(ctx context.Context) error {
	return a.adjust(ctx, "currentAmberLevel", -1)
}

func (a *BrightnessAction) IncreaseWarmth(ctx context.Context) error {
	return a.adjust(ctx, "currentAmberLevel", 1)
}

type RotationAction struct {
	client *LipcClient
}

type Orientation string

const (
	OrientationUnlocked         Orientation = ""
	OrientationPortrait         Orientation = "U"
	OrientationPortraitInverted Orientation = "D"
	OrientationLandscapeLeft    Orientation = "L"
	OrientationLandscapeRight   Orientation = "R"
)

func NewRotationAction(client *LipcClient) *RotationAction {
	return &RotationAction{client: client}
}

func orientationFromString(s string) (o Orientation, err error) {
	switch s {
	case "":
		o = OrientationUnlocked
	case "U":
		o = OrientationPortrait
	case "D":
		o = OrientationPortraitInverted
	case "L":
		o = OrientationLandscapeLeft
	case "R":
		o = OrientationLandscapeRight
	default:
		err = errors.New("unknown orientation: " + s)
	}
	return
}
func (a *RotationAction) GetOrientationLock(ctx context.Context) (Orientation, error) {
	var err error
	var o string
	if o, err = lipc.LipcGetProperty[string](ctx, a.client.conn, "com.lab126.winmgr", "orientationLock"); err != nil {
		return OrientationUnlocked, err
	}
	orientation, err := orientationFromString(o)
	if err != nil {
		return OrientationUnlocked, err
	}
	return orientation, nil
}

func (a *RotationAction) SetOrientationLock(ctx context.Context, o Orientation) error {
	oo := string(o)
	return lipc.LipcSetProperty(ctx, a.client.conn, "com.lab126.winmgr", "orientationLock", oo)
}

type RotationDirection bool

const (
	RotationClockwise        RotationDirection = true
	RotationCounterclockwise RotationDirection = false
)

func (a *RotationAction) Rotate(ctx context.Context, direction RotationDirection) error {
	currentOrientation, err := a.GetOrientationLock(ctx)
	if err != nil {
		return err
	}

	// if it's unknown/unlocked, it's treated as normal portrait
	// TODO: figure out what this does on a device with accelerometer
	current := OrientationPortrait
	if currentOrientation != OrientationUnlocked {
		current = currentOrientation
	}

	var nextOrientation Orientation
	if direction == RotationClockwise {
		switch current {
		case OrientationPortrait:
			nextOrientation = OrientationLandscapeRight
		case OrientationLandscapeRight:
			nextOrientation = OrientationPortraitInverted
		case OrientationPortraitInverted:
			nextOrientation = OrientationLandscapeLeft
		case OrientationLandscapeLeft:
			nextOrientation = OrientationPortrait
		default:
			nextOrientation = OrientationPortrait
		}
	} else { // counterclockwise
		switch current {
		case OrientationPortrait:
			nextOrientation = OrientationLandscapeLeft
		case OrientationLandscapeLeft:
			nextOrientation = OrientationPortraitInverted
		case OrientationPortraitInverted:
			nextOrientation = OrientationLandscapeRight
		case OrientationLandscapeRight:
			nextOrientation = OrientationPortrait
		default:
			nextOrientation = OrientationPortrait
		}
	}

	return a.SetOrientationLock(ctx, nextOrientation)
}
