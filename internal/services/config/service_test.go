package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/configs"
)

// TestNewService tests the creation of a new config service
func TestNewService(t *testing.T) {
	tests := []struct {
		name      string
		configDir string
		setupFunc func(t *testing.T, dir string)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid config directory",
			configDir: "",
			setupFunc: func(t *testing.T, dir string) {
				// Create minimal valid configs
				createTestConfig(t, dir, "global.yaml", `
storage_dir: /tmp/cache
config_dir: /tmp/config
cache:
  ttl: 3600
`)
			},
			wantErr: false,
		},
		{
			name:      "missing config directory",
			configDir: "/non/existent/path",
			setupFunc: func(_ *testing.T, _ string) {},
			wantErr:   true,
			errMsg:    "failed to load configurations",
		},
		{
			name:      "invalid config content",
			configDir: "",
			setupFunc: func(t *testing.T, dir string) {
				createTestConfig(t, dir, "global.yaml", `invalid: yaml: content`)
			},
			wantErr: true,
			errMsg:  "configuration validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test configs
			tempDir := t.TempDir()
			if tt.configDir == "" {
				tt.configDir = tempDir
			}

			// Set config directory environment variable
			oldConfigDir := os.Getenv("CONFIG_DIR")
			_ = os.Setenv("CONFIG_DIR", tempDir)
			defer func() { _ = os.Setenv("CONFIG_DIR", oldConfigDir) }()

			// Setup test configs
			tt.setupFunc(t, tempDir)

			// Create service
			ctx := context.Background()
			svc, err := NewService(ctx, tt.configDir)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
	}
}

// TestService_GetGlobalConfig tests retrieving global configuration
func TestService_GetGlobalConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func() Service
		wantErr bool
		check   func(t *testing.T, cfg *configs.GlobalConfig)
	}{
		{
			name: "valid global config",
			setup: func() Service {
				return &service{
					globalConfig: &configs.UnifiedConfig{
						Cache: configs.CacheConfig{
							File: configs.FileCacheConfig{
								Directory: "/tmp/storage",
							},
						},
					},
					configDir: "/tmp/config",
				}
			},
			wantErr: false,
			check: func(t *testing.T, cfg *configs.GlobalConfig) {
				assert.Equal(t, "/tmp/storage", cfg.StorageDir)
				assert.Equal(t, "/tmp/config", cfg.ConfigDir)
				assert.Equal(t, 3600, cfg.Cache.TTL)
			},
		},
		{
			name: "nil global config",
			setup: func() Service {
				return &service{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.setup()
			cfg, err := svc.GetGlobalConfig(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.check != nil {
					tt.check(t, cfg)
				}
			}
		})
	}
}

// TestService_GetProxyConfigs tests retrieving various proxy configurations
func TestService_GetProxyConfigs(t *testing.T) {
	ctx := context.Background()

	// Create a service with all configs loaded
	svc := &service{
		mavenConfig: &configs.MavenProxyConfig{
			Path:     "/maven",
			UseCache: true,
		},
		aptConfig: &configs.AptProxyConfig{
			Path:     "/apt",
			UseCache: true,
		},
		npmConfig: &configs.NpmProxyConfig{
			Path:     "/npm",
			UseCache: true,
		},
		dockerConfig: &configs.DockerProxyConfig{
			Path:     "/docker",
			UseCache: true,
		},
		pipConfig: &configs.PipProxyConfig{
			Path:     "/pip",
			UseCache: true,
		},
		yumConfig: &configs.YumProxyConfig{
			Path:     "/yum",
			UseCache: true,
		},
		apkConfig: &configs.ApkProxyConfig{
			Path:     "/apk",
			UseCache: true,
		},
	}

	// Test each getter
	tests := []struct {
		name   string
		getter func(context.Context) (interface{}, error)
		path   string
	}{
		{
			name:   "maven config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetMavenConfig(ctx) },
			path:   "/maven",
		},
		{
			name:   "apt config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetAptConfig(ctx) },
			path:   "/apt",
		},
		{
			name:   "npm config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetNpmConfig(ctx) },
			path:   "/npm",
		},
		{
			name:   "docker config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetDockerConfig(ctx) },
			path:   "/docker",
		},
		{
			name:   "pip config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetPipConfig(ctx) },
			path:   "/pip",
		},
		{
			name:   "yum config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetYumConfig(ctx) },
			path:   "/yum",
		},
		{
			name:   "apk config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetApkConfig(ctx) },
			path:   "/apk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := tt.getter(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			// Use reflection to check the Path field
			switch v := cfg.(type) {
			case *configs.MavenProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.AptProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.NpmProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.DockerProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.PipProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.YumProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			case *configs.ApkProxyConfig:
				assert.Equal(t, tt.path, v.Path)
			}
		})
	}
}

