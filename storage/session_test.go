package storage

import (
	"dashboard/config"
	"testing"
)

func TestSessionStoreGetOrCreateNew(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "10")
	t.Setenv("DEMO_SESSION_TTL", "1h")

	store := NewSessionStore(config.Config{Title: "Test"}, nil)
	ds, err := store.GetOrCreate("session1", "127.0.0.1")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if ds == nil {
		t.Fatal("expected non-nil datasource")
	}

	// Verify the session DB was seeded with the config
	settings, err := ds.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", settings.Title)
	}
}

func TestSessionStoreGetOrCreateExisting(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "10")
	t.Setenv("DEMO_SESSION_TTL", "1h")

	store := NewSessionStore(config.Config{}, nil)
	ds1, _ := store.GetOrCreate("session1", "127.0.0.1")
	ds2, _ := store.GetOrCreate("session1", "127.0.0.1")
	if ds1 != ds2 {
		t.Error("expected same datasource for same session key")
	}
}

func TestSessionStoreGetOrCreateDifferentIP(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "10")
	t.Setenv("DEMO_SESSION_TTL", "1h")

	store := NewSessionStore(config.Config{}, nil)
	ds1, _ := store.GetOrCreate("session1", "127.0.0.1")
	ds2, _ := store.GetOrCreate("session1", "10.0.0.1")
	if ds1 == ds2 {
		t.Error("expected different datasource for different IP")
	}
}

func TestSessionStoreCapEnforcement(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "2")
	t.Setenv("DEMO_SESSION_TTL", "1h")

	store := NewSessionStore(config.Config{}, nil)
	_, err := store.GetOrCreate("s1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.GetOrCreate("s2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.GetOrCreate("s3", "127.0.0.1")
	if err == nil {
		t.Error("expected error when session cap reached")
	}
}

func TestSessionStoreWrapFunction(t *testing.T) {
	t.Setenv("DEMO_MAX_SESSIONS", "10")
	t.Setenv("DEMO_SESSION_TTL", "1h")

	wrapped := false
	store := NewSessionStore(config.Config{}, func(ds Datasource) Datasource {
		wrapped = true
		return Annotated(ds)
	})
	_, err := store.GetOrCreate("s1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !wrapped {
		t.Error("expected wrap function to be called")
	}
}
