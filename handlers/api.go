package handlers

import (
	"dashboard/config"
	"dashboard/icons"
	"dashboard/model"
	"dashboard/static"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"gopkg.in/yaml.v3"
)

type APIHandler struct {
	ds       DSResolver
	icons    *icons.Loader
	api      config.APIConfig
	specJSON []byte
	specYAML []byte
}

func dataRoutes(h *APIHandler) []apiRoute {
	idParam := paramSpec{Name: "id", In: ParamPath, Required: true, Type: ParamInteger}

	return []apiRoute{
		// Categories
		{
			Method: "GET", Path: "/categories", Handler: h.listCategories,
			Summary: "List all categories with bookmarks", Tag: "Categories",
			Status: 200, Output: []model.Category{},
		},
		{
			Method: "POST", Path: "/categories", Handler: h.createCategory, Write: true,
			Summary: "Create a category", Tag: "Categories",
			Status: 201, Input: CategoryCreateInput{}, Output: model.Category{},
		},
		{
			Method: "GET", Path: "/categories/:id", Handler: h.getCategory,
			Summary: "Get a category", Tag: "Categories",
			Status: 200, Output: model.Category{}, Params: []paramSpec{idParam},
		},
		{
			Method: "PUT", Path: "/categories/:id", Handler: h.updateCategory, Write: true,
			Summary: "Update a category", Tag: "Categories",
			Status: 200, Input: CategoryUpdateInput{}, Output: model.Category{},
			Params: []paramSpec{idParam},
		},
		{
			Method: "DELETE", Path: "/categories/:id", Handler: h.deleteCategory, Write: true,
			Summary: "Delete a category", Tag: "Categories",
			Status: 204, Params: []paramSpec{idParam},
		},
		// Bookmarks
		{
			Method: "POST", Path: "/bookmarks", Handler: h.createBookmark, Write: true,
			Summary: "Create a bookmark", Tag: "Bookmarks",
			Status: 201, Input: BookmarkCreateInput{}, Output: model.Bookmark{},
		},
		{
			Method: "GET", Path: "/bookmarks/:id", Handler: h.getBookmark,
			Summary: "Get a bookmark", Tag: "Bookmarks",
			Status: 200, Output: model.Bookmark{}, Params: []paramSpec{idParam},
		},
		{
			Method: "PUT", Path: "/bookmarks/:id", Handler: h.updateBookmark, Write: true,
			Summary: "Update a bookmark", Tag: "Bookmarks",
			Status: 200, Input: BookmarkUpdateInput{}, Output: model.Bookmark{},
			Params: []paramSpec{idParam},
		},
		{
			Method: "DELETE", Path: "/bookmarks/:id", Handler: h.deleteBookmark, Write: true,
			Summary: "Delete a bookmark", Tag: "Bookmarks",
			Status: 204, Params: []paramSpec{idParam},
		},
		{
			Method: "POST", Path: "/bookmarks/:id/move", Handler: h.moveBookmark, Write: true,
			Summary: "Move a bookmark to a different category", Tag: "Bookmarks",
			Status: 200, Input: BookmarkMoveInput{}, Output: model.Bookmark{},
			Params: []paramSpec{idParam},
		},
		{
			Method: "GET", Path: "/bookmarks/search", Handler: h.searchBookmarks,
			Summary: "Search bookmarks by URL or query", Tag: "Bookmarks",
			Status: 200, Output: []model.Bookmark{},
			Params: []paramSpec{
				{Name: "url", In: ParamQuery, Type: ParamString, Description: "Exact URL match"},
				{Name: "q", In: ParamQuery, Type: ParamString, Description: "Substring search on name, URL, and keywords"},
			},
		},
		// Icons
		{
			Method: "GET", Path: "/icons", Handler: h.listIcons,
			Summary: "List or search Material Design Icons", Tag: "Icons",
			Status: 200, Output: IconListOutput{},
			Params: []paramSpec{
				{Name: "q", In: ParamQuery, Type: ParamString, Description: "Substring filter"},
				{Name: "limit", In: ParamQuery, Type: ParamInteger, Description: "Max results"},
				{Name: "offset", In: ParamQuery, Type: ParamInteger, Description: "Pagination offset"},
			},
		},
	}
}

