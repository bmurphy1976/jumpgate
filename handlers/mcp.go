package handlers

import (
	"context"
	"dashboard/config"
	"dashboard/icons"
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"dashboard/storage"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPHandler manages the MCP server endpoint.
type MCPHandler struct {
	ds     DSResolver
	icons  *icons.Loader
	tokens config.APITokens
}

func SetupMCPRoutes(e *echo.Echo, ds DSResolver, il *icons.Loader, tokens config.APITokens) {
	h := &MCPHandler{ds: ds, icons: il, tokens: tokens}

	mcpServer := mcp.NewServer(newMCPImplementation(), nil)

	h.registerTools(mcpServer)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	e.Any("/mcp", func(c *echo.Context) error {
		r := (*c).Request()

		var level string
		if tokens.HasTokens() {
			var err error
			level, err = validateBearerToken(tokens, r.Header.Get("Authorization"))
			if err != nil {
				return (*c).JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			}
		} else {
			level = "read-write"
		}

		ctx := context.WithValue(r.Context(), mcpAccessLevelKey, level)
		ctx = context.WithValue(ctx, mcpDSResolverKey, h.ds)
		ctx = context.WithValue(ctx, mcpEchoContextKey, c)
		r = r.WithContext(ctx)

		handler.ServeHTTP((*c).Response(), r)
		return nil
	})
}

func newMCPImplementation() *mcp.Implementation {
	return &mcp.Implementation{
		Name:    "jumpgate",
		Version: buildinfo.ServiceVersion(),
	}
}

type contextKey string

const (
	mcpAccessLevelKey contextKey = "mcp_access_level"
	mcpDSResolverKey  contextKey = "mcp_ds_resolver"
	mcpEchoContextKey contextKey = "mcp_echo_context"
)

func mcpRequireWrite(ctx context.Context) error {
	level, _ := ctx.Value(mcpAccessLevelKey).(string)
	if level != "read-write" {
		return fmt.Errorf("read-only token cannot perform write operations")
	}
	return nil
}

// mcpResolveDS extracts the datasource from the MCP request context.
func mcpResolveDS(ctx context.Context) (storage.Datasource, error) {
	ds, ok := ctx.Value(mcpDSResolverKey).(DSResolver)
	if !ok {
		return nil, fmt.Errorf("internal error: missing datasource resolver")
	}
	ec, ok := ctx.Value(mcpEchoContextKey).(*echo.Context)
	if !ok {
		return nil, fmt.Errorf("internal error: missing echo context")
	}
	return ds(ec)
}

func (h *MCPHandler) registerTools(server *mcp.Server) {
	// Category tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_list",
		Description: "List all categories with their bookmarks",
	}, h.categoryList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_get",
		Description: "Get a single category with its bookmarks",
	}, h.categoryGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_create",
		Description: "Create a new category",
	}, h.categoryCreateMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_update",
		Description: "Update a category name",
	}, h.categoryUpdateMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "category_delete",
		Description: "Delete a category",
	}, h.categoryDeleteMCP)

	// Bookmark tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_create",
		Description: "Create a new bookmark in a category",
	}, h.bookmarkCreateMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_get",
		Description: "Get a single bookmark by ID",
	}, h.bookmarkGetMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_update",
		Description: "Update a bookmark's fields",
	}, h.bookmarkUpdateMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_delete",
		Description: "Delete a bookmark",
	}, h.bookmarkDeleteMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_move",
		Description: "Move a bookmark to a different category",
	}, h.bookmarkMoveMCP)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bookmark_search",
		Description: "Search bookmarks by URL or query string",
	}, h.bookmarkSearchMCP)

	// Icon tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "icon_list",
		Description: "List or search Material Design Icons",
	}, h.iconListMCP)
}

// Tool handler types

type categoryGetInput struct {
	CategoryID int `json:"category_id" jsonschema:"Category ID"`
}

