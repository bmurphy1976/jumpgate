package icons

import (
	"dashboard/common"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var iconCacheFile = common.IconCacheDir + ".txt"

type Loader struct {
	Icons []string
}

func New() (*Loader, error) {
	il := &Loader{}

	if err := il.loadFromCDN(); err != nil {
		if err := il.loadFromCache(); err != nil {
			il.Icons = []string{}
		}
	} else {
		il.saveToCache()
	}

	sort.Strings(il.Icons)
	return il, nil
}

func (il *Loader) loadFromCDN() error {
	resp, err := common.HTTPClient().Get(common.MDISVGMetaURL)
	if err != nil {
		return fmt.Errorf("fetch MDI metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("fetch MDI metadata: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read MDI metadata: %w", err)
	}

	var icons []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &icons); err != nil {
		return fmt.Errorf("parse MDI metadata: %w", err)
	}

	il.Icons = make([]string, 0, len(icons))
	for _, icon := range icons {
		if icon.Name != "" {
			il.Icons = append(il.Icons, icon.Name)
		}
	}

	if len(il.Icons) == 0 {
		return fmt.Errorf("no icons found in metadata")
	}

	return nil
}

func (il *Loader) loadFromCache() error {
	data, err := os.ReadFile(iconCacheFile)
	if err != nil {
		return fmt.Errorf("read cache: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return fmt.Errorf("cache is empty")
	}

	il.Icons = strings.Split(content, "\n")
	return nil
}

func (il *Loader) saveToCache() error {
	if len(il.Icons) == 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(iconCacheFile), 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data := strings.Join(il.Icons, "\n")
	if err := os.WriteFile(iconCacheFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}

	return nil
}

func (il *Loader) Search(query string) []string {
	if query == "" {
		if len(il.Icons) > 50 {
			return il.Icons[:50]
		}
		return il.Icons
	}

	q := strings.ToLower(query)
	var results []string

	for _, icon := range il.Icons {
		if strings.Contains(icon, q) {
			results = append(results, icon)
			if len(results) >= 50 {
				break
			}
		}
	}

	return results
}
