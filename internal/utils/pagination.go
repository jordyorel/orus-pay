package utils

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

// GetPagination extracts the page and limit from the query parameters.
// It returns a Pagination struct using defaults if parsing fails.
func GetPagination(c *fiber.Ctx, defaultPage, defaultLimit int) Pagination {
	pageStr := c.Query("page", strconv.Itoa(defaultPage))
	limitStr := c.Query("limit", strconv.Itoa(defaultLimit))

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = defaultPage
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = defaultLimit
	}

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// TotalPages calculates the number of pages based on the total items and items per page.
func TotalPages(totalItems int64, limit int) int {
	pages := int(totalItems) / limit
	if int(totalItems)%limit > 0 {
		pages++
	}
	return pages
}
