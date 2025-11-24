package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"proxynd/internal/ports"
)

//go:embed schema.sql
var schemaSQL string

// Repository implements MetadataRepository using SQLite
type Repository struct {
	db     *sql.DB
	logger ports.Logger

	ansible *AnsibleRepository
	python  *PythonRepository
}

// NewRepository creates a new SQLite metadata repository
func NewRepository(dbPath string, logger ports.Logger) (*Repository, error) {
	// Open database
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	repo := &Repository{
		db:     db,
		logger: logger,
	}

	// Initialize sub-repositories
	repo.ansible = &AnsibleRepository{db: db, logger: logger}
	repo.python = &PythonRepository{db: db, logger: logger}

	return repo, nil
}

// NewSQLiteMetadataRepository is a convenience wrapper for tests without logger
func NewSQLiteMetadataRepository(dbPath string) (*Repository, error) {
	// Use a no-op logger for testing
	logger := &noOpLogger{}
	return NewRepository(dbPath, logger)
}

// noOpLogger implements ports.Logger but does nothing (for testing)
type noOpLogger struct{}

func (n *noOpLogger) Debug(ctx context.Context, msg string, fields ...ports.Field) {}
func (n *noOpLogger) Info(ctx context.Context, msg string, fields ...ports.Field)  {}
func (n *noOpLogger) Warn(ctx context.Context, msg string, fields ...ports.Field)  {}
func (n *noOpLogger) Error(ctx context.Context, msg string, fields ...ports.Field) {}
func (n *noOpLogger) Fatal(ctx context.Context, msg string, fields ...ports.Field) {}
func (n *noOpLogger) With(fields ...ports.Field) ports.Logger                      { return n }
func (n *noOpLogger) WithContext(ctx context.Context) ports.Logger                 { return n }

// AnsibleMetadata returns Ansible-specific metadata repository
func (r *Repository) AnsibleMetadata() ports.AnsibleMetadataRepository {
	return r.ansible
}

// PythonMetadata returns Python-specific metadata repository
func (r *Repository) PythonMetadata() ports.PythonMetadataRepository {
	return r.python
}

// GetArtifact retrieves artifact metadata by SHA256
func (r *Repository) GetArtifact(ctx context.Context, sha256 string) (*ports.ArtifactMetadata, error) {
	query := `
		SELECT sha256, size, content_type, ref_count, created_at, accessed_at
		FROM artifacts
		WHERE sha256 = ?
	`

	var artifact ports.ArtifactMetadata
	var createdAt, accessedAt string

	err := r.db.QueryRowContext(ctx, query, sha256).Scan(
		&artifact.SHA256,
		&artifact.Size,
		&artifact.ContentType,
		&artifact.RefCount,
		&createdAt,
		&accessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("artifact not found: %s", sha256)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	// Parse timestamps
	artifact.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	artifact.AccessedAt, _ = time.Parse("2006-01-02 15:04:05", accessedAt)

	return &artifact, nil
}

// StoreArtifact stores artifact metadata
func (r *Repository) StoreArtifact(ctx context.Context, artifact *ports.ArtifactMetadata) error {
	query := `
		INSERT INTO artifacts (sha256, size, content_type, ref_count, created_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(sha256) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		artifact.SHA256,
		artifact.Size,
		artifact.ContentType,
		artifact.RefCount,
		artifact.CreatedAt.Format("2006-01-02 15:04:05"),
		artifact.AccessedAt.Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to store artifact: %w", err)
	}

	return nil
}

// IncrementRefCount increments the reference count for an artifact
func (r *Repository) IncrementRefCount(ctx context.Context, sha256 string) error {
	query := `
		UPDATE artifacts
		SET ref_count = ref_count + 1, accessed_at = ?
		WHERE sha256 = ?
	`

	result, err := r.db.ExecContext(ctx, query, time.Now().Format("2006-01-02 15:04:05"), sha256)
	if err != nil {
		return fmt.Errorf("failed to increment ref count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("artifact not found: %s", sha256)
	}

	return nil
}

// DecrementRefCount decrements the reference count for an artifact
func (r *Repository) DecrementRefCount(ctx context.Context, sha256 string) error {
	query := `
		UPDATE artifacts
		SET ref_count = CASE
			WHEN ref_count > 0 THEN ref_count - 1
			ELSE 0
		END
		WHERE sha256 = ?
	`

	result, err := r.db.ExecContext(ctx, query, sha256)
	if err != nil {
		return fmt.Errorf("failed to decrement ref count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("artifact not found: %s", sha256)
	}

	return nil
}

// DeleteArtifact deletes an artifact from the repository
func (r *Repository) DeleteArtifact(ctx context.Context, sha256 string) error {
	query := `DELETE FROM artifacts WHERE sha256 = ?`

	result, err := r.db.ExecContext(ctx, query, sha256)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("artifact not found: %s", sha256)
	}

	return nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}
