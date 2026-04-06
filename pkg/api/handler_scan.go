package api

import (
	"log/slog"
	"net/http"

	echo "github.com/labstack/echo/v5"
)

func (s *Server) scanHandler(c *echo.Context) error {
	slog.Info("Scan triggered via API")

	scan, err := s.scanService.RunScan(c.Request().Context(), s.ghClient)
	if err != nil {
		slog.Error("Scan failed", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "scan_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, scan)
}
