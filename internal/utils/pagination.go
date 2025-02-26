package utils

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page     int   `json:"page"`
	Limit    int   `json:"limit"`
	Offset   int   `json:"offset"`
	Total    int64 `json:"total"`
	LastPage int   `json:"last_page"`
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

func (p *Pagination) SetTotal(total int64) {
	p.Total = total
	p.LastPage = int((total + int64(p.Limit) - 1) / int64(p.Limit))
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func NewPaginatedResponse(data interface{}, pagination Pagination) PaginatedResponse {
	return PaginatedResponse{
		Data:       data,
		Pagination: pagination,
	}
}
