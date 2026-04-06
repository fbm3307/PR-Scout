package api

import (
	"net/http"
	"time"

	echo "github.com/labstack/echo/v5"
)

func (s *Server) healthHandler(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":    "healthy",
		"service":   "pr-scout",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
