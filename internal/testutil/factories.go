package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"proxynd/configs"
	repocache "proxynd/internal/repositories/cache"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
)

// ConfigServiceAdapter adapts config.Service to proxy.ConfigService
type ConfigServiceAdapter struct {
	service config.Service
}

// NewConfigServiceAdapter creates a new config service adapter
func NewConfigServiceAdapter(service config.Service) proxy.ConfigService {
	return &ConfigServiceAdapter{service: service}
}

// GetProxyConfig returns configuration for a specific proxy type
func (a *ConfigServiceAdapter) GetProxyConfig(ctx context.Context, proxyType string) (interface{}, error) {
	switch proxyType {
	case "maven":
		return a.service.GetMavenConfig(ctx)
	case "apt":
		return a.service.GetAptConfig(ctx)
	case "npm":
		return a.service.GetNpmConfig(ctx)
	case "docker":
		return a.service.GetDockerConfig(ctx)
	case "pip":
		return a.service.GetPipConfig(ctx)
	case "yum":
		return a.service.GetYumConfig(ctx)
	case "apk":
		return a.service.GetApkConfig(ctx)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}
}

// GetGlobalConfig returns global configuration
func (a *ConfigServiceAdapter) GetGlobalConfig(ctx context.Context) (interface{}, error) {
	return a.service.GetGlobalConfig(ctx)
}

// ReloadConfig reloads configuration from disk
func (a *ConfigServiceAdapter) ReloadConfig(ctx context.Context) error {
	return a.service.Reload(ctx)
}

// Factory provides methods to create test objects with sensible defaults
type Factory struct {
	t       *testing.T
	tempDir string
}

// NewFactory creates a new test factory
func NewFactory(t *testing.T) *Factory {
	tempDir := t.TempDir()
	return &Factory{
		t:       t,
		tempDir: tempDir,
	}
}

// ConfigService creates a test configuration service
func (f *Factory) ConfigService() config.Service {
	configDir := filepath.Join(f.tempDir, "config")
	require.NoError(f.t, os.MkdirAll(configDir, 0755))

	// Create test config files
	f.createTestConfigFiles(configDir)

	service, err := config.NewService(context.Background(), configDir)
	require.NoError(f.t, err)

	return service
}

// CacheRepository creates a test cache repository
func (f *Factory) CacheRepository() repocache.Repository {
	cacheDir := filepath.Join(f.tempDir, "cache")
	require.NoError(f.t, os.MkdirAll(cacheDir, 0755))

	// Assuming there's a filesystem implementation
	return &fileSystemCacheRepository{
		baseDir: cacheDir,
	}
}

// CacheAdapter creates a test cache adapter
func (f *Factory) CacheAdapter() proxy.CacheService {
	repo := f.CacheRepository()
	return adapters.NewCacheAdapter(repo, 5*time.Minute)
}

// UpstreamClient creates a test upstream client
func (f *Factory) UpstreamClient() proxy.UpstreamClient {
	return adapters.NewHTTPUpstreamClient(30 * time.Second)
}

// ProxyService creates a test proxy service
func (f *Factory) ProxyService(proxyType string) proxy.ProxyService {
	configService := f.ConfigService()
	adaptedConfigService := NewConfigServiceAdapter(configService)
	cacheAdapter := f.CacheAdapter()
	upstreamClient := f.UpstreamClient()

	factory := proxy.NewServiceFactory(cacheAdapter, adaptedConfigService, upstreamClient)
	service, err := factory.GetService(proxyType)
	require.NoError(f.t, err)

	return service
}

// GlobalConfig creates a test global configuration
func (f *Factory) GlobalConfig() *configs.GlobalConfig {
	return &configs.GlobalConfig{
		StorageDir:   filepath.Join(f.tempDir, "storage"),
		ConfigDir:    filepath.Join(f.tempDir, "config"),
		CacheDir:     filepath.Join(f.tempDir, "cache"),
		CacheTTL:     3600,
		MaxCacheSize: 1024 * 1024 * 100, // 100MB for tests
	}
}

