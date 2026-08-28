// Package pagination menyediakan DTO query paginasi yang dipakai bersama oleh
// semua modul, lengkap dengan whitelist kolom sorting.
package pagination

import (
	"math"

	"verifield-be/internal/common/response"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

// Query adalah query string standar untuk endpoint list.
type Query struct {
	Page    int    `form:"page"     binding:"omitempty,min=1"           example:"1"`
	Limit   int    `form:"limit"    binding:"omitempty,min=1,max=100"   example:"10"`
	Search  string `form:"search"   binding:"omitempty,max=100"         example:"batik"`
	SortBy  string `form:"sort_by"  binding:"omitempty,max=50"          example:"created_at"`
	SortDir string `form:"sort_dir" binding:"omitempty,oneof=asc desc"  example:"desc"`
}

// Normalize mengisi nilai default dan memastikan kolom sorting berasal dari
// whitelist — mencegah SQL injection lewat parameter sort_by.
func (q *Query) Normalize(allowedSort []string, defaultSort string) {
	if q.Page < 1 {
		q.Page = defaultPage
	}
	if q.Limit < 1 {
		q.Limit = defaultLimit
	}
	if q.Limit > maxLimit {
		q.Limit = maxLimit
	}
	if q.SortDir != "asc" && q.SortDir != "desc" {
		q.SortDir = "desc"
	}

	for _, allowed := range allowedSort {
		if q.SortBy == allowed {
			return
		}
	}
	q.SortBy = defaultSort
}

// Offset menghitung OFFSET untuk query database.
func (q Query) Offset() int { return (q.Page - 1) * q.Limit }

// OrderClause menghasilkan klausa ORDER BY. Aman dipakai langsung karena
// SortBy sudah difilter oleh Normalize.
func (q Query) OrderClause() string { return q.SortBy + " " + q.SortDir }

// BuildMeta menyusun meta paginasi dari query dan total baris.
func BuildMeta(q Query, total int64) response.Meta {
	totalPage := 0
	if q.Limit > 0 {
		totalPage = int(math.Ceil(float64(total) / float64(q.Limit)))
	}
	return response.Meta{
		Page:      q.Page,
		Limit:     q.Limit,
		Total:     total,
		TotalPage: totalPage,
	}
}
