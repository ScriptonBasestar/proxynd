package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

func TestNewHotReloadManager(t *testing.T) {
	manager := config.NewHotReloadManager()
	assert.NotNil(t, manager)
}

func TestHotReloadManager_RegisterHandler(t *testing.T) {
	manager := config.NewHotReloadManager()

	handler := config.NewLoggingReloadHandler()
	manager.RegisterHandler(handler)

	// No panic means success - handlers are internal
}

func TestLoggingReloadHandler_Name(t *testing.T) {
	handler := config.NewLoggingReloadHandler()
	assert.Equal(t, "LoggingReloadHandler", handler.Name())
}

func TestLoggingReloadHandler_OnConfigReload_LevelChange(t *testing.T) {
	handler := config.NewLoggingReloadHandler()

	// Set initial level
	logging.SetLevel(logging.LevelInfo)
	assert.Equal(t, logging.LevelInfo, logging.GetLevel())

	oldConfig := &config.UnifiedConfig{
		Logging: config.UnifiedLoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	newConfig := &config.UnifiedConfig{
		Logging: config.UnifiedLoggingConfig{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	// Verify level changed
	assert.Equal(t, logging.LevelDebug, logging.GetLevel())

	// Reset back to info
	logging.SetLevel(logging.LevelInfo)
}

func TestLoggingReloadHandler_OnConfigReload_NilConfigs(t *testing.T) {
	handler := config.NewLoggingReloadHandler()

	// nil old config
	err := handler.OnConfigReload(nil, &config.UnifiedConfig{})
	assert.NoError(t, err)

	// nil new config
	err = handler.OnConfigReload(&config.UnifiedConfig{}, nil)
	assert.NoError(t, err)

	// both nil
	err = handler.OnConfigReload(nil, nil)
	assert.NoError(t, err)
}

func TestCacheReloadHandler_Name(t *testing.T) {
	handler := config.NewCacheReloadHandler()
	assert.Equal(t, "CacheReloadHandler", handler.Name())
}

func TestCacheReloadHandler_OnConfigReload_BackendChange(t *testing.T) {
	handler := config.NewCacheReloadHandler()

	oldConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "file",
			TTL:     time.Hour,
		},
	}

	newConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "s3",
			TTL:     time.Hour,
		},
	}

	// Backend change should return error (requires restart)
	err := handler.OnConfigReload(oldConfig, newConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires restart")
}

func TestCacheReloadHandler_OnConfigReload_TTLChange(t *testing.T) {
	handler := config.NewCacheReloadHandler()

	ttlChangeCalled := false
	var oldTTL, newTTL string

	handler.SetTTLChangeCallback(func(old, new string) {
		ttlChangeCalled = true
		oldTTL = old
		newTTL = new
	})

	oldConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "file",
			TTL:     time.Hour,
		},
	}

	newConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "file",
			TTL:     2 * time.Hour,
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, ttlChangeCalled)
	assert.Equal(t, "1h0m0s", oldTTL)
	assert.Equal(t, "2h0m0s", newTTL)
}

func TestCacheReloadHandler_OnConfigReload_NilConfigs(t *testing.T) {
	handler := config.NewCacheReloadHandler()

	err := handler.OnConfigReload(nil, &config.UnifiedConfig{})
	assert.NoError(t, err)

	err = handler.OnConfigReload(&config.UnifiedConfig{}, nil)
	assert.NoError(t, err)
}

func TestMetricsReloadHandler_Name(t *testing.T) {
	handler := config.NewMetricsReloadHandler()
	assert.Equal(t, "MetricsReloadHandler", handler.Name())
}

func TestMetricsReloadHandler_OnConfigReload_Enable(t *testing.T) {
	handler := config.NewMetricsReloadHandler()

	startCalled := false
	var startPort int
	var startPath string

	handler.SetStartServerCallback(func(port int, path string) {
		startCalled = true
		startPort = port
		startPath = path
	})

	oldConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	newConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, startCalled)
	assert.Equal(t, 9090, startPort)
	assert.Equal(t, "/metrics", startPath)
}

func TestMetricsReloadHandler_OnConfigReload_Disable(t *testing.T) {
	handler := config.NewMetricsReloadHandler()

	stopCalled := false

	handler.SetStopServerCallback(func() {
		stopCalled = true
	})

	oldConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    9090,
		},
	}

	newConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, stopCalled)
}

func TestMetricsReloadHandler_OnConfigReload_PortChange(t *testing.T) {
	handler := config.NewMetricsReloadHandler()

	stopCalled := false
	startCalled := false
	var newPort int

	handler.SetStopServerCallback(func() {
		stopCalled = true
	})

	handler.SetStartServerCallback(func(port int, _ string) {
		startCalled = true
		newPort = port
	})

	oldConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}

	newConfig := &config.UnifiedConfig{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    8080,
			Path:    "/metrics",
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, stopCalled, "stop should be called for port change")
	assert.True(t, startCalled, "start should be called with new port")
	assert.Equal(t, 8080, newPort)
}

func TestMetricsReloadHandler_OnConfigReload_NilConfigs(t *testing.T) {
	handler := config.NewMetricsReloadHandler()

	err := handler.OnConfigReload(nil, &config.UnifiedConfig{})
	assert.NoError(t, err)

	err = handler.OnConfigReload(&config.UnifiedConfig{}, nil)
	assert.NoError(t, err)
}

