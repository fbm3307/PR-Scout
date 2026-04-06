package api

import (
	"net/http"
	"strconv"

	echo "github.com/labstack/echo/v5"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

func (s *Server) listPRsHandler(c *echo.Context) error {
	filter := models.PRListFilter{
		Repo:             c.QueryParam("repo"),
		State:            c.QueryParam("state"),
		Author:           c.QueryParam("author"),
		MyReviewStatus:   c.QueryParam("my_review_status"),
		CIStatus:         c.QueryParam("ci_status"),
		CodeRabbitStatus: c.QueryParam("coderabbit_status"),
	}

	if v := c.QueryParam("is_new"); v != "" {
		isNew := v == "true" || v == "1"
		filter.IsNew = &isNew
	}
	if v := c.QueryParam("page"); v != "" {
		filter.Page, _ = strconv.Atoi(v)
	}
	if v := c.QueryParam("per_page"); v != "" {
		filter.PerPage, _ = strconv.Atoi(v)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 25
	}

	prs, total, err := s.prService.ListPRs(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, ListResponse[any]{
		Items:   toAnySlice(prs),
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	})
}

func (s *Server) getPRHandler(c *echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_id"})
	}

	pr, comments, err := s.prService.GetPR(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "get_failed",
			Message: err.Error(),
		})
	}
	if pr == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"pr":       pr,
		"comments": comments,
	})
}

func (s *Server) myReviewsHandler(c *echo.Context) error {
	filter := models.PRListFilter{
		MyReviewStatus: "needs_attention",
		Page:           1,
		PerPage:        50,
	}

	prs, total, err := s.prService.ListPRs(filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, ListResponse[any]{
		Items:   toAnySlice(prs),
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	})
}

func toAnySlice[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
