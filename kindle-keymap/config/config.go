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

type Device struct {
	Name string            `yaml:"name"`
	Bind map[string]string `yaml:"bind"`
}

type yamlConfig struct {
	yamlRegex *regexp.Regexp
	Device    Device `yaml:"device"`
	// TODO: make this an array and match connecting devices against all of them
}

// BindingForKey implements Config.
func (y *yamlConfig) BindingForKey(key string) string {
	if val, ok := y.Device.Bind[key]; ok {
		return val
	}
	return ""
}

// DeviceName implements Config.
func (y *yamlConfig) DeviceNamePattern() *regexp.Regexp {
	if y.yamlRegex == nil {
		y.yamlRegex = regexp.MustCompile(y.Device.Name)
	}
	return y.yamlRegex
}

type Config struct {
	DeviceNamePattern *regexp.Regexp
	DeviceType        string
	bindings          map[string]string
}

func (c *Config) BindingForKey(key string) string {
	key = strings.TrimPrefix(key, "KEY_")
	if val, ok := c.bindings[key]; ok {
		return val
	}
	return ""
}

func Load() (*Config, error) {
	var yamlCfg yamlConfig
	var file io.ReadCloser
	file, err := os.Open(configPath)
	if err == nil {
		err = yaml.NewDecoder(file).Decode(&yamlCfg)
		if err == nil {
			slog.Info("read config from file", "path", configPath, "device_name", yamlCfg.Device.Name, "bindings", yamlCfg.Device.Bind)
		}
	}
	if err != nil {
		slog.Warn("failed to load config file, using default config", "path", configPath, "error", err)
		yamlCfg = yamlConfig{
			Device: Device{
				Name: "(?i)8BitDo.*gamepad Keyboard",
				Bind: map[string]string{
					"PageUp":   "prev_page",
					"PageDown": "next_page",
					"G":        "next_page", // A button
					"J":        "next_page", // B button
					"H":        "prev_page", // X button
					"I":        "prev_page", // Y button
				},
			},
		}
	}
	defer file.Close()
	deviceName, err := regexp.Compile(yamlCfg.Device.Name)
	if err != nil {
		return nil, err
	}
	bindings := make(map[string]string, len(yamlCfg.Device.Bind))
	for k, v := range yamlCfg.Device.Bind {
		key := strings.TrimPrefix(k, "KEY_")
		key = strings.ToUpper(key)
		if _, exists := bindings[key]; exists {
			return nil, fmt.Errorf("duplicate binding for key %q", key)
		}
		if key == "" {
			return nil, fmt.Errorf("invalid key name %q", k)
		}
		bindings[key] = v
	}
	cfg := Config{
		DeviceNamePattern: deviceName,
		bindings:          bindings,
	}
	return &cfg, nil
}
