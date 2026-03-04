package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"dashboard/common"
	"dashboard/model"
	"dashboard/static"
	"dashboard/views"

	"github.com/labstack/echo/v5"
)

const defaultTheme = "default"

func discoverThemes() []string {
	entries, _ := static.FS.ReadDir("themes")
	themes := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".css") {
			continue
		}
		name = strings.TrimSuffix(name, ".css")
		themes = append(themes, name)
	}
	sort.Strings(themes)
	return themes
}

type weatherData struct {
	Temp     string
	Humidity string
	Icon     string
}

type weatherCacheEntry struct {
	data    weatherData
	expires time.Time
}

var (
	weatherCache sync.Map // "lat,lon,unit" → weatherCacheEntry
	fileHashMap  sync.Map // path → string
)

// Static assets read once from embed.FS at init.
var (
	themeJS      = readStatic("js/theme.js")
	dashboardJS  = themeJS + "\n" + readStatic("js/app.js")
	dashboardCSS = readStatic("css/style.css")
)

func readStatic(path string) string {
	b, _ := static.FS.ReadFile(path)
	return string(b)
}

type DashboardHandler struct {
	ds       DSResolver
	demoMode bool
}

func SetupDashboardRoutes(e *echo.Echo, ds DSResolver, demoMode bool) {
	h := &DashboardHandler{ds: ds, demoMode: demoMode}
	e.GET("/", h.index)
	e.POST("/set-theme", h.setTheme)
}

func (h *DashboardHandler) index(c *echo.Context) error {
	ds, err := h.ds(c)
	if err != nil {
		return err
	}
	r := (*c).Request()

	settings, err := ds.GetSettings()
	if err != nil {
		return err
	}
	categories, err := ds.GetCategoriesWithBookmarks()
	if err != nil {
		return err
	}

	authorized := isAuthorized(r) || h.demoMode
	mobile := isMobile(r)
	dashCats := buildDashCategories(categories, settings, authorized, mobile)
	themeData := resolveTheme(r)
	weather := resolveWeather(r, settings)
	iconNames := collectIconNames(dashCats, weather.Icon != "")
	symbols := buildSVGSymbols(iconNames)
	themeCSS := readStatic("themes/" + themeData.Current + ".css")

	pageData := views.DashboardPageData{
		Title:           settings.Title,
		Theme:           themeData,
		Weather:         weather,
		IsAuthorized:    authorized,
		DemoMode:        h.demoMode,
		DashboardCSSRaw: dashboardCSS,
		ThemeCSSRaw:     themeCSS,
		DashboardJSRaw:  dashboardJS,
		Categories:      dashCats,
		SVGSymbolsRaw:   symbols,
	}
	return views.DashboardPage(pageData).Render(r.Context(), (*c).Response())
}

func resolveTheme(r *http.Request) views.ThemeData {
	theme := defaultTheme
	if cookie, err := r.Cookie("theme"); err == nil && cookie.Value != "" {
		theme = cookie.Value
	}
	themes := discoverThemes()
	if !containsString(themes, theme) && len(themes) > 0 {
		theme = themes[0]
	}

	themeHashes := make(map[string]string, len(themes))
	for _, t := range themes {
		themeHashes[t] = fileHash("themes/" + t + ".css")
	}
	themeHashesJSON, _ := json.Marshal(themeHashes)

	return views.ThemeData{
		Current:        theme,
		Available:      themes,
		CacheHashesRaw: string(themeHashesJSON),
		CacheHash:      fileHash("themes/" + theme + ".css"),
	}
}

func resolveWeather(r *http.Request, settings model.Settings) views.WeatherData {
	lat, lon := parseWeatherCoords(r, settings)
	if lat == 0 && lon == 0 {
		return views.WeatherData{}
	}
	wd := getCachedWeather(lat, lon, settings.WeatherUnit, settings.WeatherCacheMinutes)
	return views.WeatherData{Temp: wd.Temp, Humidity: wd.Humidity, Icon: wd.Icon}
}

func (h *DashboardHandler) setTheme(c *echo.Context) error {
	theme := (*c).FormValue("theme")
	themes := discoverThemes()
	if !containsString(themes, theme) {
		if len(themes) > 0 {
			theme = themes[0]
		} else {
			theme = defaultTheme
		}
	}
	(*c).SetCookie(&http.Cookie{
		Name:     "theme",
		Value:    theme,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		SameSite: http.SameSiteLaxMode,
	})
	return (*c).Redirect(http.StatusSeeOther, "/")
}

