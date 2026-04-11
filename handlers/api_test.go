package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"dashboard/config"
	"dashboard/icons"
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"dashboard/storage"

	"github.com/labstack/echo/v5"
)

func setupAPIServer(t *testing.T) (*echo.Echo, *storage.SQLiteDB) {
	t.Helper()
	ds := setupTestDB(t)
	resolver := func(c *echo.Context) (storage.Datasource, error) {
		return ds, nil
	}
	e := echo.New()
	apiCfg := config.APIConfig{
		Swagger: true,
		Tokens: config.APITokens{
			ReadWrite: []string{"rw-token"},
			ReadOnly:  []string{"ro-token"},
		},
	}
	SetupAPIRoutes(e, resolver, &icons.Loader{Icons: []string{"home", "star", "newspaper", "github", "google"}}, apiCfg)
	return e, ds
}

func apiReq(method, path string, body any, token string) *http.Request {
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func setBuildInfo(t *testing.T, release, commit string) {
	t.Helper()
	oldRelease := buildinfo.ReleaseVersion
	oldCommit := buildinfo.Commit
	buildinfo.ReleaseVersion = release
	buildinfo.Commit = commit
	t.Cleanup(func() {
		buildinfo.ReleaseVersion = oldRelease
		buildinfo.Commit = oldCommit
	})
}

// Auth tests

func TestAPIRequiresToken(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/categories", nil, ""))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIInvalidToken(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/categories", nil, "bad-token"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIReadOnlyCanRead(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/categories", nil, "ro-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIReadOnlyCannotWrite(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodPost, "/api/categories", map[string]string{"name": "Test"}, "ro-token"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAPIReadWriteCanWrite(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodPost, "/api/categories", map[string]string{"name": "Test"}, "rw-token"))
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestAPIIndexIncludesVersion(t *testing.T) {
	setBuildInfo(t, "2026.04.0", "04fc78e")

	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api", nil, ""))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var index map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if index["version"] != "2026.04.0+04fc78e" {
		t.Errorf("expected stamped version, got %v", index["version"])
	}
}

// Category CRUD

func TestAPIListCategories(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/categories", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var cats []model.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &cats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(cats) == 0 {
		t.Error("expected at least one category (Favorites)")
	}
}

func TestAPICreateCategory(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodPost, "/api/categories", map[string]string{"name": "Work"}, "rw-token"))
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	var cat model.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &cat); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if cat.Name != "Work" {
		t.Errorf("expected name 'Work', got %q", cat.Name)
	}
}

func TestAPICreateCategoryRequiresName(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodPost, "/api/categories", map[string]string{}, "rw-token"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAPIGetCategory(t *testing.T) {
	e, ds := setupAPIServer(t)
	cat, _ := ds.CreateCategory("Test")
	rec := serve(e, apiReq(http.MethodGet, fmt.Sprintf("/api/categories/%d", cat.ID), nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIUpdateCategory(t *testing.T) {
	e, ds := setupAPIServer(t)
	cat, _ := ds.CreateCategory("Old")
	rec := serve(e, apiReq(http.MethodPut, fmt.Sprintf("/api/categories/%d", cat.ID), map[string]string{"name": "New"}, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var updated model.Category
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Name != "New" {
		t.Errorf("expected name 'New', got %q", updated.Name)
	}
}

func TestAPIDeleteCategory(t *testing.T) {
	e, ds := setupAPIServer(t)
	cat, _ := ds.CreateCategory("Delete Me")
	rec := serve(e, apiReq(http.MethodDelete, fmt.Sprintf("/api/categories/%d", cat.ID), nil, "rw-token"))
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

// Bookmark CRUD

func TestAPICreateBookmark(t *testing.T) {
	e, _ := setupAPIServer(t)
	body := map[string]any{
		"category_id": 1,
		"name":        "Google",
		"url":         "https://google.com",
		"icon":        "google",
		"keywords":    []string{"search", "engine"},
	}
	rec := serve(e, apiReq(http.MethodPost, "/api/bookmarks", body, "rw-token"))
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var bm model.Bookmark
	json.Unmarshal(rec.Body.Bytes(), &bm)
	if bm.Name != "Google" {
		t.Errorf("expected name 'Google', got %q", bm.Name)
	}
	if bm.Icon != "google" {
		t.Errorf("expected icon 'google', got %q", bm.Icon)
	}
	if len(bm.Keywords) != 2 || bm.Keywords[0] != "search" {
		t.Errorf("expected keywords [search engine], got %v", bm.Keywords)
	}
}

func TestAPICreateBookmarkRequiresCategoryID(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodPost, "/api/bookmarks", map[string]string{"name": "Test"}, "rw-token"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAPIGetBookmark(t *testing.T) {
	e, ds := setupAPIServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, apiReq(http.MethodGet, fmt.Sprintf("/api/bookmarks/%d", bm.ID), nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIUpdateBookmark(t *testing.T) {
	e, ds := setupAPIServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	body := map[string]any{"name": "Updated", "url": "https://example.com"}
	rec := serve(e, apiReq(http.MethodPut, fmt.Sprintf("/api/bookmarks/%d", bm.ID), body, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var updated model.Bookmark
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestAPIDeleteBookmark(t *testing.T) {
	e, ds := setupAPIServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, apiReq(http.MethodDelete, fmt.Sprintf("/api/bookmarks/%d", bm.ID), nil, "rw-token"))
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestAPIMoveBookmark(t *testing.T) {
	e, ds := setupAPIServer(t)
	cat, _ := ds.CreateCategory("Target")
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	body := map[string]any{"category_id": int(cat.ID), "position": 0}
	rec := serve(e, apiReq(http.MethodPost, fmt.Sprintf("/api/bookmarks/%d/move", bm.ID), body, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Search

func TestAPISearchBookmarks(t *testing.T) {
	e, ds := setupAPIServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	name := "SearchTest"
	url := "https://searchtest.example.com"
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Name: &name, URL: &url})

	rec := serve(e, apiReq(http.MethodGet, "/api/bookmarks/search?url=https://searchtest.example.com", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var results []model.Bookmark
	json.Unmarshal(rec.Body.Bytes(), &results)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestAPISearchBookmarksEmpty(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/bookmarks/search?q=nonexistent", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var results []model.Bookmark
	json.Unmarshal(rec.Body.Bytes(), &results)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// Icons

func TestAPIListIcons(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/icons?q=home", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)
	if _, ok := result["icons"]; !ok {
		t.Error("expected icons field in response")
	}
	if _, ok := result["total"]; !ok {
		t.Error("expected total field in response")
	}
}

// OpenAPI + Swagger

func TestAPIOpenAPISpec(t *testing.T) {
	setBuildInfo(t, "2026.04.0", "04fc78e")

	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/openapi.json", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var spec map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("failed to decode OpenAPI spec: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", spec["openapi"])
	}
	info := spec["info"].(map[string]any)
	if info["version"] != "2026.04.0+04fc78e" {
		t.Errorf("expected stamped version, got %v", info["version"])
	}
}

func TestAPISwaggerUI(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/docs", nil, "rw-token"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected HTML response")
	}
}

// Invalid input

func TestAPIInvalidCategoryID(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/categories/invalid", nil, "rw-token"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAPIInvalidBookmarkID(t *testing.T) {
	e, _ := setupAPIServer(t)
	rec := serve(e, apiReq(http.MethodGet, "/api/bookmarks/invalid", nil, "rw-token"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
