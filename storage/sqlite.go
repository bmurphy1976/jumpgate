package storage

import (
	"dashboard/config"
	"dashboard/model"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB opens a file-backed SQLite database and initializes the schema.
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// WAL mode allows concurrent reads during writes (default rollback mode blocks reads).
	// Only meaningful for file-backed databases.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	return initSQLiteDB(db)
}

// NewMemorySQLiteDB creates an in-memory SQLite database with the same schema.
// No WAL pragma — it is meaningless for :memory: databases.
func NewMemorySQLiteDB() (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open memory database: %w", err)
	}
	return initSQLiteDB(db)
}

// initSQLiteDB applies shared setup (foreign keys, schema, migrations) to a sql.DB.
func initSQLiteDB(db *sql.DB) (*SQLiteDB, error) {
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &SQLiteDB{db: db}
	if err := s.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	if err := s.migrateSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return s, nil
}

func (s *SQLiteDB) createSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS schema_version (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    title TEXT NOT NULL DEFAULT 'My Dashboard',
    weather_latitude TEXT NOT NULL DEFAULT '0',
    weather_longitude TEXT NOT NULL DEFAULT '0',
    weather_unit TEXT NOT NULL DEFAULT 'fahrenheit' CHECK (weather_unit IN ('fahrenheit', 'celsius')),
    weather_cache_minutes INTEGER NOT NULL DEFAULT 30,
    default_private BOOLEAN NOT NULL DEFAULT 1,
    default_open_in_new_tab BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    private BOOLEAN,
    is_favorites BOOLEAN NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_favorites ON categories(is_favorites) WHERE is_favorites = 1;

CREATE TABLE IF NOT EXISTS bookmarks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    mobile_url TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    open_in_new_tab BOOLEAN,
    private BOOLEAN,
    position INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_category ON bookmarks(category_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Insert schema version row if not exists
	_, err := s.db.Exec("INSERT OR IGNORE INTO schema_version (id, version) VALUES (1, 1)")
	if err != nil {
		return err
	}

	// Insert default settings row if not exists
	_, err = s.db.Exec("INSERT OR IGNORE INTO settings (id) VALUES (1)")
	if err != nil {
		return err
	}

	// Insert Favorites category if not exists
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO categories (id, name, position, is_favorites)
		SELECT 1, 'Favorites', 0, 1
		WHERE NOT EXISTS (SELECT 1 FROM categories WHERE is_favorites = 1)
	`)
	return err
}

func (s *SQLiteDB) migrateSchema() error {
	var version int
	if err := s.db.QueryRow("SELECT version FROM schema_version WHERE id = 1").Scan(&version); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version < 2 {
		if _, err := s.db.Exec(`ALTER TABLE bookmarks ADD COLUMN keywords TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if _, err := s.db.Exec("UPDATE schema_version SET version = 2 WHERE id = 1"); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err := s.db.Exec(`ALTER TABLE categories ADD COLUMN open_in_new_tab BOOLEAN`); err != nil {
			return err
		}
		if _, err := s.db.Exec("UPDATE schema_version SET version = 3 WHERE id = 1"); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// Settings

func (s *SQLiteDB) GetSettings() (model.Settings, error) {
	var settings model.Settings
	err := s.db.QueryRow(`
		SELECT title, weather_latitude, weather_longitude, weather_unit,
		       weather_cache_minutes, default_private, default_open_in_new_tab
		FROM settings WHERE id = 1
	`).Scan(
		&settings.Title,
		&settings.WeatherLatitude,
		&settings.WeatherLongitude,
		&settings.WeatherUnit,
		&settings.WeatherCacheMinutes,
		&settings.DefaultPrivate,
		&settings.DefaultOpenInNewTab,
	)
	return settings, err
}

func (s *SQLiteDB) UpdateSettings(update model.SettingsUpdate) error {
	var setClauses []string
	var args []any

	if update.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *update.Title)
	}
	if update.WeatherLatitude != nil {
		setClauses = append(setClauses, "weather_latitude = ?")
		args = append(args, *update.WeatherLatitude)
	}
	if update.WeatherLongitude != nil {
		setClauses = append(setClauses, "weather_longitude = ?")
		args = append(args, *update.WeatherLongitude)
	}
	if update.WeatherUnit != nil {
		setClauses = append(setClauses, "weather_unit = ?")
		args = append(args, *update.WeatherUnit)
	}
	if update.WeatherCacheMinutes != nil {
		setClauses = append(setClauses, "weather_cache_minutes = ?")
		args = append(args, *update.WeatherCacheMinutes)
	}
	if update.DefaultPrivate != nil {
		setClauses = append(setClauses, "default_private = ?")
		args = append(args, *update.DefaultPrivate)
	}
	if update.DefaultOpenInNewTab != nil {
		setClauses = append(setClauses, "default_open_in_new_tab = ?")
		args = append(args, *update.DefaultOpenInNewTab)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE settings SET %s WHERE id = 1", strings.Join(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	return err
}

// Categories

func (s *SQLiteDB) GetCategoriesWithBookmarks() ([]model.Category, error) {
	rows, err := s.db.Query(`
		SELECT id, name, position, enabled, private, open_in_new_tab, is_favorites
		FROM categories
		ORDER BY position
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		var private, openInNewTab sql.NullBool
		if err := rows.Scan(&c.ID, &c.Name, &c.Position, &c.Enabled, &private, &openInNewTab, &c.IsFavorites); err != nil {
			return nil, err
		}
		c.Private = nullBoolPtr(private)
		c.OpenInNewTab = nullBoolPtr(openInNewTab)
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load bookmarks for each category
	for i := range categories {
		bookmarks, err := s.getBookmarksByCategory(categories[i].ID)
		if err != nil {
			return nil, err
		}
		categories[i].Bookmarks = bookmarks
	}

	return categories, nil
}

func (s *SQLiteDB) getBookmarksByCategory(categoryID model.CategoryID) ([]model.Bookmark, error) {
	rows, err := s.db.Query(`
		SELECT id, category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, position, keywords
		FROM bookmarks
		WHERE category_id = ?
		ORDER BY position
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		var openInNewTab, private sql.NullBool
		var keywords string
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Name, &b.URL, &b.MobileURL, &b.Icon, &b.Enabled, &openInNewTab, &private, &b.Position, &keywords); err != nil {
			return nil, err
		}
		b.OpenInNewTab = nullBoolPtr(openInNewTab)
		b.Private = nullBoolPtr(private)
		b.Keywords = strings.Fields(keywords)
		bookmarks = append(bookmarks, b)
	}
	return bookmarks, rows.Err()
}

