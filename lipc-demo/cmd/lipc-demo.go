package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clintharrison/bueno/lipc"
	"github.com/godbus/dbus/v5"
	"github.com/lmittmann/tint"
)

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

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	configureLogger()

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		slog.Error("Failed to connect to system bus", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Skip requesting a name, like LipcOpenNoName()
	// conn.RequestName("com.example.lipc-go", dbus.NameFlagReplaceExisting)

	// conn.Eavesdrop() will cause us to miss any replies to our own calls in Demo(),
	// so only one or the other can be run at a time.
	// EavesdropAll(ctx, conn)

	if err = Demo(ctx, conn); err != nil {
		slog.Error("Demo()", "error", err)
		os.Exit(1)
	}

}

// EavesdropAll sets up match rules to listen for all message types on the given bus.
// This function does not return until the context is cancelled.
func EavesdropAll(ctx context.Context, conn *dbus.Conn) error {
	// I think this is equivalent to conn.AddMatchSignalContext(ctx)? But it doesn't expose any other match types :/
	conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='signal'").Store()
	conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='method_call'").Store()
	conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='method_return'").Store()
	conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, "type='error'").Store()

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
	intensity, err := lipc.LipcGetProperty[int32](ctx, conn, "com.lab126.powerd", "flIntensity")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "intensity", intensity)

	cvmLogLevel, err := lipc.LipcGetProperty[string](ctx, conn, "com.lab126.cvm", "logLevel")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "cvm log level", cvmLogLevel)

	powerStatus, err := lipc.LipcGetProperty[string](ctx, conn, "com.lab126.powerd", "status")
	if err != nil {
		slog.Error("Failed to get property", "error", err)
		return err
	}
	slog.Info("got property", "power status", powerStatus)

	if err = lipc.LipcSetProperty(ctx, conn, "com.lab126.powerd", "flIntensity", intensity+1); err != nil {
		slog.Error("Failed to set property", "error", err)
		return err
	}
	slog.Info("wrote property", "intensity", intensity+1)
	return nil
}