func SetupAPIRoutes(e *echo.Echo, ds DSResolver, il *icons.Loader, api config.APIConfig) {
	h := &APIHandler{ds: ds, icons: il, api: api}
	routes := dataRoutes(h)

	// Build and cache the OpenAPI spec
	spec := buildSpecFromRoutes(routes)
	h.specJSON, _ = json.Marshal(spec)
	h.specYAML, _ = yaml.Marshal(spec)

	// Public routes — no auth required
	e.GET("/api", h.apiIndex)
	e.GET("/api/openapi.json", h.openAPISpecJSON)
	e.GET("/api/openapi.yaml", h.openAPISpecYAML)
	if api.Swagger {
		e.GET("/api/docs", h.swaggerUI)
	}

	// Data routes — auth required when tokens are configured
	apiGroup := e.Group("/api", requireAPIToken(api.Tokens))
	for _, r := range routes {
		var mw []echo.MiddlewareFunc
		if r.Write {
			mw = append(mw, requireWriteAccess)
		}
		switch r.Method {
		case "GET":
			apiGroup.GET(r.Path, r.Handler, mw...)
		case "POST":
			apiGroup.POST(r.Path, r.Handler, mw...)
		case "PUT":
			apiGroup.PUT(r.Path, r.Handler, mw...)
		case "DELETE":
			apiGroup.DELETE(r.Path, r.Handler, mw...)
		}
	}
}

// Helpers

func apiError(c *echo.Context, status int, msg string) error {
	return (*c).JSON(status, map[string]string{"error": msg})
}

func apiCategoryID(c *echo.Context) (model.CategoryID, error) {
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return 0, fmt.Errorf("invalid category id")
	}
	return model.CategoryID(id), nil
}

func apiBookmarkID(c *echo.Context) (model.BookmarkID, error) {
	id, err := strconv.Atoi((*c).Param("id"))
	if err != nil {
		return 0, fmt.Errorf("invalid bookmark id")
	}
	return model.BookmarkID(id), nil
}

// Category handlers

func (h *APIHandler) listCategories(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	cats, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	return (*c).JSON(http.StatusOK, cats)
}

func (h *APIHandler) getCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiCategoryID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	cat, err := ds.GetCategory(id)
	if err != nil {
		return apiError(c, http.StatusNotFound, "category not found")
	}
	return (*c).JSON(http.StatusOK, cat)
}

func (h *APIHandler) createCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	var body CategoryCreateInput
	if err := (*c).Bind(&body); err != nil {
		return apiError(c, http.StatusBadRequest, "invalid request body")
	}
	if body.Name == "" {
		return apiError(c, http.StatusBadRequest, "name is required")
	}
	cat, err := ds.CreateCategory(body.Name)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	return (*c).JSON(http.StatusCreated, cat)
}

func (h *APIHandler) updateCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiCategoryID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	var body CategoryUpdateInput
	if err := (*c).Bind(&body); err != nil {
		return apiError(c, http.StatusBadRequest, "invalid request body")
	}
	update := model.CategoryUpdate{Name: body.Name}
	if err := ds.UpdateCategory(id, update); err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	cat, err := ds.GetCategory(id)
	if err != nil {
		return apiError(c, http.StatusNotFound, "category not found")
	}
	return (*c).JSON(http.StatusOK, cat)
}

func (h *APIHandler) deleteCategory(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiCategoryID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	if err := ds.DeleteCategory(id); err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	return (*c).NoContent(http.StatusNoContent)
}

// Bookmark handlers

func (h *APIHandler) createBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	var body BookmarkCreateInput
	if err := (*c).Bind(&body); err != nil {
		return apiError(c, http.StatusBadRequest, "invalid request body")
	}
	if body.CategoryID == 0 {
		return apiError(c, http.StatusBadRequest, "category_id is required")
	}

	bm, err := ds.CreateBookmark(model.CategoryID(body.CategoryID))
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}

	// Apply fields if provided
	update := model.BookmarkUpdate{}
	if body.Name != "" {
		update.Name = &body.Name
	}
	if body.URL != "" {
		update.URL = &body.URL
	}
	if body.MobileURL != "" {
		update.MobileURL = &body.MobileURL
	}
	if body.Icon != "" {
		update.Icon = &body.Icon
	}
	if body.Keywords != nil {
		update.Keywords = body.Keywords
	}

	if update.Name != nil || update.URL != nil || update.MobileURL != nil || update.Icon != nil || update.Keywords != nil {
		if err := ds.UpdateBookmark(bm.ID, update); err != nil {
			return apiError(c, http.StatusInternalServerError, err.Error())
		}
		bm, err = ds.GetBookmark(bm.ID)
		if err != nil {
			return apiError(c, http.StatusInternalServerError, err.Error())
		}
	}

	return (*c).JSON(http.StatusCreated, bm)
}

func (h *APIHandler) getBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiBookmarkID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	bm, err := ds.GetBookmark(id)
	if err != nil {
		return apiError(c, http.StatusNotFound, "bookmark not found")
	}
	return (*c).JSON(http.StatusOK, bm)
}

