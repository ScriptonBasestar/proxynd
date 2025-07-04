package handlers

import (
	"context"
	"fmt"
	"io"

	"proxynd/internal/domain/maven"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/pkg/types"
)

// MavenHandler implements the unified ProxyHandler interface for Maven
type MavenHandler struct {
	service proxy.ProxyService
	domain  *maven.Domain
	logger  logging.Logger
}

// NewMavenHandler creates a new Maven proxy handler
func NewMavenHandler(service proxy.ProxyService) *MavenHandler {
	return &MavenHandler{
		service: service,
		domain:  maven.NewDomain(),
		logger:  logging.GetLogger(),
	}
}

// Handle processes a Maven proxy request
func (h *MavenHandler) Handle(ctx context.Context, req *types.ProxyRequest) (*types.ProxyResponse, error) {
	// Convert to service request
	serviceReq := proxy.ProxyRequest{
		Path:       req.Path,
		ProxyType:  string(req.Type),
		Method:     req.Method,
		Headers:    req.Headers,
		RemoteAddr: req.RemoteAddr,
	}

	// Handle through service
	serviceResp, err := h.service.HandleRequest(ctx, serviceReq)
	if err != nil {
		return nil, types.NewProxyError(
			types.ProxyTypeMaven,
			500,
			"Failed to handle Maven request",
			err,
		)
	}

	// Convert to unified response
	return &types.ProxyResponse{
		StatusCode:  serviceResp.StatusCode,
		Headers:     serviceResp.Headers,
		Body:        serviceResp.Body,
		ContentType: serviceResp.ContentType,
		Cached:      serviceResp.Cached,
		CacheKey:    h.buildCacheKey(req.Path),
	}, nil
}

// GetType returns the proxy type this handler serves
func (h *MavenHandler) GetType() types.ProxyType {
	return types.ProxyTypeMaven
}

// ValidateRequest validates if the request is valid for Maven
func (h *MavenHandler) ValidateRequest(req *types.ProxyRequest) error {
	// Use domain logic for validation
	return h.domain.ValidatePath(req.Path)
}

// IsHealthy checks if the Maven proxy handler is healthy
func (h *MavenHandler) IsHealthy(ctx context.Context) bool {
	// Could check service health, upstream connectivity, etc.
	// For now, return true if service exists
	return h.service != nil
}

// buildCacheKey builds a cache key for the request
func (h *MavenHandler) buildCacheKey(path string) string {
	return fmt.Sprintf("maven:%s", path)
}

// Ensure MavenHandler implements ProxyHandler
var _ types.ProxyHandler = (*MavenHandler)(nil)

// MavenHandlerCreator creates Maven handlers for the factory
func MavenHandlerCreator(serviceFactory *proxy.ServiceFactory) func() types.ProxyHandler {
	return func() types.ProxyHandler {
		service, err := serviceFactory.GetService("maven")
		if err != nil {
			// Log error and return nil
			logging.GetLogger().Error("Failed to create Maven service", logging.F("error", err))
			return nil
		}
		return NewMavenHandler(service)
	}
}