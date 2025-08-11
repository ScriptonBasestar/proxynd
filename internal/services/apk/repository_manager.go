package apk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"proxynd/internal/domain/apk"
	"proxynd/logging"
)

type repositoryManagerImpl struct {
	config     apk.ProxyConfig
	logger     logging.Logger
	storageDir string
	repos      map[string]*apk.RepositoryInfo
}

func NewRepositoryManager(config apk.ProxyConfig, logger logging.Logger, storageDir string) apk.RepositoryManager {
	return &repositoryManagerImpl{
		config:     config,
		logger:     logger,
		storageDir: storageDir,
		repos:      make(map[string]*apk.RepositoryInfo),
	}
}

func (r *repositoryManagerImpl) GetRepositoryInfo(ctx context.Context, architecture, branch, component string) (*apk.RepositoryInfo, error) { //nolint:lll
	key := fmt.Sprintf("%s:%s:%s", branch, component, architecture)

	if info, exists := r.repos[key]; exists {
		return info, nil
	}

	indexPath := filepath.Join(r.storageDir, r.config.GetPath(), branch, component, architecture, "APKINDEX.tar.gz")
	stat, err := os.Stat(indexPath)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %s", key)
	}

	info := &apk.RepositoryInfo{
		Architecture: architecture,
		Branch:       branch,
		Component:    component,
		IndexFile:    "APKINDEX.tar.gz",
		LastModified: stat.ModTime(),
		Size:         stat.Size(),
	}

	r.repos[key] = info
	return info, nil
}

func (r *repositoryManagerImpl) UpdateRepositoryIndex(ctx context.Context, info *apk.RepositoryInfo) error {
	key := fmt.Sprintf("%s:%s:%s", info.Branch, info.Component, info.Architecture)
	info.LastModified = time.Now()
	r.repos[key] = info
	return nil
}

func (r *repositoryManagerImpl) ValidateRepository(architecture, branch, component string) error {
	if architecture == "" || branch == "" || component == "" {
		return fmt.Errorf("architecture, branch, and component are required")
	}

	validArches := []string{"x86_64", "x86", "aarch64", "armhf", "armv7", "ppc64le", "s390x"}
	validBranches := []string{"edge", "v3.18", "v3.17", "v3.16", "v3.15"}
	validComponents := []string{"main", "community", "testing"}

	if !r.contains(validArches, architecture) {
		return fmt.Errorf("invalid architecture: %s", architecture)
	}
	if !r.contains(validBranches, branch) {
		return fmt.Errorf("invalid branch: %s", branch)
	}
	if !r.contains(validComponents, component) {
		return fmt.Errorf("invalid component: %s", component)
	}

	return nil
}

func (r *repositoryManagerImpl) GetAvailableArchitectures(ctx context.Context) ([]string, error) {
	return []string{"x86_64", "x86", "aarch64", "armhf", "armv7", "ppc64le", "s390x"}, nil
}

func (r *repositoryManagerImpl) GetAvailableBranches(ctx context.Context) ([]string, error) {
	return []string{"edge", "v3.18", "v3.17", "v3.16", "v3.15"}, nil
}

func (r *repositoryManagerImpl) GetAvailableComponents(ctx context.Context) ([]string, error) {
	return []string{"main", "community", "testing"}, nil
}

func (r *repositoryManagerImpl) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
