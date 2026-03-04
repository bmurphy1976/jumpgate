package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"dashboard/config"
	"dashboard/handlers"
	"dashboard/icons"
	"dashboard/model"
	"dashboard/storage"

	"github.com/labstack/echo/v5"
)

func setupCLITest(t *testing.T) (*apiClient, *storage.SQLiteDB) {
	t.Helper()
	ds, err := storage.NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ds.Close() })

	resolver := func(c *echo.Context) (storage.Datasource, error) {
		return ds, nil
	}
	e := echo.New()
	apiCfg := config.APIConfig{} // no tokens = no auth
	handlers.SetupAPIRoutes(e, resolver, &icons.Loader{Icons: []string{"home", "star", "newspaper", "github", "google"}}, apiCfg)

	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	client := &apiClient{baseURL: ts.URL}
	return client, ds
}

// getBookmarkViaAPI fetches a bookmark by ID using the test client.
func getBookmarkViaAPI(c *apiClient, id model.BookmarkID) (model.Bookmark, error) {
	data, err := c.get(fmt.Sprintf("/api/bookmarks/%d", id))
	if err != nil {
		return model.Bookmark{}, err
	}
	var bm model.Bookmark
	if err := json.Unmarshal(data, &bm); err != nil {
		return model.Bookmark{}, err
	}
	return bm, nil
}

// Category commands

func TestCLICategoryList(t *testing.T) {
	c, _ := setupCLITest(t)
	if err := categoryList(c); err != nil {
		t.Fatalf("categoryList: %v", err)
	}
}

func TestCLICategoryCreateAndGet(t *testing.T) {
	c, _ := setupCLITest(t)
	if err := categoryCreate(c, []string{"--name", "Work"}); err != nil {
		t.Fatalf("categoryCreate: %v", err)
	}
	// Favorites=1, Work=2
	if err := categoryGet(c, []string{"--category-id", "2"}); err != nil {
		t.Fatalf("categoryGet: %v", err)
	}
}

func TestCLICategoryUpdate(t *testing.T) {
	c, ds := setupCLITest(t)
	cat, _ := ds.CreateCategory("Old")
	if err := categoryUpdate(c, []string{"--category-id", fmt.Sprint(cat.ID), "--name", "New"}); err != nil {
		t.Fatalf("categoryUpdate: %v", err)
	}
	updated, _ := ds.GetCategory(cat.ID)
	if updated.Name != "New" {
		t.Errorf("expected name 'New', got %q", updated.Name)
	}
}

func TestCLICategoryDelete(t *testing.T) {
	c, ds := setupCLITest(t)
	cat, _ := ds.CreateCategory("Delete Me")
	if err := categoryDelete(c, []string{"--category-id", fmt.Sprint(cat.ID)}); err != nil {
		t.Fatalf("categoryDelete: %v", err)
	}
	_, err := ds.GetCategory(cat.ID)
	if err == nil {
		t.Error("expected category to be deleted")
	}
}

// Bookmark commands

func TestCLIBookmarkCreate(t *testing.T) {
	c, _ := setupCLITest(t)
	if err := bookmarkCreate(c, []string{"--category-id", "1", "--name", "Google", "--url", "https://google.com", "--icon", "google", "--keyword", "search", "--keyword", "engine"}); err != nil {
		t.Fatalf("bookmarkCreate: %v", err)
	}
}

func TestCLIBookmarkGet(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	if err := bookmarkGet(c, []string{"--bookmark-id", fmt.Sprint(bm.ID)}); err != nil {
		t.Fatalf("bookmarkGet: %v", err)
	}
}

