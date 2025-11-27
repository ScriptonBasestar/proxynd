package ansible

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/ports"
)

// DownloadCollection handles collection download
// (GET /api/v3/collections/{namespace}/{name}/versions/{version}/)
//
// Galaxy v3 API Specification:
// - Method: GET
// - URL params: namespace, name, version
// - Response: Binary tarball with Content-Type: application/gzip
// - Headers: Content-Disposition, ETag (SHA256), Cache-Control
func (h *AnsibleHandler) DownloadCollection(c *fiber.Ctx) error {
	ctx := c.Context()

	namespace := c.Params("namespace")
	name := c.Params("name")
	version := c.Params("version")

	h.logger.Info(ctx, "Ansible collection download request",
		zap.NewField("repository", h.config.Name),
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", version),
		zap.NewField("remote_ip", c.IP()),
	)

	// Validate parameters
	if namespace == "" || name == "" || version == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Detail: "Missing required parameters: namespace, name, or version",
		})
	}

	// Call HostedDriver.GetPackage
	req := &ports.GetPackageRequest{
		Name:    fmt.Sprintf("%s.%s", namespace, name),
		Version: version,
	}

	resp, err := h.hostedDriver.GetPackage(ctx, req)
	if err != nil {
		h.logger.Warn(ctx, "Collection not found",
			zap.NewField("namespace", namespace),
			zap.NewField("name", name),
			zap.NewField("version", version),
			zap.NewField("error", err.Error()),
		)
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Detail: fmt.Sprintf("Collection %s.%s version %s not found",
				namespace, name, version),
		})
	}
	defer func() { _ = resp.Content.Close() }()

	// Set response headers
	c.Set("Content-Type", "application/gzip")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-%s.tar.gz",
		namespace, name, version))

	// Set ETag with SHA256 for client-side caching
	if resp.SHA256 != "" {
		c.Set("ETag", fmt.Sprintf("\"%s\"", resp.SHA256))
	}

	// Set cache headers (collections are immutable)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")

	// Set content length if available
	if resp.Size > 0 {
		c.Set("Content-Length", fmt.Sprintf("%d", resp.Size))
	}

	h.logger.Info(ctx, "Streaming collection download",
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", version),
		zap.NewField("size", resp.Size),
		zap.NewField("sha256", resp.SHA256),
	)

	// Stream the content to the client
	c.Status(fiber.StatusOK)

	// Use io.Copy to stream the content
	_, err = io.Copy(c.Response().BodyWriter(), resp.Content)
	if err != nil {
		h.logger.Error(ctx, "Failed to stream collection content",
			zap.NewField("error", err.Error()),
		)
		// Can't send error response here as headers are already sent
		return err
	}

	h.logger.Info(ctx, "Collection download completed",
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", version),
	)

	return nil
}
