package api

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
)

func (s *Server) digestHandler(c *echo.Context) error {
	digest, err := s.digestService.GetLatestDigest()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "digest_failed",
			Message: err.Error(),
		})
	}
	return c.JSON(http.StatusOK, digest)
}
