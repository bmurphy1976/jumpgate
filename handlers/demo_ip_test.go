package handlers

import (
	"dashboard/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func demoConfigWithAllowedProxies(proxies ...string) config.DemoConfig {
	return config.DemoConfig{
		Enabled:        true,
		AllowedProxies: proxies,
	}
}

func demoConfigWithProxyHeadersDisabled() config.DemoConfig {
	disabled := true
	return config.DemoConfig{
		Enabled:             true,
		DisableProxyHeaders: &disabled,
	}
}

func newRequest(remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	return req
}

func runDemoSessionRequest(t *testing.T, demo config.DemoConfig, req *http.Request) (string, string) {
	t.Helper()

	extractor, err := newDemoIPExtractor(demo)
	if err != nil {
		t.Fatalf("newDemoIPExtractor returned error: %v", err)
	}

	e := echo.New()
	e.IPExtractor = extractor

	var sessionID, sessionIP string
	inner := func(c *echo.Context) error {
		sessionID, _ = (*c).Get("session_id").(string)
		sessionIP, _ = (*c).Get("session_ip").(string)
		return (*c).String(http.StatusOK, "ok")
	}

	handler := sessionMiddleware()(inner)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Fatalf("sessionMiddleware returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if sessionID == "" {
		t.Fatal("expected session_id to be set")
	}

	return sessionID, sessionIP
}

func TestSessionMiddlewareUsesRemoteAddrWhenProxyHeadersDisabled(t *testing.T) {
	req := newRequest("203.0.113.40:1234")
	req.Header.Set(echo.HeaderXForwardedFor, "198.51.100.20")

	_, sessionIP := runDemoSessionRequest(t, demoConfigWithProxyHeadersDisabled(), req)
	if sessionIP != "203.0.113.40" {
		t.Fatalf("expected remote addr IP, got %q", sessionIP)
	}
}

func TestSessionMiddlewareUsesRemoteAddrWithoutXFF(t *testing.T) {
	req := newRequest("203.0.113.10:1234")

	_, sessionIP := runDemoSessionRequest(t, demoConfigWithAllowedProxies("198.51.100.10"), req)
	if sessionIP != "203.0.113.10" {
		t.Fatalf("expected remote addr IP, got %q", sessionIP)
	}
}

func TestSessionMiddlewareUsesXForwardedForForAllowlistedProxy(t *testing.T) {
	req := newRequest("198.51.100.10:1234")
	req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.20")

	_, sessionIP := runDemoSessionRequest(t, demoConfigWithAllowedProxies("198.51.100.10"), req)
	if sessionIP != "203.0.113.20" {
		t.Fatalf("expected client IP from X-Forwarded-For, got %q", sessionIP)
	}
}

func TestSessionMiddlewareFallsBackToRemoteAddrOnMalformedXFF(t *testing.T) {
	req := newRequest("198.51.100.10:1234")
	req.Header.Set(echo.HeaderXForwardedFor, "bad-ip")

	_, sessionIP := runDemoSessionRequest(t, demoConfigWithAllowedProxies("198.51.100.10"), req)
	if sessionIP != "198.51.100.10" {
		t.Fatalf("expected remote addr IP, got %q", sessionIP)
	}
}

func TestSessionMiddlewareUsesNearestUntrustedHopFromXFF(t *testing.T) {
	req := newRequest("198.51.100.10:1234")
	req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.20, 192.0.2.10")

	_, sessionIP := runDemoSessionRequest(t, demoConfigWithAllowedProxies("198.51.100.10"), req)
	if sessionIP != "192.0.2.10" {
		t.Fatalf("expected nearest untrusted hop, got %q", sessionIP)
	}
}

func TestNewServerReturnsErrorForInvalidDemoProxyConfig(t *testing.T) {
	cfg := config.ServerConfig{
		Demo: config.DemoConfig{
			Enabled:        true,
			AllowedProxies: []string{"bad-proxy"},
		},
	}

	srv, err := NewServer(cfg, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if srv != nil {
		t.Fatal("expected nil server on error")
	}
}
