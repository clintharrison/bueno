package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/clintharrison/bueno/core/logutil"
	"github.com/clintharrison/bueno/lipc"
	"github.com/clintharrison/bueno/quietly"
	"github.com/godbus/dbus/v5"
)

func main() {
	err := doMain()
	if err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func doMain() error {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logutil.ConfigureInteractiveLogger()

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer quietly.Close(conn)

	// Skip requesting a name, like LipcOpenNoName()
	// conn.RequestName("com.example.lipc-go", dbus.NameFlagReplaceExisting)

	// conn.Eavesdrop() will cause us to miss any replies to our own calls in Demo(),
	// so only one or the other can be run at a time.
	// EavesdropAll(ctx, conn)
	err = Demo(ctx, conn)
	if err != nil {
		return fmt.Errorf("error in demo: %w", err)
	}
	return nil
}

// EavesdropAll sets up match rules to listen for all message types on the given bus.
// This function does not return until the context is cancelled.
func EavesdropAll(ctx context.Context, conn *dbus.Conn) error {
	// I think this is equivalent to conn.AddMatchSignalContext(ctx)? But it doesn't expose any other match types :/
	_ = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='signal'").Store()
	_ = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='method_call'").Store()
	_ = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='method_return'").Store()
	_ = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='error'").Store()

	ms := make(chan *dbus.Message, 10)
	conn.Eavesdrop(ms)

	slog.Info("listening for messages")
	for {
		select {
		case msg := <-ms:
			slog.Info("got message", "msg", *msg)
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				slog.Info("exiting")
				return nil
			}
			slog.Error("context done", "error", ctx.Err())
			return ctx.Err()
		}
	}
}

func Demo(ctx context.Context, conn *dbus.Conn) error {
	intensity, err := lipc.GetProperty[int32](ctx, conn, "com.lab126.powerd", "flIntensity")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "intensity", intensity)

	cvmLogLevel, err := lipc.GetProperty[string](ctx, conn, "com.lab126.cvm", "logLevel")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "cvm log level", cvmLogLevel)

	powerStatus, err := lipc.GetProperty[string](ctx, conn, "com.lab126.powerd", "status")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "power status", powerStatus)

	err = lipc.SetProperty(ctx, conn, "com.lab126.powerd", "flIntensity", intensity+1)
	if err != nil {
		slog.Error("Failed to set property", "error", err)
		return err
	}
	slog.Info("wrote property", "intensity", intensity+1)
	return nil
}
