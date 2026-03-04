package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CLIConfig holds configuration for the jumpgate-cli tool.
type CLIConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

// DefaultCLIConfigPaths returns candidate config file paths in priority order.
// It checks $XDG_CONFIG_HOME first, then falls back to ~/.config/.
func DefaultCLIConfigPaths() []string {
	var paths []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "jumpgate-cli", "config.yaml"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "jumpgate-cli", "config.yaml"))
	}

	return paths
}

// LoadCLIConfig reads and parses a CLI config YAML file.
func LoadCLIConfig(path string) (CLIConfig, error) {
	cfg := CLIConfig{URL: "http://localhost:8080"}

	data, err := os.ReadFile(path)
	if err != nil {
		return CLIConfig{}, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return CLIConfig{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
