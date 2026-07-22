package query

import "math"

// Meta holds standardized pagination metadata for API responses.
type Meta struct {
	Page        int   `json:"page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

// NewMeta calculates pagination metadata given current page, per_page, and total record count.
func NewMeta(page, perPage int, total int64) Meta {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	return Meta{
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

// PaginatedResult wraps a slice of data items along with pagination metadata.
type PaginatedResult[T any] struct {
	Items []T  `json:"items"`
	Meta  Meta `json:"meta"`
}