type categoryCreateInput struct {
	Name string `json:"name" jsonschema:"Category name"`
}

type categoryUpdateInput struct {
	CategoryID int    `json:"category_id" jsonschema:"Category ID"`
	Name       string `json:"name" jsonschema:"New category name"`
}

type categoryDeleteInput struct {
	CategoryID int `json:"category_id" jsonschema:"Category ID"`
}

type bookmarkCreateInput struct {
	CategoryID int      `json:"category_id" jsonschema:"Category ID"`
	Name       string   `json:"name,omitempty" jsonschema:"Bookmark name"`
	URL        string   `json:"url,omitempty" jsonschema:"Bookmark URL"`
	MobileURL  string   `json:"mobile_url,omitempty" jsonschema:"Mobile URL"`
	Icon       string   `json:"icon,omitempty" jsonschema:"MDI icon name"`
	Keywords   []string `json:"keywords,omitempty" jsonschema:"Search keywords"`
}

type bookmarkGetInput struct {
	BookmarkID int `json:"bookmark_id" jsonschema:"Bookmark ID"`
}

type bookmarkUpdateInput struct {
	BookmarkID int      `json:"bookmark_id" jsonschema:"Bookmark ID"`
	Name       string   `json:"name,omitempty" jsonschema:"Bookmark name"`
	URL        string   `json:"url,omitempty" jsonschema:"Bookmark URL"`
	MobileURL  string   `json:"mobile_url,omitempty" jsonschema:"Mobile URL"`
	Icon       string   `json:"icon,omitempty" jsonschema:"MDI icon name"`
	Keywords   []string `json:"keywords,omitempty" jsonschema:"Search keywords"`
}

type bookmarkDeleteInput struct {
	BookmarkID int `json:"bookmark_id" jsonschema:"Bookmark ID"`
}

type bookmarkMoveInput struct {
	BookmarkID int `json:"bookmark_id" jsonschema:"Bookmark ID"`
	CategoryID int `json:"category_id" jsonschema:"Target category ID"`
	Position   int `json:"position,omitempty" jsonschema:"Position in target category"`
}

type bookmarkSearchInput struct {
	URL   string `json:"url,omitempty" jsonschema:"Exact URL match"`
	Query string `json:"query,omitempty" jsonschema:"Substring search on name/URL/keywords"`
}

type iconListInput struct {
	Query  string `json:"query,omitempty" jsonschema:"Substring filter"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Max results"`
	Offset int    `json:"offset,omitempty" jsonschema:"Pagination offset"`
}

// Tool implementations

func (h *MCPHandler) categoryList(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	cats, err := store.GetCategoriesWithBookmarks()
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]any{"categories": cats}, nil
}

func (h *MCPHandler) categoryGet(ctx context.Context, req *mcp.CallToolRequest, input categoryGetInput) (*mcp.CallToolResult, any, error) {
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	cat, err := store.GetCategory(model.CategoryID(input.CategoryID))
	if err != nil {
		return nil, nil, err
	}
	return nil, cat, nil
}

func (h *MCPHandler) categoryCreateMCP(ctx context.Context, req *mcp.CallToolRequest, input categoryCreateInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	cat, err := store.CreateCategory(input.Name)
	if err != nil {
		return nil, nil, err
	}
	return nil, cat, nil
}

func (h *MCPHandler) categoryUpdateMCP(ctx context.Context, req *mcp.CallToolRequest, input categoryUpdateInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	update := model.CategoryUpdate{Name: &input.Name}
	if err := store.UpdateCategory(model.CategoryID(input.CategoryID), update); err != nil {
		return nil, nil, err
	}
	cat, err := store.GetCategory(model.CategoryID(input.CategoryID))
	if err != nil {
		return nil, nil, err
	}
	return nil, cat, nil
}

func (h *MCPHandler) categoryDeleteMCP(ctx context.Context, req *mcp.CallToolRequest, input categoryDeleteInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := store.DeleteCategory(model.CategoryID(input.CategoryID)); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"status": "deleted"}, nil
}

