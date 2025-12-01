package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// PythonRepository implements PythonMetadataRepository using SQLite
type PythonRepository struct {
	db     *sql.DB
	logger ports.Logger
}

// normalizeName normalizes Python package names according to PEP 503
// Converts to lowercase and replaces [-_.] with hyphens
func normalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// CreatePackage creates a new Python package
func (r *PythonRepository) CreatePackage(ctx context.Context, req *ports.CreatePythonPackageRequest) (*ports.PythonPackage, error) {
	normalizedName := normalizeName(req.Name)

	query := `
		INSERT INTO python_packages (
			name, normalized_name, description, home_page, author, author_email, license,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now().Format("2006-01-02 15:04:05")
	var id int64

	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		normalizedName,
		req.Description,
		req.HomePage,
		req.Author,
		req.AuthorEmail,
		req.License,
		now,
		now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", now)

	return &ports.PythonPackage{
		ID:             id,
		Name:           req.Name,
		NormalizedName: normalizedName,
		Description:    req.Description,
		HomePage:       req.HomePage,
		Author:         req.Author,
		AuthorEmail:    req.AuthorEmail,
		License:        req.License,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}, nil
}

// GetPackage retrieves a Python package by name (normalized lookup)
func (r *PythonRepository) GetPackage(ctx context.Context, name string) (*ports.PythonPackage, error) {
	normalizedName := normalizeName(name)

	query := `
		SELECT id, name, normalized_name, description, home_page, author, author_email, license, created_at, updated_at
		FROM python_packages
		WHERE normalized_name = ?
	`

	var pkg ports.PythonPackage
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, normalizedName).Scan(
		&pkg.ID,
		&pkg.Name,
		&pkg.NormalizedName,
		&pkg.Description,
		&pkg.HomePage,
		&pkg.Author,
		&pkg.AuthorEmail,
		&pkg.License,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("package not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	pkg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	pkg.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &pkg, nil
}

// ListPackages lists all Python packages
func (r *PythonRepository) ListPackages(ctx context.Context, req *ports.ListPythonPackagesRequest) (*ports.ListPythonPackagesResponse, error) {
	baseQuery := `
		SELECT id, name, normalized_name, description, home_page, author, author_email, license, created_at, updated_at
		FROM python_packages
	`

	// Count total
	countQuery := "SELECT COUNT(*) FROM python_packages"
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count packages: %w", err)
	}

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	query := baseQuery + " ORDER BY normalized_name LIMIT ? OFFSET ?"

	rows, err := r.db.QueryContext(ctx, query, req.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var packages []*ports.PythonPackage
	for rows.Next() {
		var pkg ports.PythonPackage
		var createdAt, updatedAt string

		if err := rows.Scan(
			&pkg.ID,
			&pkg.Name,
			&pkg.NormalizedName,
			&pkg.Description,
			&pkg.HomePage,
			&pkg.Author,
			&pkg.AuthorEmail,
			&pkg.License,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
		}

		pkg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		pkg.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		packages = append(packages, &pkg)
	}

	return &ports.ListPythonPackagesResponse{
		Packages:   packages,
		Total:      total,
		Page:       req.Page,
		TotalPages: (total + req.PageSize - 1) / req.PageSize,
	}, nil
}

// CreateVersion creates a new package version
func (r *PythonRepository) CreateVersion(ctx context.Context, req *ports.CreatePythonVersionRequest) (*ports.PythonPackageVersion, error) {
	// First, get or create package
	pkg, err := r.GetPackage(ctx, req.PackageName)
	if err != nil {
		pkg, err = r.CreatePackage(ctx, &ports.CreatePythonPackageRequest{
			Name: req.PackageName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create package: %w", err)
		}
	}

	// Marshal JSON fields
	classifiersJSON, _ := json.Marshal(req.Classifiers)
	depsJSON, _ := json.Marshal(req.Dependencies)

	query := `
		INSERT INTO python_package_versions (
			package_id, version, artifact_sha256,
			filename, python_version, requires_python, packagetype,
			summary, keywords, classifiers, dependencies,
			uploaded_by, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now().Format("2006-01-02 15:04:05")
	var id int64

	err = r.db.QueryRowContext(ctx, query,
		pkg.ID,
		req.Version,
		req.ArtifactSHA256,
		req.Filename,
		req.PythonVersion,
		req.RequiresPython,
		req.PackageType,
		req.Summary,
		req.Keywords,
		string(classifiersJSON),
		string(depsJSON),
		req.UploadedBy,
		now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", now)

	return &ports.PythonPackageVersion{
		ID:             id,
		PackageName:    req.PackageName,
		Version:        req.Version,
		ArtifactSHA256: req.ArtifactSHA256,
		Filename:       req.Filename,
		PythonVersion:  req.PythonVersion,
		RequiresPython: req.RequiresPython,
		PackageType:    req.PackageType,
		Summary:        req.Summary,
		Keywords:       req.Keywords,
		Classifiers:    req.Classifiers,
		Dependencies:   req.Dependencies,
		UploadedBy:     req.UploadedBy,
		CreatedAt:      createdAt,
	}, nil
}

// GetVersion retrieves a specific package version
func (r *PythonRepository) GetVersion(ctx context.Context, packageName, version string) (*ports.PythonPackageVersion, error) {
	normalizedName := normalizeName(packageName)

	query := `
		SELECT
			pv.id, p.name, pv.version, pv.artifact_sha256,
			pv.filename, pv.python_version, pv.requires_python, pv.packagetype,
			pv.summary, pv.keywords, pv.classifiers, pv.dependencies,
			pv.uploaded_by, pv.created_at
		FROM python_package_versions pv
		JOIN python_packages p ON pv.package_id = p.id
		WHERE p.normalized_name = ? AND pv.version = ?
	`

	var ver ports.PythonPackageVersion
	var createdAt string
	var classifiersJSON, depsJSON string

	err := r.db.QueryRowContext(ctx, query, normalizedName, version).Scan(
		&ver.ID,
		&ver.PackageName,
		&ver.Version,
		&ver.ArtifactSHA256,
		&ver.Filename,
		&ver.PythonVersion,
		&ver.RequiresPython,
		&ver.PackageType,
		&ver.Summary,
		&ver.Keywords,
		&classifiersJSON,
		&depsJSON,
		&ver.UploadedBy,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("version not found: %s==%s", packageName, version)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// Parse JSON fields (best-effort, ignore errors for malformed JSON)
	_ = json.Unmarshal([]byte(classifiersJSON), &ver.Classifiers)
	_ = json.Unmarshal([]byte(depsJSON), &ver.Dependencies)

	ver.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	return &ver, nil
}

// ListVersions lists all versions of a package
func (r *PythonRepository) ListVersions(ctx context.Context, req *ports.ListPythonVersionsRequest) (*ports.ListPythonVersionsResponse, error) {
	normalizedName := normalizeName(req.PackageName)

	query := `
		SELECT
			pv.id, p.name, pv.version, pv.artifact_sha256,
			pv.filename, pv.python_version, pv.requires_python, pv.packagetype,
			pv.summary, pv.keywords, pv.classifiers, pv.dependencies,
			pv.uploaded_by, pv.created_at
		FROM python_package_versions pv
		JOIN python_packages p ON pv.package_id = p.id
		WHERE p.normalized_name = ?
		ORDER BY pv.created_at DESC
		LIMIT ? OFFSET ?
	`

	offset := (req.Page - 1) * req.PageSize

	rows, err := r.db.QueryContext(ctx, query, normalizedName, req.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var versions []*ports.PythonPackageVersion
	for rows.Next() {
		var ver ports.PythonPackageVersion
		var createdAt string
		var classifiersJSON, depsJSON string

		if err := rows.Scan(
			&ver.ID,
			&ver.PackageName,
			&ver.Version,
			&ver.ArtifactSHA256,
			&ver.Filename,
			&ver.PythonVersion,
			&ver.RequiresPython,
			&ver.PackageType,
			&ver.Summary,
			&ver.Keywords,
			&classifiersJSON,
			&depsJSON,
			&ver.UploadedBy,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		_ = json.Unmarshal([]byte(classifiersJSON), &ver.Classifiers)
		_ = json.Unmarshal([]byte(depsJSON), &ver.Dependencies)

		ver.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		versions = append(versions, &ver)
	}

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM python_package_versions pv
		JOIN python_packages p ON pv.package_id = p.id
		WHERE p.normalized_name = ?
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, normalizedName).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count versions: %w", err)
	}

	return &ports.ListPythonVersionsResponse{
		Versions:   versions,
		Total:      total,
		Page:       req.Page,
		TotalPages: (total + req.PageSize - 1) / req.PageSize,
	}, nil
}

// DeleteVersion deletes a package version
func (r *PythonRepository) DeleteVersion(ctx context.Context, packageName, version string) error {
	normalizedName := normalizeName(packageName)

	query := `
		DELETE FROM python_package_versions
		WHERE id IN (
			SELECT pv.id
			FROM python_package_versions pv
			JOIN python_packages p ON pv.package_id = p.id
			WHERE p.normalized_name = ? AND pv.version = ?
		)
	`

	result, err := r.db.ExecContext(ctx, query, normalizedName, version)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("version not found: %s==%s", packageName, version)
	}

	return nil
}