func (s *SQLiteDB) GetCategory(id model.CategoryID) (model.Category, error) {
	var c model.Category
	var private, openInNewTab sql.NullBool
	err := s.db.QueryRow(`
		SELECT id, name, position, enabled, private, open_in_new_tab, is_favorites
		FROM categories WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Position, &c.Enabled, &private, &openInNewTab, &c.IsFavorites)
	if err != nil {
		return c, err
	}
	c.Private = nullBoolPtr(private)
	c.OpenInNewTab = nullBoolPtr(openInNewTab)

	bookmarks, err := s.getBookmarksByCategory(id)
	if err != nil {
		return c, err
	}
	c.Bookmarks = bookmarks
	return c, nil
}

func (s *SQLiteDB) CreateCategory(name string) (model.Category, error) {
	// Shift all non-favorites categories down to make room at the top
	_, err := s.db.Exec("UPDATE categories SET position = position + 1 WHERE is_favorites = 0")
	if err != nil {
		return model.Category{}, err
	}

	// Insert at position 1 (after favorites at position 0)
	result, err := s.db.Exec(`
		INSERT INTO categories (name, position, is_favorites)
		VALUES (?, 1, 0)
	`, name)
	if err != nil {
		return model.Category{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Category{}, err
	}

	return s.GetCategory(model.CategoryID(id))
}

func (s *SQLiteDB) UpdateCategory(id model.CategoryID, update model.CategoryUpdate) error {
	if update.Name == nil {
		return nil
	}
	_, err := s.db.Exec("UPDATE categories SET name = ? WHERE id = ?", *update.Name, id)
	return err
}

func (s *SQLiteDB) DeleteCategory(id model.CategoryID) error {
	_, err := s.db.Exec("DELETE FROM categories WHERE id = ?", id)
	return err
}

func (s *SQLiteDB) ReorderCategories(ids []model.CategoryID) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE categories SET position = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for pos, id := range ids {
		if _, err := stmt.Exec(pos, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteDB) cycleNullBool(table, column string, id any) (*bool, error) {
	var current sql.NullBool
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", column, table)
	if err := s.db.QueryRow(query, id).Scan(&current); err != nil {
		return nil, err
	}

	var next *bool
	if !current.Valid {
		t := true
		next = &t
	} else if current.Bool {
		f := false
		next = &f
	} else {
		next = nil
	}

	var nextVal any
	if next != nil {
		nextVal = *next
	}

	update := fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", table, column)
	_, err := s.db.Exec(update, nextVal, id)
	return next, err
}

func (s *SQLiteDB) ToggleCategoryPrivate(id model.CategoryID) (*bool, error) {
	return s.cycleNullBool("categories", "private", id)
}

func (s *SQLiteDB) ToggleCategoryOpenInNewTab(id model.CategoryID) (*bool, error) {
	return s.cycleNullBool("categories", "open_in_new_tab", id)
}

// Bookmarks

func (s *SQLiteDB) GetBookmark(id model.BookmarkID) (model.Bookmark, error) {
	var b model.Bookmark
	var openInNewTab, private sql.NullBool
	var keywords string
	err := s.db.QueryRow(`
		SELECT id, category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, position, keywords
		FROM bookmarks WHERE id = ?
	`, id).Scan(&b.ID, &b.CategoryID, &b.Name, &b.URL, &b.MobileURL, &b.Icon, &b.Enabled, &openInNewTab, &private, &b.Position, &keywords)
	if err != nil {
		return b, err
	}
	b.OpenInNewTab = nullBoolPtr(openInNewTab)
	b.Private = nullBoolPtr(private)
	b.Keywords = strings.Fields(keywords)
	return b, nil
}

func (s *SQLiteDB) CreateBookmark(categoryID model.CategoryID) (model.Bookmark, error) {
	_, err := s.db.Exec("UPDATE bookmarks SET position = position + 1 WHERE category_id = ?", categoryID)
	if err != nil {
		return model.Bookmark{}, err
	}

	result, err := s.db.Exec(`
		INSERT INTO bookmarks (category_id, name, url, position)
		VALUES (?, '', '', 0)
	`, categoryID)
	if err != nil {
		return model.Bookmark{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Bookmark{}, err
	}

	return s.GetBookmark(model.BookmarkID(id))
}

func (s *SQLiteDB) UpdateBookmark(id model.BookmarkID, update model.BookmarkUpdate) error {
	var setClauses []string
	var args []any

	if update.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *update.Name)
	}
	if update.URL != nil {
		setClauses = append(setClauses, "url = ?")
		args = append(args, *update.URL)
	}
	if update.MobileURL != nil {
		setClauses = append(setClauses, "mobile_url = ?")
		args = append(args, *update.MobileURL)
	}
	if update.Icon != nil {
		setClauses = append(setClauses, "icon = ?")
		args = append(args, *update.Icon)
	}
	if update.Keywords != nil {
		setClauses = append(setClauses, "keywords = ?")
		args = append(args, strings.Join(*update.Keywords, " "))
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE bookmarks SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	return err
}

func (s *SQLiteDB) DeleteBookmark(id model.BookmarkID) error {
	_, err := s.db.Exec("DELETE FROM bookmarks WHERE id = ?", id)
	return err
}

func (s *SQLiteDB) DuplicateBookmark(id model.BookmarkID) (model.Bookmark, error) {
	var maxPos int
	var categoryID model.CategoryID
	err := s.db.QueryRow("SELECT category_id FROM bookmarks WHERE id = ?", id).Scan(&categoryID)
	if err != nil {
		return model.Bookmark{}, err
	}

	if err := s.db.QueryRow("SELECT COALESCE(MAX(position), -1) FROM bookmarks WHERE category_id = ?", categoryID).Scan(&maxPos); err != nil {
		return model.Bookmark{}, fmt.Errorf("max position: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO bookmarks (category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, position, keywords)
		SELECT category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, ?, keywords
		FROM bookmarks WHERE id = ?
	`, maxPos+1, id)
	if err != nil {
		return model.Bookmark{}, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return model.Bookmark{}, err
	}

	return s.GetBookmark(model.BookmarkID(newID))
}