// TestService_GetProxyConfigs_Nil tests error cases for nil configs
func TestService_GetProxyConfigs_Nil(t *testing.T) {
	ctx := context.Background()
	svc := &service{} // All configs are nil

	tests := []struct {
		name   string
		getter func(context.Context) (interface{}, error)
		errMsg string
	}{
		{
			name:   "nil maven config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetMavenConfig(ctx) },
			errMsg: "maven configuration not loaded",
		},
		{
			name:   "nil apt config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetAptConfig(ctx) },
			errMsg: "apt configuration not loaded",
		},
		{
			name:   "nil npm config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetNpmConfig(ctx) },
			errMsg: "npm configuration not loaded",
		},
		{
			name:   "nil docker config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetDockerConfig(ctx) },
			errMsg: "docker configuration not loaded",
		},
		{
			name:   "nil pip config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetPipConfig(ctx) },
			errMsg: "pip configuration not loaded",
		},
		{
			name:   "nil yum config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetYumConfig(ctx) },
			errMsg: "yum configuration not loaded",
		},
		{
			name:   "nil apk config",
			getter: func(ctx context.Context) (interface{}, error) { return svc.GetApkConfig(ctx) },
			errMsg: "apk configuration not loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := tt.getter(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
			assert.Nil(t, cfg)
		})
	}
}

// TestService_ValidateAll tests configuration validation
func TestService_ValidateAll(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func() *service
		wantErr bool
		errMsg  string
	}{
		{
			name: "all configs valid",
			setup: func() *service {
				svc := &service{
					validators: make(map[string]Validator),
					globalConfig: &configs.UnifiedConfig{
						Cache: configs.CacheConfig{
							File: configs.FileCacheConfig{
								Directory: "/tmp/storage",
							},
						},
					},
					aptConfig: &configs.AptProxyConfig{
						Path: "/apt",
						Proxies: map[string][]configs.AptProxy{
							"default": {
								{Name: "ubuntu", URL: "http://archive.ubuntu.com"},
							},
						},
					},
				}
				svc.registerValidators()
				return svc
			},
			wantErr: false,
		},
		{
			name: "validation failure",
			setup: func() *service {
				svc := &service{
					validators: make(map[string]Validator),
					globalConfig: &configs.UnifiedConfig{
						Cache: configs.CacheConfig{
							File: configs.FileCacheConfig{
								Directory: "", // This should fail validation
							},
						},
					},
				}
				svc.validators["global"] = &mockValidator{
					shouldFail: true,
					errMsg:     "cache directory cannot be empty",
				}
				return svc
			},
			wantErr: true,
			errMsg:  "global config validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.setup()
			err := svc.ValidateAll(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestService_Reload tests configuration reload
func TestService_Reload(t *testing.T) {
	ctx := context.Background()

	// Create temp directory with test configs
	tempDir := t.TempDir()
	createTestConfig(t, tempDir, "global.yaml", `
storage_dir: /tmp/storage
config_dir: /tmp/config
cache:
  ttl: 3600
`)

	// Set config directory
	oldConfigDir := os.Getenv("CONFIG_DIR")
	_ = os.Setenv("CONFIG_DIR", tempDir)
	defer func() { _ = os.Setenv("CONFIG_DIR", oldConfigDir) }()

	svc := &service{
		configDir:  tempDir,
		validators: make(map[string]Validator),
	}
	svc.registerValidators()

	// Test reload
	err := svc.Reload(ctx)
	require.NoError(t, err)

	// Verify config was loaded
	cfg, err := svc.GetGlobalConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/storage", cfg.StorageDir)
}

// TestService_Concurrency tests concurrent access to configs
func TestService_Concurrency(_ *testing.T) {
	ctx := context.Background()
	svc := &service{
		globalConfig: &configs.UnifiedConfig{
			Cache: configs.CacheConfig{
				File: configs.FileCacheConfig{
					Directory: "/tmp/storage",
				},
			},
		},
		mavenConfig: &configs.MavenProxyConfig{
			Path: "/maven",
		},
	}

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = svc.GetGlobalConfig(ctx)
				_, _ = svc.GetMavenConfig(ctx)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper functions

func createTestConfig(t *testing.T, dir, filename, content string) {
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// mockValidator is a mock implementation of Validator for testing
type mockValidator struct {
	shouldFail bool
	errMsg     string
}

func (m *mockValidator) Validate(_ interface{}) error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}
