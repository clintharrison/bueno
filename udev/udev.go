// Package udev provides a udev-based input device watcher.
package udev

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"github.com/holoplot/go-evdev"
	"github.com/pilebones/go-udev/netlink"
)

type InputDeviceWatcher struct {
	Patterns   []*regexp.Regexp
	AddFunc    func(dev *evdev.InputDevice)
	RemoveFunc func(uevent netlink.UEvent)
}

func ptrTo[T any](v T) *T {
	return &v
}

func (w *InputDeviceWatcher) Start(ctx context.Context) {
	conn := netlink.UEventConn{}
	if err := conn.Connect(netlink.KernelEvent); err != nil {
		slog.Error("conn.Connect()", "error", err)
		return
	}
	defer conn.Close()
	slog.Info("connected to udev netlink socket", "fd", conn.Fd, "addr", conn.Addr)

	queue := make(chan netlink.UEvent)
	errors := make(chan error)
	rules := &netlink.RuleDefinitions{}

	rules.AddRule(netlink.RuleDefinition{
		Action: ptrTo("add"),
		Env: map[string]string{
			"SUBSYSTEM": "input",
		},
	})
	rules.AddRule(netlink.RuleDefinition{
		Action: ptrTo("remove"),
		Env: map[string]string{
			"SUBSYSTEM": "input",
		},
	})
	quit := conn.Monitor(queue, errors, rules)

	for {
		select {
		case uevent := <-queue:
			w.handleEvent(uevent)
		case err := <-errors:
			slog.Error("conn.Monitor() sent an error", "error", err)
		case <-ctx.Done():
			close(quit)
			return
		}
	}
}

func pollForDevice(path string) (*evdev.InputDevice, error) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timed out waiting for device at %s", path)
		case <-ticker.C:
			dev, err := evdev.Open(path)
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				continue
			} else if err != nil {
				slog.Warn("evdev.Open()", "path", path, "error", err)
				return nil, err
			}
			return dev, nil
		}
	}
}

func (w *InputDeviceWatcher) handleEvent(uevent netlink.UEvent) {
	switch uevent.Action {
	case "add":
		// slog.Debug("udev event", "action", uevent.Action)
		// for k, v := range uevent.Env {
		// 	slog.Debug("  env", "key", k, "value", v)
		// }
		// add events need to first be filtered by name patterns
		devname := uevent.Env["DEVNAME"]
		if devname == "" {
			slog.Debug("ignoring event with no DEVNAME")
			return
		}
		devPath := fmt.Sprintf("/dev/%s", devname)
		// Unfortunately the event that has DEVNAME set is not the one that has properties like device name :(
		// So, we have to open the device to get its name.
		// We're also responding to this event concurrently with udevd running its rules, so
		// we have to poll for the device to exist for opening.
		dev, err := pollForDevice(devPath)
		if err != nil {
			slog.Error("evdev.Open()", "path", devPath, "error", err)
			return
		}

		if len(w.Patterns) > 0 {
			if name, err := dev.Name(); err == nil {
				matched := slices.ContainsFunc(w.Patterns, func(pattern *regexp.Regexp) bool {
					return pattern.MatchString(name)
				})
				if !matched {
					slog.Debug("ignoring device not matching patterns", "devname", name, "path", dev.Path())
					dev.Close()
					return
				}
			} else {
				slog.Error("dev.Name()", "path", dev.Path(), "error", err)
				dev.Close()
				return
			}
		}
		// If we get here, we have a device that matches our patterns (or we have no patterns)
		w.AddFunc(dev)
	case "remove":
		w.RemoveFunc(uevent)
	}
}
