package config

import (
	"fmt"
	"net"
	"os"
	"strings"

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
	Enabled             bool     `yaml:"enabled"`
	Source              string   `yaml:"source"`
	DisableProxyHeaders *bool    `yaml:"disable_proxy_headers"`
	AllowedProxies      []string `yaml:"allowed_proxies"`
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
	if err := cfg.Validate(); err != nil {
		return ServerConfig{}, err
	}
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

func (c *ServerConfig) Validate() error {
	if !c.Demo.Enabled {
		return nil
	}
	return c.Demo.ValidateProxyHeaders()
}

func (d DemoConfig) ValidateProxyHeaders() error {
	if d.DisableProxyHeaders != nil && *d.DisableProxyHeaders {
		if len(d.AllowedProxies) > 0 {
			return fmt.Errorf("demo.allowed_proxies must not be set when demo.disable_proxy_headers is true")
		}
		return nil
	}
	if len(d.AllowedProxies) == 0 {
		return fmt.Errorf("demo.allowed_proxies must be set unless demo.disable_proxy_headers is true when demo.enabled is true")
	}
	if _, err := d.AllowedProxyNetworks(); err != nil {
		return err
	}
	return nil
}

func (d DemoConfig) AllowedProxyNetworks() ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(d.AllowedProxies))
	for _, proxy := range d.AllowedProxies {
		network, err := parseAllowedProxy(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid demo.allowed_proxies entry %q: %w", proxy, err)
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func parseAllowedProxy(value string) (*net.IPNet, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("must not be empty")
	}
	if ip := net.ParseIP(value); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}, nil
		}
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}, nil
	}
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		return nil, fmt.Errorf("must be an IP address or CIDR")
	}
	return network, nil
}

// AuthEnabled returns whether admin auth is enabled.
func (c *ServerConfig) AuthEnabled() bool {
	if c.Auth == nil {
		return true
	}
	return *c.Auth
}
