package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/config"
	"proxynd/internal/domain/ansible"
	"proxynd/internal/ports"
)

// HostedDriver implements HostedRepository interface for Ansible Collections
type HostedDriver struct {
	cas       ports.ContentAddressableStorage
	metadata  ports.MetadataRepository
	validator *ansible.TarballValidator
	config    *config.AnsibleRepositoryConfig
	logger    ports.Logger
}

// NewHostedDriver creates a new Ansible hosted driver
func NewHostedDriver(
	cas ports.ContentAddressableStorage,
	metadata ports.MetadataRepository,
	cfg *config.AnsibleRepositoryConfig,
	logger ports.Logger,
) (*HostedDriver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if !cfg.IsHosted() {
		return nil, fmt.Errorf("repository type must be 'hosted', got '%s'", cfg.Type)
	}

	return &HostedDriver{
		cas:       cas,
		metadata:  metadata,
		validator: ansible.NewTarballValidator(cfg.MaxUploadSize),
		config:    cfg,
		logger:    logger,
	}, nil
}

// StorePackage stores an Ansible collection package
func (d *HostedDriver) StorePackage(ctx context.Context, req *ports.StorePackageRequest) (*ports.StorePackageResponse, error) {
	d.logger.Info(ctx, "Storing Ansible collection",
		zap.NewField("repository", d.config.Name),
		zap.NewField("uploaded_by", req.UploadedBy),
	)

	// 1. Validate file size
	if err := d.validator.ValidateSize(req.Size); err != nil {
		d.logger.Warn(ctx, "File size validation failed",
			zap.NewField("size", req.Size),
			zap.NewField("error", err.Error()),
		)
		return nil, err
	}

	// 2. Read content into buffer
	content, err := io.ReadAll(req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// 3. Parse manifest from tarball
	manifest, err := ansible.ParseManifestFromTarball(io.NopCloser(bytes.NewReader(content)))
	if err != nil {
		d.logger.Warn(ctx, "Manifest parsing failed",
			zap.NewField("error", err.Error()),
		)
		return nil, err
	}

	d.logger.Info(ctx, "Manifest parsed successfully",
		zap.NewField("namespace", manifest.Namespace),
		zap.NewField("name", manifest.Name),
		zap.NewField("version", manifest.Version),
	)

	// 4. Check namespace permission
	if !d.config.IsNamespaceAllowed(manifest.Namespace) {
		d.logger.Warn(ctx, "Namespace not allowed",
			zap.NewField("namespace", manifest.Namespace),
		)
		return nil, fmt.Errorf("namespace '%s' is not allowed in this repository", manifest.Namespace)
	}

	// 5. Validate tarball structure
	if err := d.validator.ValidateStructure(io.NopCloser(bytes.NewReader(content))); err != nil {
		d.logger.Warn(ctx, "Tarball validation failed",
			zap.NewField("error", err.Error()),
		)
		return nil, err
	}

	// 6. Store blob in CAS
	putResp, err := d.cas.PutBlob(ctx, &ports.PutBlobRequest{
		Content: bytes.NewReader(content),
		SHA256:  req.SHA256,
	})
	if err != nil {
		d.logger.Error(ctx, "Failed to store blob in CAS",
			zap.NewField("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to store blob: %w", err)
	}

	d.logger.Info(ctx, "Blob stored in CAS",
		zap.NewField("sha256", putResp.SHA256),
		zap.NewField("already_exists", putResp.AlreadyExists),
	)

	// 7. Store artifact metadata if new
	if !putResp.AlreadyExists {
		artifactMeta := &ports.ArtifactMetadata{
			SHA256:      putResp.SHA256,
			Size:        req.Size,
			ContentType: "application/gzip",
			RefCount:    0,
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
		}
		if err := d.metadata.StoreArtifact(ctx, artifactMeta); err != nil {
			d.logger.Error(ctx, "Failed to store artifact metadata",
				zap.NewField("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to store artifact metadata: %w", err)
		}
	}

	// 8. Check if version already exists
	existingVersion, err := d.metadata.AnsibleMetadata().GetVersion(
		ctx, manifest.Namespace, manifest.Name, manifest.Version,
	)
	if err == nil && existingVersion != nil {
		d.logger.Warn(ctx, "Collection version already exists",
			zap.NewField("namespace", manifest.Namespace),
			zap.NewField("name", manifest.Name),
			zap.NewField("version", manifest.Version),
		)
		return nil, ansible.ErrVersionAlreadyExists
	}

	// 9. Store collection version metadata
	versionReq := &ports.CreateVersionRequest{
		Namespace:        manifest.Namespace,
		Name:             manifest.Name,
		Version:          manifest.Version,
		ArtifactSHA256:   putResp.SHA256,
		License:          strings.Join(manifest.License, ", "),
		Tags:             manifest.Tags,
		Authors:          manifest.Authors,
		Dependencies:     manifest.Dependencies,
		RepositoryURL:    manifest.Repository,
		DocumentationURL: manifest.Documentation,
		HomepageURL:      manifest.Homepage,
		IssuesURL:        manifest.Issues,
		RequiresAnsible:  manifest.RequiresAnsible,
		UploadedBy:       req.UploadedBy,
	}

	version, err := d.metadata.AnsibleMetadata().CreateVersion(ctx, versionReq)
	if err != nil {
		d.logger.Error(ctx, "Failed to create collection version",
			zap.NewField("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// 10. Increment reference count
	if err := d.metadata.IncrementRefCount(ctx, putResp.SHA256); err != nil {
		d.logger.Error(ctx, "Failed to increment ref count",
			zap.NewField("error", err.Error()),
		)
	}

	d.logger.Info(ctx, "Collection stored successfully",
		zap.NewField("namespace", manifest.Namespace),
		zap.NewField("name", manifest.Name),
		zap.NewField("version", manifest.Version),
		zap.NewField("sha256", putResp.SHA256),
	)

	return &ports.StorePackageResponse{
		SHA256:      putResp.SHA256,
		Size:        req.Size,
		StoredAt:    version.CreatedAt,
		PackagePath: fmt.Sprintf("%s-%s-%s.tar.gz", manifest.Namespace, manifest.Name, manifest.Version),
	}, nil
}

// GetPackage retrieves an Ansible collection package
func (d *HostedDriver) GetPackage(ctx context.Context, req *ports.GetPackageRequest) (*ports.GetPackageResponse, error) {
	d.logger.Info(ctx, "Retrieving Ansible collection",
		zap.NewField("repository", d.config.Name),
		zap.NewField("name", req.Name),
		zap.NewField("version", req.Version),
	)

	// Parse namespace and name from req.Name (format: "namespace.name")
	namespace, name, err := parseCollectionName(req.Name)
	if err != nil {
		return nil, err
	}

	// 1. Get version metadata
	version, err := d.metadata.AnsibleMetadata().GetVersion(ctx, namespace, name, req.Version)
	if err != nil {
		d.logger.Warn(ctx, "Version not found",
			zap.NewField("namespace", namespace),
			zap.NewField("name", name),
			zap.NewField("version", req.Version),
		)
		return nil, fmt.Errorf("version not found: %w", err)
	}

	// 2. Get blob from CAS
	reader, err := d.cas.GetBlob(ctx, version.ArtifactSHA256)
	if err != nil {
		d.logger.Error(ctx, "Failed to get blob from CAS",
			zap.NewField("sha256", version.ArtifactSHA256),
			zap.NewField("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get blob: %w", err)
	}

	// 3. Get artifact metadata
	artifactMeta, err := d.metadata.GetArtifact(ctx, version.ArtifactSHA256)
	if err != nil {
		d.logger.Warn(ctx, "Failed to get artifact metadata",
			zap.NewField("error", err.Error()),
		)
	}

	d.logger.Info(ctx, "Collection retrieved successfully",
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", req.Version),
	)

	var size int64
	if artifactMeta != nil {
		size = artifactMeta.Size
	}

	return &ports.GetPackageResponse{
		Content:     reader,
		ContentType: "application/gzip",
		Size:        size,
		SHA256:      version.ArtifactSHA256,
		Metadata: &ports.PackageMetadata{
			Namespace:   namespace,
			Name:        name,
			Version:     req.Version,
			Description: "",
		},
	}, nil
}

// DeletePackage deletes an Ansible collection version
func (d *HostedDriver) DeletePackage(ctx context.Context, req *ports.DeletePackageRequest) error {
	d.logger.Info(ctx, "Deleting Ansible collection",
		zap.NewField("repository", d.config.Name),
		zap.NewField("name", req.Name),
		zap.NewField("version", req.Version),
	)

	namespace, name, err := parseCollectionName(req.Name)
	if err != nil {
		return err
	}

	// 1. Get version to obtain artifact SHA256
	version, err := d.metadata.AnsibleMetadata().GetVersion(ctx, namespace, name, req.Version)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 2. Delete version metadata
	if err := d.metadata.AnsibleMetadata().DeleteVersion(ctx, namespace, name, req.Version); err != nil {
		d.logger.Error(ctx, "Failed to delete version metadata",
			zap.NewField("error", err.Error()),
		)
		return fmt.Errorf("failed to delete version: %w", err)
	}

	// 3. Decrement reference count
	if err := d.metadata.DecrementRefCount(ctx, version.ArtifactSHA256); err != nil {
		d.logger.Error(ctx, "Failed to decrement ref count",
			zap.NewField("error", err.Error()),
		)
	}

	// 4. Check if artifact should be deleted (ref count = 0)
	artifactMeta, err := d.metadata.GetArtifact(ctx, version.ArtifactSHA256)
	if err == nil && artifactMeta.RefCount == 0 {
		if err := d.cas.DeleteBlob(ctx, version.ArtifactSHA256); err != nil {
			d.logger.Warn(ctx, "Failed to delete blob from CAS",
				zap.NewField("error", err.Error()),
			)
		}
	}

	d.logger.Info(ctx, "Collection deleted successfully",
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", req.Version),
	)

	return nil
}

// ListPackages lists collections
func (d *HostedDriver) ListPackages(ctx context.Context, req *ports.ListPackagesRequest) (*ports.ListPackagesResponse, error) {
	namespace := ""
	if req.Filters != nil {
		namespace = req.Filters["namespace"]
	}

	collections, err := d.metadata.AnsibleMetadata().ListCollections(ctx, &ports.ListCollectionsRequest{
		Namespace: namespace,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	packages := make([]*ports.PackageMetadata, len(collections.Collections))
	for i, coll := range collections.Collections {
		packages[i] = &ports.PackageMetadata{
			Namespace:   coll.Namespace,
			Name:        coll.Name,
			Description: coll.Description,
			Deprecated:  coll.Deprecated,
		}
	}

	return &ports.ListPackagesResponse{
		Packages:   packages,
		Total:      collections.Total,
		Page:       collections.Page,
		PageSize:   req.PageSize,
		TotalPages: collections.TotalPages,
	}, nil
}

// GetMetadata retrieves collection metadata
func (d *HostedDriver) GetMetadata(ctx context.Context, req *ports.GetMetadataRequest) (*ports.PackageMetadata, error) {
	namespace, name, err := parseCollectionName(req.Name)
	if err != nil {
		return nil, err
	}

	collection, err := d.metadata.AnsibleMetadata().GetCollection(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	return &ports.PackageMetadata{
		Namespace:   collection.Namespace,
		Name:        collection.Name,
		Description: collection.Description,
		Deprecated:  collection.Deprecated,
	}, nil
}

// parseCollectionName parses "namespace.name" into separate parts
func parseCollectionName(fullName string) (namespace, name string, err error) {
	parts := strings.Split(fullName, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid collection name format: expected 'namespace.name', got '%s'", fullName)
	}
	return parts[0], parts[1], nil
}