func (s *SQLiteDB) SortBookmarksAlpha(categoryID model.CategoryID) error {
	rows, err := s.db.Query(`
		SELECT id FROM bookmarks
		WHERE category_id = ?
		ORDER BY LOWER(name)
	`, categoryID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []model.BookmarkID
	for rows.Next() {
		var id model.BookmarkID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return s.ReorderBookmarks(categoryID, ids)
}

func (s *SQLiteDB) ReorderBookmarks(categoryID model.CategoryID, ids []model.BookmarkID) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE bookmarks SET position = ? WHERE id = ? AND category_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for pos, id := range ids {
		if _, err := stmt.Exec(pos, id, categoryID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteDB) MoveBookmark(id model.BookmarkID, targetCategoryID model.CategoryID, position int) error {
	_, err := s.db.Exec(`
		UPDATE bookmarks
		SET category_id = ?, position = ?
		WHERE id = ?
	`, targetCategoryID, position, id)
	return err
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (s *SQLiteDB) SearchBookmarks(url, query string) ([]model.Bookmark, error) {
	var conditions []string
	var args []any

	if url != "" {
		conditions = append(conditions, "b.url = ?")
		args = append(args, url)
	}
	if query != "" {
		conditions = append(conditions, "(b.name LIKE ? ESCAPE '\\' OR b.url LIKE ? ESCAPE '\\' OR b.keywords LIKE ? ESCAPE '\\')")
		pattern := "%" + escapeLike(query) + "%"
		args = append(args, pattern, pattern, pattern)
	}

	if len(conditions) == 0 {
		return []model.Bookmark{}, nil
	}

	q := "SELECT id, category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, position, keywords FROM bookmarks b WHERE " + strings.Join(conditions, " AND ") + " ORDER BY name"
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		var openInNewTab, private sql.NullBool
		var keywords string
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Name, &b.URL, &b.MobileURL, &b.Icon, &b.Enabled, &openInNewTab, &private, &b.Position, &keywords); err != nil {
			return nil, err
		}
		b.OpenInNewTab = nullBoolPtr(openInNewTab)
		b.Private = nullBoolPtr(private)
		b.Keywords = strings.Fields(keywords)
		bookmarks = append(bookmarks, b)
	}
	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	return bookmarks, rows.Err()
}

func (s *SQLiteDB) ToggleBookmarkPrivate(id model.BookmarkID) (*bool, error) {
	return s.cycleNullBool("bookmarks", "private", id)
}

func (s *SQLiteDB) ToggleBookmarkOpenInNewTab(id model.BookmarkID) (*bool, error) {
	return s.cycleNullBool("bookmarks", "open_in_new_tab", id)
}

func queryIDs[T ~int](db *sql.DB, query string, args ...any) ([]T, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []T
	for rows.Next() {
		var id T
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SQLiteDB) GetCategoryIDsWithInheritedPrivate() ([]model.CategoryID, error) {
	return queryIDs[model.CategoryID](s.db, "SELECT id FROM categories WHERE private IS NULL")
}

func (s *SQLiteDB) GetBookmarkIDsWithInheritedPrivate() ([]model.BookmarkID, error) {
	return queryIDs[model.BookmarkID](s.db, `
		SELECT b.id FROM bookmarks b
		JOIN categories c ON b.category_id = c.id
		WHERE b.private IS NULL AND c.private IS NULL
	`)
}

func (s *SQLiteDB) GetCategoryIDsWithInheritedNewTab() ([]model.CategoryID, error) {
	return queryIDs[model.CategoryID](s.db, "SELECT id FROM categories WHERE open_in_new_tab IS NULL")
}

func (s *SQLiteDB) GetBookmarkIDsWithInheritedNewTab() ([]model.BookmarkID, error) {
	return queryIDs[model.BookmarkID](s.db, `
		SELECT b.id FROM bookmarks b
		JOIN categories c ON b.category_id = c.id
		WHERE b.open_in_new_tab IS NULL AND c.open_in_new_tab IS NULL
	`)
}

func (s *SQLiteDB) GetBookmarkIDsInCategoryWithInheritedNewTab(categoryID model.CategoryID) ([]model.BookmarkID, error) {
	return queryIDs[model.BookmarkID](s.db, "SELECT id FROM bookmarks WHERE category_id = ? AND open_in_new_tab IS NULL", categoryID)
}

func (s *SQLiteDB) GetBookmarkIDsInCategoryWithInheritedPrivate(categoryID model.CategoryID) ([]model.BookmarkID, error) {
	return queryIDs[model.BookmarkID](s.db, "SELECT id FROM bookmarks WHERE category_id = ? AND private IS NULL", categoryID)
}

func (s *SQLiteDB) ToggleCategoryEnabled(id model.CategoryID) (bool, error) {
	var current bool
	err := s.db.QueryRow("SELECT enabled FROM categories WHERE id = ?", id).Scan(&current)
	if err != nil {
		return false, err
	}
	next := !current
	_, err = s.db.Exec("UPDATE categories SET enabled = ? WHERE id = ?", next, id)
	return next, err
}

func (s *SQLiteDB) ToggleBookmarkEnabled(id model.BookmarkID) (bool, error) {
	var current bool
	err := s.db.QueryRow("SELECT enabled FROM bookmarks WHERE id = ?", id).Scan(&current)
	if err != nil {
		return false, err
	}
	next := !current
	_, err = s.db.Exec("UPDATE bookmarks SET enabled = ? WHERE id = ?", next, id)
	return next, err
}

// ImportConfig wipes all existing data and repopulates from the given config.
func (s *SQLiteDB) ImportConfig(cfg config.Config) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM categories"); err != nil {
		return fmt.Errorf("clear categories: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM settings"); err != nil {
		return fmt.Errorf("clear settings: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO settings (id) VALUES (1)"); err != nil {
		return fmt.Errorf("reset settings: %w", err)
	}
	if cfg.Title != "" {
		if _, err := tx.Exec("UPDATE settings SET title = ? WHERE id = 1", cfg.Title); err != nil {
			return fmt.Errorf("update title: %w", err)
		}
	}
	if cfg.Weather != nil {
		_, err := tx.Exec(
			"UPDATE settings SET weather_latitude = ?, weather_longitude = ?, weather_unit = ?, weather_cache_minutes = ? WHERE id = 1",
			strconv.FormatFloat(cfg.Weather.Latitude, 'f', -1, 64),
			strconv.FormatFloat(cfg.Weather.Longitude, 'f', -1, 64),
			cfg.Weather.Unit,
			cfg.Weather.CacheMinutes,
		)
		if err != nil {
			return fmt.Errorf("update weather settings: %w", err)
		}
	}

	for i, cat := range cfg.Categories {
		catEnabled := cat.Enabled == nil || *cat.Enabled
		result, err := tx.Exec(
			"INSERT INTO categories (name, position, is_favorites, enabled, private, open_in_new_tab) VALUES (?, ?, ?, ?, ?, ?)",
			cat.Name, i, i == 0, catEnabled, nullBool(cat.Private), nullBool(cat.OpenInNewTab),
		)
		if err != nil {
			return fmt.Errorf("insert category %q: %w", cat.Name, err)
		}
		catID, _ := result.LastInsertId()
		for j, link := range cat.Links {
			linkEnabled := link.Enabled == nil || *link.Enabled
			_, err := tx.Exec(
				"INSERT INTO bookmarks (category_id, name, url, mobile_url, icon, enabled, open_in_new_tab, private, position, keywords) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				catID, link.Name, link.URL, link.MobileURL, link.Icon,
				linkEnabled, nullBool(link.OpenInNewTab), nullBool(link.Private), j, strings.Join(link.Keywords, " "),
			)
			if err != nil {
				return fmt.Errorf("insert bookmark %q: %w", link.Name, err)
			}
		}
	}

	return tx.Commit()
}

// ExportConfig reads the current database state and returns it as a config.Config.
func (s *SQLiteDB) ExportConfig() (config.Config, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return config.Config{}, fmt.Errorf("read settings: %w", err)
	}

	cats, err := s.GetCategoriesWithBookmarks()
	if err != nil {
		return config.Config{}, fmt.Errorf("read categories: %w", err)
	}

	cfg := config.Config{Title: settings.Title}

	lat, _ := strconv.ParseFloat(settings.WeatherLatitude, 64)
	lon, _ := strconv.ParseFloat(settings.WeatherLongitude, 64)
	if lat != 0 || lon != 0 {
		cfg.Weather = &config.Weather{
			Latitude:     lat,
			Longitude:    lon,
			Unit:         settings.WeatherUnit,
			CacheMinutes: settings.WeatherCacheMinutes,
		}
	}

	for _, cat := range cats {
		cfgCat := config.Category{Name: cat.Name, Private: cat.Private, OpenInNewTab: cat.OpenInNewTab}
		if !cat.Enabled {
			f := false
			cfgCat.Enabled = &f
		}
		for _, b := range cat.Bookmarks {
			link := config.Link{
				Name:         b.Name,
				URL:          b.URL,
				MobileURL:    b.MobileURL,
				Icon:         b.Icon,
				OpenInNewTab: b.OpenInNewTab,
				Private:      b.Private,
				Keywords:     b.Keywords,
			}
			if !b.Enabled {
				f := false
				link.Enabled = &f
			}
			cfgCat.Links = append(cfgCat.Links, link)
		}
		cfg.Categories = append(cfg.Categories, cfgCat)
	}

	return cfg, nil
}

func nullBoolPtr(nb sql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	return &nb.Bool
}

func nullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Valid: true, Bool: *b}
}
