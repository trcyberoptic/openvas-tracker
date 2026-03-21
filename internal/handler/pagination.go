package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

const (
	defaultLimit = 500
	maxLimit     = 5000
)

func paginate(c echo.Context) (limit, offset int32) {
	limit = defaultLimit
	offset = 0

	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	return limit, offset
}
