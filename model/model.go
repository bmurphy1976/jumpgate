package model

// Typed IDs prevent passing a bookmark ID where a category ID is expected
type CategoryID int
type BookmarkID int

// Settings holds global dashboard configuration (single-row table)
type Settings struct {
	Title               string `json:"title"`
	WeatherLatitude     string `json:"weather_latitude"`
	WeatherLongitude    string `json:"weather_longitude"`
	WeatherUnit         string `json:"weather_unit"`
	WeatherCacheMinutes int    `json:"weather_cache_minutes"`
	DefaultPrivate      bool   `json:"default_private"`
	DefaultOpenInNewTab bool   `json:"default_open_in_new_tab"`
}

// Bookmark represents a single bookmark/link
type Bookmark struct {
	ID           BookmarkID `json:"id"`
	CategoryID   CategoryID `json:"category_id"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	MobileURL    string     `json:"mobile_url"`
	Icon         string     `json:"icon"`
	Enabled      bool       `json:"enabled"`
	OpenInNewTab *bool      `json:"open_in_new_tab"` // nil = inherit from global default
	Private      *bool      `json:"private"`         // nil = inherit from global default
	Position     int        `json:"position"`
	Keywords     []string   `json:"keywords"`
}

// Category represents a group of bookmarks
type Category struct {
	ID           CategoryID `json:"id"`
	Name         string     `json:"name"`
	Position     int        `json:"position"`
	Enabled      bool       `json:"enabled"`
	Private      *bool      `json:"private"`         // nil = inherit from global default
	OpenInNewTab *bool      `json:"open_in_new_tab"` // nil = inherit from global default
	IsFavorites  bool       `json:"is_favorites"`
	Bookmarks    []Bookmark `json:"bookmarks"`
}

// Update structs — pointer fields distinguish "not sent" from "set to this value"

type SettingsUpdate struct {
	Title               *string
	WeatherLatitude     *string
	WeatherLongitude    *string
	WeatherUnit         *string
	WeatherCacheMinutes *int
	DefaultPrivate      *bool
	DefaultOpenInNewTab *bool
}

type CategoryUpdate struct {
	Name *string
}

type BookmarkUpdate struct {
	Name      *string
	URL       *string
	MobileURL *string
	Icon      *string
	Keywords  *[]string // nil = no update
}
