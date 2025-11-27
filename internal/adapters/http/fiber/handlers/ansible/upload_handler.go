package ansible

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/observability/zap"
	"proxynd/internal/domain/ansible"
	"proxynd/internal/ports"
)

// UploadCollection handles collection upload (POST /api/v3/collections/)
//
// Galaxy v3 API Specification:
// - Method: POST
// - Content-Type: multipart/form-data
// - Form field: "file" (the tarball)
// - Response: CollectionUploadResponse (201 Created)
func (h *AnsibleHandler) UploadCollection(c *fiber.Ctx) error {
	ctx := c.Context()

	h.logger.Info(ctx, "Ansible collection upload request",
		zap.NewField("repository", h.config.Name),
		zap.NewField("remote_ip", c.IP()),
	)

	// 1. Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn(ctx, "Failed to parse form file",
			zap.NewField("error", err.Error()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Detail: "Missing or invalid 'file' field in multipart form",
		})
	}

	// 2. Validate file size
	if file.Size > h.config.MaxUploadSize {
		h.logger.Warn(ctx, "File size exceeds limit",
			zap.NewField("size", file.Size),
			zap.NewField("limit", h.config.MaxUploadSize),
		)
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(ErrorResponse{
			Detail: fmt.Sprintf("File size (%d bytes) exceeds maximum allowed size (%d bytes)",
				file.Size, h.config.MaxUploadSize),
		})
	}

	// 3. Open uploaded file
	fileContent, err := file.Open()
	if err != nil {
		h.logger.Error(ctx, "Failed to open uploaded file",
			zap.NewField("error", err.Error()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Detail: "Failed to process uploaded file",
		})
	}
	defer func() { _ = fileContent.Close() }()

	// 4. Read file content and calculate SHA256
	fileBytes, err := io.ReadAll(fileContent)
	if err != nil {
		h.logger.Error(ctx, "Failed to read uploaded file",
			zap.NewField("error", err.Error()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Detail: "Failed to read uploaded file",
		})
	}

	hash := sha256.Sum256(fileBytes)
	sha256Hash := fmt.Sprintf("%x", hash)

	h.logger.Info(ctx, "File uploaded successfully",
		zap.NewField("filename", file.Filename),
		zap.NewField("size", file.Size),
		zap.NewField("sha256", sha256Hash),
	)

	// 5. Get uploaded user from context (set by auth middleware)
	uploadedBy := "anonymous"
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		uploadedBy = userID
	}

	// 6. Call HostedDriver.StorePackage
	storeReq := &ports.StorePackageRequest{
		Content:    io.NopCloser(bytes.NewReader(fileBytes)),
		Size:       file.Size,
		SHA256:     sha256Hash,
		UploadedBy: uploadedBy,
	}

	storeResp, err := h.hostedDriver.StorePackage(ctx, storeReq)
	if err != nil {
		// Check for domain-specific errors
		switch err {
		case ansible.ErrVersionAlreadyExists:
			h.logger.Warn(ctx, "Collection version already exists",
				zap.NewField("error", err.Error()),
			)
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Detail: "This collection version already exists",
			})

		case ansible.ErrInvalidNamespace, ansible.ErrInvalidName:
			h.logger.Warn(ctx, "Invalid collection metadata",
				zap.NewField("error", err.Error()),
			)
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Detail: err.Error(),
			})

		case ansible.ErrManifestNotFound, ansible.ErrManifestInvalid:
			h.logger.Warn(ctx, "Invalid manifest",
				zap.NewField("error", err.Error()),
			)
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Detail: "Invalid or missing galaxy.yml in tarball",
			})

		case ansible.ErrTarballTooLarge:
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(ErrorResponse{
				Detail: err.Error(),
			})

		default:
			// Check if it's a namespace permission error (contains "not allowed")
			if contains(err.Error(), "not allowed") {
				h.logger.Warn(ctx, "Namespace permission denied",
					zap.NewField("error", err.Error()),
				)
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Detail: err.Error(),
				})
			}

			// Generic server error
			h.logger.Error(ctx, "Failed to store collection",
				zap.NewField("error", err.Error()),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: "Failed to store collection",
			})
		}
	}

	// 7. Parse namespace, name, version from package path
	// Package path format: "namespace-name-version.tar.gz"
	namespace, name, version, err := parsePackagePath(storeResp.PackagePath)
	if err != nil {
		h.logger.Error(ctx, "Failed to parse package path",
			zap.NewField("package_path", storeResp.PackagePath),
			zap.NewField("error", err.Error()),
		)
		// Fallback: still return success but with empty fields
		namespace, name, version = "", "", ""
	}

	// 8. Build response
	response := CollectionUploadResponse{
		Namespace:   namespace,
		Name:        name,
		Version:     version,
		SHA256:      storeResp.SHA256,
		UploadedAt:  storeResp.StoredAt,
		UploadedBy:  uploadedBy,
		PackagePath: storeResp.PackagePath,
	}

	h.logger.Info(ctx, "Collection uploaded successfully",
		zap.NewField("namespace", namespace),
		zap.NewField("name", name),
		zap.NewField("version", version),
		zap.NewField("sha256", storeResp.SHA256),
	)

	return c.Status(fiber.StatusCreated).JSON(response)
}

// parsePackagePath parses "namespace-name-version.tar.gz" into components
func parsePackagePath(packagePath string) (namespace, name, version string, err error) {
	// Remove .tar.gz suffix
	if len(packagePath) < 7 || packagePath[len(packagePath)-7:] != ".tar.gz" {
		return "", "", "", fmt.Errorf("invalid package path: %s", packagePath)
	}

	base := packagePath[:len(packagePath)-7] // Remove ".tar.gz"

	// Split by "-"
	parts := []string{}
	current := ""
	for i := 0; i < len(base); i++ {
		if base[i] == '-' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(base[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Expected format: namespace-name-version
	// But name might contain hyphens, so we need at least 3 parts
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid package path format: %s", packagePath)
	}

	namespace = parts[0]
	version = parts[len(parts)-1]
	name = ""
	for i := 1; i < len(parts)-1; i++ {
		if name != "" {
			name += "-"
		}
		name += parts[i]
	}

	return namespace, name, version, nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring finds a substring in a string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
