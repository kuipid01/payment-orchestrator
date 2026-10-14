package apihttp

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Pagination bounds for list endpoints.
const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// Page is the pagination block returned alongside list responses.
type Page struct {
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Total      *int64 `json:"total,omitempty"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	// Summary is an optional facet tally for the same filter the page was
	// computed from — for payments, one entry per status. A client can label its
	// own filters ("failed: 4") without a second, differently-filtered request.
	Summary map[string]int64 `json:"summary,omitempty"`
}

// Pagination is the parsed request-side pagination.
type Pagination struct {
	Page   int
	Limit  int
	Cursor string
}

// ParsePagination reads page/limit/cursor with sane clamps so a client cannot
// request an unbounded result set.
func ParsePagination(c *fiber.Ctx) Pagination {
	p := Pagination{
		Page:   atoiOr(c.Query("page"), 1),
		Limit:  atoiOr(c.Query("limit"), DefaultLimit),
		Cursor: c.Query("cursor"),
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	return p
}

// Offset converts page/limit into an SQL offset.
func (p Pagination) Offset() int { return (p.Page - 1) * p.Limit }

func atoiOr(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
