package storage

import (
	"dashboard/config"
	"dashboard/model"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *SQLiteDB {
	t.Helper()
	ds, err := NewSQLiteDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ds.Close() })
	return ds
}

// Test initialization creates default settings and Favorites category
func TestNewSQLiteDB(t *testing.T) {
	ds := setupTestDB(t)

	// Default settings row exists
	settings, err := ds.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.Title != "My Dashboard" {
		t.Errorf("expected default title 'My Dashboard', got %q", settings.Title)
	}
	if settings.DefaultPrivate != true {
		t.Error("expected default_private = true")
	}

	// Favorites category exists
	cats, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		t.Fatalf("GetCategoriesWithBookmarks failed: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}
	if cats[0].Name != "Favorites" || !cats[0].IsFavorites {
		t.Error("expected Favorites category with is_favorites=true")
	}
}

// Test settings CRUD
func TestSettings(t *testing.T) {
	ds := setupTestDB(t)

	// Update title
	title := "Test Dashboard"
	err := ds.UpdateSettings(model.SettingsUpdate{Title: &title})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	settings, err := ds.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.Title != "Test Dashboard" {
		t.Errorf("expected title 'Test Dashboard', got %q", settings.Title)
	}

	// Update weather unit with CHECK constraint
	validUnit := "celsius"
	err = ds.UpdateSettings(model.SettingsUpdate{WeatherUnit: &validUnit})
	if err != nil {
		t.Fatalf("UpdateSettings with valid unit failed: %v", err)
	}

	invalidUnit := "kelvin"
	err = ds.UpdateSettings(model.SettingsUpdate{WeatherUnit: &invalidUnit})
	if err == nil {
		t.Error("expected error for invalid weather_unit, got nil")
	}
}

// Test category CRUD
func TestCategories(t *testing.T) {
	ds := setupTestDB(t)

	// Create category
	cat, err := ds.CreateCategory("Work")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if cat.Name != "Work" {
		t.Errorf("expected name 'Work', got %q", cat.Name)
	}
	if cat.Position != 1 { // Favorites is position 0
		t.Errorf("expected position 1, got %d", cat.Position)
	}

	// Update category
	newName := "Personal"
	err = ds.UpdateCategory(cat.ID, model.CategoryUpdate{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	updated, err := ds.GetCategory(cat.ID)
	if err != nil {
		t.Fatalf("GetCategory failed: %v", err)
	}
	if updated.Name != "Personal" {
		t.Errorf("expected name 'Personal', got %q", updated.Name)
	}

	// Reorder categories
	favID := model.CategoryID(1) // Favorites is always ID 1
	err = ds.ReorderCategories([]model.CategoryID{cat.ID, favID})
	if err != nil {
		t.Fatalf("ReorderCategories failed: %v", err)
	}

	cats, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		t.Fatalf("GetCategoriesWithBookmarks failed: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0].ID != cat.ID || cats[1].ID != favID {
		t.Error("categories not reordered correctly")
	}

	// Delete category
	err = ds.DeleteCategory(cat.ID)
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	cats, err = ds.GetCategoriesWithBookmarks()
	if err != nil {
		t.Fatalf("GetCategoriesWithBookmarks failed: %v", err)
	}
	if len(cats) != 1 {
		t.Errorf("expected 1 category after delete, got %d", len(cats))
	}
}

// Test only one Favorites category allowed
func TestOneFavoritesCategory(t *testing.T) {
	ds := setupTestDB(t)

	// Try to create second favorites category via direct SQL (should fail)
	_, err := ds.db.Exec("INSERT INTO categories (name, position, is_favorites) VALUES ('Favorites2', 1, 1)")
	if err == nil {
		t.Error("expected error creating second favorites category, got nil")
	}
}

// Test bookmark CRUD
func TestBookmarks(t *testing.T) {
	ds := setupTestDB(t)
	favID := model.CategoryID(1)

	// Create bookmark
	bm, err := ds.CreateBookmark(favID)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
	if bm.Name != "" || bm.URL != "" {
		t.Error("expected empty name and URL for new bookmark")
	}
	if bm.Position != 0 {
		t.Errorf("expected position 0, got %d", bm.Position)
	}

	// Update bookmark
	name := "Hacker News"
	url := "https://news.ycombinator.com"
	icon := "mdi-newspaper"
	err = ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{
		Name: &name,
		URL:  &url,
		Icon: &icon,
	})
	if err != nil {
		t.Fatalf("UpdateBookmark failed: %v", err)
	}

	updated, err := ds.GetBookmark(bm.ID)
	if err != nil {
		t.Fatalf("GetBookmark failed: %v", err)
	}
	if updated.Name != "Hacker News" || updated.URL != url || updated.Icon != icon {
		t.Error("bookmark not updated correctly")
	}

	// Create second bookmark
	bm2, err := ds.CreateBookmark(favID)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
	if bm2.Position != 0 {
		t.Errorf("expected position 0 for second bookmark, got %d", bm2.Position)
	}

	// Duplicate bookmark
	dup, err := ds.DuplicateBookmark(bm.ID)
	if err != nil {
		t.Fatalf("DuplicateBookmark failed: %v", err)
	}
	if dup.Name != updated.Name || dup.URL != updated.URL || dup.Icon != updated.Icon {
		t.Errorf("duplicated bookmark fields don't match: got name=%q url=%q icon=%q, want name=%q url=%q icon=%q",
			dup.Name, dup.URL, dup.Icon, updated.Name, updated.URL, updated.Icon)
	}
	if dup.Position != 2 {
		t.Errorf("expected position 2 for duplicate, got %d", dup.Position)
	}

	// Sort alphabetically
	name2 := "Apple"
	ds.UpdateBookmark(bm2.ID, model.BookmarkUpdate{Name: &name2})

	err = ds.SortBookmarksAlpha(favID)
	if err != nil {
		t.Fatalf("SortBookmarksAlpha failed: %v", err)
	}

	cat, err := ds.GetCategory(favID)
	if err != nil {
		t.Fatalf("GetCategory failed: %v", err)
	}
	if len(cat.Bookmarks) < 2 {
		t.Fatal("expected at least 2 bookmarks")
	}
	if cat.Bookmarks[0].Name != "Apple" {
		t.Errorf("expected 'Apple' first after alpha sort, got %q", cat.Bookmarks[0].Name)
	}

	// Reorder bookmarks
	err = ds.ReorderBookmarks(favID, []model.BookmarkID{bm.ID, bm2.ID})
	if err != nil {
		t.Fatalf("ReorderBookmarks failed: %v", err)
	}

	cat, err = ds.GetCategory(favID)
	if err != nil {
		t.Fatalf("GetCategory failed: %v", err)
	}
	if len(cat.Bookmarks) < 2 {
		t.Fatal("expected at least 2 bookmarks")
	}
	if cat.Bookmarks[0].ID != bm.ID {
		t.Error("bookmarks not reordered correctly")
	}

	// Delete bookmark
	err = ds.DeleteBookmark(bm.ID)
	if err != nil {
		t.Fatalf("DeleteBookmark failed: %v", err)
	}

	cat, err = ds.GetCategory(favID)
	if err != nil {
		t.Fatalf("GetCategory failed: %v", err)
	}
	for _, b := range cat.Bookmarks {
		if b.ID == bm.ID {
			t.Error("bookmark still exists after delete")
		}
	}
}

// Test CASCADE delete
func TestCascadeDelete(t *testing.T) {
	ds := setupTestDB(t)

	cat, err := ds.CreateCategory("Test")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	bm, err := ds.CreateBookmark(cat.ID)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	// Delete category should cascade to bookmarks
	err = ds.DeleteCategory(cat.ID)
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	// Bookmark should be gone
	_, err = ds.GetBookmark(bm.ID)
	if err == nil {
		t.Error("expected error getting deleted bookmark, got nil")
	}
}

// Test toggle cycling: nil → true → false → nil
func TestToggleCategoryPrivate(t *testing.T) {
	ds := setupTestDB(t)

	cat, err := ds.CreateCategory("Test")
	if err != nil {
		t.Fatal(err)
	}

	// Initial state: nil (inherit)
	c, err := ds.GetCategory(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Private != nil {
		t.Errorf("expected nil, got %v", *c.Private)
	}

	// nil → true
	next, err := ds.ToggleCategoryPrivate(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || *next != true {
		t.Error("expected true")
	}

	// true → false
	next, err = ds.ToggleCategoryPrivate(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || *next != false {
		t.Error("expected false")
	}

	// false → nil
	next, err = ds.ToggleCategoryPrivate(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next != nil {
		t.Error("expected nil")
	}
}

func TestToggleBookmarkPrivate(t *testing.T) {
	ds := setupTestDB(t)
	bm, err := ds.CreateBookmark(model.CategoryID(1))
	if err != nil {
		t.Fatal(err)
	}

	// nil → true → false → nil
	next, _ := ds.ToggleBookmarkPrivate(bm.ID)
	if next == nil || *next != true {
		t.Error("expected true")
	}

	next, _ = ds.ToggleBookmarkPrivate(bm.ID)
	if next == nil || *next != false {
		t.Error("expected false")
	}

	next, _ = ds.ToggleBookmarkPrivate(bm.ID)
	if next != nil {
		t.Error("expected nil")
	}
}

func TestSearchBookmarksLikeEscape(t *testing.T) {
	ds := setupTestDB(t)
	favID := model.CategoryID(1)

	bm, err := ds.CreateBookmark(favID)
	if err != nil {
		t.Fatal(err)
	}
	name := "100% off"
	url := "https://example.com/100%25off"
	ds.UpdateBookmark(bm.ID, model.BookmarkUpdate{Name: &name, URL: &url})

	bm2, err := ds.CreateBookmark(favID)
	if err != nil {
		t.Fatal(err)
	}
	name2 := "1000 items"
	url2 := "https://example.com/1000"
	ds.UpdateBookmark(bm2.ID, model.BookmarkUpdate{Name: &name2, URL: &url2})

	// Search for literal "100%" — should match only the first bookmark
	results, err := ds.SearchBookmarks("", "100%")
	if err != nil {
		t.Fatalf("SearchBookmarks failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for '100%%', got %d", len(results))
	} else if results[0].ID != bm.ID {
		t.Errorf("expected bookmark %d, got %d", bm.ID, results[0].ID)
	}

	// Search for "100" — should match both
	results, err = ds.SearchBookmarks("", "100")
	if err != nil {
		t.Fatalf("SearchBookmarks failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for '100', got %d", len(results))
	}
}

func TestToggleBookmarkOpenInNewTab(t *testing.T) {
	ds := setupTestDB(t)
	bm, err := ds.CreateBookmark(model.CategoryID(1))
	if err != nil {
		t.Fatal(err)
	}

	// nil → true → false → nil
	next, _ := ds.ToggleBookmarkOpenInNewTab(bm.ID)
	if next == nil || *next != true {
		t.Error("expected true")
	}

	next, _ = ds.ToggleBookmarkOpenInNewTab(bm.ID)
	if next == nil || *next != false {
		t.Error("expected false")
	}

	next, _ = ds.ToggleBookmarkOpenInNewTab(bm.ID)
	if next != nil {
		t.Error("expected nil")
	}
}

func TestToggleCategoryEnabled(t *testing.T) {
	ds := setupTestDB(t)
	cat, err := ds.CreateCategory("Test")
	if err != nil {
		t.Fatal(err)
	}

	// starts enabled (true) → toggle to false
	next, err := ds.ToggleCategoryEnabled(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next != false {
		t.Errorf("expected false, got %v", next)
	}

	// false → true
	next, err = ds.ToggleCategoryEnabled(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next != true {
		t.Errorf("expected true, got %v", next)
	}
}

func TestToggleBookmarkEnabled(t *testing.T) {
	ds := setupTestDB(t)
	bm, err := ds.CreateBookmark(model.CategoryID(1))
	if err != nil {
		t.Fatal(err)
	}

	// starts enabled (true) → toggle to false
	next, err := ds.ToggleBookmarkEnabled(bm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next != false {
		t.Errorf("expected false, got %v", next)
	}

	// false → true
	next, err = ds.ToggleBookmarkEnabled(bm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next != true {
		t.Errorf("expected true, got %v", next)
	}
}

func TestMoveBookmark(t *testing.T) {
	ds := setupTestDB(t)

	catA, err := ds.CreateCategory("CatA")
	if err != nil {
		t.Fatal(err)
	}
	catB, err := ds.CreateCategory("CatB")
	if err != nil {
		t.Fatal(err)
	}

	bm, err := ds.CreateBookmark(catA.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = ds.MoveBookmark(bm.ID, catB.ID, 0)
	if err != nil {
		t.Fatal(err)
	}

	moved, err := ds.GetBookmark(bm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.CategoryID != catB.ID {
		t.Errorf("expected category_id %d, got %d", catB.ID, moved.CategoryID)
	}
	if moved.Position != 0 {
		t.Errorf("expected position 0, got %d", moved.Position)
	}
}

func TestMoveBookmarkSameCategory(t *testing.T) {
	ds := setupTestDB(t)

	cat, err := ds.CreateCategory("Test")
	if err != nil {
		t.Fatal(err)
	}

	bm1, err := ds.CreateBookmark(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ds.CreateBookmark(cat.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = ds.MoveBookmark(bm1.ID, cat.ID, 1)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := ds.GetBookmark(bm1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Position != 1 {
		t.Errorf("expected position 1, got %d", updated.Position)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestImportExportRoundTrip(t *testing.T) {
	ds := setupTestDB(t)

	input := config.Config{
		Title: "Test",
		Weather: &config.Weather{
			Latitude:     40.7,
			Longitude:    -74.0,
			Unit:         "fahrenheit",
			CacheMinutes: 30,
		},
		Categories: []config.Category{
			{
				Name: "Favorites",
				Links: []config.Link{
					{Name: "Go", URL: "https://go.dev", Keywords: []string{"golang", "programming"}},
					{Name: "Rust", URL: "https://rust-lang.org"},
				},
			},
			{
				Name: "Work",
				Links: []config.Link{
					{Name: "Internal", URL: "https://internal.example.com", Private: boolPtr(true)},
				},
			},
		},
	}

	err := ds.ImportConfig(input)
	if err != nil {
		t.Fatalf("ImportConfig failed: %v", err)
	}

	output, err := ds.ExportConfig()
	if err != nil {
		t.Fatalf("ExportConfig failed: %v", err)
	}

	if output.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", output.Title)
	}

	if output.Weather == nil {
		t.Fatal("expected weather config, got nil")
	}
	if output.Weather.Latitude != 40.7 {
		t.Errorf("expected latitude 40.7, got %f", output.Weather.Latitude)
	}
	if output.Weather.Longitude != -74.0 {
		t.Errorf("expected longitude -74.0, got %f", output.Weather.Longitude)
	}
	if output.Weather.Unit != "fahrenheit" {
		t.Errorf("expected unit 'fahrenheit', got %q", output.Weather.Unit)
	}
	if output.Weather.CacheMinutes != 30 {
		t.Errorf("expected cache_minutes 30, got %d", output.Weather.CacheMinutes)
	}

	if len(output.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(output.Categories))
	}
	if output.Categories[0].Name != "Favorites" {
		t.Errorf("expected first category 'Favorites', got %q", output.Categories[0].Name)
	}
	if output.Categories[1].Name != "Work" {
		t.Errorf("expected second category 'Work', got %q", output.Categories[1].Name)
	}
	if len(output.Categories[0].Links) != 2 {
		t.Errorf("expected 2 links in Favorites, got %d", len(output.Categories[0].Links))
	}
	if len(output.Categories[1].Links) != 1 {
		t.Errorf("expected 1 link in Work, got %d", len(output.Categories[1].Links))
	}

	workLink := output.Categories[1].Links[0]
	if workLink.Private == nil || *workLink.Private != true {
		t.Error("expected private=true on Work link, got nil or false")
	}
}

func TestImportConfigWipesExisting(t *testing.T) {
	ds := setupTestDB(t)

	// Create some data manually
	_, err := ds.CreateCategory("OldCategory")
	if err != nil {
		t.Fatal(err)
	}

	// Import a config with different data
	cfg := config.Config{
		Title: "Fresh",
		Categories: []config.Category{
			{Name: "NewCat", Links: []config.Link{{Name: "Link1", URL: "https://example.com"}}},
		},
	}
	err = ds.ImportConfig(cfg)
	if err != nil {
		t.Fatalf("ImportConfig failed: %v", err)
	}

	cats, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		t.Fatalf("GetCategoriesWithBookmarks failed: %v", err)
	}

	if len(cats) != 1 {
		t.Errorf("expected 1 category after import, got %d", len(cats))
	}
	if cats[0].Name != "NewCat" {
		t.Errorf("expected category 'NewCat', got %q", cats[0].Name)
	}
}

func TestInheritedIDQueries(t *testing.T) {
	ds := setupTestDB(t)

	// catA: private=nil (inherits)
	catA, err := ds.CreateCategory("CatA")
	if err != nil {
		t.Fatal(err)
	}

	// catB: private=true (explicit)
	catB, err := ds.CreateCategory("CatB")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ds.ToggleCategoryPrivate(catB.ID) // nil → true
	if err != nil {
		t.Fatal(err)
	}

	// bm1 in catA: private=nil (inherits from catA which inherits)
	bm1, err := ds.CreateBookmark(catA.ID)
	if err != nil {
		t.Fatal(err)
	}

	// bm2 in catA: private=true (explicit)
	bm2, err := ds.CreateBookmark(catA.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ds.ToggleBookmarkPrivate(bm2.ID) // nil → true
	if err != nil {
		t.Fatal(err)
	}

	// bm3 in catB: private=nil (but catB has private=true)
	_, err = ds.CreateBookmark(catB.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Test GetCategoryIDsWithInheritedPrivate
	catIDs, err := ds.GetCategoryIDsWithInheritedPrivate()
	if err != nil {
		t.Fatal(err)
	}
	catIDSet := make(map[model.CategoryID]bool)
	for _, id := range catIDs {
		catIDSet[id] = true
	}
	if !catIDSet[catA.ID] {
		t.Errorf("expected catA (id=%d) in inherited private categories", catA.ID)
	}
	favID := model.CategoryID(1)
	if !catIDSet[favID] {
		t.Errorf("expected Favorites (id=%d) in inherited private categories", favID)
	}
	if catIDSet[catB.ID] {
		t.Errorf("catB (id=%d) should NOT be in inherited private categories", catB.ID)
	}

	// Test GetBookmarkIDsWithInheritedPrivate
	bmIDs, err := ds.GetBookmarkIDsWithInheritedPrivate()
	if err != nil {
		t.Fatal(err)
	}
	bmIDSet := make(map[model.BookmarkID]bool)
	for _, id := range bmIDs {
		bmIDSet[id] = true
	}
	if !bmIDSet[bm1.ID] {
		t.Errorf("expected bm1 (id=%d) in inherited private bookmarks", bm1.ID)
	}
	if bmIDSet[bm2.ID] {
		t.Errorf("bm2 (id=%d) should NOT be in inherited private bookmarks", bm2.ID)
	}

	// Test GetBookmarkIDsInCategoryWithInheritedPrivate for catA
	catABmIDs, err := ds.GetBookmarkIDsInCategoryWithInheritedPrivate(catA.ID)
	if err != nil {
		t.Fatal(err)
	}
	catABmIDSet := make(map[model.BookmarkID]bool)
	for _, id := range catABmIDs {
		catABmIDSet[id] = true
	}
	if !catABmIDSet[bm1.ID] {
		t.Errorf("expected bm1 (id=%d) in catA inherited private bookmarks", bm1.ID)
	}
	if catABmIDSet[bm2.ID] {
		t.Errorf("bm2 (id=%d) should NOT be in catA inherited private bookmarks", bm2.ID)
	}

	// Same pattern for NewTab queries
	_, err = ds.ToggleCategoryOpenInNewTab(catB.ID) // nil → true
	if err != nil {
		t.Fatal(err)
	}
	_, err = ds.ToggleBookmarkOpenInNewTab(bm2.ID) // nil → true
	if err != nil {
		t.Fatal(err)
	}

	// Test GetCategoryIDsWithInheritedNewTab
	catNewTabIDs, err := ds.GetCategoryIDsWithInheritedNewTab()
	if err != nil {
		t.Fatal(err)
	}
	catNewTabSet := make(map[model.CategoryID]bool)
	for _, id := range catNewTabIDs {
		catNewTabSet[id] = true
	}
	if !catNewTabSet[catA.ID] {
		t.Errorf("expected catA (id=%d) in inherited new tab categories", catA.ID)
	}
	if catNewTabSet[catB.ID] {
		t.Errorf("catB (id=%d) should NOT be in inherited new tab categories", catB.ID)
	}

	// Test GetBookmarkIDsWithInheritedNewTab
	bmNewTabIDs, err := ds.GetBookmarkIDsWithInheritedNewTab()
	if err != nil {
		t.Fatal(err)
	}
	bmNewTabSet := make(map[model.BookmarkID]bool)
	for _, id := range bmNewTabIDs {
		bmNewTabSet[id] = true
	}
	if !bmNewTabSet[bm1.ID] {
		t.Errorf("expected bm1 (id=%d) in inherited new tab bookmarks", bm1.ID)
	}
	if bmNewTabSet[bm2.ID] {
		t.Errorf("bm2 (id=%d) should NOT be in inherited new tab bookmarks", bm2.ID)
	}

	// Test GetBookmarkIDsInCategoryWithInheritedNewTab for catA
	catANewTabIDs, err := ds.GetBookmarkIDsInCategoryWithInheritedNewTab(catA.ID)
	if err != nil {
		t.Fatal(err)
	}
	catANewTabSet := make(map[model.BookmarkID]bool)
	for _, id := range catANewTabIDs {
		catANewTabSet[id] = true
	}
	if !catANewTabSet[bm1.ID] {
		t.Errorf("expected bm1 (id=%d) in catA inherited new tab bookmarks", bm1.ID)
	}
	if catANewTabSet[bm2.ID] {
		t.Errorf("bm2 (id=%d) should NOT be in catA inherited new tab bookmarks", bm2.ID)
	}
}
