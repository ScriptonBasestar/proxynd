package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configTypes "proxynd/internal/config"
	"proxynd/internal/ports"
)

func TestNewUnifiedConfigLoader(t *testing.T) {
	t.Run("successful creation with valid config", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Create a minimal valid config file
		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Act
		loader, err := NewUnifiedConfigLoader(configPath)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, loader)
		assert.Implements(t, (*ports.ConfigLoader)(nil), loader)
		assert.Implements(t, (*ports.ConfigProvider)(nil), loader)
	})

	t.Run("error when config file does not exist", func(t *testing.T) {
		// Arrange
		configPath := "/nonexistent/path/config.yaml"

		// Act
		loader, err := NewUnifiedConfigLoader(configPath)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, loader)
	})

	t.Run("error when config file is invalid yaml", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Create invalid YAML
		invalidContent := `
invalid: yaml: content:
  - this is not: valid
		[broken
`
		err := os.WriteFile(configPath, []byte(invalidContent), 0644)
		require.NoError(t, err)

		// Act
		loader, err := NewUnifiedConfigLoader(configPath)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, loader)
	})
}

func TestUnifiedConfigLoader_GetCurrent(t *testing.T) {
	t.Run("returns current config after successful load", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act
		cfg := loader.GetCurrent()

		// Assert
		assert.NotNil(t, cfg)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	})
}

func TestUnifiedConfigLoader_GetPath(t *testing.T) {
	t.Run("returns config file path", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act
		path := loader.GetPath()

		// Assert
		assert.Equal(t, configPath, path)
	})
}

func TestUnifiedConfigLoader_Load(t *testing.T) {
	t.Run("loads config from new path", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath1 := filepath.Join(tempDir, "config1.yaml")
		configPath2 := filepath.Join(tempDir, "config2.yaml")

		config1Content := validTestConfigWithPort(8080)
		config2Content := validTestConfigWithPort(9090)
		err := os.WriteFile(configPath1, []byte(config1Content), 0644)
		require.NoError(t, err)
		err = os.WriteFile(configPath2, []byte(config2Content), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath1)
		require.NoError(t, err)
		assert.Equal(t, 8080, loader.GetCurrent().Server.Port)

		// Act
		cfg, err := loader.Load(configPath2)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, 9090, loader.GetCurrent().Server.Port)
		assert.Equal(t, configPath2, loader.GetPath())
	})

	t.Run("error when loading invalid config", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath1 := filepath.Join(tempDir, "config1.yaml")
		configPath2 := filepath.Join(tempDir, "config2.yaml")

		config1Content := validTestConfig()
		err := os.WriteFile(configPath1, []byte(config1Content), 0644)
		require.NoError(t, err)
		err = os.WriteFile(configPath2, []byte("invalid: yaml: ["), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath1)
		require.NoError(t, err)

		// Act
		cfg, err := loader.Load(configPath2)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cfg)
		// Current config should remain unchanged
		assert.Equal(t, 8080, loader.GetCurrent().Server.Port)
		assert.Equal(t, configPath1, loader.GetPath())
	})
}

func TestUnifiedConfigLoader_Reload(t *testing.T) {
	t.Run("reloads config from same path", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := validTestConfigWithPort(8080)
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)
		assert.Equal(t, 8080, loader.GetCurrent().Server.Port)

		// Modify the config file
		updatedContent := validTestConfigWithPort(9090)
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Act
		cfg, err := loader.Reload()

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, 9090, loader.GetCurrent().Server.Port)
	})
}

func TestUnifiedConfigLoader_Validate(t *testing.T) {
	t.Run("validates valid config", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		cfg := loader.GetCurrent()

		// Act
		err = loader.Validate(cfg)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("error for nil config", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act
		err = loader.Validate(nil)

		// Assert
		assert.Error(t, err)
	})
}

func TestUnifiedConfigLoader_ObserverPattern(t *testing.T) {
	t.Run("notifies observers on config change", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := validTestConfigWithPort(8080)
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Create mock observer
		var notified bool
		var oldCfg, newCfg *configTypes.RootConfig
		mockObserver := &mockConfigObserver{
			name: "test-observer",
			onConfigChange: func(old, new *configTypes.RootConfig) error {
				notified = true
				oldCfg = old
				newCfg = new
				return nil
			},
		}

		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(mockObserver)

		// Update config file
		updatedContent := validTestConfigWithPort(9090)
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Act
		_, err = loader.Reload()

		// Assert
		assert.NoError(t, err)
		assert.True(t, notified, "Observer should be notified")
		assert.NotNil(t, oldCfg)
		assert.NotNil(t, newCfg)
		assert.Equal(t, 8080, oldCfg.Server.Port)
		assert.Equal(t, 9090, newCfg.Server.Port)
	})

	t.Run("multiple observers all notified", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := validTestConfigWithPort(8080)
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		var notified1, notified2, notified3 bool
		observer1 := &mockConfigObserver{
			name:           "observer-1",
			onConfigChange: func(old, new *configTypes.RootConfig) error { notified1 = true; return nil },
		}
		observer2 := &mockConfigObserver{
			name:           "observer-2",
			onConfigChange: func(old, new *configTypes.RootConfig) error { notified2 = true; return nil },
		}
		observer3 := &mockConfigObserver{
			name:           "observer-3",
			onConfigChange: func(old, new *configTypes.RootConfig) error { notified3 = true; return nil },
		}

		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(observer1)
		provider.Subscribe(observer2)
		provider.Subscribe(observer3)

		// Update config
		updatedContent := validTestConfigWithPort(9090)
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Act
		_, err = loader.Reload()

		// Assert
		assert.NoError(t, err)
		assert.True(t, notified1, "Observer 1 should be notified")
		assert.True(t, notified2, "Observer 2 should be notified")
		assert.True(t, notified3, "Observer 3 should be notified")
	})

	t.Run("unsubscribe removes observer", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := validTestConfigWithPort(8080)
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		var notified bool
		observer := &mockConfigObserver{
			name:           "test-observer",
			onConfigChange: func(old, new *configTypes.RootConfig) error { notified = true; return nil },
		}

		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(observer)
		provider.Unsubscribe(observer)

		// Update config
		updatedContent := validTestConfigWithPort(9090)
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Act
		_, err = loader.Reload()

		// Assert
		assert.NoError(t, err)
		assert.False(t, notified, "Unsubscribed observer should not be notified")
	})
}

func TestUnifiedConfigLoader_ThreadSafety(t *testing.T) {
	t.Run("concurrent GetCurrent calls are safe", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := validTestConfig()
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act - concurrent reads
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				cfg := loader.GetCurrent()
				assert.NotNil(t, cfg)
				done <- true
			}()
		}

		// Assert - all goroutines complete without panic
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// mockConfigObserver is a test double for ports.ConfigObserver
type mockConfigObserver struct {
	name           string
	onConfigChange func(old, new *configTypes.RootConfig) error
}

func (m *mockConfigObserver) OnConfigChange(old, new *configTypes.RootConfig) error {
	if m.onConfigChange != nil {
		return m.onConfigChange(old, new)
	}
	return nil
}

func (m *mockConfigObserver) Name() string {
	return m.name
}
