package icons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchWithQuery(t *testing.T) {
	il := &Loader{Icons: []string{"home", "home-outline", "star", "star-outline", "newspaper"}}
	results := il.Search("home")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'home', got %d", len(results))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	icons := make([]string, 100)
	for i := range icons {
		icons[i] = "icon-" + string(rune('a'+i%26))
	}
	il := &Loader{Icons: icons}
	results := il.Search("")
	if len(results) != 50 {
		t.Errorf("expected 50 results (capped), got %d", len(results))
	}
}

func TestSearchEmptyQuerySmallList(t *testing.T) {
	il := &Loader{Icons: []string{"home", "star"}}
	results := il.Search("")
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearchAllWithQuery(t *testing.T) {
	il := &Loader{Icons: []string{"home", "home-outline", "star"}}
	results := il.SearchAll("home")
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearchAllEmptyQuery(t *testing.T) {
	il := &Loader{Icons: []string{"a", "b", "c"}}
	results := il.SearchAll("")
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	tmpDir := t.TempDir()
	origCacheFile := iconCacheFile
	iconCacheFile = filepath.Join(tmpDir, "icons.txt")
	defer func() { iconCacheFile = origCacheFile }()

	il := &Loader{Icons: []string{"home", "star", "newspaper"}}
	if err := il.saveToCache(); err != nil {
		t.Fatalf("saveToCache failed: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(iconCacheFile); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	il2 := &Loader{}
	if err := il2.loadFromCache(); err != nil {
		t.Fatalf("loadFromCache failed: %v", err)
	}
	if len(il2.Icons) != 3 {
		t.Errorf("expected 3 icons from cache, got %d", len(il2.Icons))
	}
}

func TestLoadFromCacheEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origCacheFile := iconCacheFile
	iconCacheFile = filepath.Join(tmpDir, "empty.txt")
	defer func() { iconCacheFile = origCacheFile }()

	os.WriteFile(iconCacheFile, []byte(""), 0644)

	il := &Loader{}
	err := il.loadFromCache()
	if err == nil {
		t.Error("expected error for empty cache")
	}
}

func TestSaveToCacheEmpty(t *testing.T) {
	il := &Loader{Icons: []string{}}
	err := il.saveToCache()
	if err != nil {
		t.Errorf("expected nil error for empty icons, got %v", err)
	}
}
