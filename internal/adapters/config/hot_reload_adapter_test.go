package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configTypes "proxynd/internal/config"
	"proxynd/internal/ports"
)

func TestNewHotReloadAdapter(t *testing.T) {
	t.Run("creates adapter with manager", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()

		// Act
		adapter := NewHotReloadAdapter(manager)

		// Assert
		assert.NotNil(t, adapter)
		assert.Implements(t, (*ports.ConfigObserver)(nil), adapter)
	})
}

func TestHotReloadAdapter_Name(t *testing.T) {
	t.Run("returns correct name", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()
		adapter := NewHotReloadAdapter(manager)

		// Act
		name := adapter.Name()

		// Assert
		assert.Equal(t, "HotReloadAdapter", name)
	})
}

func TestHotReloadAdapter_OnConfigChange(t *testing.T) {
	t.Run("delegates to hot reload manager", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()

		// Track handler calls
		var handlerCalled bool
		var receivedOld, receivedNew *configTypes.RootConfig
		mockHandler := &mockReloadHandler{
			name: "test-handler",
			onReload: func(old, new *configTypes.RootConfig) error {
				handlerCalled = true
				receivedOld = old
				receivedNew = new
				return nil
			},
		}
		manager.RegisterHandler(mockHandler)

		adapter := NewHotReloadAdapter(manager)

		oldConfig := &configTypes.RootConfig{
			Server: configTypes.ServerConfig{Port: 8080},
		}
		newConfig := &configTypes.RootConfig{
			Server: configTypes.ServerConfig{Port: 9090},
		}

		// Act
		err := adapter.OnConfigChange(oldConfig, newConfig)

		// Assert
		assert.NoError(t, err)
		assert.True(t, handlerCalled, "Handler should be called")
		assert.Equal(t, oldConfig, receivedOld)
		assert.Equal(t, newConfig, receivedNew)
	})

	t.Run("returns error from handler", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()

		expectedErr := errors.New("handler error")
		mockHandler := &mockReloadHandler{
			name: "failing-handler",
			onReload: func(old, new *configTypes.RootConfig) error {
				return expectedErr
			},
		}
		manager.RegisterHandler(mockHandler)

		adapter := NewHotReloadAdapter(manager)

		oldConfig := &configTypes.RootConfig{}
		newConfig := &configTypes.RootConfig{}

		// Act
		err := adapter.OnConfigChange(oldConfig, newConfig)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failing-handler")
	})

	t.Run("continues notifying handlers after one fails", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()

		var handler1Called, handler2Called, handler3Called bool

		handler1 := &mockReloadHandler{
			name:     "handler-1",
			onReload: func(old, new *configTypes.RootConfig) error { handler1Called = true; return nil },
		}
		handler2 := &mockReloadHandler{
			name:     "handler-2",
			onReload: func(old, new *configTypes.RootConfig) error { handler2Called = true; return errors.New("error") },
		}
		handler3 := &mockReloadHandler{
			name:     "handler-3",
			onReload: func(old, new *configTypes.RootConfig) error { handler3Called = true; return nil },
		}

		manager.RegisterHandler(handler1)
		manager.RegisterHandler(handler2)
		manager.RegisterHandler(handler3)

		adapter := NewHotReloadAdapter(manager)

		oldConfig := &configTypes.RootConfig{}
		newConfig := &configTypes.RootConfig{}

		// Act
		err := adapter.OnConfigChange(oldConfig, newConfig)

		// Assert
		assert.Error(t, err) // Should return first error
		assert.True(t, handler1Called, "Handler 1 should be called")
		assert.True(t, handler2Called, "Handler 2 should be called")
		assert.True(t, handler3Called, "Handler 3 should be called despite handler 2 failing")
	})

	t.Run("handles nil configs gracefully", func(t *testing.T) {
		// Arrange
		manager := configTypes.NewHotReloadManager()

		var handlerCalled bool
		mockHandler := &mockReloadHandler{
			name:     "test-handler",
			onReload: func(old, new *configTypes.RootConfig) error { handlerCalled = true; return nil },
		}
		manager.RegisterHandler(mockHandler)

		adapter := NewHotReloadAdapter(manager)

		// Act
		err := adapter.OnConfigChange(nil, nil)

		// Assert
		assert.NoError(t, err)
		assert.True(t, handlerCalled, "Handler should be called even with nil configs")
	})
}

func TestHotReloadAdapter_Integration(t *testing.T) {
	t.Run("adapter works with config loader observer pattern", func(t *testing.T) {
		// Arrange - setup config loader
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := validTestConfigWithPort(8080)
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Setup hot reload manager with handler
		manager := configTypes.NewHotReloadManager()
		var handlerCalled bool
		var oldPort, newPort int

		mockHandler := &mockReloadHandler{
			name: "port-tracker",
			onReload: func(old, new *configTypes.RootConfig) error {
				handlerCalled = true
				if old != nil {
					oldPort = old.Server.Port
				}
				if new != nil {
					newPort = new.Server.Port
				}
				return nil
			},
		}
		manager.RegisterHandler(mockHandler)

		// Connect adapter to loader
		adapter := NewHotReloadAdapter(manager)
		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(adapter)

		// Act - update config
		updatedContent := validTestConfigWithPort(9090)
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		_, err = loader.Reload()
		require.NoError(t, err)

		// Assert
		assert.True(t, handlerCalled, "Handler should be called via adapter")
		assert.Equal(t, 8080, oldPort)
		assert.Equal(t, 9090, newPort)
	})
}

// mockReloadHandler is a test double for config.ReloadHandler
type mockReloadHandler struct {
	name     string
	onReload func(old, new *configTypes.RootConfig) error
}

func (m *mockReloadHandler) Name() string {
	return m.name
}

func (m *mockReloadHandler) OnConfigReload(old, new *configTypes.RootConfig) error {
	if m.onReload != nil {
		return m.onReload(old, new)
	}
	return nil
}
