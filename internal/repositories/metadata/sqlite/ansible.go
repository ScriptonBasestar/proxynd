package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"proxynd/internal/ports"
)

// AnsibleRepository implements AnsibleMetadataRepository using SQLite
type AnsibleRepository struct {
	db     *sql.DB
	logger ports.Logger
}

// CreateNamespace creates a new Ansible namespace
func (r *AnsibleRepository) CreateNamespace(ctx context.Context, req *ports.CreateNamespaceRequest) (*ports.AnsibleNamespace, error) {
	query := `
		INSERT INTO ansible_namespaces (name, description, company, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now().Format("2006-01-02 15:04:05")
	var id int64

	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Description,
		req.Company,
		req.AvatarURL,
		now,
		now,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", now)

	return &ports.AnsibleNamespace{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Company:     req.Company,
		AvatarURL:   req.AvatarURL,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

// GetNamespace retrieves a namespace by name
func (r *AnsibleRepository) GetNamespace(ctx context.Context, name string) (*ports.AnsibleNamespace, error) {
	query := `
		SELECT id, name, description, company, avatar_url, created_at, updated_at
		FROM ansible_namespaces
		WHERE name = ?
	`

	var ns ports.AnsibleNamespace
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&ns.ID,
		&ns.Name,
		&ns.Description,
		&ns.Company,
		&ns.AvatarURL,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("namespace not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	ns.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	ns.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &ns, nil
}

// ListNamespaces lists all namespaces
func (r *AnsibleRepository) ListNamespaces(ctx context.Context, req *ports.ListNamespacesRequest) (*ports.ListNamespacesResponse, error) {
	// Build query
	baseQuery := `
		SELECT id, name, description, company, avatar_url, created_at, updated_at
		FROM ansible_namespaces
	`

	// Count total
	countQuery := "SELECT COUNT(*) FROM ansible_namespaces"
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count namespaces: %w", err)
	}

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	query := baseQuery + " ORDER BY name LIMIT ? OFFSET ?"

	rows, err := r.db.QueryContext(ctx, query, req.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	defer rows.Close()

	var namespaces []*ports.AnsibleNamespace
	for rows.Next() {
		var ns ports.AnsibleNamespace
		var createdAt, updatedAt string

		if err := rows.Scan(
			&ns.ID,
			&ns.Name,
			&ns.Description,
			&ns.Company,
			&ns.AvatarURL,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan namespace: %w", err)
		}

		ns.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		ns.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		namespaces = append(namespaces, &ns)
	}

	return &ports.ListNamespacesResponse{
		Namespaces: namespaces,
		Total:      total,
		Page:       req.Page,
		TotalPages: (total + req.PageSize - 1) / req.PageSize,
	}, nil
}

// CreateCollection creates a new Ansible collection
func (r *AnsibleRepository) CreateCollection(ctx context.Context, req *ports.CreateCollectionRequest) (*ports.AnsibleCollection, error) {
	// First, get or create namespace
	namespace, err := r.GetNamespace(ctx, req.Namespace)
	if err != nil {
		// Namespace doesn't exist, create it
		namespace, err = r.CreateNamespace(ctx, &ports.CreateNamespaceRequest{
			Name: req.Namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	query := `
		INSERT INTO ansible_collections (namespace_id, name, description, deprecated, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now().Format("2006-01-02 15:04:05")
	var id int64

	err = r.db.QueryRowContext(ctx, query,
		namespace.ID,
		req.Name,
		req.Description,
		false,
		now,
		now,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", now)

	return &ports.AnsibleCollection{
		ID:          id,
		Namespace:   req.Namespace,
		Name:        req.Name,
		Description: req.Description,
		Deprecated:  false,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

// GetCollection retrieves a collection
func (r *AnsibleRepository) GetCollection(ctx context.Context, namespace, name string) (*ports.AnsibleCollection, error) {
	query := `
		SELECT c.id, n.name as namespace, c.name, c.description, c.deprecated, c.created_at, c.updated_at
		FROM ansible_collections c
		JOIN ansible_namespaces n ON c.namespace_id = n.id
		WHERE n.name = ? AND c.name = ?
	`

	var coll ports.AnsibleCollection
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, namespace, name).Scan(
		&coll.ID,
		&coll.Namespace,
		&coll.Name,
		&coll.Description,
		&coll.Deprecated,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection not found: %s.%s", namespace, name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	coll.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	coll.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &coll, nil
}

// ListCollections lists collections
func (r *AnsibleRepository) ListCollections(ctx context.Context, req *ports.ListCollectionsRequest) (*ports.ListCollectionsResponse, error) {
	baseQuery := `
		SELECT c.id, n.name as namespace, c.name, c.description, c.deprecated, c.created_at, c.updated_at
		FROM ansible_collections c
		JOIN ansible_namespaces n ON c.namespace_id = n.id
	`

	// Build where clause
	whereClause := ""
	args := []interface{}{}

	if req.Namespace != "" {
		whereClause = " WHERE n.name = ?"
		args = append(args, req.Namespace)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM ansible_collections c JOIN ansible_namespaces n ON c.namespace_id = n.id" + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count collections: %w", err)
	}

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	query := baseQuery + whereClause + " ORDER BY c.name LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	var collections []*ports.AnsibleCollection
	for rows.Next() {
		var coll ports.AnsibleCollection
		var createdAt, updatedAt string

		if err := rows.Scan(
			&coll.ID,
			&coll.Namespace,
			&coll.Name,
			&coll.Description,
			&coll.Deprecated,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}

		coll.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		coll.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		collections = append(collections, &coll)
	}

	return &ports.ListCollectionsResponse{
		Collections: collections,
		Total:       total,
		Page:        req.Page,
		TotalPages:  (total + req.PageSize - 1) / req.PageSize,
	}, nil
}

// CreateVersion creates a new collection version
func (r *AnsibleRepository) CreateVersion(ctx context.Context, req *ports.CreateVersionRequest) (*ports.AnsibleCollectionVersion, error) {
	// First, get or create collection
	collection, err := r.GetCollection(ctx, req.Namespace, req.Name)
	if err != nil {
		collection, err = r.CreateCollection(ctx, &ports.CreateCollectionRequest{
			Namespace:   req.Namespace,
			Name:        req.Name,
			Description: "",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Marshal JSON fields
	tagsJSON, _ := json.Marshal(req.Tags)
	authorsJSON, _ := json.Marshal(req.Authors)
	depsJSON, _ := json.Marshal(req.Dependencies)

	query := `
		INSERT INTO ansible_collection_versions (
			collection_id, version, artifact_sha256,
			license, tags, authors, dependencies,
			repository_url, documentation_url, homepage_url, issues_url,
			requires_ansible, uploaded_by, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	now := time.Now().Format("2006-01-02 15:04:05")
	var id int64

	err = r.db.QueryRowContext(ctx, query,
		collection.ID,
		req.Version,
		req.ArtifactSHA256,
		req.License,
		string(tagsJSON),
		string(authorsJSON),
		string(depsJSON),
		req.RepositoryURL,
		req.DocumentationURL,
		req.HomepageURL,
		req.IssuesURL,
		req.RequiresAnsible,
		req.UploadedBy,
		now,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	createdAt, _ := time.Parse("2006-01-02 15:04:05", now)

	return &ports.AnsibleCollectionVersion{
		ID:               id,
		Namespace:        req.Namespace,
		Name:             req.Name,
		Version:          req.Version,
		ArtifactSHA256:   req.ArtifactSHA256,
		License:          req.License,
		Tags:             req.Tags,
		Authors:          req.Authors,
		Dependencies:     req.Dependencies,
		RepositoryURL:    req.RepositoryURL,
		DocumentationURL: req.DocumentationURL,
		HomepageURL:      req.HomepageURL,
		IssuesURL:        req.IssuesURL,
		RequiresAnsible:  req.RequiresAnsible,
		UploadedBy:       req.UploadedBy,
		CreatedAt:        createdAt,
	}, nil
}

// GetVersion retrieves a specific collection version
func (r *AnsibleRepository) GetVersion(ctx context.Context, namespace, name, version string) (*ports.AnsibleCollectionVersion, error) {
	query := `
		SELECT
			cv.id, n.name as namespace, c.name, cv.version, cv.artifact_sha256,
			cv.license, cv.tags, cv.authors, cv.dependencies,
			cv.repository_url, cv.documentation_url, cv.homepage_url, cv.issues_url,
			cv.requires_ansible, cv.uploaded_by, cv.created_at
		FROM ansible_collection_versions cv
		JOIN ansible_collections c ON cv.collection_id = c.id
		JOIN ansible_namespaces n ON c.namespace_id = n.id
		WHERE n.name = ? AND c.name = ? AND cv.version = ?
	`

	var ver ports.AnsibleCollectionVersion
	var createdAt string
	var tagsJSON, authorsJSON, depsJSON string

	err := r.db.QueryRowContext(ctx, query, namespace, name, version).Scan(
		&ver.ID,
		&ver.Namespace,
		&ver.Name,
		&ver.Version,
		&ver.ArtifactSHA256,
		&ver.License,
		&tagsJSON,
		&authorsJSON,
		&depsJSON,
		&ver.RepositoryURL,
		&ver.DocumentationURL,
		&ver.HomepageURL,
		&ver.IssuesURL,
		&ver.RequiresAnsible,
		&ver.UploadedBy,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("version not found: %s.%s-%s", namespace, name, version)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(tagsJSON), &ver.Tags)
	json.Unmarshal([]byte(authorsJSON), &ver.Authors)
	json.Unmarshal([]byte(depsJSON), &ver.Dependencies)

	ver.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	return &ver, nil
}

// ListVersions lists all versions of a collection
func (r *AnsibleRepository) ListVersions(ctx context.Context, req *ports.ListVersionsRequest) (*ports.ListVersionsResponse, error) {
	query := `
		SELECT
			cv.id, n.name as namespace, c.name, cv.version, cv.artifact_sha256,
			cv.license, cv.tags, cv.authors, cv.dependencies,
			cv.repository_url, cv.documentation_url, cv.homepage_url, cv.issues_url,
			cv.requires_ansible, cv.uploaded_by, cv.created_at
		FROM ansible_collection_versions cv
		JOIN ansible_collections c ON cv.collection_id = c.id
		JOIN ansible_namespaces n ON c.namespace_id = n.id
		WHERE n.name = ? AND c.name = ?
		ORDER BY cv.created_at DESC
		LIMIT ? OFFSET ?
	`

	offset := (req.Page - 1) * req.PageSize

	rows, err := r.db.QueryContext(ctx, query, req.Namespace, req.Name, req.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*ports.AnsibleCollectionVersion
	for rows.Next() {
		var ver ports.AnsibleCollectionVersion
		var createdAt string
		var tagsJSON, authorsJSON, depsJSON string

		if err := rows.Scan(
			&ver.ID,
			&ver.Namespace,
			&ver.Name,
			&ver.Version,
			&ver.ArtifactSHA256,
			&ver.License,
			&tagsJSON,
			&authorsJSON,
			&depsJSON,
			&ver.RepositoryURL,
			&ver.DocumentationURL,
			&ver.HomepageURL,
			&ver.IssuesURL,
			&ver.RequiresAnsible,
			&ver.UploadedBy,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		json.Unmarshal([]byte(tagsJSON), &ver.Tags)
		json.Unmarshal([]byte(authorsJSON), &ver.Authors)
		json.Unmarshal([]byte(depsJSON), &ver.Dependencies)

		ver.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		versions = append(versions, &ver)
	}

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM ansible_collection_versions cv
		JOIN ansible_collections c ON cv.collection_id = c.id
		JOIN ansible_namespaces n ON c.namespace_id = n.id
		WHERE n.name = ? AND c.name = ?
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, req.Namespace, req.Name).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count versions: %w", err)
	}

	return &ports.ListVersionsResponse{
		Versions:   versions,
		Total:      total,
		Page:       req.Page,
		TotalPages: (total + req.PageSize - 1) / req.PageSize,
	}, nil
}

// DeleteVersion deletes a collection version
func (r *AnsibleRepository) DeleteVersion(ctx context.Context, namespace, name, version string) error {
	query := `
		DELETE FROM ansible_collection_versions
		WHERE id IN (
			SELECT cv.id
			FROM ansible_collection_versions cv
			JOIN ansible_collections c ON cv.collection_id = c.id
			JOIN ansible_namespaces n ON c.namespace_id = n.id
			WHERE n.name = ? AND c.name = ? AND cv.version = ?
		)
	`

	result, err := r.db.ExecContext(ctx, query, namespace, name, version)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("version not found: %s.%s-%s", namespace, name, version)
	}

	return nil
}