// createTestConfigFiles creates necessary test configuration files
func (f *Factory) createTestConfigFiles(configDir string) {
	// Global config
	globalConfig := `
storage_dir: ./storage
cache_dir: ./cache
cache_ttl: 3600
max_cache_size: 104857600
`
	require.NoError(f.t, os.WriteFile(
		filepath.Join(configDir, "global.yaml"),
		[]byte(globalConfig),
		0644,
	))

	// APT config
	aptConfig := `
name: apt-proxy
enabled: true
servers:
  - name: ubuntu
    enabled: true
    server:
      url: http://archive.ubuntu.com/ubuntu
`
	require.NoError(f.t, os.WriteFile(
		filepath.Join(configDir, "apt-proxy.yaml"),
		[]byte(aptConfig),
		0644,
	))

	// Maven config
	mavenConfig := `
name: maven-proxy
enabled: true
servers:
  - name: central
    enabled: true
    url: https://repo1.maven.org/maven2
`
	require.NoError(f.t, os.WriteFile(
		filepath.Join(configDir, "maven-proxy.yaml"),
		[]byte(mavenConfig),
		0644,
	))

	// NPM config
	npmConfig := `
name: npm-proxy
enabled: true
servers:
  - name: npmjs
    enabled: true
    url: https://registry.npmjs.org
`
	require.NoError(f.t, os.WriteFile(
		filepath.Join(configDir, "npm-proxy.yaml"),
		[]byte(npmConfig),
		0644,
	))
}

// fileSystemCacheRepository is a simple implementation for testing
type fileSystemCacheRepository struct {
	baseDir string
}

// Get returns the
func (r *fileSystemCacheRepository) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(r.baseDir, key)
	return os.Open(path)
}

// Put performs put operation

func (r *fileSystemCacheRepository) Put(_ context.Context, key string, content io.Reader, _ time.Duration) error {
	path := filepath.Join(r.baseDir, key)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, content)
	return err
	// Exists performs exists operation
}

func (r *fileSystemCacheRepository) Exists(_ context.Context, key string) (bool, error) {
	path := filepath.Join(r.baseDir, key)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	// Delete performs delete operation
	return err == nil, err
}

func (r *fileSystemCacheRepository) Delete(_ context.Context, key string) error {
	// List performs list operation
	path := filepath.Join(r.baseDir, key)
	return os.Remove(path)
}

func (r *fileSystemCacheRepository) List(_ context.Context, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(r.baseDir, pattern))
	if err != nil {
		return nil, err
	}

	// Remove base directory from paths
	for i, match := range matches {
		// 상대 경로 변환 오류시 원본 경로 사용
		if relPath, err := filepath.Rel(r.baseDir, match); err == nil {
			matches[i] = relPath
		} else {
			matches[i] = match
		}
		// Size performs size operation
	}

	return matches, nil
}

func (r *fileSystemCacheRepository) Size(_ context.Context, key string) (int64, error) {
	path := filepath.Join(r.baseDir, key)
	info, err := os.Stat(path)
	// Clear performs clear operation
	if err != nil {
		return 0, err
	}
	// Stats performs stats operation
	return info.Size(), nil
}

func (r *fileSystemCacheRepository) Clear(_ context.Context) error {
	return os.RemoveAll(r.baseDir)
}

func (r *fileSystemCacheRepository) Stats(_ context.Context) (*repocache.CacheStats, error) {
	var stats repocache.CacheStats

	err := filepath.Walk(r.baseDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			stats.TotalItems++
			stats.TotalSize += info.Size()

			if stats.OldestItem.IsZero() || info.ModTime().Before(stats.OldestItem) {
				stats.OldestItem = info.ModTime()
			}
			if info.ModTime().After(stats.NewestItem) {
				stats.NewestItem = info.ModTime()
			}
		}
		return nil
	})

	return &stats, err
}
