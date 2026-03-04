package model

// Typed IDs prevent passing a bookmark ID where a category ID is expected
type CategoryID int
type BookmarkID int

// Settings holds global dashboard configuration (single-row table)
type Settings struct {
	Title               string
	WeatherLatitude     string
	WeatherLongitude    string
	WeatherUnit         string
	WeatherCacheMinutes int
	DefaultPrivate      bool
	DefaultOpenInNewTab bool
}

// Bookmark represents a single bookmark/link
type Bookmark struct {
	ID           BookmarkID
	CategoryID   CategoryID
	Name         string
	URL          string
	MobileURL    string
	Icon         string
	Enabled      bool
	OpenInNewTab *bool // nil = inherit from global default
	Private      *bool // nil = inherit from global default
	Position     int
	Keywords     []string
}

// Category represents a group of bookmarks
type Category struct {
	ID           CategoryID
	Name         string
	Position     int
	Enabled      bool
	Private      *bool // nil = inherit from global default
	OpenInNewTab *bool // nil = inherit from global default
	IsFavorites  bool
	Bookmarks    []Bookmark
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
	Keywords  *string // space-separated; nil = no update
}
