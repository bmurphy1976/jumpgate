package handlers

import (
	"dashboard/config"
	"dashboard/icons"
	"dashboard/static"
	"dashboard/storage"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
)

// DSResolver returns the datasource for the current request.
type DSResolver func(*echo.Context) (storage.Datasource, error)

// StaticResolver returns a DSResolver that always returns the same datasource.
func StaticResolver(ds storage.Datasource) DSResolver {
	return func(c *echo.Context) (storage.Datasource, error) {
		return ds, nil
	}
}

// SessionResolver returns a DSResolver that looks up per-session datasources from the store.
func SessionResolver(store *storage.SessionStore) DSResolver {
	return func(c *echo.Context) (storage.Datasource, error) {
		sessionID, _ := (*c).Get("session_id").(string)
		clientIP, _ := (*c).Get("session_ip").(string)
		return store.GetOrCreate(sessionID, clientIP)
	}
}

func newDemoIPExtractor(cfg config.DemoConfig) (echo.IPExtractor, error) {
	if err := cfg.ValidateProxyHeaders(); err != nil {
		return nil, err
	}
	if cfg.DisableProxyHeaders != nil && *cfg.DisableProxyHeaders {
		return echo.ExtractIPDirect(), nil
	}

	networks, err := cfg.AllowedProxyNetworks()
	if err != nil {
		return nil, err
	}

	options := make([]echo.TrustOption, 0, len(networks))
	for _, network := range networks {
		options = append(options, echo.TrustIPRange(network))
	}

	return echo.ExtractIPFromXFFHeader(options...), nil
}

func NewServer(cfg config.ServerConfig, ds DSResolver, il *icons.Loader, store *storage.SessionStore) (*echo.Echo, error) {
	e := echo.New()
	e.Use(echomiddleware.RequestLogger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.Gzip())

	if cfg.Demo.Enabled {
		ipExtractor, err := newDemoIPExtractor(cfg.Demo)
		if err != nil {
			return nil, err
		}
		e.IPExtractor = ipExtractor
		e.Use(sessionMiddleware())
	}

	if cfg.Slow > 0 {
		e.Use(slowMiddleware(time.Duration(cfg.Slow) * time.Second))
	}

	SetupAdminRoutes(e, ds, il, !cfg.AuthEnabled(), cfg.Demo.Enabled)
	SetupDashboardRoutes(e, ds, cfg.Demo.Enabled)

	if cfg.API.Tokens.HasTokens() || cfg.API.Swagger {
		SetupAPIRoutes(e, ds, il, cfg.API)
	}

	if cfg.MCP.Enabled {
		SetupMCPRoutes(e, ds, il, cfg.API.Tokens)
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := (*c).Request().URL.Path
			switch {
			case strings.HasPrefix(path, "/static/") ||
				strings.HasPrefix(path, "/css/") ||
				strings.HasPrefix(path, "/js/") ||
				strings.HasPrefix(path, "/themes/"):
				(*c).Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			case path == "/favicon.ico":
				(*c).Response().Header().Set("Cache-Control", "public, max-age=604800")
			}
			return next(c)
		}
	})

	e.Use(echomiddleware.StaticWithConfig(echomiddleware.StaticConfig{
		Root:       ".",
		Filesystem: static.FS,
	}))
	e.StaticFS("/static", static.FS)

	return e, nil
}

func slowMiddleware(d time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := (*c).Request().URL.Path
			if path != "/" && path != "/admin" &&
				!strings.HasPrefix(path, "/static/") &&
				!strings.HasPrefix(path, "/css/") &&
				!strings.HasPrefix(path, "/js/") &&
				!strings.HasPrefix(path, "/themes/") &&
				path != "/favicon.ico" {
				time.Sleep(d)
			}
			return next(c)
		}
	}
}