func (h *MCPHandler) bookmarkCreateMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkCreateInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	bm, err := store.CreateBookmark(model.CategoryID(input.CategoryID))
	if err != nil {
		return nil, nil, err
	}
	update := model.BookmarkUpdate{}
	if input.Name != "" {
		update.Name = &input.Name
	}
	if input.URL != "" {
		update.URL = &input.URL
	}
	if input.MobileURL != "" {
		update.MobileURL = &input.MobileURL
	}
	if input.Icon != "" {
		update.Icon = &input.Icon
	}
	if len(input.Keywords) > 0 {
		update.Keywords = &input.Keywords
	}
	if update.Name != nil || update.URL != nil || update.MobileURL != nil || update.Icon != nil || update.Keywords != nil {
		if err := store.UpdateBookmark(bm.ID, update); err != nil {
			return nil, nil, err
		}
		bm, err = store.GetBookmark(bm.ID)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, bm, nil
}

func (h *MCPHandler) bookmarkGetMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkGetInput) (*mcp.CallToolResult, any, error) {
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	bm, err := store.GetBookmark(model.BookmarkID(input.BookmarkID))
	if err != nil {
		return nil, nil, err
	}
	return nil, bm, nil
}

func (h *MCPHandler) bookmarkUpdateMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkUpdateInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	update := model.BookmarkUpdate{}
	if input.Name != "" {
		update.Name = &input.Name
	}
	if input.URL != "" {
		update.URL = &input.URL
	}
	if input.MobileURL != "" {
		update.MobileURL = &input.MobileURL
	}
	if input.Icon != "" {
		update.Icon = &input.Icon
	}
	if input.Keywords != nil {
		update.Keywords = &input.Keywords
	}
	if err := store.UpdateBookmark(model.BookmarkID(input.BookmarkID), update); err != nil {
		return nil, nil, err
	}
	bm, err := store.GetBookmark(model.BookmarkID(input.BookmarkID))
	if err != nil {
		return nil, nil, err
	}
	return nil, bm, nil
}

func (h *MCPHandler) bookmarkDeleteMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkDeleteInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := store.DeleteBookmark(model.BookmarkID(input.BookmarkID)); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"status": "deleted"}, nil
}

func (h *MCPHandler) bookmarkMoveMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkMoveInput) (*mcp.CallToolResult, any, error) {
	if err := mcpRequireWrite(ctx); err != nil {
		return nil, nil, err
	}
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := store.MoveBookmark(model.BookmarkID(input.BookmarkID), model.CategoryID(input.CategoryID), input.Position); err != nil {
		return nil, nil, err
	}
	bm, err := store.GetBookmark(model.BookmarkID(input.BookmarkID))
	if err != nil {
		return nil, nil, err
	}
	return nil, bm, nil
}

func (h *MCPHandler) bookmarkSearchMCP(ctx context.Context, req *mcp.CallToolRequest, input bookmarkSearchInput) (*mcp.CallToolResult, any, error) {
	store, err := mcpResolveDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	bookmarks, err := store.SearchBookmarks(input.URL, input.Query)
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]any{"bookmarks": bookmarks}, nil
}

func (h *MCPHandler) iconListMCP(ctx context.Context, req *mcp.CallToolRequest, input iconListInput) (*mcp.CallToolResult, any, error) {
	var allIcons []string
	if input.Query != "" {
		allIcons = h.icons.SearchAll(input.Query)
	} else {
		allIcons = h.icons.Icons
	}

	total := len(allIcons)
	offset := input.Offset
	if offset > total {
		offset = total
	}
	limit := total - offset
	if input.Limit > 0 && input.Limit < limit {
		limit = input.Limit
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return nil, map[string]any{
		"icons": allIcons[offset:end],
		"total": total,
	}, nil
}