func TestCLIBookmarkUpdate(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	if err := bookmarkUpdate(c, []string{"--bookmark-id", fmt.Sprint(bm.ID), "--name", "Updated"}); err != nil {
		t.Fatalf("bookmarkUpdate: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestCLIBookmarkDelete(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	if err := bookmarkDelete(c, []string{"--bookmark-id", fmt.Sprint(bm.ID)}); err != nil {
		t.Fatalf("bookmarkDelete: %v", err)
	}
	_, err := ds.GetBookmark(bm.ID)
	if err == nil {
		t.Error("expected bookmark to be deleted")
	}
}

func TestCLIBookmarkMove(t *testing.T) {
	c, ds := setupCLITest(t)
	cat, _ := ds.CreateCategory("Target")
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	if err := bookmarkMove(c, []string{"--bookmark-id", fmt.Sprint(bm.ID), "--category-id", fmt.Sprint(cat.ID)}); err != nil {
		t.Fatalf("bookmarkMove: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if updated.CategoryID != cat.ID {
		t.Errorf("expected category_id %d, got %d", cat.ID, updated.CategoryID)
	}
}

func TestCLIBookmarkSearch(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	name := "SearchTarget"
	url := "https://search.example.com"
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Name: &name, URL: &url})

	if err := bookmarkSearch(c, []string{"--url", "https://search.example.com"}); err != nil {
		t.Fatalf("bookmarkSearch: %v", err)
	}
	if err := bookmarkSearch(c, []string{"--query", "SearchTarget"}); err != nil {
		t.Fatalf("bookmarkSearch by query: %v", err)
	}
}

// Keyword commands

func TestCLIKeywordAddAndList(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	id := fmt.Sprint(bm.ID)

	if err := keywordAdd(c, []string{"--bookmark-id", id, "alpha", "beta"}); err != nil {
		t.Fatalf("keywordAdd: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if len(updated.Keywords) != 2 || updated.Keywords[0] != "alpha" || updated.Keywords[1] != "beta" {
		t.Errorf("expected [alpha beta], got %v", updated.Keywords)
	}

	if err := keywordList(c, []string{"--bookmark-id", id}); err != nil {
		t.Fatalf("keywordList: %v", err)
	}
}

func TestCLIKeywordAddDeduplicates(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	kw := []string{"alpha"}
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Keywords: &kw})
	id := fmt.Sprint(bm.ID)

	if err := keywordAdd(c, []string{"--bookmark-id", id, "alpha", "beta"}); err != nil {
		t.Fatalf("keywordAdd: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if len(updated.Keywords) != 2 {
		t.Errorf("expected 2 keywords (deduplicated), got %v", updated.Keywords)
	}
}

func TestCLIKeywordDelete(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	kw := []string{"alpha", "beta", "gamma"}
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Keywords: &kw})
	id := fmt.Sprint(bm.ID)

	if err := keywordDelete(c, []string{"--bookmark-id", id, "beta"}); err != nil {
		t.Fatalf("keywordDelete: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if len(updated.Keywords) != 2 || updated.Keywords[0] != "alpha" || updated.Keywords[1] != "gamma" {
		t.Errorf("expected [alpha gamma], got %v", updated.Keywords)
	}
}

func TestCLIKeywordSet(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	id := fmt.Sprint(bm.ID)

	if err := keywordSet(c, []string{"--bookmark-id", id, "one", "two"}); err != nil {
		t.Fatalf("keywordSet: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if len(updated.Keywords) != 2 || updated.Keywords[0] != "one" || updated.Keywords[1] != "two" {
		t.Errorf("expected [one two], got %v", updated.Keywords)
	}
}

func TestCLIKeywordClear(t *testing.T) {
	c, ds := setupCLITest(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	kw := []string{"alpha", "beta"}
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Keywords: &kw})
	id := fmt.Sprint(bm.ID)

	if err := keywordClear(c, []string{"--bookmark-id", id}); err != nil {
		t.Fatalf("keywordClear: %v", err)
	}
	updated, _ := getBookmarkViaAPI(c, bm.ID)
	if len(updated.Keywords) != 0 {
		t.Errorf("expected empty keywords, got %v", updated.Keywords)
	}
}

// Icon commands

func TestCLIIconList(t *testing.T) {
	c, _ := setupCLITest(t)
	if err := iconList(c, []string{"--query", "home"}); err != nil {
		t.Fatalf("iconList: %v", err)
	}
}

func TestCLIIconListWithPagination(t *testing.T) {
	c, _ := setupCLITest(t)
	if err := iconList(c, []string{"--limit", "2", "--offset", "1"}); err != nil {
		t.Fatalf("iconList with pagination: %v", err)
	}
}
