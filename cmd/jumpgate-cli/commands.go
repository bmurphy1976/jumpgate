package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
)

// Category commands

func runCategory(c *apiClient, verb string, args []string) error {
	switch verb {
	case "list":
		return categoryList(c)
	case "get":
		return categoryGet(c, args)
	case "create":
		return categoryCreate(c, args)
	case "update":
		return categoryUpdate(c, args)
	case "delete":
		return categoryDelete(c, args)
	default:
		return fmt.Errorf("unknown category verb: %s", verb)
	}
}

func categoryList(c *apiClient) error {
	data, err := c.get("/api/categories")
	if err != nil {
		return err
	}
	return printJSON(data)
}

func categoryGet(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("category get", flag.ExitOnError)
	id := fs.Int("category-id", 0, "category ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--category-id is required")
	}
	data, err := c.get(fmt.Sprintf("/api/categories/%d", *id))
	if err != nil {
		return err
	}
	return printJSON(data)
}

func categoryCreate(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("category create", flag.ExitOnError)
	name := fs.String("name", "", "category name")
	fs.Parse(args)
	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	data, err := c.post("/api/categories", map[string]string{"name": *name})
	if err != nil {
		return err
	}
	return printJSON(data)
}

func categoryUpdate(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("category update", flag.ExitOnError)
	id := fs.Int("category-id", 0, "category ID")
	name := fs.String("name", "", "new name")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--category-id is required")
	}
	body := map[string]any{}
	if *name != "" {
		body["name"] = *name
	}
	data, err := c.put(fmt.Sprintf("/api/categories/%d", *id), body)
	if err != nil {
		return err
	}
	return printJSON(data)
}

func categoryDelete(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("category delete", flag.ExitOnError)
	id := fs.Int("category-id", 0, "category ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--category-id is required")
	}
	_, err := c.delete(fmt.Sprintf("/api/categories/%d", *id))
	return err
}

// Bookmark commands

func runBookmark(c *apiClient, verb string, args []string) error {
	switch verb {
	case "create":
		return bookmarkCreate(c, args)
	case "get":
		return bookmarkGet(c, args)
	case "update":
		return bookmarkUpdate(c, args)
	case "delete":
		return bookmarkDelete(c, args)
	case "move":
		return bookmarkMove(c, args)
	case "search":
		return bookmarkSearch(c, args)
	default:
		return fmt.Errorf("unknown bookmark verb: %s", verb)
	}
}

type multiString []string

func (m *multiString) String() string { return fmt.Sprintf("%v", *m) }
func (m *multiString) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func bookmarkCreate(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark create", flag.ExitOnError)
	catID := fs.Int("category-id", 0, "category ID")
	name := fs.String("name", "", "bookmark name")
	bmURL := fs.String("url", "", "bookmark URL")
	icon := fs.String("icon", "", "icon name")
	var keywords multiString
	fs.Var(&keywords, "keyword", "keyword (repeatable)")
	fs.Parse(args)
	if *catID == 0 {
		return fmt.Errorf("--category-id is required")
	}
	body := map[string]any{"category_id": *catID}
	if *name != "" {
		body["name"] = *name
	}
	if *bmURL != "" {
		body["url"] = *bmURL
	}
	if *icon != "" {
		body["icon"] = *icon
	}
	if len(keywords) > 0 {
		body["keywords"] = []string(keywords)
	}
	data, err := c.post("/api/bookmarks", body)
	if err != nil {
		return err
	}
	return printJSON(data)
}

func bookmarkGet(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark get", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	data, err := c.get(fmt.Sprintf("/api/bookmarks/%d", *id))
	if err != nil {
		return err
	}
	return printJSON(data)
}

func bookmarkUpdate(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark update", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	name := fs.String("name", "", "bookmark name")
	bmURL := fs.String("url", "", "bookmark URL")
	mobileURL := fs.String("mobile-url", "", "mobile URL")
	icon := fs.String("icon", "", "icon name")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	body := map[string]any{}
	if *name != "" {
		body["name"] = *name
	}
	if *bmURL != "" {
		body["url"] = *bmURL
	}
	if *mobileURL != "" {
		body["mobile_url"] = *mobileURL
	}
	if *icon != "" {
		body["icon"] = *icon
	}
	data, err := c.put(fmt.Sprintf("/api/bookmarks/%d", *id), body)
	if err != nil {
		return err
	}
	return printJSON(data)
}

func bookmarkDelete(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark delete", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	_, err := c.delete(fmt.Sprintf("/api/bookmarks/%d", *id))
	return err
}

