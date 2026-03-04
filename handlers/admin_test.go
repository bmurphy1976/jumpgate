package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"dashboard/icons"
	"dashboard/model"
	"dashboard/storage"

	"github.com/labstack/echo/v5"
)

func setupTestDB(t *testing.T) *storage.SQLiteDB {
	t.Helper()
	ds, err := storage.NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ds.Close() })
	return ds
}

func setupTestServer(t *testing.T) (*echo.Echo, *storage.SQLiteDB) {
	t.Helper()
	ds := setupTestDB(t)
	resolver := func(c *echo.Context) (storage.Datasource, error) {
		return ds, nil
	}
	e := echo.New()
	SetupAdminRoutes(e, resolver, &icons.Loader{Icons: []string{"home", "star", "newspaper"}}, false, false)
	return e, ds
}

func authGet(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Authorized-User", "test")
	return req
}

func authForm(method, path string, values url.Values) *http.Request {
	body := ""
	if values != nil {
		body = values.Encode()
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorized-User", "test")
	return req
}

func serve(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// Auth

func TestAdminRequiresAuth(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAdminIndex(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authGet("/admin"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("expected HTML response")
	}
}

// Settings

func TestUpdateSettings(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPut, "/admin/settings", url.Values{
		"title": {"Test Dashboard"},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestToggleDefaultPrivate(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPost, "/admin/settings/toggle/private", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected HTML response")
	}
}

func TestToggleDefaultNewTab(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPost, "/admin/settings/toggle/new-tab", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected HTML response")
	}
}

// Categories

func TestCreateCategory(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPost, "/admin/categories", url.Values{}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "category-card") {
		t.Error("expected category card HTML")
	}
}

func TestUpdateCategory(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Old Name")
	rec := serve(e, authForm(http.MethodPut, fmt.Sprintf("/admin/categories/%d", cat.ID), url.Values{
		"name": {"New Name"},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestDeleteCategory(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Delete Me")
	rec := serve(e, authForm(http.MethodDelete, fmt.Sprintf("/admin/categories/%d", cat.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReorderCategories(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Second")
	order, _ := json.Marshal([]int{int(cat.ID), 1})
	rec := serve(e, authForm(http.MethodPost, "/admin/categories/reorder", url.Values{
		"order": {string(order)},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSortBookmarks(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Sort Test")
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/categories/%d/sort", cat.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "category-card") {
		t.Error("expected category card HTML")
	}
}

func TestToggleCategoryPrivateRoute(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Toggle Test")
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/categories/%d/toggle/private", cat.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected toggle HTML")
	}
}

// Bookmarks

func TestCreateBookmark(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPost, "/admin/bookmarks", url.Values{
		"category_id": {"1"},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bookmark-entry") {
		t.Error("expected bookmark row HTML")
	}
}

func TestUpdateBookmark(t *testing.T) {
	e, ds := setupTestServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, authForm(http.MethodPut, fmt.Sprintf("/admin/bookmarks/%d", bm.ID), url.Values{
		"name": {"Updated"},
		"url":  {"https://example.com"},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestDeleteBookmark(t *testing.T) {
	e, ds := setupTestServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, authForm(http.MethodDelete, fmt.Sprintf("/admin/bookmarks/%d", bm.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestDuplicateBookmark(t *testing.T) {
	e, ds := setupTestServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/bookmarks/%d/duplicate", bm.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bookmark-entry") {
		t.Error("expected bookmark row HTML")
	}
}

func TestToggleBookmarkPrivateRoute(t *testing.T) {
	e, ds := setupTestServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/bookmarks/%d/toggle/private", bm.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected toggle HTML")
	}
}

func TestToggleBookmarkNewTabRoute(t *testing.T) {
	e, ds := setupTestServer(t)
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/bookmarks/%d/toggle/new-tab", bm.ID), nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected toggle HTML")
	}
}

func TestMoveBookmark(t *testing.T) {
	e, ds := setupTestServer(t)
	cat, _ := ds.CreateCategory("Target")
	bm, _ := ds.CreateBookmark(model.CategoryID(1))
	order, _ := json.Marshal([]int{int(bm.ID)})
	rec := serve(e, authForm(http.MethodPost, fmt.Sprintf("/admin/bookmarks/%d/move", bm.ID), url.Values{
		"target_category_id": {fmt.Sprintf("%d", cat.ID)},
		"order":              {string(order)},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReorderBookmarks(t *testing.T) {
	e, ds := setupTestServer(t)
	bm1, _ := ds.CreateBookmark(model.CategoryID(1))
	bm2, _ := ds.CreateBookmark(model.CategoryID(1))
	order, _ := json.Marshal([]int{int(bm2.ID), int(bm1.ID)})
	rec := serve(e, authForm(http.MethodPost, "/admin/bookmarks/reorder", url.Values{
		"category_id": {"1"},
		"order":       {string(order)},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSearchIcons(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authGet("/admin/icons?q=home"))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected icon grid HTML")
	}
}

// Invalid input

func TestInvalidCategoryID(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPut, "/admin/categories/invalid", url.Values{"name": {"x"}}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestInvalidBookmarkID(t *testing.T) {
	e, _ := setupTestServer(t)
	rec := serve(e, authForm(http.MethodPut, "/admin/bookmarks/invalid", url.Values{"name": {"x"}}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
