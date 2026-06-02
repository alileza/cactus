// Package config loads and validates the cactus configuration file.
package config

import (
	"fmt"
	"os"
	"time"

	"cactus/notifier"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration loaded from the YAML file.
type Config struct {
	Probes    []ProbeConfig   `yaml:"probes"`
	Receivers ReceiversConfig `yaml:"receivers"`
}

// ProbeConfig describes a single HTTP endpoint to probe.
type ProbeConfig struct {
	Name           string            `yaml:"name"`
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Interval       Duration          `yaml:"interval"`
	Auth           *AuthConfig       `yaml:"auth,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	ExpectedStatus int               `yaml:"expected_status"`
}

// AuthConfig holds HTTP Basic Auth credentials for a probe.
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ReceiversConfig lists the alert destinations to notify on state changes.
type ReceiversConfig struct {
	Telegram *notifier.TelegramConfig `yaml:"telegram,omitempty"`
}

// Duration wraps time.Duration to support YAML unmarshaling from strings like "30s".
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

// Load reads and parses the YAML config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