func buildDashCategories(categories []model.Category, settings model.Settings, authorized, mobile bool) []views.DashCategory {
	out := make([]views.DashCategory, 0, len(categories))
	for _, cat := range categories {
		if !cat.Enabled {
			continue
		}
		catPrivate := common.ResolveNullBool(cat.Private, settings.DefaultPrivate)
		if !authorized && catPrivate {
			continue
		}
		dc := views.DashCategory{
			Name:        cat.Name,
			IsFavorites: cat.IsFavorites,
		}
		for _, bm := range cat.Bookmarks {
			if !bm.Enabled {
				continue
			}
			if !authorized && common.ResolveNullBool(bm.Private, catPrivate) {
				continue
			}
			href := bm.URL
			if mobile && bm.MobileURL != "" {
				href = bm.MobileURL
			}
			dc.Links = append(dc.Links, views.DashBookmark{
				Name:         bm.Name,
				URL:          href,
				Domain:       extractDomain(bm.URL),
				Icon:         bm.Icon,
				OpenInNewTab: common.ResolveNullBool(bm.OpenInNewTab, settings.DefaultOpenInNewTab),
				Keywords:     bm.Keywords,
			})
		}
		out = append(out, dc)
	}
	return out
}

func isAuthorized(r *http.Request) bool {
	for _, h := range []string{"X-Authorized-User", "X-User", "X-Remote-User"} {
		if strings.TrimSpace(r.Header.Get(h)) != "" {
			return true
		}
	}
	return false
}

var mobileRE = regexp.MustCompile(`(?i)android|webos|iphone|ipad|ipod|blackberry|iemobile|opera mini`)

