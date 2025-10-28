package plugins

import (
	"fmt"
	"sync"
)

// Plugin exposes the minimal identity that every plugin must provide.
// Additional behaviour is discovered through the optional interfaces
// (Initializer, MiddlewareProvider, etc).
type Plugin interface {
	Name() string
}

// Registry is the in-memory list of all plugins discovered via the `Register` helper.
var (
	registryMu sync.RWMutex
	registry   []Plugin
)

// Register adds a plugin implementation to the in-memory registry.
// It panics when the implementation is nil or when another plugin with the same
// name has already been registered. Registration is typically done inside an init() function.
func Register(p Plugin) {
	if p == nil {
		panic("plugins: attempted to register a nil plugin")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	name := p.Name()
	for _, existing := range registry {
		if existing.Name() == name {
			panic(fmt.Sprintf("plugins: duplicate plugin registration for %q", name))
		}
	}

	registry = append(registry, p)
}

// All returns a copy of the currently registered plugins.
func All() []Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()

	plugins := make([]Plugin, len(registry))
	copy(plugins, registry)
	return plugins
}
