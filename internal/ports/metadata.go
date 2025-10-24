package ports

import (
	"context"
	"time"
)

// MetadataRepository provides database operations for package metadata
type MetadataRepository interface {
	// Ansible Collections
	AnsibleMetadata() AnsibleMetadataRepository

	// Python Packages
	PythonMetadata() PythonMetadataRepository

	// Common artifact operations
	GetArtifact(ctx context.Context, sha256 string) (*ArtifactMetadata, error)
	StoreArtifact(ctx context.Context, artifact *ArtifactMetadata) error
	DeleteArtifact(ctx context.Context, sha256 string) error
	IncrementRefCount(ctx context.Context, sha256 string) error
	DecrementRefCount(ctx context.Context, sha256 string) error
}

// ArtifactMetadata represents artifact metadata
type ArtifactMetadata struct {
	SHA256      string    `json:"sha256" db:"sha256"`
	Size        int64     `json:"size" db:"size"`
	ContentType string    `json:"content_type" db:"content_type"`
	RefCount    int       `json:"ref_count" db:"ref_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	AccessedAt  time.Time `json:"accessed_at" db:"accessed_at"`
}

// ============================================================================
// Ansible Metadata Repository
// ============================================================================

// AnsibleMetadataRepository manages Ansible Collections metadata
type AnsibleMetadataRepository interface {
	// Namespace operations
	GetNamespace(ctx context.Context, name string) (*AnsibleNamespace, error)
	CreateNamespace(ctx context.Context, req *CreateNamespaceRequest) (*AnsibleNamespace, error)
	ListNamespaces(ctx context.Context, req *ListNamespacesRequest) (*ListNamespacesResponse, error)

	// Collection operations
	GetCollection(ctx context.Context, namespace, name string) (*AnsibleCollection, error)
	CreateCollection(ctx context.Context, req *CreateCollectionRequest) (*AnsibleCollection, error)
	ListCollections(ctx context.Context, req *ListCollectionsRequest) (*ListCollectionsResponse, error)

	// Collection Version operations
	GetVersion(ctx context.Context, namespace, name, version string) (*AnsibleCollectionVersion, error)
	CreateVersion(ctx context.Context, req *CreateVersionRequest) (*AnsibleCollectionVersion, error)
	DeleteVersion(ctx context.Context, namespace, name, version string) error
	ListVersions(ctx context.Context, req *ListVersionsRequest) (*ListVersionsResponse, error)
}

// AnsibleNamespace represents an Ansible namespace
type AnsibleNamespace struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Company     string    `json:"company" db:"company"`
	AvatarURL   string    `json:"avatar_url" db:"avatar_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AnsibleCollection represents an Ansible collection
type AnsibleCollection struct {
	ID          int64     `json:"id" db:"id"`
	Namespace   string    `json:"namespace" db:"namespace"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Deprecated  bool      `json:"deprecated" db:"deprecated"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AnsibleCollectionVersion represents a collection version
type AnsibleCollectionVersion struct {
	ID               int64             `json:"id" db:"id"`
	Namespace        string            `json:"namespace" db:"namespace"`
	Name             string            `json:"name" db:"name"`
	Version          string            `json:"version" db:"version"`
	ArtifactSHA256   string            `json:"artifact_sha256" db:"artifact_sha256"`
	License          string            `json:"license" db:"license"`
	Tags             []string          `json:"tags" db:"tags"`
	Authors          []string          `json:"authors" db:"authors"`
	Dependencies     map[string]string `json:"dependencies" db:"dependencies"`
	RepositoryURL    string            `json:"repository_url" db:"repository_url"`
	DocumentationURL string            `json:"documentation_url" db:"documentation_url"`
	HomepageURL      string            `json:"homepage_url" db:"homepage_url"`
	IssuesURL        string            `json:"issues_url" db:"issues_url"`
	RequiresAnsible  string            `json:"requires_ansible" db:"requires_ansible"`
	UploadedBy       string            `json:"uploaded_by" db:"uploaded_by"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
}

// Request/Response types
type CreateNamespaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Company     string `json:"company"`
	AvatarURL   string `json:"avatar_url"`
}

type ListNamespacesRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type ListNamespacesResponse struct {
	Namespaces []*AnsibleNamespace `json:"namespaces"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"total_pages"`
}

type CreateCollectionRequest struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListCollectionsRequest struct {
	Namespace string `json:"namespace"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

type ListCollectionsResponse struct {
	Collections []*AnsibleCollection `json:"collections"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	TotalPages  int                  `json:"total_pages"`
}

type CreateVersionRequest struct {
	Namespace        string            `json:"namespace"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	ArtifactSHA256   string            `json:"artifact_sha256"`
	License          string            `json:"license"`
	Tags             []string          `json:"tags"`
	Authors          []string          `json:"authors"`
	Dependencies     map[string]string `json:"dependencies"`
	RepositoryURL    string            `json:"repository_url"`
	DocumentationURL string            `json:"documentation_url"`
	HomepageURL      string            `json:"homepage_url"`
	IssuesURL        string            `json:"issues_url"`
	RequiresAnsible  string            `json:"requires_ansible"`
	UploadedBy       string            `json:"uploaded_by"`
}

type ListVersionsRequest struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

type ListVersionsResponse struct {
	Versions   []*AnsibleCollectionVersion `json:"versions"`
	Total      int                         `json:"total"`
	Page       int                         `json:"page"`
	TotalPages int                         `json:"total_pages"`
}

// ============================================================================
// Python Metadata Repository
// ============================================================================

// PythonMetadataRepository manages Python Packages metadata
type PythonMetadataRepository interface {
	// Package operations
	GetPackage(ctx context.Context, packageName string) (*PythonPackage, error)
	CreatePackage(ctx context.Context, req *CreatePythonPackageRequest) (*PythonPackage, error)
	ListPackages(ctx context.Context, req *ListPythonPackagesRequest) (*ListPythonPackagesResponse, error)

	// Version operations
	GetVersion(ctx context.Context, packageName, version string) (*PythonPackageVersion, error)
	CreateVersion(ctx context.Context, req *CreatePythonVersionRequest) (*PythonPackageVersion, error)
	DeleteVersion(ctx context.Context, packageName, version string) error
	ListVersions(ctx context.Context, req *ListPythonVersionsRequest) (*ListPythonVersionsResponse, error)
}

// PythonPackage represents a Python package
type PythonPackage struct {
	ID             int64     `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	NormalizedName string    `json:"normalized_name" db:"normalized_name"`
	Description    string    `json:"description" db:"description"`
	HomePage       string    `json:"home_page" db:"home_page"`
	Author         string    `json:"author" db:"author"`
	AuthorEmail    string    `json:"author_email" db:"author_email"`
	License        string    `json:"license" db:"license"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// PythonPackageVersion represents a package version
type PythonPackageVersion struct {
	ID             int64     `json:"id" db:"id"`
	PackageName    string    `json:"package_name" db:"package_name"`
	Version        string    `json:"version" db:"version"`
	ArtifactSHA256 string    `json:"artifact_sha256" db:"artifact_sha256"`
	Filename       string    `json:"filename" db:"filename"`
	PythonVersion  string    `json:"python_version" db:"python_version"`
	RequiresPython string    `json:"requires_python" db:"requires_python"`
	PackageType    string    `json:"package_type" db:"package_type"`
	Summary        string    `json:"summary" db:"summary"`
	Keywords       string    `json:"keywords" db:"keywords"`
	Classifiers    []string  `json:"classifiers" db:"classifiers"`
	Dependencies   []string  `json:"dependencies" db:"dependencies"`
	UploadedBy     string    `json:"uploaded_by" db:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Request/Response types
type CreatePythonPackageRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	HomePage    string `json:"home_page"`
	Author      string `json:"author"`
	AuthorEmail string `json:"author_email"`
	License     string `json:"license"`
}

type ListPythonPackagesRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type ListPythonPackagesResponse struct {
	Packages   []*PythonPackage `json:"packages"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	TotalPages int              `json:"total_pages"`
}

type CreatePythonVersionRequest struct {
	PackageName    string   `json:"package_name"`
	Version        string   `json:"version"`
	ArtifactSHA256 string   `json:"artifact_sha256"`
	Filename       string   `json:"filename"`
	PythonVersion  string   `json:"python_version"`
	RequiresPython string   `json:"requires_python"`
	PackageType    string   `json:"package_type"`
	Summary        string   `json:"summary"`
	Keywords       string   `json:"keywords"`
	Classifiers    []string `json:"classifiers"`
	Dependencies   []string `json:"dependencies"`
	UploadedBy     string   `json:"uploaded_by"`
}

type ListPythonVersionsRequest struct {
	PackageName string `json:"package_name"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

type ListPythonVersionsResponse struct {
	Versions   []*PythonPackageVersion `json:"versions"`
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	TotalPages int                     `json:"total_pages"`
}
