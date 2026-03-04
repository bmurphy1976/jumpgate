package storage

import (
	"dashboard/config"
	"dashboard/model"
)

// Datasource defines the contract all storage backends must implement
type Datasource interface {
	// Settings
	GetSettings() (model.Settings, error)
	UpdateSettings(update model.SettingsUpdate) error

	// Categories
	GetCategoriesWithBookmarks() ([]model.Category, error)
	GetCategory(id model.CategoryID) (model.Category, error)
	CreateCategory(name string) (model.Category, error)
	UpdateCategory(id model.CategoryID, update model.CategoryUpdate) error
	DeleteCategory(id model.CategoryID) error
	ReorderCategories(ids []model.CategoryID) error

	// Bookmarks
	GetBookmark(id model.BookmarkID) (model.Bookmark, error)
	CreateBookmark(categoryID model.CategoryID) (model.Bookmark, error)
	UpdateBookmark(id model.BookmarkID, update model.BookmarkUpdate) error
	DeleteBookmark(id model.BookmarkID) error
	DuplicateBookmark(id model.BookmarkID) (model.Bookmark, error)
	SortBookmarksAlpha(categoryID model.CategoryID) error
	ReorderBookmarks(categoryID model.CategoryID, ids []model.BookmarkID) error
	MoveBookmark(id model.BookmarkID, targetCategoryID model.CategoryID, position int) error

	// Toggle cycling (nil → true → false → nil)
	ToggleCategoryPrivate(id model.CategoryID) (*bool, error)
	ToggleCategoryOpenInNewTab(id model.CategoryID) (*bool, error)
	ToggleBookmarkPrivate(id model.BookmarkID) (*bool, error)
	ToggleBookmarkOpenInNewTab(id model.BookmarkID) (*bool, error)

	// IDs of items currently inheriting the default value
	GetCategoryIDsWithInheritedPrivate() ([]model.CategoryID, error)
	GetCategoryIDsWithInheritedNewTab() ([]model.CategoryID, error)
	GetBookmarkIDsWithInheritedPrivate() ([]model.BookmarkID, error)
	GetBookmarkIDsWithInheritedNewTab() ([]model.BookmarkID, error)
	GetBookmarkIDsInCategoryWithInheritedPrivate(categoryID model.CategoryID) ([]model.BookmarkID, error)
	GetBookmarkIDsInCategoryWithInheritedNewTab(categoryID model.CategoryID) ([]model.BookmarkID, error)

	// Toggle enabled (true ↔ false)
	ToggleCategoryEnabled(id model.CategoryID) (bool, error)
	ToggleBookmarkEnabled(id model.BookmarkID) (bool, error)

	// Import / Export
	ImportConfig(cfg config.Config) error
	ExportConfig() (config.Config, error)

	// Lifecycle
	Close() error
}
