package handlers

import (
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

func NewServer(ds DSResolver, il *icons.Loader, noAuth bool, demoMode bool, store *storage.SessionStore, slow bool) *echo.Echo {
	e := echo.New()
	e.Use(echomiddleware.RequestLogger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.Gzip())

	if store != nil {
		e.Use(sessionMiddleware(store))
	}

	if slow {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				path := (*c).Request().URL.Path
				if path != "/" && path != "/admin" &&
					!strings.HasPrefix(path, "/static/") &&
					!strings.HasPrefix(path, "/css/") &&
					!strings.HasPrefix(path, "/js/") &&
					!strings.HasPrefix(path, "/themes/") &&
					path != "/favicon.ico" {
					time.Sleep(2 * time.Second)
				}
				return next(c)
			}
		})
	}

	SetupAdminRoutes(e, ds, il, noAuth, demoMode)
	SetupDashboardRoutes(e, ds, demoMode)

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

	return e
}