func (h *APIHandler) updateBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiBookmarkID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	var body BookmarkUpdateInput
	if err := (*c).Bind(&body); err != nil {
		return apiError(c, http.StatusBadRequest, "invalid request body")
	}
	update := model.BookmarkUpdate{
		Name:      body.Name,
		URL:       body.URL,
		MobileURL: body.MobileURL,
		Icon:      body.Icon,
		Keywords:  body.Keywords,
	}
	if err := ds.UpdateBookmark(id, update); err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	bm, err := ds.GetBookmark(id)
	if err != nil {
		return apiError(c, http.StatusNotFound, "bookmark not found")
	}
	return (*c).JSON(http.StatusOK, bm)
}

func (h *APIHandler) deleteBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiBookmarkID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	if err := ds.DeleteBookmark(id); err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	return (*c).NoContent(http.StatusNoContent)
}

func (h *APIHandler) moveBookmark(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	id, err := apiBookmarkID(c)
	if err != nil {
		return apiError(c, http.StatusBadRequest, err.Error())
	}
	var body BookmarkMoveInput
	if err := (*c).Bind(&body); err != nil {
		return apiError(c, http.StatusBadRequest, "invalid request body")
	}
	if body.CategoryID == 0 {
		return apiError(c, http.StatusBadRequest, "category_id is required")
	}
	if err := ds.MoveBookmark(id, model.CategoryID(body.CategoryID), body.Position); err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	bm, err := ds.GetBookmark(id)
	if err != nil {
		return apiError(c, http.StatusNotFound, "bookmark not found")
	}
	return (*c).JSON(http.StatusOK, bm)
}

func (h *APIHandler) searchBookmarks(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	urlParam := (*c).QueryParam("url")
	query := (*c).QueryParam("q")
	bookmarks, err := ds.SearchBookmarks(urlParam, query)
	if err != nil {
		return apiError(c, http.StatusInternalServerError, err.Error())
	}
	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	return (*c).JSON(http.StatusOK, bookmarks)
}

// Icons

func (h *APIHandler) listIcons(c *echo.Context) error {
	query := (*c).QueryParam("q")
	limitStr := (*c).QueryParam("limit")
	offsetStr := (*c).QueryParam("offset")

	var allIcons []string
	if query != "" {
		allIcons = h.icons.SearchAll(query)
	} else {
		allIcons = h.icons.Icons
	}

	total := len(allIcons)

	offset := 0
	if offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			return apiError(c, http.StatusBadRequest, "invalid offset")
		}
		offset = v
	}

	limit := total
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			return apiError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = v
	}

	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	icons := allIcons[offset:end]

	return (*c).JSON(http.StatusOK, map[string]any{
		"icons": icons,
		"total": total,
	})
}

// API index

func (h *APIHandler) apiIndex(c *echo.Context) error {
	index := map[string]any{
		"name":    "Jumpgate API",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"categories": "/api/categories",
			"bookmarks":  "/api/bookmarks",
			"search":     "/api/bookmarks/search",
			"icons":      "/api/icons",
		},
		"openapi_json": "/api/openapi.json",
		"openapi_yaml": "/api/openapi.yaml",
	}
	if h.api.Swagger {
		index["docs"] = "/api/docs"
	}
	return (*c).JSON(http.StatusOK, index)
}

// OpenAPI spec

func (h *APIHandler) openAPISpecJSON(c *echo.Context) error {
	return (*c).Blob(http.StatusOK, "application/json", h.specJSON)
}

func (h *APIHandler) openAPISpecYAML(c *echo.Context) error {
	return (*c).Blob(http.StatusOK, "text/yaml", h.specYAML)
}

// Swagger UI

func (h *APIHandler) swaggerUI(c *echo.Context) error {
	html, _ := static.FS.ReadFile("swagger.html")
	return (*c).HTMLBlob(http.StatusOK, html)
}

// accessLevel key used in echo context
const accessLevelKey = "api_access_level"

func validateBearerToken(tokens config.APITokens, authHeader string) (string, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("missing or invalid authorization header")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	for _, t := range tokens.ReadWrite {
		if t == token {
			return "read-write", nil
		}
	}
	for _, t := range tokens.ReadOnly {
		if t == token {
			return "read-only", nil
		}
	}
	return "", fmt.Errorf("invalid token")
}

func requireAPIToken(tokens config.APITokens) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !tokens.HasTokens() {
				(*c).Set(accessLevelKey, "read-write")
				return next(c)
			}
			level, err := validateBearerToken(tokens, (*c).Request().Header.Get("Authorization"))
			if err != nil {
				return (*c).JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			}
			(*c).Set(accessLevelKey, level)
			return next(c)
		}
	}
}

func requireWriteAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		level, _ := (*c).Get(accessLevelKey).(string)
		if level != "read-write" {
			return (*c).JSON(http.StatusForbidden, map[string]string{"error": "read-only token cannot perform write operations"})
		}
		return next(c)
	}
}
