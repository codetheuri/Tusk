package query

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Apply applies whitelisted filtering, searching, and sorting to a GORM query session.
func Apply(db *gorm.DB, q Query, cfg Config) *gorm.DB {
	queryDB := db

	// 1. Apply Whitelisted Filters
	if len(q.Filters) > 0 && len(cfg.AllowedFilters) > 0 {
		for key, val := range q.Filters {
			// Handle range suffix conventions: _from, _to, _min, _max, _start, _end
			cleanKey := key
			op := "="
			if strings.HasSuffix(key, "_from") || strings.HasSuffix(key, "_start") || strings.HasSuffix(key, "_min") {
				cleanKey = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(key, "_from"), "_start"), "_min")
				op = ">="
			} else if strings.HasSuffix(key, "_to") || strings.HasSuffix(key, "_end") || strings.HasSuffix(key, "_max") {
				cleanKey = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(key, "_to"), "_end"), "_max")
				op = "<="
			}

			dbColumn, allowed := cfg.AllowedFilters[cleanKey]
			if !allowed {
				// Check direct match
				dbColumn, allowed = cfg.AllowedFilters[key]
				if !allowed {
					continue
				}
				op = "="
			}

			// Parameterized query execution prevents SQL injection
			queryDB = queryDB.Where(fmt.Sprintf("%s %s ?", dbColumn, op), val)
		}
	}

	// 2. Apply Whitelisted Multi-Column Search
	if q.Search != "" && len(cfg.AllowedSearches) > 0 {
		var searchConditions []string
		var searchArgs []interface{}
		pattern := "%" + q.Search + "%"

		for _, col := range cfg.AllowedSearches {
			searchConditions = append(searchConditions, fmt.Sprintf("%s LIKE ?", col))
			searchArgs = append(searchArgs, pattern)
		}

		searchClause := strings.Join(searchConditions, " OR ")
		queryDB = queryDB.Where("("+searchClause+")", searchArgs...)
	}

	// 3. Apply Whitelisted Sorting
	var appliedSorts []string

	if len(q.Sorts) > 0 && len(cfg.AllowedSorts) > 0 {
		for _, s := range q.Sorts {
			if dbColumn, allowed := cfg.AllowedSorts[s.Field]; allowed {
				appliedSorts = append(appliedSorts, fmt.Sprintf("%s %s", dbColumn, s.Order))
			}
		}
	}

	// Fallback to DefaultSort if no valid sort fields were supplied
	if len(appliedSorts) == 0 && cfg.DefaultSort != "" {
		defaultField := cfg.DefaultSort
		order := SortAsc
		if strings.HasPrefix(defaultField, "-") {
			order = SortDesc
			defaultField = strings.TrimPrefix(defaultField, "-")
		} else if strings.HasPrefix(defaultField, "+") {
			defaultField = strings.TrimPrefix(defaultField, "+")
		}

		if dbColumn, allowed := cfg.AllowedSorts[defaultField]; allowed {
			appliedSorts = append(appliedSorts, fmt.Sprintf("%s %s", dbColumn, order))
		} else {
			appliedSorts = append(appliedSorts, fmt.Sprintf("%s %s", defaultField, order))
		}
	}

	for _, sortExpr := range appliedSorts {
		queryDB = queryDB.Order(sortExpr)
	}

	return queryDB
}

// Paginate executes a paginated query on a GORM DB session using Go generics.
// It applies filters, search, sorting, counts total matching records, and fetches the page slice.
func Paginate[T any](ctx context.Context, db *gorm.DB, q Query, cfg Config) ([]T, Meta, error) {
	page := cfg.SanitizePage(q.Page)
	perPage := cfg.SanitizePerPage(q.PerPage)

	// Apply filtering & searching (without sorting/pagination) for total count calculation
	filteredDB := Apply(db.WithContext(ctx), q, cfg)

	var total int64
	var model T
	if err := filteredDB.Model(&model).Count(&total).Error; err != nil {
		return nil, Meta{}, fmt.Errorf("failed to count records: %w", err)
	}

	// Calculate offset and fetch records
	offset := (page - 1) * perPage
	var items []T

	if err := filteredDB.Limit(perPage).Offset(offset).Find(&items).Error; err != nil {
		return nil, Meta{}, fmt.Errorf("failed to fetch paginated records: %w", err)
	}

	meta := NewMeta(page, perPage, total)
	return items, meta, nil
}
