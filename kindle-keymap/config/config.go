// Package config loads the YAML config and provides an API for accessing key mappings.
package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/clintharrison/bueno/ace/address"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "/mnt/us/extensions/kindle-keymap/kindle-keymap.yaml"
const configEnvVar = "KINDLE_KEYMAP_CONFIG"

type yamlConfig struct {
	Devices []yamlDevice `yaml:"device"`
}

type yamlDevice struct {
	Name string            `yaml:"name"`
	Addr string            `yaml:"mac,omitempty"`
	Bind map[string]string `yaml:"bind"`
}

type Device struct {
	address  address.Address
	bindings map[string]string
}

func isSpecificName(key string) bool {
	return strings.HasPrefix(key, "KEY_") || strings.HasPrefix(key, "BTN_")
}

func (d *Device) BindingForKey(keyName string) string {
	// key may be several names separated by /
	keys := strings.Split(keyName, "/")
	for _, key := range keys {
		if val, ok := d.bindings[key]; ok {
			return val
		}
	}
	return ""
}

func newDevice(addr string, bindings map[string]string) (*Device, error) {
	mac, err := address.NewFromString(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid device address %q: %w", addr, err)
	}
	return &Device{
		address:  mac,
		bindings: bindings,
	}, nil
}

func (d *Device) Address() address.Address {
	return d.address
}

func (d *Device) Dump() string {
	return fmt.Sprintf("Device{address=%q, bindings=%v}", d.address.ToString(), d.bindings)
}

type Config struct {
	Devices []Device
}

func getConfigPath() string {
	if path := os.Getenv(configEnvVar); path != "" {
		fi, err := os.Stat(path)
		if err == nil && !fi.IsDir() {
			return path
		}
		slog.Warn(fmt.Sprintf("%s is set but not a valid file, using default", configEnvVar), "path", path, "error", err)
	}
	return defaultConfigPath
}

func Load() (*Config, error) {
	var yamlCfg yamlConfig
	var file io.ReadCloser
	configPath := getConfigPath()
	file, err := os.Open(configPath)
	if err == nil {
		err = yaml.NewDecoder(file).Decode(&yamlCfg)
		if err == nil {
			deviceNames := make([]string, 0, len(yamlCfg.Devices))
			for _, d := range yamlCfg.Devices {
				deviceNames = append(deviceNames, d.Name)
			}
			slog.Info("config read from file", "path", configPath, "device_names", deviceNames)
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
			if isSpecificName(k) {
				// Keep the original name if it has a KEY_ prefix to disambiguate from BTN_.
				if err := addToBindings(bindings, k, v); err != nil {
					return nil, fmt.Errorf("error in device %q binding for key %q: %w", d.Name, k, err)
				}
			} else {
				if err := addToBindings(bindings, "BTN_"+k, v); err != nil {
					return nil, fmt.Errorf("error in device %q binding for key %q: %w", d.Name, "BTN_"+k, err)
				}
				if err := addToBindings(bindings, "KEY_"+k, v); err != nil {
					return nil, fmt.Errorf("error in device %q binding for key %q: %w", d.Name, "KEY_"+k, err)
				}
			}
		}
		dev, err := newDevice(d.Addr, bindings)
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

func addToBindings(bindings map[string]string, key string, action string) error {
	key = strings.ToUpper(key)
	if _, exists := bindings[key]; exists {
		return fmt.Errorf("duplicate binding for key %q", key)
	}
	bindings[key] = action
	return nil
}

func (c *Config) FirstMatchingDevice(addr address.Address) *Device {
	for _, d := range c.Devices {
		if addr == d.Address() {
			return &d
		}
	}
	return nil
}
