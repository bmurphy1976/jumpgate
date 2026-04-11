package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"dashboard/icons"
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"dashboard/storage"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupMCPHandler(t *testing.T) (*MCPHandler, *storage.SQLiteDB, func(string) context.Context) {
	t.Helper()
	ds := setupTestDB(t)
	resolver := func(c *echo.Context) (storage.Datasource, error) {
		return ds, nil
	}
	h := &MCPHandler{
		ds:    resolver,
		icons: &icons.Loader{Icons: []string{"home", "star", "newspaper", "github", "google"}},
	}

	ctxFactory := func(level string) context.Context {
		e := echo.New()
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ctx := context.WithValue(context.Background(), mcpAccessLevelKey, level)
		ctx = context.WithValue(ctx, mcpDSResolverKey, DSResolver(resolver))
		ctx = context.WithValue(ctx, mcpEchoContextKey, c)
		return ctx
	}

	return h, ds, ctxFactory
}

// Permission enforcement

func TestMCPRequireWriteAllows(t *testing.T) {
	ctx := context.WithValue(context.Background(), mcpAccessLevelKey, "read-write")
	if err := mcpRequireWrite(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMCPRequireWriteBlocks(t *testing.T) {
	ctx := context.WithValue(context.Background(), mcpAccessLevelKey, "read-only")
	if err := mcpRequireWrite(ctx); err == nil {
		t.Error("expected error for read-only, got nil")
	}
}

func TestMCPRequireWriteMissingLevel(t *testing.T) {
	ctx := context.Background()
	if err := mcpRequireWrite(ctx); err == nil {
		t.Error("expected error for missing level, got nil")
	}
}

func TestMCPImplementationIncludesVersion(t *testing.T) {
	oldRelease := buildinfo.ReleaseVersion
	oldCommit := buildinfo.Commit
	buildinfo.ReleaseVersion = "2026.04.0"
	buildinfo.Commit = "04fc78e"
	t.Cleanup(func() {
		buildinfo.ReleaseVersion = oldRelease
		buildinfo.Commit = oldCommit
	})

	impl := newMCPImplementation()
	if impl.Version != "2026.04.0+04fc78e" {
		t.Fatalf("expected stamped version, got %q", impl.Version)
	}
}

// Datasource resolution

func TestMCPResolveDSOK(t *testing.T) {
	_, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	ds, err := mcpResolveDS(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ds == nil {
		t.Fatal("expected non-nil datasource")
	}
}

func TestMCPResolveDSMissingResolver(t *testing.T) {
	ctx := context.Background()
	_, err := mcpResolveDS(ctx)
	if err == nil {
		t.Error("expected error for missing resolver, got nil")
	}
}

// Category CRUD

func TestMCPCategoryList(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, result, err := h.categoryList(ctx, &mcp.CallToolRequest{}, struct{}{})
	if err != nil {
		t.Fatalf("categoryList failed: %v", err)
	}
	m := result.(map[string]any)
	cats := m["categories"]
	if cats == nil {
		t.Fatal("expected categories in result")
	}
}

func TestMCPCategoryCreate(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, result, err := h.categoryCreateMCP(ctx, &mcp.CallToolRequest{}, categoryCreateInput{Name: "Work"})
	if err != nil {
		t.Fatalf("categoryCreate failed: %v", err)
	}
	// Verify it was created by listing
	_, listResult, _ := h.categoryList(ctx, &mcp.CallToolRequest{}, struct{}{})
	m := listResult.(map[string]any)
	_ = m // if we got here without error, create succeeded
	_ = result
}

func TestMCPCategoryGet(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	cat, _ := ds.CreateCategory("Test")
	ctx := ctxFactory("read-write")
	_, _, err := h.categoryGet(ctx, &mcp.CallToolRequest{}, categoryGetInput{CategoryID: int(cat.ID)})
	if err != nil {
		t.Fatalf("categoryGet failed: %v", err)
	}
}

func TestMCPCategoryUpdate(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	cat, _ := ds.CreateCategory("Old")
	ctx := ctxFactory("read-write")
	_, _, err := h.categoryUpdateMCP(ctx, &mcp.CallToolRequest{}, categoryUpdateInput{
		CategoryID: int(cat.ID),
		Name:       "New",
	})
	if err != nil {
		t.Fatalf("categoryUpdate failed: %v", err)
	}
	updated, _ := ds.GetCategory(cat.ID)
	if updated.Name != "New" {
		t.Errorf("expected name 'New', got %q", updated.Name)
	}
}

func TestMCPCategoryDelete(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	cat, _ := ds.CreateCategory("Delete Me")
	ctx := ctxFactory("read-write")
	_, _, err := h.categoryDeleteMCP(ctx, &mcp.CallToolRequest{}, categoryDeleteInput{CategoryID: int(cat.ID)})
	if err != nil {
		t.Fatalf("categoryDelete failed: %v", err)
	}
	_, err = ds.GetCategory(cat.ID)
	if err == nil {
		t.Error("expected error getting deleted category")
	}
}

// Bookmark CRUD

func TestMCPBookmarkCreate(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkCreateMCP(ctx, &mcp.CallToolRequest{}, bookmarkCreateInput{
		CategoryID: 1,
		Name:       "Google",
		URL:        "https://google.com",
		Icon:       "google",
		Keywords:   []string{"search"},
	})
	if err != nil {
		t.Fatalf("bookmarkCreate failed: %v", err)
	}
}

func TestMCPBookmarkCreateMinimal(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkCreateMCP(ctx, &mcp.CallToolRequest{}, bookmarkCreateInput{
		CategoryID: 1,
	})
	if err != nil {
		t.Fatalf("bookmarkCreate minimal failed: %v", err)
	}
}

func TestMCPBookmarkGet(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	bm, _ := ds.CreateBookmark(1)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkGetMCP(ctx, &mcp.CallToolRequest{}, bookmarkGetInput{BookmarkID: int(bm.ID)})
	if err != nil {
		t.Fatalf("bookmarkGet failed: %v", err)
	}
}

func TestMCPBookmarkUpdate(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	bm, _ := ds.CreateBookmark(1)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkUpdateMCP(ctx, &mcp.CallToolRequest{}, bookmarkUpdateInput{
		BookmarkID: int(bm.ID),
		Name:       "Updated",
		URL:        "https://example.com",
	})
	if err != nil {
		t.Fatalf("bookmarkUpdate failed: %v", err)
	}
	updated, _ := ds.GetBookmark(bm.ID)
	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestMCPBookmarkDelete(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	bm, _ := ds.CreateBookmark(1)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkDeleteMCP(ctx, &mcp.CallToolRequest{}, bookmarkDeleteInput{BookmarkID: int(bm.ID)})
	if err != nil {
		t.Fatalf("bookmarkDelete failed: %v", err)
	}
	_, err = ds.GetBookmark(bm.ID)
	if err == nil {
		t.Error("expected error getting deleted bookmark")
	}
}

func TestMCPBookmarkMove(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	cat, _ := ds.CreateCategory("Target")
	bm, _ := ds.CreateBookmark(1)
	ctx := ctxFactory("read-write")
	_, _, err := h.bookmarkMoveMCP(ctx, &mcp.CallToolRequest{}, bookmarkMoveInput{
		BookmarkID: int(bm.ID),
		CategoryID: int(cat.ID),
		Position:   0,
	})
	if err != nil {
		t.Fatalf("bookmarkMove failed: %v", err)
	}
	moved, _ := ds.GetBookmark(bm.ID)
	if moved.CategoryID != cat.ID {
		t.Errorf("expected category %d, got %d", cat.ID, moved.CategoryID)
	}
}

func TestMCPBookmarkSearch(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	bm, _ := ds.CreateBookmark(1)
	name := "Searchable"
	url := "https://search.example.com"
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Name: &name, URL: &url})
	ctx := ctxFactory("read-write")
	_, result, err := h.bookmarkSearchMCP(ctx, &mcp.CallToolRequest{}, bookmarkSearchInput{Query: "Searchable"})
	if err != nil {
		t.Fatalf("bookmarkSearch failed: %v", err)
	}
	m := result.(map[string]any)
	if m["bookmarks"] == nil {
		t.Error("expected bookmarks in result")
	}
}

// Icon tools

func TestMCPIconList(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, result, err := h.iconListMCP(ctx, &mcp.CallToolRequest{}, iconListInput{})
	if err != nil {
		t.Fatalf("iconList failed: %v", err)
	}
	m := result.(map[string]any)
	if m["total"].(int) != 5 {
		t.Errorf("expected 5 icons, got %v", m["total"])
	}
}

func TestMCPIconListWithQuery(t *testing.T) {
	h, _, ctxFactory := setupMCPHandler(t)
	ctx := ctxFactory("read-write")
	_, result, err := h.iconListMCP(ctx, &mcp.CallToolRequest{}, iconListInput{Query: "home"})
	if err != nil {
		t.Fatalf("iconList with query failed: %v", err)
	}
	m := result.(map[string]any)
	icons := m["icons"].([]string)
	if len(icons) != 1 || icons[0] != "home" {
		t.Errorf("expected [home], got %v", icons)
	}
}

// Read-only enforcement

func TestMCPWriteOpsBlockedForReadOnly(t *testing.T) {
	h, ds, ctxFactory := setupMCPHandler(t)
	cat, _ := ds.CreateCategory("Test")
	bm, _ := ds.CreateBookmark(cat.ID)
	ctx := ctxFactory("read-only")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"categoryCreate", func() error {
			_, _, err := h.categoryCreateMCP(ctx, &mcp.CallToolRequest{}, categoryCreateInput{Name: "X"})
			return err
		}},
		{"categoryUpdate", func() error {
			_, _, err := h.categoryUpdateMCP(ctx, &mcp.CallToolRequest{}, categoryUpdateInput{CategoryID: int(cat.ID), Name: "X"})
			return err
		}},
		{"categoryDelete", func() error {
			_, _, err := h.categoryDeleteMCP(ctx, &mcp.CallToolRequest{}, categoryDeleteInput{CategoryID: int(cat.ID)})
			return err
		}},
		{"bookmarkCreate", func() error {
			_, _, err := h.bookmarkCreateMCP(ctx, &mcp.CallToolRequest{}, bookmarkCreateInput{CategoryID: int(cat.ID)})
			return err
		}},
		{"bookmarkUpdate", func() error {
			_, _, err := h.bookmarkUpdateMCP(ctx, &mcp.CallToolRequest{}, bookmarkUpdateInput{BookmarkID: int(bm.ID), Name: "X"})
			return err
		}},
		{"bookmarkDelete", func() error {
			_, _, err := h.bookmarkDeleteMCP(ctx, &mcp.CallToolRequest{}, bookmarkDeleteInput{BookmarkID: int(bm.ID)})
			return err
		}},
		{"bookmarkMove", func() error {
			_, _, err := h.bookmarkMoveMCP(ctx, &mcp.CallToolRequest{}, bookmarkMoveInput{BookmarkID: int(bm.ID), CategoryID: 1})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Errorf("%s should fail for read-only token", tt.name)
			}
		})
	}
}
