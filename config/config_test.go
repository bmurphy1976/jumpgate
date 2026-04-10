package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Config (general)

func TestLoadConfig(t *testing.T) {
	path := writeTempYAML(t, `
title: "My Links"
categories:
  - name: Favorites
    links:
      - name: Google
        url: https://google.com
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Title != "My Links" {
		t.Errorf("expected title 'My Links', got %q", cfg.Title)
	}
	if len(cfg.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cfg.Categories))
	}
	if len(cfg.Categories[0].Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(cfg.Categories[0].Links))
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ServerConfig

func TestLoadServerConfig(t *testing.T) {
	path := writeTempYAML(t, `
db: "/tmp/test.db"
addr: ":9090"
auth: false
slow: 100
api:
  swagger: true
  tokens:
    read-write:
      - "rw-token"
    read-only:
      - "ro-token"
mcp:
  enabled: true
demo:
  enabled: true
  source: "config.yaml"
  allowed_proxies:
    - "127.0.0.1"
`)
	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}
	if cfg.DB != "/tmp/test.db" {
		t.Errorf("expected DB '/tmp/test.db', got %q", cfg.DB)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("expected Addr ':9090', got %q", cfg.Addr)
	}
	if cfg.AuthEnabled() {
		t.Error("expected auth disabled")
	}
	if cfg.Slow != 100 {
		t.Errorf("expected Slow 100, got %d", cfg.Slow)
	}
	if !cfg.API.Swagger {
		t.Error("expected swagger enabled")
	}
	if len(cfg.API.Tokens.ReadWrite) != 1 || cfg.API.Tokens.ReadWrite[0] != "rw-token" {
		t.Errorf("unexpected ReadWrite tokens: %v", cfg.API.Tokens.ReadWrite)
	}
	if !cfg.MCP.Enabled {
		t.Error("expected MCP enabled")
	}
	if !cfg.Demo.Enabled {
		t.Error("expected Demo enabled")
	}
	if cfg.Demo.DisableProxyHeaders != nil {
		t.Error("expected demo proxy headers to be enabled by allowlist")
	}
	if len(cfg.Demo.AllowedProxies) != 1 || cfg.Demo.AllowedProxies[0] != "127.0.0.1" {
		t.Errorf("unexpected AllowedProxies: %v", cfg.Demo.AllowedProxies)
	}
}

func TestLoadServerConfigDemoRequiresAllowlistOrDisableFlag(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
`)

	_, err := LoadServerConfig(path)
	if err == nil || err.Error() != "demo.allowed_proxies must be set unless demo.disable_proxy_headers is true when demo.enabled is true" {
		t.Fatalf("expected demo allowlist/disable error, got %v", err)
	}
}

func TestLoadServerConfigDemoAllowsAllowlistMode(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
  allowed_proxies:
    - "127.0.0.1"
`)

	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}
	if len(cfg.Demo.AllowedProxies) != 1 {
		t.Fatalf("expected one allowed proxy, got %d", len(cfg.Demo.AllowedProxies))
	}
}

func TestLoadServerConfigRejectsInvalidAllowedProxy(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
  allowed_proxies:
    - "not-an-ip"
`)

	_, err := LoadServerConfig(path)
	if err == nil || err.Error() != `invalid demo.allowed_proxies entry "not-an-ip": must be an IP address or CIDR` {
		t.Fatalf("expected invalid allowed proxy error, got %v", err)
	}
}

func TestLoadServerConfigAllowsDemoWithProxyHeadersDisabled(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
  disable_proxy_headers: true
`)

	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}
	if cfg.Demo.DisableProxyHeaders == nil || !*cfg.Demo.DisableProxyHeaders {
		t.Fatal("expected demo proxy headers disabled")
	}
}

func TestLoadServerConfigRejectsContradictoryDisableAndAllowlist(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
  disable_proxy_headers: true
  allowed_proxies:
    - "127.0.0.1"
`)

	_, err := LoadServerConfig(path)
	if err == nil || err.Error() != "demo.allowed_proxies must not be set when demo.disable_proxy_headers is true" {
		t.Fatalf("expected contradictory config error, got %v", err)
	}
}

func TestLoadServerConfigExplicitFalseStillRequiresAllowlist(t *testing.T) {
	path := writeTempYAML(t, `
demo:
  enabled: true
  source: "config.yaml"
  disable_proxy_headers: false
`)

	_, err := LoadServerConfig(path)
	if err == nil || err.Error() != "demo.allowed_proxies must be set unless demo.disable_proxy_headers is true when demo.enabled is true" {
		t.Fatalf("expected missing allowlist error, got %v", err)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := ServerConfig{}
	cfg.ApplyDefaults()
	if cfg.DB != "data/jumpgate.db" {
		t.Errorf("expected default DB, got %q", cfg.DB)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("expected default Addr, got %q", cfg.Addr)
	}
	if cfg.Auth == nil || !*cfg.Auth {
		t.Error("expected default Auth = true")
	}
}

func TestApplyDefaultsPreservesExisting(t *testing.T) {
	f := false
	cfg := ServerConfig{DB: "/custom.db", Addr: ":3000", Auth: &f}
	cfg.ApplyDefaults()
	if cfg.DB != "/custom.db" {
		t.Errorf("expected preserved DB, got %q", cfg.DB)
	}
	if cfg.Addr != ":3000" {
		t.Errorf("expected preserved Addr, got %q", cfg.Addr)
	}
	if *cfg.Auth != false {
		t.Error("expected preserved Auth = false")
	}
}

func TestAuthEnabledDefault(t *testing.T) {
	cfg := ServerConfig{}
	if !cfg.AuthEnabled() {
		t.Error("expected AuthEnabled() = true when Auth is nil")
	}
}

func TestHasTokens(t *testing.T) {
	tokens := APITokens{ReadWrite: []string{"t1"}}
	if !tokens.HasTokens() {
		t.Error("expected HasTokens() = true")
	}
	empty := APITokens{}
	if empty.HasTokens() {
		t.Error("expected HasTokens() = false for empty")
	}
}

// CLIConfig

func TestLoadCLIConfig(t *testing.T) {
	path := writeTempYAML(t, `
url: "http://myserver:8080"
token: "my-token"
`)
	cfg, err := LoadCLIConfig(path)
	if err != nil {
		t.Fatalf("LoadCLIConfig failed: %v", err)
	}
	if cfg.URL != "http://myserver:8080" {
		t.Errorf("expected URL 'http://myserver:8080', got %q", cfg.URL)
	}
	if cfg.Token != "my-token" {
		t.Errorf("expected token 'my-token', got %q", cfg.Token)
	}
}

func TestLoadCLIConfigDefaultURL(t *testing.T) {
	path := writeTempYAML(t, `
token: "my-token"
`)
	cfg, err := LoadCLIConfig(path)
	if err != nil {
		t.Fatalf("LoadCLIConfig failed: %v", err)
	}
	if cfg.URL != "http://localhost:8080" {
		t.Errorf("expected default URL, got %q", cfg.URL)
	}
}

func TestDefaultCLIConfigPaths(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	paths := DefaultCLIConfigPaths()
	if len(paths) == 0 {
		t.Fatal("expected at least 1 path")
	}
	if paths[0] != "/tmp/xdg/jumpgate-cli/config.yaml" {
		t.Errorf("expected XDG path first, got %q", paths[0])
	}
}
