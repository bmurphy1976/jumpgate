package handlers

import (
	"dashboard/common"
	"dashboard/icons"
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"dashboard/storage"
	"dashboard/views"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

type AdminHandler struct {
	ds       DSResolver
	icons    *icons.Loader
	demoMode bool
}

// formStr returns a *string if the key is present in form data (even if empty), nil if absent.
func formStr(c *echo.Context, key string) *string {
	_ = (*c).Request().ParseForm()
	form := (*c).Request().Form
	if _, ok := form[key]; ok {
		v := form.Get(key)
		return &v
	}
	return nil
}

func parseIntIDs(raw string) ([]int, error) {
	var ids []int
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		var strIDs []string
		if err := json.Unmarshal([]byte(raw), &strIDs); err != nil {
			return nil, err
		}
		ids = make([]int, len(strIDs))
		for i, s := range strIDs {
			id, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}
			ids[i] = id
		}
	}
	return ids, nil
}

func SetupAdminRoutes(e *echo.Echo, ds DSResolver, il *icons.Loader, noAuth bool, demoMode bool) {
	h := &AdminHandler{ds: ds, icons: il, demoMode: demoMode}
	var admin *echo.Group
	if noAuth {
		admin = e.Group("/admin")
	} else {
		admin = e.Group("/admin", requireAuth)
	}

	admin.GET("", h.index)
	admin.PUT("/settings", h.updateSettings)
	admin.POST("/settings/toggle/private", h.toggleDefaultPrivate)
	admin.POST("/settings/toggle/new-tab", h.toggleDefaultNewTab)
	admin.POST("/categories", h.createCategory)
	admin.PUT("/categories/:id", h.updateCategory)
	admin.DELETE("/categories/:id", h.deleteCategory)
	admin.POST("/categories/reorder", h.reorderCategories)
	admin.POST("/categories/:id/sort", h.sortBookmarks)
	admin.POST("/categories/:id/toggle/private", h.toggleCategoryPrivate)
	admin.POST("/categories/:id/toggle/new-tab", h.toggleCategoryNewTab)
	admin.POST("/categories/:id/toggle/enabled", h.toggleCategoryEnabled)
	admin.POST("/bookmarks", h.createBookmark)
	admin.PUT("/bookmarks/:id", h.updateBookmark)
	admin.DELETE("/bookmarks/:id", h.deleteBookmark)
	admin.POST("/bookmarks/:id/duplicate", h.duplicateBookmark)
	admin.POST("/bookmarks/:id/toggle/private", h.toggleBookmarkPrivate)
	admin.POST("/bookmarks/:id/toggle/new-tab", h.toggleBookmarkNewTab)
	admin.POST("/bookmarks/:id/toggle/enabled", h.toggleBookmarkEnabled)
	admin.POST("/bookmarks/:id/move", h.moveBookmark)
	admin.POST("/bookmarks/reorder", h.reorderBookmarks)
	admin.GET("/icons", h.searchIcons)
}

func (h *AdminHandler) index(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	categories, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		return err
	}
	r := (*c).Request()
	themeData := resolveTheme(r)
	weather := resolveWeather(r, settings)
	authorized := isAuthorized(r) || h.demoMode
	layout := views.AdminLayoutData{
		Title:        settings.Title,
		Version:      buildinfo.ServiceVersion(),
		Theme:        themeData,
		Weather:      weather,
		IsAuthorized: authorized,
		DemoMode:     h.demoMode,
		AdminCSSHash: fileHash("css/admin.css"),
		AdminJSHash:  fileHash("js/admin.js"),
		ThemeJSRaw:   themeJS,
		Deps: views.CDNDeps{
			HtmxURL:       common.HTMXURL,
			SortableJSURL: common.SortableJSURL,
			MDIFontCSSURL: common.MDIFontCSSURL,
		},
	}
	return views.AdminPage(settings, categories, layout).Render(r.Context(), (*c).Response())
}

