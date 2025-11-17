// Package enterprise provides domain models for enterprise features
package enterprise

import (
	"time"
)

// Role represents a user role in the RBAC system
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a system permission
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`    // e.g., "cache", "config", "audit"
	Action      string `json:"action"`      // e.g., "read", "write", "delete"
	Description string `json:"description"`
}

// UserRole represents a user-role assignment
type UserRole struct {
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	AssignedBy string    `json:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at"`
}

// RoleRequest represents a request to create or update a role
type RoleRequest struct {
	Name        string   `json:"name" validate:"required,min=3,max=50"`
	Description string   `json:"description" validate:"max=500"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
}

// Validate performs business validation on a Role
func (r *Role) Validate() error {
	if r.Name == "" {
		return ErrInvalidRoleName
	}
	if len(r.Permissions) == 0 {
		return ErrNoPermissions
	}
	return nil
}

// HasPermission checks if role has a specific permission
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// PermissionCatalog returns all available system permissions
func PermissionCatalog() []Permission {
	return []Permission{
		{ID: "cache:read", Name: "Cache Read", Resource: "cache", Action: "read", Description: "View cache statistics and contents"},
		{ID: "cache:write", Name: "Cache Write", Resource: "cache", Action: "write", Description: "Clear cache and modify cache settings"},
		{ID: "config:read", Name: "Config Read", Resource: "config", Action: "read", Description: "View system configuration"},
		{ID: "config:write", Name: "Config Write", Resource: "config", Action: "write", Description: "Modify system configuration"},
		{ID: "audit:read", Name: "Audit Read", Resource: "audit", Action: "read", Description: "View audit logs"},
		{ID: "audit:export", Name: "Audit Export", Resource: "audit", Action: "export", Description: "Export audit logs"},
		{ID: "rbac:read", Name: "RBAC Read", Resource: "rbac", Action: "read", Description: "View roles and permissions"},
		{ID: "rbac:write", Name: "RBAC Write", Resource: "rbac", Action: "write", Description: "Manage roles and permissions"},
		{ID: "users:read", Name: "Users Read", Resource: "users", Action: "read", Description: "View user information"},
		{ID: "users:write", Name: "Users Write", Resource: "users", Action: "write", Description: "Manage users"},
		{ID: "analytics:read", Name: "Analytics Read", Resource: "analytics", Action: "read", Description: "View analytics and reports"},
		{ID: "security:read", Name: "Security Read", Resource: "security", Action: "read", Description: "View security scan results"},
		{ID: "security:scan", Name: "Security Scan", Resource: "security", Action: "scan", Description: "Trigger security scans"},
		{ID: "admin:*", Name: "Admin All", Resource: "admin", Action: "*", Description: "Full administrative access"},
	}
}

// Domain errors
var (
	ErrInvalidRoleName = &DomainError{Code: "INVALID_ROLE_NAME", Message: "Role name is required and must be valid"}
	ErrNoPermissions   = &DomainError{Code: "NO_PERMISSIONS", Message: "Role must have at least one permission"}
	ErrRoleNotFound    = &DomainError{Code: "ROLE_NOT_FOUND", Message: "Role not found"}
	ErrRoleExists      = &DomainError{Code: "ROLE_EXISTS", Message: "Role with this name already exists"}
)
