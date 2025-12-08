package middleware

import (
	"net/http"
	"strconv"
)

// PaginationConfig holds configuration for pagination
type PaginationConfig struct {
	DefaultLimit int
	MaxLimit     int
}

// DefaultPaginationConfig returns default pagination configuration
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultLimit: 50,
		MaxLimit:     500,
	}
}

// Pagination holds parsed pagination parameters
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Page   int `json:"page"`
}

// ParsePagination extracts pagination parameters from the request
func ParsePagination(r *http.Request, config PaginationConfig) Pagination {
	p := Pagination{
		Limit:  config.DefaultLimit,
		Offset: 0,
		Page:   1,
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			p.Limit = limit
			if p.Limit > config.MaxLimit {
				p.Limit = config.MaxLimit
			}
		}
	}

	// Parse offset (takes precedence over page)
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			p.Offset = offset
			p.Page = (p.Offset / p.Limit) + 1
		}
	} else if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		// Parse page number
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			p.Page = page
			p.Offset = (page - 1) * p.Limit
		}
	}

	return p
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Total      int  `json:"total"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Page       int  `json:"page"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// NewPaginatedResponse creates a paginated response with metadata
func NewPaginatedResponse(data interface{}, total int, p Pagination) PaginatedResponse {
	totalPages := total / p.Limit
	if total%p.Limit > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Total:      total,
			Limit:      p.Limit,
			Offset:     p.Offset,
			Page:       p.Page,
			TotalPages: totalPages,
			HasNext:    p.Offset+p.Limit < total,
			HasPrev:    p.Offset > 0,
		},
	}
}

// ApplyPagination applies pagination to a slice of any type
// Returns the paginated slice and the total count
func ApplyPagination[T any](items []T, p Pagination) ([]T, int) {
	total := len(items)

	if p.Offset >= total {
		return []T{}, total
	}

	end := p.Offset + p.Limit
	if end > total {
		end = total
	}

	return items[p.Offset:end], total
}
