package query

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SortOrder defines ascending or descending order.
type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

// Sort represents a single field sorting directive.
type Sort struct {
	Field string
	Order SortOrder
}

// Query encapsulates normalized parameters for pagination, searching, sorting, and filtering.
type Query struct {
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
	Search  string            `json:"search"`
	Sorts   []Sort            `json:"sorts"`
	Filters map[string]string `json:"filters"`
}

// ParseRequest extracts query parameters from an HTTP request into a normalized Query struct.
func ParseRequest(r *http.Request) Query {
	if r == nil {
		return NewDefaultQuery()
	}
	return ParseURLValues(r.URL.Query())
}

// ParseURLValues parses url.Values into a normalized Query struct.
func ParseURLValues(v url.Values) Query {
	q := NewDefaultQuery()

	if pageStr := v.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			q.Page = p
		}
	}

	if perPageStr := v.Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			q.PerPage = pp
		}
	}

	if searchStr := strings.TrimSpace(v.Get("search")); searchStr != "" {
		q.Search = searchStr
	}

	// Parse sort parameters (e.g., sort=-created_at,name or sort=-created_at&sort=name)
	sortParams := v["sort"]
	if len(sortParams) > 0 {
		var sorts []Sort
		for _, param := range sortParams {
			fields := strings.Split(param, ",")
			for _, field := range fields {
				field = strings.TrimSpace(field)
				if field == "" {
					continue
				}

				order := SortAsc
				if strings.HasPrefix(field, "-") {
					order = SortDesc
					field = strings.TrimPrefix(field, "-")
				} else if strings.HasPrefix(field, "+") {
					field = strings.TrimPrefix(field, "+")
				}

				sorts = append(sorts, Sort{
					Field: field,
					Order: order,
				})
			}
		}
		q.Sorts = sorts
	}

	// Extract generic filters (excluding pagination, search, and sort keywords)
	reservedKeys := map[string]bool{
		"page":     true,
		"per_page": true,
		"search":   true,
		"sort":     true,
	}

	for key, values := range v {
		if reservedKeys[key] || len(values) == 0 {
			continue
		}
		val := strings.TrimSpace(values[0])
		if val != "" {
			q.Filters[key] = val
		}
	}

	return q
}

// NewDefaultQuery initializes a Query with safe default values.
func NewDefaultQuery() Query {
	return Query{
		Page:    1,
		PerPage: 20,
		Filters: make(map[string]string),
	}
}
