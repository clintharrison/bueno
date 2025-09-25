// Package config loads the YAML config and provides an API for accessing key mappings.
package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const configPath = "/mnt/us/kindle-keymap.yaml"

type yamlConfig struct {
	Devices []yamlDevice `yaml:"device"`
}

type yamlDevice struct {
	Name string            `yaml:"name"`
	Bind map[string]string `yaml:"bind"`
}

type Device struct {
	nameRegex *regexp.Regexp
	bindings  map[string]string
}

func normalizeKeyName(key string) string {
	key = strings.TrimPrefix(key, "KEY_")
	key = strings.TrimPrefix(key, "BTN_")
	key = strings.ToUpper(key)
	return key
}

func (d *Device) BindingForKey(keyName string) string {
	// key may be several names separated by /
	keys := strings.Split(keyName, "/")
	nk := make([]string, 0, len(keys))
	for _, k := range keys {
		k = normalizeKeyName(k)
		if k != "" {
			nk = append(nk, k)
		}
	}
	slog.Debug("looking up binding for key", "keyName", keyName, "keys", nk, "bindings", d.bindings)
	for _, key := range nk {
		key = normalizeKeyName(key)
		slog.Debug("checking for binding for key", "key", key)
		if val, ok := d.bindings[key]; ok {
			return val
		}
	}
	return ""
}

func newDevice(name string, bindings map[string]string) (*Device, error) {
	re, err := regexp.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("invalid device name regex %q: %w", name, err)
	}
	return &Device{
		nameRegex: re,
		bindings:  bindings,
	}, nil
}

func (d *Device) NamePattern() *regexp.Regexp {
	return d.nameRegex
}

type Config struct {
	Devices []Device
}

func Load() (*Config, error) {
	var yamlCfg yamlConfig
	var file io.ReadCloser
	file, err := os.Open(configPath)
	if err == nil {
		err = yaml.NewDecoder(file).Decode(&yamlCfg)
		if err == nil {
			deviceNames := make([]string, 0, len(yamlCfg.Devices))
			for _, d := range yamlCfg.Devices {
				deviceNames = append(deviceNames, d.Name)
			}
			slog.Info("read config from file", "path", configPath, "device_names", deviceNames)
		}
	}
	if err != nil {
		slog.Error("failed to load config file", "path", configPath, "error", err)
		return nil, err
	}
	defer file.Close()
	devices := make([]Device, 0, len(yamlCfg.Devices))
	for _, d := range yamlCfg.Devices {
		bindings := make(map[string]string, len(d.Bind))
		for k, v := range d.Bind {
			// TODO: validate these keys exist on this device?

			// Hopefully you don't have KEY_B and BTN_B on the same device :)
			key := normalizeKeyName(k)
			if _, exists := bindings[key]; exists {
				return nil, fmt.Errorf("duplicate binding for key %q", key)
			}
			bindings[key] = v
		}
		dev, err := newDevice(d.Name, bindings)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *dev)
	}
	cfg := Config{
		Devices: devices,
	}
	return &cfg, nil
}

func (c *Config) MatchingDevice(devName string) *Device {
	for _, d := range c.Devices {
		if d.NamePattern().MatchString(devName) {
			return &d
		}
	}
	return nil
}
