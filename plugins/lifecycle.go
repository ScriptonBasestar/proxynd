package plugins

import (
	"context"
	"fmt"
	"path"

	"github.com/gofiber/fiber/v2"
)

// Initialize iterates over registered plugins, triggering their Init method,
// mounting middleware, and registering routes.
func Initialize(app *fiber.App, ctx Context) error {
	for _, plugin := range All() {
		if initializer, ok := plugin.(Initializer); ok {
			if err := initializer.Init(ctx); err != nil {
				return fmt.Errorf("plugin %s init failed: %w", plugin.Name(), err)
			}
		}

		if middlewareProvider, ok := plugin.(MiddlewareProvider); ok {
			if middleware := middlewareProvider.Middleware(); middleware != nil {
				app.Use(middleware)
			}
		}

		if routesProvider, ok := plugin.(RoutesProvider); ok {
			basePath := defaultMountPath(plugin)
			group := app.Group(basePath)
			routesProvider.Routes(group)
		}
	}

	for _, plugin := range All() {
		if readyHandler, ok := plugin.(ReadyHandler); ok {
			if err := readyHandler.OnReady(ctx); err != nil {
				return fmt.Errorf("plugin %s ready hook failed: %w", plugin.Name(), err)
			}
		}
	}

	return nil
}

// Shutdown propagates the shutdown signal to all registered plugins.
func Shutdown(ctx context.Context) error {
	for _, plugin := range All() {
		if shutdownHandler, ok := plugin.(ShutdownHandler); ok {
			if err := shutdownHandler.OnShutdown(ctx); err != nil {
				return fmt.Errorf("plugin %s shutdown hook failed: %w", plugin.Name(), err)
			}
		}
	}
	return nil
}

func defaultMountPath(p Plugin) string {
	if mountProvider, ok := p.(MountPathProvider); ok {
		if mount := mountProvider.MountPath(); mount != "" {
			return mount
		}
	}
	return path.Join("/plugins", p.Name())
}
