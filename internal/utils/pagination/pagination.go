package pagination

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Pagination struct {
	Page   int
	Limit  int
	Offset int
	Total  int64
}

// ParseFromRequest handles pagination parameters from Fiber context
func ParseFromRequest(c *fiber.Ctx) Pagination {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// Response creates a standardized pagination response
func Response(p Pagination, data interface{}) fiber.Map {
	totalPages := p.Total / int64(p.Limit)
	if p.Total%int64(p.Limit) > 0 {
		totalPages++
	}

	return fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"current_page": p.Page,
			"per_page":     p.Limit,
			"total_items":  p.Total,
			"total_pages":  totalPages,
		},
	}
}
