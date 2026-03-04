package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	DB   string     `yaml:"db"`
	Addr string     `yaml:"addr"`
	Auth *bool      `yaml:"auth"`
	Slow int        `yaml:"slow"`
	Demo DemoConfig `yaml:"demo"`
	API  APIConfig  `yaml:"api"`
	MCP  MCPConfig  `yaml:"mcp"`
}

type DemoConfig struct {
	Enabled bool   `yaml:"enabled"`
	Source  string `yaml:"source"`
}

type APIConfig struct {
	Swagger bool      `yaml:"swagger"`
	Tokens  APITokens `yaml:"tokens"`
}

type APITokens struct {
	ReadWrite []string `yaml:"read-write"`
	ReadOnly  []string `yaml:"read-only"`
}

// HasTokens returns true if any API tokens are configured.
func (t APITokens) HasTokens() bool {
	return len(t.ReadWrite) > 0 || len(t.ReadOnly) > 0
}

type MCPConfig struct {
	Enabled bool `yaml:"enabled"`
}

// LoadServerConfig reads and parses a server config YAML file.
func LoadServerConfig(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read config: %w", err)
	}
	var cfg ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

// ApplyDefaults fills in zero-value fields with sensible defaults.
func (c *ServerConfig) ApplyDefaults() {
	if c.DB == "" {
		c.DB = "data/jumpgate.db"
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.Auth == nil {
		t := true
		c.Auth = &t
	}
}

// AuthEnabled returns whether admin auth is enabled.
func (c *ServerConfig) AuthEnabled() bool {
	if c.Auth == nil {
		return true
	}
	return *c.Auth
}