func (h *AdminHandler) updateSettings(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	var update model.SettingsUpdate
	update.Title = formStr(c, "title")
	update.WeatherLatitude = formStr(c, "weather_latitude")
	update.WeatherLongitude = formStr(c, "weather_longitude")
	update.WeatherUnit = formStr(c, "weather_unit")

	if raw := formStr(c, "weather_cache_minutes"); raw != nil {
		if v, err := strconv.Atoi(*raw); err == nil {
			update.WeatherCacheMinutes = &v
		}
	}
	if raw := formStr(c, "default_private"); raw != nil {
		v := *raw == "true"
		update.DefaultPrivate = &v
	}
	if raw := formStr(c, "default_open_in_new_tab"); raw != nil {
		v := *raw == "true"
		update.DefaultOpenInNewTab = &v
	}

	if err := ds.UpdateSettings(update); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) toggleDefaultPrivate(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	newValue := !settings.DefaultPrivate
	update := model.SettingsUpdate{DefaultPrivate: &newValue}
	if err := ds.UpdateSettings(update); err != nil {
		return err
	}
	ctx := (*c).Request().Context()
	w := (*c).Response()
	if err := views.DefaultToggle("default_private", "/admin/settings/toggle/private", newValue).Render(ctx, w); err != nil {
		return err
	}
	catIDs, err := ds.GetCategoryIDsWithInheritedPrivate()
	if err != nil {
		return err
	}
	for _, catID := range catIDs {
		if err := views.HeaderToggle(fmt.Sprintf("cat-%d-private-toggle", catID), fmt.Sprintf("/admin/categories/%d/toggle/private", catID), nil, newValue, true).Render(ctx, w); err != nil {
			return err
		}
	}
	bmIDs, err := ds.GetBookmarkIDsWithInheritedPrivate()
	if err != nil {
		return err
	}
	for _, bmID := range bmIDs {
		if err := views.BookmarkHeaderToggle("private", fmt.Sprintf("bm-%d-private-toggle", bmID), fmt.Sprintf("/admin/bookmarks/%d/toggle/private", bmID), nil, newValue, true).Render(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminHandler) toggleDefaultNewTab(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	newValue := !settings.DefaultOpenInNewTab
	update := model.SettingsUpdate{DefaultOpenInNewTab: &newValue}
	if err := ds.UpdateSettings(update); err != nil {
		return err
	}
	ctx := (*c).Request().Context()
	w := (*c).Response()
	if err := views.DefaultToggle("default_open_in_new_tab", "/admin/settings/toggle/new-tab", newValue).Render(ctx, w); err != nil {
		return err
	}
	catIDs, err := ds.GetCategoryIDsWithInheritedNewTab()
	if err != nil {
		return err
	}
	for _, catID := range catIDs {
		if err := views.BookmarkHeaderToggle("newtab", fmt.Sprintf("cat-%d-newtab-toggle", catID), fmt.Sprintf("/admin/categories/%d/toggle/new-tab", catID), nil, newValue, true).Render(ctx, w); err != nil {
			return err
		}
	}
	bmIDs, err := ds.GetBookmarkIDsWithInheritedNewTab()
	if err != nil {
		return err
	}
	for _, bmID := range bmIDs {
		if err := views.BookmarkHeaderToggle("newtab", fmt.Sprintf("bm-%d-newtab-toggle", bmID), fmt.Sprintf("/admin/bookmarks/%d/toggle/new-tab", bmID), nil, newValue, true).Render(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminHandler) createCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	name := (*c).FormValue("name")
	if name == "" {
		name = "New Category"
	}
	cat, err := ds.CreateCategory(name)
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	return views.CategoryCard(cat, settings).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) updateCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	update := model.CategoryUpdate{
		Name: formStr(c, "name"),
	}
	if err := ds.UpdateCategory(model.CategoryID(id), update); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) deleteCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	if err := ds.DeleteCategory(model.CategoryID(id)); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) reorderCategories(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	ids, err := parseIntIDs((*c).FormValue("order"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid order")
	}

	categoryIDs := make([]model.CategoryID, len(ids))
	for i, id := range ids {
		categoryIDs[i] = model.CategoryID(id)
	}
	if err := ds.ReorderCategories(categoryIDs); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) sortBookmarks(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	if err := ds.SortBookmarksAlpha(model.CategoryID(id)); err != nil {
		return err
	}
	cat, err := ds.GetCategory(model.CategoryID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	return views.CategoryCard(cat, settings).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) toggleCategoryPrivate(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	newValue, err := ds.ToggleCategoryPrivate(model.CategoryID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	ctx := (*c).Request().Context()
	w := (*c).Response()
	if err := views.HeaderToggle(fmt.Sprintf("cat-%d-private-toggle", id), fmt.Sprintf("/admin/categories/%d/toggle/private", id), newValue, settings.DefaultPrivate, false).Render(ctx, w); err != nil {
		return err
	}
	bmIDs, err := ds.GetBookmarkIDsInCategoryWithInheritedPrivate(model.CategoryID(id))
	if err != nil {
		return err
	}
	effPrivate := settings.DefaultPrivate
	if newValue != nil {
		effPrivate = *newValue
	}
	for _, bmID := range bmIDs {
		if err := views.BookmarkHeaderToggle("private", fmt.Sprintf("bm-%d-private-toggle", bmID), fmt.Sprintf("/admin/bookmarks/%d/toggle/private", bmID), nil, effPrivate, true).Render(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminHandler) toggleCategoryEnabled(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	newValue, err := ds.ToggleCategoryEnabled(model.CategoryID(id))
	if err != nil {
		return err
	}
	return views.CategoryEnabledToggle(fmt.Sprintf("/admin/categories/%d/toggle/enabled", id), newValue).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) toggleCategoryNewTab(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	newValue, err := ds.ToggleCategoryOpenInNewTab(model.CategoryID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	ctx := (*c).Request().Context()
	w := (*c).Response()
	if err := views.BookmarkHeaderToggle("newtab", fmt.Sprintf("cat-%d-newtab-toggle", id), fmt.Sprintf("/admin/categories/%d/toggle/new-tab", id), newValue, settings.DefaultOpenInNewTab, false).Render(ctx, w); err != nil {
		return err
	}
	bmIDs, err := ds.GetBookmarkIDsInCategoryWithInheritedNewTab(model.CategoryID(id))
	if err != nil {
		return err
	}
	effNewTab := settings.DefaultOpenInNewTab
	if newValue != nil {
		effNewTab = *newValue
	}
	for _, bmID := range bmIDs {
		if err := views.BookmarkHeaderToggle("newtab", fmt.Sprintf("bm-%d-newtab-toggle", bmID), fmt.Sprintf("/admin/bookmarks/%d/toggle/new-tab", bmID), nil, effNewTab, true).Render(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminHandler) createBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	catID, err := strconv.Atoi((*c).FormValue("category_id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}
	bm, err := ds.CreateBookmark(model.CategoryID(catID))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	effPrivate, effNewTab, err := effectiveCategoryDefaults(ds, model.CategoryID(catID), settings.DefaultPrivate, settings.DefaultOpenInNewTab)
	if err != nil {
		return err
	}
	return views.BookmarkRow(bm, settings, effPrivate, effNewTab).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) updateBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	update := model.BookmarkUpdate{
		Name:      formStr(c, "name"),
		URL:       formStr(c, "url"),
		MobileURL: formStr(c, "mobile_url"),
		Icon:      formStr(c, "icon"),
	}
	if raw := formStr(c, "keywords"); raw != nil {
		kw := strings.Fields(*raw)
		update.Keywords = &kw
	}
	if err := ds.UpdateBookmark(model.BookmarkID(id), update); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) deleteBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	if err := ds.DeleteBookmark(model.BookmarkID(id)); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) duplicateBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	bm, err := ds.DuplicateBookmark(model.BookmarkID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	effPrivate, effNewTab, err := effectiveCategoryDefaults(ds, bm.CategoryID, settings.DefaultPrivate, settings.DefaultOpenInNewTab)
	if err != nil {
		return err
	}
	return views.BookmarkRow(bm, settings, effPrivate, effNewTab).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) toggleBookmarkPrivate(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	newValue, err := ds.ToggleBookmarkPrivate(model.BookmarkID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	bm, err := ds.GetBookmark(model.BookmarkID(id))
	if err != nil {
		return err
	}
	effPrivate, _, err := effectiveCategoryDefaults(ds, bm.CategoryID, settings.DefaultPrivate, settings.DefaultOpenInNewTab)
	if err != nil {
		return err
	}
	return views.BookmarkHeaderToggle("private", fmt.Sprintf("bm-%d-private-toggle", id), fmt.Sprintf("/admin/bookmarks/%d/toggle/private", id), newValue, effPrivate, false).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) toggleBookmarkNewTab(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	newValue, err := ds.ToggleBookmarkOpenInNewTab(model.BookmarkID(id))
	if err != nil {
		return err
	}
	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	bm, err := ds.GetBookmark(model.BookmarkID(id))
	if err != nil {
		return err
	}
	_, effNewTab, err := effectiveCategoryDefaults(ds, bm.CategoryID, settings.DefaultPrivate, settings.DefaultOpenInNewTab)
	if err != nil {
		return err
	}
	return views.BookmarkHeaderToggle("newtab", fmt.Sprintf("bm-%d-newtab-toggle", id), fmt.Sprintf("/admin/bookmarks/%d/toggle/new-tab", id), newValue, effNewTab, false).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) toggleBookmarkEnabled(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	newValue, err := ds.ToggleBookmarkEnabled(model.BookmarkID(id))
	if err != nil {
		return err
	}
	return views.BookmarkEnabledToggle(fmt.Sprintf("/admin/bookmarks/%d/toggle/enabled", id), newValue).Render((*c).Request().Context(), (*c).Response())
}

func (h *AdminHandler) moveBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid bookmark id")
	}
	targetCatID, err := strconv.Atoi((*c).FormValue("target_category_id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid target category id")
	}
	if err := ds.MoveBookmark(model.BookmarkID(id), model.CategoryID(targetCatID), 0); err != nil {
		return err
	}
	orderJSON := (*c).FormValue("order")
	var ids []int
	if err := json.Unmarshal([]byte(orderJSON), &ids); err != nil {
		return echo.NewHTTPError(400, "invalid order")
	}
	bmIDs := make([]model.BookmarkID, len(ids))
	for i, v := range ids {
		bmIDs[i] = model.BookmarkID(v)
	}
	if err := ds.ReorderBookmarks(model.CategoryID(targetCatID), bmIDs); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func (h *AdminHandler) reorderBookmarks(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	catID, err := strconv.Atoi((*c).FormValue("category_id"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid category id")
	}

	ids, err := parseIntIDs((*c).FormValue("order"))
	if err != nil {
		return echo.NewHTTPError(400, "invalid order")
	}

	bookmarkIDs := make([]model.BookmarkID, len(ids))
	for i, id := range ids {
		bookmarkIDs[i] = model.BookmarkID(id)
	}
	if err := ds.ReorderBookmarks(model.CategoryID(catID), bookmarkIDs); err != nil {
		return err
	}
	return (*c).NoContent(200)
}

func effectiveCategoryDefaults(ds storage.Datasource, categoryID model.CategoryID, globalPrivate, globalNewTab bool) (private, newTab bool, err error) {
	cat, err := ds.GetCategory(categoryID)
	if err != nil {
		return globalPrivate, globalNewTab, err
	}
	return common.ResolveNullBool(cat.Private, globalPrivate),
		common.ResolveNullBool(cat.OpenInNewTab, globalNewTab), nil
}

func (h *AdminHandler) searchIcons(c *echo.Context) error {
	query := (*c).QueryParam("q")
	results := h.icons.Search(query)
	return views.IconGrid(results).Render((*c).Request().Context(), (*c).Response())
}
