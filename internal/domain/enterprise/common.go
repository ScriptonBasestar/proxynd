package enterprise

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// DomainError represents a domain-level error
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// NewPagination creates pagination metadata
func NewPagination(page, perPage int, total int64) *Pagination {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// generateID generates a random hex ID
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Common validation errors
var (
	ErrInvalidPagination = &DomainError{Code: "INVALID_PAGINATION", Message: "Invalid pagination parameters"}
	ErrUnauthorized      = &DomainError{Code: "UNAUTHORIZED", Message: "Unauthorized access"}
	ErrForbidden         = &DomainError{Code: "FORBIDDEN", Message: "Insufficient permissions"}
)
