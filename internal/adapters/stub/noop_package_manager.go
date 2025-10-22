package stub

import (
	"context"
	"fmt"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpPackageManager is a no-operation implementation of ports.PackageManager
// Used for testing and gradual migration to hexagonal architecture
type NoOpPackageManager struct {
	logger logging.Logger
}

// NewNoOpPackageManager creates a new NoOp package manager
func NewNoOpPackageManager(logger logging.Logger) ports.PackageManager {
	return &NoOpPackageManager{logger: logger}
}

// GetPackage retrieves a package from repository (NoOp implementation)
func (pm *NoOpPackageManager) GetPackage(ctx context.Context, req *ports.PackageRequest) (*ports.PackageResponse, error) {
	pm.logger.Debug("NoOpPackageManager.GetPackage called (stub)",
		logging.F("repository", req.Repository),
		logging.F("name", req.Name),
		logging.F("version", req.Version))

	return nil, fmt.Errorf("NoOpPackageManager: GetPackage not implemented")
}

// ListPackages lists available packages in repository (NoOp implementation)
func (pm *NoOpPackageManager) ListPackages(ctx context.Context, repo string) (*ports.PackageListResponse, error) {
	pm.logger.Debug("NoOpPackageManager.ListPackages called (stub)",
		logging.F("repository", repo))

	return &ports.PackageListResponse{
		Packages: []ports.PackageInfo{},
		Total:    0,
		Page:     1,
		PageSize: 10,
	}, nil
}

// GetMetadata retrieves package metadata (NoOp implementation)
func (pm *NoOpPackageManager) GetMetadata(ctx context.Context, req *ports.MetadataRequest) (*ports.MetadataResponse, error) {
	pm.logger.Debug("NoOpPackageManager.GetMetadata called (stub)",
		logging.F("repository", req.Repository),
		logging.F("name", req.Name))

	return nil, fmt.Errorf("NoOpPackageManager: GetMetadata not implemented")
}

// UploadPackage uploads a package to repository (NoOp implementation)
func (pm *NoOpPackageManager) UploadPackage(ctx context.Context, req *ports.UploadRequest) error {
	pm.logger.Debug("NoOpPackageManager.UploadPackage called (stub)",
		logging.F("repository", req.Repository),
		logging.F("name", req.Name))

	return fmt.Errorf("NoOpPackageManager: UploadPackage not implemented")
}

// DeletePackage removes a package from repository (NoOp implementation)
func (pm *NoOpPackageManager) DeletePackage(ctx context.Context, req *ports.DeleteRequest) error {
	pm.logger.Debug("NoOpPackageManager.DeletePackage called (stub)",
		logging.F("repository", req.Repository),
		logging.F("name", req.Name))

	return fmt.Errorf("NoOpPackageManager: DeletePackage not implemented")
}
