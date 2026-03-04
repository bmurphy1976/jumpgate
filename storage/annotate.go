package storage

import (
	"dashboard/config"
	"dashboard/model"
	"fmt"
)

type annotatedDS struct {
	ds Datasource
}

// Annotated wraps a Datasource so every error includes the operation that failed.
func Annotated(ds Datasource) Datasource {
	return &annotatedDS{ds: ds}
}

func (a *annotatedDS) GetSettings() (model.Settings, error) {
	v, err := a.ds.GetSettings()
	if err != nil {
		return v, fmt.Errorf("get settings: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) UpdateSettings(update model.SettingsUpdate) error {
	if err := a.ds.UpdateSettings(update); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	return nil
}

func (a *annotatedDS) GetCategoriesWithBookmarks() ([]model.Category, error) {
	v, err := a.ds.GetCategoriesWithBookmarks()
	if err != nil {
		return v, fmt.Errorf("get categories: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) GetCategory(id model.CategoryID) (model.Category, error) {
	v, err := a.ds.GetCategory(id)
	if err != nil {
		return v, fmt.Errorf("get category %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) CreateCategory(name string) (model.Category, error) {
	v, err := a.ds.CreateCategory(name)
	if err != nil {
		return v, fmt.Errorf("create category: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) UpdateCategory(id model.CategoryID, update model.CategoryUpdate) error {
	if err := a.ds.UpdateCategory(id, update); err != nil {
		return fmt.Errorf("update category %d: %w", id, err)
	}
	return nil
}

func (a *annotatedDS) DeleteCategory(id model.CategoryID) error {
	if err := a.ds.DeleteCategory(id); err != nil {
		return fmt.Errorf("delete category %d: %w", id, err)
	}
	return nil
}

func (a *annotatedDS) ReorderCategories(ids []model.CategoryID) error {
	if err := a.ds.ReorderCategories(ids); err != nil {
		return fmt.Errorf("reorder categories: %w", err)
	}
	return nil
}

func (a *annotatedDS) GetBookmark(id model.BookmarkID) (model.Bookmark, error) {
	v, err := a.ds.GetBookmark(id)
	if err != nil {
		return v, fmt.Errorf("get bookmark %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) CreateBookmark(categoryID model.CategoryID) (model.Bookmark, error) {
	v, err := a.ds.CreateBookmark(categoryID)
	if err != nil {
		return v, fmt.Errorf("create bookmark: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) UpdateBookmark(id model.BookmarkID, update model.BookmarkUpdate) error {
	if err := a.ds.UpdateBookmark(id, update); err != nil {
		return fmt.Errorf("update bookmark %d: %w", id, err)
	}
	return nil
}

func (a *annotatedDS) DeleteBookmark(id model.BookmarkID) error {
	if err := a.ds.DeleteBookmark(id); err != nil {
		return fmt.Errorf("delete bookmark %d: %w", id, err)
	}
	return nil
}

func (a *annotatedDS) DuplicateBookmark(id model.BookmarkID) (model.Bookmark, error) {
	v, err := a.ds.DuplicateBookmark(id)
	if err != nil {
		return v, fmt.Errorf("duplicate bookmark %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) SortBookmarksAlpha(categoryID model.CategoryID) error {
	if err := a.ds.SortBookmarksAlpha(categoryID); err != nil {
		return fmt.Errorf("sort bookmarks in category %d: %w", categoryID, err)
	}
	return nil
}

func (a *annotatedDS) ReorderBookmarks(categoryID model.CategoryID, ids []model.BookmarkID) error {
	if err := a.ds.ReorderBookmarks(categoryID, ids); err != nil {
		return fmt.Errorf("reorder bookmarks in category %d: %w", categoryID, err)
	}
	return nil
}

func (a *annotatedDS) MoveBookmark(id model.BookmarkID, targetCategoryID model.CategoryID, position int) error {
	if err := a.ds.MoveBookmark(id, targetCategoryID, position); err != nil {
		return fmt.Errorf("move bookmark %d: %w", id, err)
	}
	return nil
}

func (a *annotatedDS) ToggleCategoryPrivate(id model.CategoryID) (*bool, error) {
	v, err := a.ds.ToggleCategoryPrivate(id)
	if err != nil {
		return v, fmt.Errorf("toggle category private %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) ToggleCategoryOpenInNewTab(id model.CategoryID) (*bool, error) {
	v, err := a.ds.ToggleCategoryOpenInNewTab(id)
	if err != nil {
		return v, fmt.Errorf("toggle category new-tab %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) ToggleBookmarkPrivate(id model.BookmarkID) (*bool, error) {
	v, err := a.ds.ToggleBookmarkPrivate(id)
	if err != nil {
		return v, fmt.Errorf("toggle bookmark private %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) ToggleBookmarkOpenInNewTab(id model.BookmarkID) (*bool, error) {
	v, err := a.ds.ToggleBookmarkOpenInNewTab(id)
	if err != nil {
		return v, fmt.Errorf("toggle bookmark new-tab %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) GetCategoryIDsWithInheritedPrivate() ([]model.CategoryID, error) {
	v, err := a.ds.GetCategoryIDsWithInheritedPrivate()
	if err != nil {
		return v, fmt.Errorf("get inherited-private category ids: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) GetCategoryIDsWithInheritedNewTab() ([]model.CategoryID, error) {
	v, err := a.ds.GetCategoryIDsWithInheritedNewTab()
	if err != nil {
		return v, fmt.Errorf("get inherited-new-tab category ids: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) GetBookmarkIDsWithInheritedPrivate() ([]model.BookmarkID, error) {
	v, err := a.ds.GetBookmarkIDsWithInheritedPrivate()
	if err != nil {
		return v, fmt.Errorf("get inherited-private bookmark ids: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) GetBookmarkIDsWithInheritedNewTab() ([]model.BookmarkID, error) {
	v, err := a.ds.GetBookmarkIDsWithInheritedNewTab()
	if err != nil {
		return v, fmt.Errorf("get inherited-new-tab bookmark ids: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) GetBookmarkIDsInCategoryWithInheritedPrivate(categoryID model.CategoryID) ([]model.BookmarkID, error) {
	v, err := a.ds.GetBookmarkIDsInCategoryWithInheritedPrivate(categoryID)
	if err != nil {
		return v, fmt.Errorf("get inherited-private bookmark ids in category %d: %w", categoryID, err)
	}
	return v, nil
}

func (a *annotatedDS) GetBookmarkIDsInCategoryWithInheritedNewTab(categoryID model.CategoryID) ([]model.BookmarkID, error) {
	v, err := a.ds.GetBookmarkIDsInCategoryWithInheritedNewTab(categoryID)
	if err != nil {
		return v, fmt.Errorf("get inherited-new-tab bookmark ids in category %d: %w", categoryID, err)
	}
	return v, nil
}

func (a *annotatedDS) ToggleCategoryEnabled(id model.CategoryID) (bool, error) {
	v, err := a.ds.ToggleCategoryEnabled(id)
	if err != nil {
		return v, fmt.Errorf("toggle category enabled %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) ToggleBookmarkEnabled(id model.BookmarkID) (bool, error) {
	v, err := a.ds.ToggleBookmarkEnabled(id)
	if err != nil {
		return v, fmt.Errorf("toggle bookmark enabled %d: %w", id, err)
	}
	return v, nil
}

func (a *annotatedDS) ImportConfig(cfg config.Config) error {
	if err := a.ds.ImportConfig(cfg); err != nil {
		return fmt.Errorf("import config: %w", err)
	}
	return nil
}

func (a *annotatedDS) ExportConfig() (config.Config, error) {
	v, err := a.ds.ExportConfig()
	if err != nil {
		return v, fmt.Errorf("export config: %w", err)
	}
	return v, nil
}

func (a *annotatedDS) Close() error {
	if err := a.ds.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}
