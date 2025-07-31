package pagination

import (
	"math"
)

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100
)

type Params struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func NewPaginationParams(page, limit int) *Params {
	p := &Params{Page: page, Limit: limit}
	p.ApplyDefaults()
	return p
}
func (p *Params) ApplyDefaults() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.Limit < 1 || p.Limit > MaxLimit {
		p.Limit = DefaultLimit
	}
}
func (p *Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

type Metadata struct {
	Page       int   `json:"current_page"`
	Limit      int   `json:"per_page"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func NewPaginationmetadata(page, limit int, totalItems int64) *Metadata {
	metadata := &Metadata{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
	}
	metadata.CalculateTotalPages()
	return metadata
}
func (m *Metadata) CalculateTotalPages() {
	if m.Limit > 0 {
		m.TotalPages = int(math.Ceil(float64(m.TotalItems) / float64(m.Limit)))
	} else {
		m.TotalPages = 0
	}
}

type PaginationResponse struct {
	Data     interface{} `json:"data"`
	Metadata *Metadata   `json:"metadata"`
}
