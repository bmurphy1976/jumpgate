package views

// ThemeData holds theme-related data shared by all page templates.
type ThemeData struct {
	Current   string
	Available []string
	// CacheHashesRaw is a pre-serialized JSON object mapping theme names to
	// CSS file hashes (e.g. {"monokai":"abc123"}), embedded into a <script> tag
	// as window.THEME_HASHES for client-side cache-busted theme switching.
	CacheHashesRaw string
	CacheHash      string // cache-busting hash of the current theme's CSS file
}

// WeatherData holds pre-formatted weather display data shared by all page templates.
type WeatherData struct {
	Temp     string
	Humidity string
	Icon     string
}

// CDNDeps holds URLs for third-party CDN dependencies used by the admin layout.
type CDNDeps struct {
	HtmxURL       string
	SortableJSURL string
	MDIFontCSSURL string
}

// AdminLayoutData bundles all parameters for the admin Layout template.
type AdminLayoutData struct {
	Title        string
	Theme        ThemeData
	Weather      WeatherData
	IsAuthorized bool
	DemoMode     bool
	AdminCSSHash string
	AdminJSHash  string
	// ThemeJSRaw is the shared theme switcher JS, embedded inline in a <script> tag.
	ThemeJSRaw string
	Deps       CDNDeps
}

// DashboardPageData bundles all parameters for the DashboardPage template.
type DashboardPageData struct {
	Title            string
	Theme            ThemeData
	Weather          WeatherData
	IsAuthorized     bool
	DemoMode         bool
	// DashboardCSSRaw is the full contents of style.css, embedded inline in a <style> tag.
	DashboardCSSRaw string
	// ThemeCSSRaw is the full contents of the active theme's CSS, embedded inline in a <style> tag.
	ThemeCSSRaw string
	// DashboardJSRaw is theme.js + app.js concatenated, embedded inline in a <script> tag.
	DashboardJSRaw string
	Categories     []DashCategory
	// SVGSymbolsRaw is pre-built <symbol> elements for all icons used on the page,
	// embedded inside a hidden <svg> so individual icons can reference them via <use>.
	SVGSymbolsRaw string
}
