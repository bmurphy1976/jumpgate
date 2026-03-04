package common

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// HTTPClient returns a shared HTTP client with a 10-second timeout.
func HTTPClient() *http.Client { return httpClient }

// IconCacheDir is the directory for cached SVG icon files.
const IconCacheDir = "data/icons"

// Third-party CDN dependencies.
// To check for updates: /update-deps
const (
	HTMXVersion = "2.0.8"
	HTMXURL     = "https://unpkg.com/htmx.org@" + HTMXVersion

	SortableJSVersion = "1.15.7"
	SortableJSURL     = "https://cdn.jsdelivr.net/npm/sortablejs@" + SortableJSVersion + "/Sortable.min.js"

	MDIVersion    = "7"
	MDIFontCSSURL = "https://cdn.jsdelivr.net/npm/@mdi/font@" + MDIVersion + "/css/materialdesignicons.min.css"
	MDISVGMetaURL = "https://cdn.jsdelivr.net/npm/@mdi/svg@" + MDIVersion + "/meta.json"
	MDISVGBaseURL = "https://raw.githubusercontent.com/Templarian/MaterialDesign-SVG/master/svg/"
)

// Demo mode defaults.
const (
	DemoMaxSessionsDefault = 1000
	DemoSessionTTLDefault  = 30 * time.Minute
)

// DemoMaxSessions returns the DEMO_MAX_SESSIONS env var as int, or DemoMaxSessionsDefault.
func DemoMaxSessions() (int, error) {
	if v := os.Getenv("DEMO_MAX_SESSIONS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid DEMO_MAX_SESSIONS %q: %w", v, err)
		}
		if n <= 0 {
			return 0, fmt.Errorf("invalid DEMO_MAX_SESSIONS %q: must be positive", v)
		}
		return n, nil
	}
	return DemoMaxSessionsDefault, nil
}

// DemoSessionTTL returns the DEMO_SESSION_TTL env var as duration, or DemoSessionTTLDefault.
func DemoSessionTTL() (time.Duration, error) {
	if v := os.Getenv("DEMO_SESSION_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("invalid DEMO_SESSION_TTL %q: %w", v, err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("invalid DEMO_SESSION_TTL %q: must be positive", v)
		}
		return d, nil
	}
	return DemoSessionTTLDefault, nil
}

// ResolveNullBool returns *val if non-nil, otherwise defaultVal.
func ResolveNullBool(val *bool, defaultVal bool) bool {
	if val != nil {
		return *val
	}
	return defaultVal
}
