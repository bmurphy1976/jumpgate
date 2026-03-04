package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRequireAuthAllows(t *testing.T) {
	e := echo.New()
	inner := func(c *echo.Context) error {
		return (*c).String(http.StatusOK, "ok")
	}
	handler := requireAuth(inner)

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("X-Authorized-User", "test")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAuthBlocks(t *testing.T) {
	e := echo.New()
	inner := func(c *echo.Context) error {
		return (*c).String(http.StatusOK, "ok")
	}
	handler := requireAuth(inner)

	req := httptest.NewRequest("GET", "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if he.Code != 401 {
		t.Errorf("expected 401, got %d", he.Code)
	}
}
