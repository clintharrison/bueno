// Package lipcaction holds key map actions that use LIPC to run.
package lipcaction

import (
	"context"
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
