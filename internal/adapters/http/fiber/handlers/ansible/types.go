package ansible

import "time"

// Galaxy v3 API Response Types

// CollectionUploadResponse represents the response after uploading a collection
type CollectionUploadResponse struct {
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	SHA256      string    `json:"sha256"`
	UploadedAt  time.Time `json:"uploaded_at"`
	UploadedBy  string    `json:"uploaded_by"`
	PackagePath string    `json:"package_path"`
}

// CollectionVersionDetail represents detailed information about a collection version
type CollectionVersionDetail struct {
	Namespace       string            `json:"namespace"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Authors         []string          `json:"authors"`
	License         []string          `json:"license"`
	Tags            []string          `json:"tags"`
	Dependencies    map[string]string `json:"dependencies"`
	Repository      string            `json:"repository"`
	Documentation   string            `json:"documentation"`
	Homepage        string            `json:"homepage"`
	Issues          string            `json:"issues"`
	RequiresAnsible string            `json:"requires_ansible"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DownloadURL     string            `json:"download_url"`
	ArtifactSHA256  string            `json:"artifact_sha256"`
}

// CollectionListItem represents a collection in the list response
type CollectionListItem struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Deprecated  bool   `json:"deprecated"`
	VersionsURL string `json:"versions_url"`
}

// CollectionListResponse represents the paginated list of collections
type CollectionListResponse struct {
	Data  []CollectionListItem `json:"data"`
	Links *PaginationLinks     `json:"links"`
	Meta  *PaginationMeta      `json:"meta"`
}

// PaginationLinks contains pagination URLs
type PaginationLinks struct {
	First    *string `json:"first"`
	Previous *string `json:"previous"`
	Next     *string `json:"next"`
	Last     *string `json:"last"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Count int64 `json:"count"` // Total number of items
	Page  int   `json:"page"`  // Current page number
	Limit int   `json:"limit"` // Items per page
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Detail string                 `json:"detail"`
	Errors map[string]interface{} `json:"errors,omitempty"`
}
