package plugins

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// Context provides host application information to plugins during lifecycle callbacks.
type Context struct {
	Logger      Logger
	ConfigDir   string
	DataDir     string
	Environment map[string]string
}

// Logger is a lightweight logging interface that plugins can rely on without
// coupling to the host's concrete logging implementation.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Initializer is implemented by plugins that require initialization.
type Initializer interface {
	Init(ctx Context) error
}

// ReadyHandler is invoked after the application has fully initialised.
type ReadyHandler interface {
	OnReady(ctx Context) error
}

// ShutdownHandler is invoked during graceful shutdown.
type ShutdownHandler interface {
	OnShutdown(ctx context.Context) error
}

// MiddlewareProvider supplies a Fiber middleware that should be mounted globally.
type MiddlewareProvider interface {
	Middleware() fiber.Handler
}

// RoutesProvider allows the plugin to mount its routes under a dedicated router group.
type RoutesProvider interface {
	Routes(router fiber.Router)
}

// MountPathProvider allows a plugin to override the base route group used when
// registering HTTP handlers. When not implemented, "/plugins/<plugin-name>" is used.
type MountPathProvider interface {
	MountPath() string
}
