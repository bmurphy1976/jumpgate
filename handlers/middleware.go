package handlers

import (
	"dashboard/common"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func requireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if !isAuthorized((*c).Request()) {
			return echo.NewHTTPError(401, "Unauthorized")
		}
		return next(c)
	}
}

func sessionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := (*c).Request().URL.Path
			if strings.HasPrefix(path, "/static/") || path == "/favicon.ico" {
				return next(c)
			}

			r := (*c).Request()
			sessionID := ""
			if cookie, err := r.Cookie("session_id"); err == nil {
				sessionID = cookie.Value
			}
			if sessionID == "" {
				sessionID = uuid.NewString()
				ttl, err := common.DemoSessionTTL()
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				(*c).SetCookie(&http.Cookie{
					Name:     "session_id",
					Value:    sessionID,
					Path:     "/",
					MaxAge:   int(ttl.Seconds()),
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			(*c).Set("session_id", sessionID)
			(*c).Set("session_ip", (*c).RealIP())

			return next(c)
		}
	}
}