func bookmarkMove(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark move", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	catID := fs.Int("category-id", 0, "target category ID")
	pos := fs.Int("position", 0, "position in target category")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	if *catID == 0 {
		return fmt.Errorf("--category-id is required")
	}
	data, err := c.post(fmt.Sprintf("/api/bookmarks/%d/move", *id), map[string]any{
		"category_id": *catID,
		"position":    *pos,
	})
	if err != nil {
		return err
	}
	return printJSON(data)
}

func bookmarkSearch(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("bookmark search", flag.ExitOnError)
	searchURL := fs.String("url", "", "exact URL match")
	query := fs.String("query", "", "substring search")
	fs.Parse(args)
	params := url.Values{}
	if *searchURL != "" {
		params.Set("url", *searchURL)
	}
	if *query != "" {
		params.Set("q", *query)
	}
	path := "/api/bookmarks/search"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	data, err := c.get(path)
	if err != nil {
		return err
	}
	return printJSON(data)
}

// Keyword commands

func runKeyword(c *apiClient, verb string, args []string) error {
	switch verb {
	case "list":
		return keywordList(c, args)
	case "add":
		return keywordAdd(c, args)
	case "delete":
		return keywordDelete(c, args)
	case "set":
		return keywordSet(c, args)
	case "clear":
		return keywordClear(c, args)
	default:
		return fmt.Errorf("unknown keyword verb: %s", verb)
	}
}

func keywordList(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("keyword list", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	data, err := c.get(fmt.Sprintf("/api/bookmarks/%d", *id))
	if err != nil {
		return err
	}
	var bm struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal(data, &bm); err != nil {
		return err
	}
	out, _ := json.MarshalIndent(bm.Keywords, "", "  ")
	fmt.Println(string(out))
	return nil
}

func keywordAdd(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("keyword add", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	words := fs.Args()
	if len(words) == 0 {
		return fmt.Errorf("at least one keyword is required")
	}

	// Read current keywords
	data, err := c.get(fmt.Sprintf("/api/bookmarks/%d", *id))
	if err != nil {
		return err
	}
	var bm struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal(data, &bm); err != nil {
		return fmt.Errorf("parse bookmark: %w", err)
	}

	// Merge (deduplicate)
	existing := make(map[string]bool)
	for _, k := range bm.Keywords {
		existing[k] = true
	}
	merged := bm.Keywords
	for _, w := range words {
		if !existing[w] {
			merged = append(merged, w)
		}
	}

	data, err = c.put(fmt.Sprintf("/api/bookmarks/%d", *id), map[string]any{"keywords": merged})
	if err != nil {
		return err
	}
	return printJSON(data)
}

func keywordDelete(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("keyword delete", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	words := fs.Args()
	if len(words) == 0 {
		return fmt.Errorf("at least one keyword is required")
	}

	data, err := c.get(fmt.Sprintf("/api/bookmarks/%d", *id))
	if err != nil {
		return err
	}
	var bm struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal(data, &bm); err != nil {
		return fmt.Errorf("parse bookmark: %w", err)
	}

	remove := make(map[string]bool)
	for _, w := range words {
		remove[w] = true
	}
	var filtered []string
	for _, k := range bm.Keywords {
		if !remove[k] {
			filtered = append(filtered, k)
		}
	}

	data, err = c.put(fmt.Sprintf("/api/bookmarks/%d", *id), map[string]any{"keywords": filtered})
	if err != nil {
		return err
	}
	return printJSON(data)
}

func keywordSet(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("keyword set", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	data, err := c.put(fmt.Sprintf("/api/bookmarks/%d", *id), map[string]any{"keywords": fs.Args()})
	if err != nil {
		return err
	}
	return printJSON(data)
}

func keywordClear(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("keyword clear", flag.ExitOnError)
	id := fs.Int("bookmark-id", 0, "bookmark ID")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("--bookmark-id is required")
	}
	data, err := c.put(fmt.Sprintf("/api/bookmarks/%d", *id), map[string]any{"keywords": []string{}})
	if err != nil {
		return err
	}
	return printJSON(data)
}

// Icon commands

func runIcon(c *apiClient, verb string, args []string) error {
	switch verb {
	case "list":
		return iconList(c, args)
	default:
		return fmt.Errorf("unknown icon verb: %s", verb)
	}
}

func iconList(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("icon list", flag.ExitOnError)
	query := fs.String("query", "", "substring filter")
	limit := fs.Int("limit", 0, "max results")
	offset := fs.Int("offset", 0, "pagination start")
	fs.Parse(args)
	params := url.Values{}
	if *query != "" {
		params.Set("q", *query)
	}
	if *limit > 0 {
		params.Set("limit", strconv.Itoa(*limit))
	}
	if *offset > 0 {
		params.Set("offset", strconv.Itoa(*offset))
	}
	path := "/api/icons"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	data, err := c.get(path)
	if err != nil {
		return err
	}
	return printJSON(data)
}
