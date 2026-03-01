// Package install handles setting up the system for kindle-keymap
package install

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
)

const (
	udevRulePath = "/etc/udev/rules.d/99-kindle-keymap.rules"
	udevRuleBody = `KERNEL=="uhid", MODE="0660", GROUP="bluetooth"

ACTION=="add", SUBSYSTEM=="input", IMPORT+="/usr/local/bin/dev_is_keyboard.sh %N"
`
	devIsKeyboardScriptPath = "/usr/local/bin/dev_is_keyboard.sh"
	devIsKeyboardScriptBody = `#!/bin/sh
DEVICE=$1
if evtest info $DEVICE | grep -q 'Event type 1 (Key)'; then
  if evtest info $DEVICE | grep -q 'Event code 16 (Q)'; then
    # Don't set these just because Key is supported -- that will
    # detect the touchscreen as a keyboard which breaks the UI
    echo ID_INPUT=1
    echo ID_INPUT_KEY=1
    echo ID_INPUT_KEYBOARD=1
  fi
fi
`
)

func MaybeInstallUdevRule(ctx context.Context) error {
	// do we need to mntroot rw?
	scriptExists := false
	ruleExists := false

	if _, err := os.Stat(devIsKeyboardScriptPath); err == nil {
		scriptExists = true
	}
	if _, err := os.Stat(udevRulePath); err == nil {
		ruleExists = true
	}

	if scriptExists && ruleExists {
		slog.Debug("udev rule and script already installed")
		return nil
	}

	slog.Info("running 'mntroot rw'")
	cmd := exec.CommandContext(ctx, "/usr/sbin/mntroot", "rw")
	err := cmd.Run()
	if err != nil {
		return err
	}

	defer func() {
		cmd := exec.Command("/usr/sbin/mntroot", "ro")
		if err := cmd.Run(); err != nil {
			slog.Error("failed to remount rootfs as read-only", "error", err)
		}
	}()

	if !scriptExists {
		slog.Info("installing dev_is_keyboard.sh script", "path", devIsKeyboardScriptPath)
		err := os.WriteFile(devIsKeyboardScriptPath, []byte(devIsKeyboardScriptBody), 0755)
		if err != nil {
			return err
		}
	}

	if !ruleExists {
		slog.Info("installing udev rule", "path", udevRulePath)
		err := os.WriteFile(udevRulePath, []byte(udevRuleBody), 0644)
		if err != nil {
			return err
		}
	}

	// if either of these were missing, we also need to reload udev rules and trigger
	// if this doesn't work, :shrug: reboot time
	slog.Info("reloading udev rules")
	cmd = exec.CommandContext(ctx, "udevadm", "control", "--reload-rules")
	err = cmd.Run()
	if err != nil {
		return err
	}

	slog.Info("triggering udev")
	cmd = exec.CommandContext(ctx, "udevadm", "trigger")
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