func TestSecurityReloadHandler_Name(t *testing.T) {
	handler := config.NewSecurityReloadHandler()
	assert.Equal(t, "SecurityReloadHandler", handler.Name())
}

func TestSecurityReloadHandler_OnConfigReload_IPWhitelistEnable(t *testing.T) {
	handler := config.NewSecurityReloadHandler()

	ipWhitelistCalled := false
	var enabled bool
	var allowedIPs []string

	handler.SetIPWhitelistChangeCallback(func(e bool, ips []string) {
		ipWhitelistCalled = true
		enabled = e
		allowedIPs = ips
	})

	oldConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			AccessControl: config.AccessControlConfig{
				IPWhitelist: config.IPWhitelistConfig{
					Enabled: false,
				},
			},
		},
	}

	newConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			AccessControl: config.AccessControlConfig{
				IPWhitelist: config.IPWhitelistConfig{
					Enabled: true,
					IPs:     []string{"10.0.0.1"},
					CIDRs:   []string{"192.168.1.0/24"},
				},
			},
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, ipWhitelistCalled)
	assert.True(t, enabled)
	assert.Equal(t, []string{"10.0.0.1", "192.168.1.0/24"}, allowedIPs)
}

func TestSecurityReloadHandler_OnConfigReload_IPWhitelistUpdate(t *testing.T) {
	handler := config.NewSecurityReloadHandler()

	ipWhitelistCalled := false
	var allowedIPs []string

	handler.SetIPWhitelistChangeCallback(func(_ bool, ips []string) {
		ipWhitelistCalled = true
		allowedIPs = ips
	})

	oldConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			AccessControl: config.AccessControlConfig{
				IPWhitelist: config.IPWhitelistConfig{
					Enabled: true,
					CIDRs:   []string{"192.168.1.0/24"},
				},
			},
		},
	}

	newConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			AccessControl: config.AccessControlConfig{
				IPWhitelist: config.IPWhitelistConfig{
					Enabled: true,
					CIDRs:   []string{"192.168.1.0/24", "10.0.0.0/8"},
				},
			},
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, ipWhitelistCalled)
	assert.Equal(t, []string{"192.168.1.0/24", "10.0.0.0/8"}, allowedIPs)
}

func TestSecurityReloadHandler_OnConfigReload_UserChange(t *testing.T) {
	handler := config.NewSecurityReloadHandler()

	usersCalled := false
	var newUsers map[string]string

	handler.SetUsersChangeCallback(func(users map[string]string) {
		usersCalled = true
		newUsers = users
	})

	oldConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			Authentication: config.AuthenticationConfig{
				BasicAuth: &config.BasicAuthConfig{
					Realm: "ProxyND",
					Users: map[string]string{
						"admin": "oldpass",
					},
				},
			},
		},
	}

	newConfig := &config.UnifiedConfig{
		Security: config.UnifiedSecurityConfig{
			Authentication: config.AuthenticationConfig{
				BasicAuth: &config.BasicAuthConfig{
					Realm: "ProxyND",
					Users: map[string]string{
						"admin": "newpass",
						"user":  "userpass",
					},
				},
			},
		},
	}

	err := handler.OnConfigReload(oldConfig, newConfig)
	require.NoError(t, err)

	assert.True(t, usersCalled)
	assert.Equal(t, map[string]string{
		"admin": "newpass",
		"user":  "userpass",
	}, newUsers)
}

func TestSecurityReloadHandler_OnConfigReload_NilConfigs(t *testing.T) {
	handler := config.NewSecurityReloadHandler()

	err := handler.OnConfigReload(nil, &config.UnifiedConfig{})
	assert.NoError(t, err)

	err = handler.OnConfigReload(&config.UnifiedConfig{}, nil)
	assert.NoError(t, err)
}

func TestHotReloadManager_OnConfigChange(t *testing.T) {
	manager := config.NewHotReloadManager()

	// Register handlers
	loggingHandler := config.NewLoggingReloadHandler()
	cacheHandler := config.NewCacheReloadHandler()
	metricsHandler := config.NewMetricsReloadHandler()
	securityHandler := config.NewSecurityReloadHandler()

	manager.RegisterHandler(loggingHandler)
	manager.RegisterHandler(cacheHandler)
	manager.RegisterHandler(metricsHandler)
	manager.RegisterHandler(securityHandler)

	oldConfig := &config.UnifiedConfig{
		Logging: config.UnifiedLoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Cache: config.CacheConfig{
			Backend: "file",
			TTL:     time.Hour,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	newConfig := &config.UnifiedConfig{
		Logging: config.UnifiedLoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		Cache: config.CacheConfig{
			Backend: "file",
			TTL:     2 * time.Hour,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	err := manager.OnConfigChange(oldConfig, newConfig)
	assert.NoError(t, err)
}

func TestHotReloadManager_OnConfigChange_WithError(t *testing.T) {
	manager := config.NewHotReloadManager()

	cacheHandler := config.NewCacheReloadHandler()
	manager.RegisterHandler(cacheHandler)

	// Backend change should cause error
	oldConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "file",
		},
	}

	newConfig := &config.UnifiedConfig{
		Cache: config.CacheConfig{
			Backend: "s3",
		},
	}

	err := manager.OnConfigChange(oldConfig, newConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CacheReloadHandler")
}
