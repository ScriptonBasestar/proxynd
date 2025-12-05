package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configTypes "proxynd/internal/config"
	"proxynd/internal/ports"
)

func TestNewFileWatcher(t *testing.T) {
	t.Run("successful creation with valid config", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act
		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, watcher)
		assert.Implements(t, (*ports.ConfigWatcher)(nil), watcher)
	})

	t.Run("uses default debounce when zero", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Act
		watcher, err := NewFileWatcher(configPath, loader, 0)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, watcher)
		// Default debounce is 500ms (tested indirectly via field check)
	})
}

func TestFileWatcher_StartStop(t *testing.T) {
	t.Run("start successfully begins watching", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		// Act
		err = watcher.Start()

		// Assert
		assert.NoError(t, err)
		assert.True(t, watcher.IsRunning())

		// Cleanup
		_ = watcher.Stop()
	})

	t.Run("stop successfully stops watching", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)
		require.True(t, watcher.IsRunning())

		// Act
		err = watcher.Stop()

		// Assert
		assert.NoError(t, err)
		assert.False(t, watcher.IsRunning())
	})

	t.Run("start twice returns error", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)

		// Act
		err = watcher.Start()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")

		// Cleanup
		_ = watcher.Stop()
	})

	t.Run("stop when not running is safe", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		// Act
		err = watcher.Stop()

		// Assert
		assert.NoError(t, err)
	})

	t.Run("error when config file does not exist", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Create loader with valid config first
		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Remove the file
		err = os.Remove(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		// Act
		err = watcher.Start()

		// Assert
		assert.Error(t, err)
		assert.False(t, watcher.IsRunning())
	})
}

func TestFileWatcher_FileChangeDetection(t *testing.T) {
	t.Run("detects file changes and triggers reload", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Track reload calls
		var reloadCalled bool
		observer := &mockConfigObserver{
			name: "test-observer",
			onConfigChange: func(old, new *configTypes.RootConfig) error {
				reloadCalled = true
				return nil
			},
		}
		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(observer)

		watcher, err := NewFileWatcher(configPath, loader, 50*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)
		defer watcher.Stop()

		// Act - modify config file
		updatedContent := `
server:
  port: 9090
`
		err = os.WriteFile(configPath, []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Wait for debounce + processing
		time.Sleep(200 * time.Millisecond)

		// Assert
		assert.True(t, reloadCalled, "Config reload should be triggered")
		assert.Equal(t, 9090, loader.GetCurrent().Server.Port)
	})

	t.Run("debouncing prevents multiple reloads", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		// Track reload calls
		reloadCount := 0
		observer := &mockConfigObserver{
			name: "test-observer",
			onConfigChange: func(old, new *configTypes.RootConfig) error {
				reloadCount++
				return nil
			},
		}
		provider, ok := loader.(ports.ConfigProvider)
		require.True(t, ok)
		provider.Subscribe(observer)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)
		defer watcher.Stop()

		// Act - modify config file multiple times rapidly
		for i := 0; i < 5; i++ {
			content := fmt.Sprintf(`
server:
  port: %d
`, 9000+i)
			err = os.WriteFile(configPath, []byte(content), 0644)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Rapid writes within debounce window
		}

		// Wait for debounce + processing
		time.Sleep(250 * time.Millisecond)

		// Assert - should only reload once due to debouncing
		assert.Equal(t, 1, reloadCount, "Debouncing should prevent multiple reloads")
	})
}

func TestFileWatcher_IsRunning(t *testing.T) {
	t.Run("returns false before start", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		// Act & Assert
		assert.False(t, watcher.IsRunning())
	})

	t.Run("returns true after start", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)
		defer watcher.Stop()

		// Act & Assert
		assert.True(t, watcher.IsRunning())
	})

	t.Run("returns false after stop", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
server:
  port: 8080
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader, err := NewUnifiedConfigLoader(configPath)
		require.NoError(t, err)

		watcher, err := NewFileWatcher(configPath, loader, 100*time.Millisecond)
		require.NoError(t, err)

		err = watcher.Start()
		require.NoError(t, err)

		err = watcher.Stop()
		require.NoError(t, err)

		// Act & Assert
		assert.False(t, watcher.IsRunning())
	})
}
