package common

import (
	"testing"
	"time"
)

func TestResolveNullBoolNilReturnsDefault(t *testing.T) {
	if !ResolveNullBool(nil, true) {
		t.Error("expected true when val is nil and default is true")
	}
	if ResolveNullBool(nil, false) {
		t.Error("expected false when val is nil and default is false")
	}
}

func TestResolveNullBoolOverridesDefault(t *testing.T) {
	tr := true
	fa := false
	if !ResolveNullBool(&tr, false) {
		t.Error("expected true to override false default")
	}
	if ResolveNullBool(&fa, true) {
		t.Error("expected false to override true default")
	}
}

func TestDemoMaxSessionsDefault(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "")
	n, err := DemoMaxSessions()
	if err != nil {
		t.Fatal(err)
	}
	if n != DemoMaxSessionsDefault {
		t.Errorf("expected %d, got %d", DemoMaxSessionsDefault, n)
	}
}

func TestDemoMaxSessionsFromEnv(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "50")
	n, err := DemoMaxSessions()
	if err != nil {
		t.Fatal(err)
	}
	if n != 50 {
		t.Errorf("expected 50, got %d", n)
	}
}

func TestDemoMaxSessionsInvalid(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "abc")
	_, err := DemoMaxSessions()
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestDemoMaxSessionsNonPositive(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "0")
	_, err := DemoMaxSessions()
	if err == nil {
		t.Error("expected error for non-positive value")
	}
}

func TestDemoSessionTTLDefault(t *testing.T) {
	t.Setenv("DEMO_SESSION_TTL", "")
	d, err := DemoSessionTTL()
	if err != nil {
		t.Fatal(err)
	}
	if d != DemoSessionTTLDefault {
		t.Errorf("expected %v, got %v", DemoSessionTTLDefault, d)
	}
}

func TestDemoSessionTTLFromEnv(t *testing.T) {
	t.Setenv("DEMO_SESSION_TTL", "1h")
	d, err := DemoSessionTTL()
	if err != nil {
		t.Fatal(err)
	}
	if d != time.Hour {
		t.Errorf("expected 1h, got %v", d)
	}
}

func TestDemoSessionTTLInvalid(t *testing.T) {
	t.Setenv("DEMO_SESSION_TTL", "abc")
	_, err := DemoSessionTTL()
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestDemoSessionTTLNonPositive(t *testing.T) {
	t.Setenv("DEMO_SESSION_TTL", "-5m")
	_, err := DemoSessionTTL()
	if err == nil {
		t.Error("expected error for non-positive duration")
	}
}
