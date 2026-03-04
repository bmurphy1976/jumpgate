package handlers

import (
	"dashboard/model"
	"dashboard/views"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildDashCategoriesAllVisible(t *testing.T) {
	cats := []model.Category{
		{ID: 1, Name: "Favorites", Enabled: true, IsFavorites: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Google", URL: "https://google.com", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, true, false)
	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}
	if len(result[0].Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(result[0].Links))
	}
}

func TestBuildDashCategoriesFilterDisabledCategory(t *testing.T) {
	cats := []model.Category{
		{ID: 1, Name: "Disabled", Enabled: false, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Test", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, true, false)
	if len(result) != 0 {
		t.Errorf("expected 0 categories, got %d", len(result))
	}
}

func TestBuildDashCategoriesFilterPrivateCategoryUnauthorized(t *testing.T) {
	priv := true
	cats := []model.Category{
		{ID: 1, Name: "Secret", Enabled: true, Private: &priv, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Test", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, false, false)
	if len(result) != 0 {
		t.Errorf("expected 0 categories for unauthorized, got %d", len(result))
	}
}

func TestBuildDashCategoriesFilterPrivateBookmarkUnauthorized(t *testing.T) {
	priv := true
	cats := []model.Category{
		{ID: 1, Name: "Public", Enabled: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Secret", Enabled: true, Private: &priv},
			{ID: 2, Name: "Visible", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, false, false)
	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}
	if len(result[0].Links) != 1 {
		t.Errorf("expected 1 visible link, got %d", len(result[0].Links))
	}
	if result[0].Links[0].Name != "Visible" {
		t.Errorf("expected 'Visible', got %q", result[0].Links[0].Name)
	}
}

func TestBuildDashCategoriesInheritedPrivacy(t *testing.T) {
	// Category private=nil, settings default_private=true → category inherits private
	cats := []model.Category{
		{ID: 1, Name: "Inherits", Enabled: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Test", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: true}
	result := buildDashCategories(cats, settings, false, false)
	if len(result) != 0 {
		t.Errorf("expected 0 categories (inherited private), got %d", len(result))
	}
}

func TestBuildDashCategoriesMobileURL(t *testing.T) {
	cats := []model.Category{
		{ID: 1, Name: "Test", Enabled: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "App", URL: "https://example.com", MobileURL: "https://m.example.com", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, true, true)
	if result[0].Links[0].URL != "https://m.example.com" {
		t.Errorf("expected mobile URL, got %q", result[0].Links[0].URL)
	}
	// Desktop should use regular URL
	result = buildDashCategories(cats, settings, true, false)
	if result[0].Links[0].URL != "https://example.com" {
		t.Errorf("expected desktop URL, got %q", result[0].Links[0].URL)
	}
}

func TestBuildDashCategoriesOpenInNewTab(t *testing.T) {
	newTab := true
	cats := []model.Category{
		{ID: 1, Name: "Test", Enabled: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Explicit", Enabled: true, OpenInNewTab: &newTab},
			{ID: 2, Name: "Inherit", Enabled: true},
		}},
	}
	settings := model.Settings{DefaultPrivate: false, DefaultOpenInNewTab: false}
	result := buildDashCategories(cats, settings, true, false)
	if !result[0].Links[0].OpenInNewTab {
		t.Error("expected OpenInNewTab=true for explicit bookmark")
	}
	if result[0].Links[1].OpenInNewTab {
		t.Error("expected OpenInNewTab=false for inherited bookmark")
	}
}

func TestBuildDashCategoriesFilterDisabledBookmark(t *testing.T) {
	cats := []model.Category{
		{ID: 1, Name: "Test", Enabled: true, Bookmarks: []model.Bookmark{
			{ID: 1, Name: "Active", Enabled: true},
			{ID: 2, Name: "Disabled", Enabled: false},
		}},
	}
	settings := model.Settings{DefaultPrivate: false}
	result := buildDashCategories(cats, settings, true, false)
	if len(result[0].Links) != 1 {
		t.Errorf("expected 1 link (disabled filtered), got %d", len(result[0].Links))
	}
}

// isAuthorized tests

func TestIsAuthorizedXAuthorizedUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Authorized-User", "test")
	if !isAuthorized(req) {
		t.Error("expected authorized with X-Authorized-User")
	}
}

func TestIsAuthorizedXUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User", "test")
	if !isAuthorized(req) {
		t.Error("expected authorized with X-User")
	}
}

func TestIsAuthorizedXRemoteUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Remote-User", "test")
	if !isAuthorized(req) {
		t.Error("expected authorized with X-Remote-User")
	}
}

func TestIsAuthorizedNoHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if isAuthorized(req) {
		t.Error("expected unauthorized with no headers")
	}
}

// isMobile tests

func TestIsMobileWithMobileUA(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)")
	if !isMobile(req) {
		t.Error("expected mobile detection for iPhone UA")
	}
}

func TestIsMobileWithDesktopUA(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) Chrome/120")
	if isMobile(req) {
		t.Error("expected non-mobile for desktop UA")
	}
}

// extractDomain tests

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"https://www.google.com/search?q=test", "google.com"},
		{"https://github.com/user/repo", "github.com"},
		{"http://localhost:8080/api", "localhost"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractDomain(tt.input)
		if got != tt.expected {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// weatherIconName tests

func TestWeatherIconName(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "weather-sunny"},
		{3, "weather-cloudy"},
		{61, "weather-rainy"},
		{95, "weather-lightning"},
		{999, "weather-cloudy"}, // unknown code
	}
	for _, tt := range tests {
		got := weatherIconName(tt.code)
		if got != tt.want {
			t.Errorf("weatherIconName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// parseWeatherCoords tests

func TestParseWeatherCoords(t *testing.T) {
	settings := model.Settings{WeatherLatitude: "40.7128", WeatherLongitude: "-74.0060"}
	req := httptest.NewRequest("GET", "/", nil)
	lat, lon := parseWeatherCoords(req, settings)
	if lat != 40.7128 {
		t.Errorf("expected lat 40.7128, got %f", lat)
	}
	if lon != -74.0060 {
		t.Errorf("expected lon -74.0060, got %f", lon)
	}
}

func TestParseWeatherCoordsCookieOverride(t *testing.T) {
	settings := model.Settings{WeatherLatitude: "0", WeatherLongitude: "0"}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "weather_lat", Value: "51.5074"})
	req.AddCookie(&http.Cookie{Name: "weather_lon", Value: "-0.1278"})
	lat, lon := parseWeatherCoords(req, settings)
	if lat != 51.5074 {
		t.Errorf("expected lat 51.5074, got %f", lat)
	}
	if lon != -0.1278 {
		t.Errorf("expected lon -0.1278, got %f", lon)
	}
}

// collectIconNames tests

func TestCollectIconNames(t *testing.T) {
	cats := []views.DashCategory{
		{Links: []views.DashBookmark{
			{Icon: "home"},
			{Icon: "star"},
			{Icon: "home"}, // duplicate
		}},
	}
	names := collectIconNames(cats, false)
	if len(names) != 2 {
		t.Errorf("expected 2 unique icons, got %d", len(names))
	}
}

func TestCollectIconNamesWithWeather(t *testing.T) {
	cats := []views.DashCategory{}
	names := collectIconNames(cats, true)
	if len(names) != 11 {
		t.Errorf("expected 11 weather icons, got %d", len(names))
	}
}

// svgToSymbol test

func TestSvgToSymbol(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000" d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>`
	result := svgToSymbol("home", svg)
	if result == "" {
		t.Fatal("expected non-empty symbol")
	}
	// Should have symbol ID and viewBox, fill attribute should be stripped
	expected := `<symbol id="mdi-home" viewBox="0 0 24 24"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></symbol>`
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSvgToSymbolInvalid(t *testing.T) {
	result := svgToSymbol("bad", "not an svg")
	if result != "" {
		t.Errorf("expected empty for invalid SVG, got %q", result)
	}
}
