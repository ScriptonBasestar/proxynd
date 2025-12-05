package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// fileWatcher implements ports.ConfigWatcher using fsnotify.
type fileWatcher struct {
	configPath string
	loader     ports.ConfigLoader
	watcher    *fsnotify.Watcher
	stopChan   chan struct{}
	running    bool
	mu         sync.RWMutex
	logger     logging.Logger
	debounce   time.Duration
}

// NewFileWatcher creates a new file watcher for configuration changes.
// configPath: path to the configuration file to watch
// loader: config loader to trigger reload when file changes
// debounce: debounce duration to avoid multiple reloads (default: 500ms)
func NewFileWatcher(configPath string, loader ports.ConfigLoader, debounce time.Duration) (ports.ConfigWatcher, error) {
	if debounce == 0 {
		debounce = 500 * time.Millisecond
	}

	return &fileWatcher{
		configPath: configPath,
		loader:     loader,
		stopChan:   make(chan struct{}),
		logger:     logging.GetLogger(),
		debounce:   debounce,
	}, nil
}

// Start begins watching the configuration file for changes.
func (fw *fileWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.running {
		return fmt.Errorf("file watcher is already running")
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add config file to watch list
	if err := watcher.Add(fw.configPath); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("failed to watch config file %s: %w", fw.configPath, err)
	}

	fw.watcher = watcher
	fw.running = true

	// Start watching in a goroutine
	go fw.watch()

	fw.logger.Info("File watcher started", logging.F("path", fw.configPath))

	return nil
}

// Stop stops watching for configuration changes.
func (fw *fileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		return nil // Already stopped
	}

	close(fw.stopChan)
	fw.running = false

	if fw.watcher != nil {
		if err := fw.watcher.Close(); err != nil {
			fw.logger.Error("Error closing file watcher", logging.F("error", err))
			return fmt.Errorf("failed to close file watcher: %w", err)
		}
		fw.watcher = nil
	}

	fw.logger.Info("File watcher stopped", logging.F("path", fw.configPath))

	return nil
}

// IsRunning returns true if the watcher is currently active.
func (fw *fileWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	return fw.running
}

// watch is the main watch loop that processes file system events.
func (fw *fileWatcher) watch() {
	var debounceTimer *time.Timer
	var debounceMu sync.Mutex

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return // Watcher closed
			}

			// Only react to Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				fw.logger.Debug("Config file changed",
					logging.F("file", event.Name),
					logging.F("op", event.Op.String()))

				// Debounce: wait for rapid successive changes to settle
				debounceMu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(fw.debounce, func() {
					fw.reloadConfig()
				})
				debounceMu.Unlock()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return // Watcher closed
			}
			fw.logger.Error("File watcher error", logging.F("error", err))

		case <-fw.stopChan:
			fw.logger.Debug("File watcher stop signal received")
			return
		}
	}
}

// reloadConfig triggers a config reload through the loader.
func (fw *fileWatcher) reloadConfig() {
	fw.logger.Info("Triggering config reload due to file change",
		logging.F("path", fw.configPath))

	if _, err := fw.loader.Reload(); err != nil {
		fw.logger.Error("Failed to reload config after file change",
			logging.F("path", fw.configPath),
			logging.F("error", err))
	} else {
		fw.logger.Info("Config reloaded successfully after file change",
			logging.F("path", fw.configPath))
	}
}

// Verify that fileWatcher implements the required interface at compile time
var _ ports.ConfigWatcher = (*fileWatcher)(nil)
