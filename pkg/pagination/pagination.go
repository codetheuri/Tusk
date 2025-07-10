package pagination

import (
	"math"

	"gorm.io/gorm"
)

type Pagination struct {
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalRows  int64           `json:"total_rows"`
	TotalPages int           `json:"total_pages"`
	Rows       interface{} `json:"data"`
}

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100
)

func Paginate( p *Pagination) func(db *gorm.DB) *gorm.DB {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.Limit < 1 || p.Limit > MaxLimit {
		p.Limit = DefaultLimit
	}

	offset := (p.Page - 1) * p.Limit
	return func(tx *gorm.DB) *gorm.DB {
		var totalRows int64


		// tx.Model(model).Count(&totalRows)
		// tx.Count(&totalRows)
		tx.Session(&gorm.Session{}).Count(&totalRows)

		p.TotalRows = totalRows

		p.TotalPages = int(math.Ceil(float64(totalRows) / float64(p.Limit)))

		// tx.Statement.Selects = nil

		return tx.Offset(offset).Limit(p.Limit)
	}
}