func isMobile(r *http.Request) bool {
	return mobileRE.MatchString(r.Header.Get("User-Agent"))
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := parsed.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

func containsString(ss []string, s string) bool {
	return slices.Contains(ss, s)
}

func fileHash(path string) string {
	if v, ok := fileHashMap.Load(path); ok {
		return v.(string)
	}
	data, err := static.FS.ReadFile(path)
	if err != nil {
		return "static"
	}
	sum := md5.Sum(data)
	hash := fmt.Sprintf("%x", sum)[:8]
	fileHashMap.Store(path, hash)
	return hash
}

// Weather

var weatherIconMap = map[int]string{
	0: "weather-sunny", 1: "weather-sunny", 2: "weather-partly-cloudy",
	3: "weather-cloudy", 45: "weather-fog", 48: "weather-fog",
	51: "weather-rainy", 53: "weather-rainy", 55: "weather-rainy",
	56: "weather-snowy-rainy", 57: "weather-snowy-rainy",
	61: "weather-rainy", 63: "weather-rainy", 65: "weather-pouring",
	66: "weather-snowy-rainy", 67: "weather-snowy-rainy",
	71: "weather-snowy", 73: "weather-snowy", 75: "weather-snowy-heavy",
	77: "weather-snowy",
	80: "weather-rainy", 81: "weather-rainy", 82: "weather-pouring",
	85: "weather-snowy", 86: "weather-snowy-heavy",
	95: "weather-lightning", 96: "weather-lightning-rainy", 99: "weather-lightning-rainy",
}

func weatherIconName(code int) string {
	if icon, ok := weatherIconMap[code]; ok {
		return icon
	}
	return "weather-cloudy"
}

func parseWeatherCoords(r *http.Request, s model.Settings) (float64, float64) {
	latStr := s.WeatherLatitude
	lonStr := s.WeatherLongitude
	if c, err := r.Cookie("weather_lat"); err == nil {
		latStr = c.Value
	}
	if c, err := r.Cookie("weather_lon"); err == nil {
		lonStr = c.Value
	}
	var lat, lon float64
	fmt.Sscanf(latStr, "%f", &lat)
	fmt.Sscanf(lonStr, "%f", &lon)
	return lat, lon
}

func getCachedWeather(lat, lon float64, unit string, cacheMinutes int) weatherData {
	key := fmt.Sprintf("%.6f,%.6f,%s", lat, lon, unit)
	if v, ok := weatherCache.Load(key); ok {
		entry := v.(weatherCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.data
		}
	}
	data := fetchWeather(lat, lon, unit)
	ttl := time.Duration(cacheMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	weatherCache.Store(key, weatherCacheEntry{data: data, expires: time.Now().Add(ttl)})
	return data
}

func fetchWeather(lat, lon float64, unit string) weatherData {
	u := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.6f&longitude=%.6f"+
			"&current=temperature_2m,relative_humidity_2m,weather_code"+
			"&temperature_unit=%s&timezone=auto",
		lat, lon, unit,
	)
	resp, err := common.HTTPClient().Get(u)
	if err != nil {
		slog.Warn("weather fetch failed", "error", err)
		return weatherData{Temp: "N/A", Humidity: "N/A", Icon: "weather-cloudy"}
	}
	defer resp.Body.Close()

	var result struct {
		Current struct {
			Temperature2m      float64 `json:"temperature_2m"`
			RelativeHumidity2m int     `json:"relative_humidity_2m"`
			WeatherCode        int     `json:"weather_code"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("weather decode failed", "error", err)
		return weatherData{Temp: "N/A", Humidity: "N/A", Icon: "weather-cloudy"}
	}

	unitSymbol := "°F"
	if unit == "celsius" {
		unitSymbol = "°C"
	}
	c := result.Current
	return weatherData{
		Temp:     fmt.Sprintf("%.0f%s", c.Temperature2m, unitSymbol),
		Humidity: fmt.Sprintf("%d%%", c.RelativeHumidity2m),
		Icon:     weatherIconName(c.WeatherCode),
	}
}

// SVG symbols

var svgFillRE = regexp.MustCompile(` fill="[^"]*"`)

func collectIconNames(cats []views.DashCategory, includeWeather bool) []string {
	seen := make(map[string]struct{})
	for _, cat := range cats {
		for _, bm := range cat.Links {
			if bm.Icon != "" {
				seen[bm.Icon] = struct{}{}
			}
		}
	}
	if includeWeather {
		for _, icon := range []string{
			"weather-cloudy", "weather-fog", "weather-lightning", "weather-lightning-rainy",
			"weather-partly-cloudy", "weather-pouring", "weather-rainy", "weather-snowy",
			"weather-snowy-heavy", "weather-snowy-rainy", "weather-sunny",
		} {
			seen[icon] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildSVGSymbols(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if err := os.MkdirAll(common.IconCacheDir, 0755); err != nil {
		slog.Warn("create icons dir failed", "error", err)
		return ""
	}
	var sb strings.Builder
	for _, name := range names {
		svg, err := fetchSVG(name)
		if err != nil {
			slog.Warn("fetch SVG failed", "icon", name, "error", err)
			continue
		}
		if symbol := svgToSymbol(name, svg); symbol != "" {
			sb.WriteString(symbol)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func fetchSVG(name string) (string, error) {
	name = filepath.Base(name) // prevent path traversal from user-entered icon names
	cachePath := filepath.Join(common.IconCacheDir, name+".svg")
	if data, err := os.ReadFile(cachePath); err == nil {
		return string(data), nil
	}
	u := common.MDISVGBaseURL + name + ".svg"
	resp, err := common.HTTPClient().Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cachePath, body, 0644); err != nil {
		slog.Warn("SVG cache write failed", "path", cachePath, "error", err)
	}
	return string(body), nil
}

func svgToSymbol(name, svg string) string {
	vbStart := strings.Index(svg, `viewBox="`)
	if vbStart < 0 {
		return ""
	}
	vbStart += 9
	vbEnd := strings.Index(svg[vbStart:], `"`)
	if vbEnd < 0 {
		return ""
	}
	viewBox := svg[vbStart : vbStart+vbEnd]

	innerStart := strings.Index(svg, ">")
	if innerStart < 0 {
		return ""
	}
	innerEnd := strings.LastIndex(svg, "</svg>")
	if innerEnd < 0 {
		return ""
	}
	inner := strings.TrimSpace(svg[innerStart+1 : innerEnd])
	inner = svgFillRE.ReplaceAllString(inner, "")
	return fmt.Sprintf(`<symbol id="mdi-%s" viewBox="%s">%s</symbol>`, name, viewBox, inner)
}
