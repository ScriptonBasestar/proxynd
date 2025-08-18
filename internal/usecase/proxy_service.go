package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"proxynd/internal/ports"
)

// ProxyService implements core proxy business logic
type ProxyService struct {
	packageManager ports.PackageManager
	cacheManager   ports.CacheManager
	cacheStrategy  *CacheStrategyService
	authService    ports.AuthService
	logger         ports.Logger
	metrics        ports.MetricsCollector
	rateLimiter    ports.RateLimiter
}

// NewProxyService creates a new proxy service
func NewProxyService(
	pm ports.PackageManager,
	cache ports.CacheManager,
	cacheStrategy *CacheStrategyService,
	auth ports.AuthService,
	logger ports.Logger,
	metrics ports.MetricsCollector,
	rateLimiter ports.RateLimiter,
) *ProxyService {
	return &ProxyService{
		packageManager: pm,
		cacheManager:   cache,
		cacheStrategy:  cacheStrategy,
		authService:    auth,
		logger:         logger,
		metrics:        metrics,
		rateLimiter:    rateLimiter,
	}
}

// ProxyRequest represents a proxy request
type ProxyRequest struct {
	PackageType   string            `json:"package_type"`
	Repository    string            `json:"repository"`
	Path          string            `json:"path"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	QueryParams   map[string]string `json:"query_params"`
	Body          []byte            `json:"body"`
	UserID        string            `json:"user_id"`
	Authorization string            `json:"authorization"`
}

// ProxyResponse represents a proxy response
type ProxyResponse struct {
	Content     []byte            `json:"-"`
	ContentType string            `json:"content_type"`
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Cached      bool              `json:"cached"`
	CacheKey    string            `json:"cache_key"`
	Error       error             `json:"error,omitempty"`
}

// HandleProxyRequest processes a proxy request following the complete proxy flow:
// auth/permission → rate limit → cache lookup → remote fetch → signature verification → cache write → response
func (ps *ProxyService) HandleProxyRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	startTime := time.Now()

	// Step 1: Authentication and Authorization
	if err := ps.authenticateAndAuthorize(ctx, req); err != nil {
		ps.logger.Error(ctx, "Authentication/authorization failed",
			&LogField{key: "error", value: err},
			&LogField{key: "user_id", value: req.UserID})
		ps.metrics.IncCounter("proxy_auth_failures", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
		})
		return &ProxyResponse{
			StatusCode: 401,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Content:    []byte(`{"error": "authentication failed"}`),
			Error:      err,
		}, nil
	}

	// Step 2: Rate Limiting
	if err := ps.checkRateLimit(ctx, req); err != nil {
		ps.logger.Warn(ctx, "Rate limit exceeded",
			&LogField{key: "user_id", value: req.UserID},
			&LogField{key: "package_type", value: req.PackageType})
		ps.metrics.IncCounter("proxy_rate_limit_exceeded", map[string]string{
			"package_type": req.PackageType,
			"user_id":      req.UserID,
		})
		return &ProxyResponse{
			StatusCode: 429,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Content:    []byte(`{"error": "rate limit exceeded"}`),
			Error:      err,
		}, nil
	}

	// Step 3: Cache Lookup
	cacheReq := &CacheRequest{
		PackageType: req.PackageType,
		Repository:  req.Repository,
		Path:        req.Path,
		Headers:     req.Headers,
		Operation:   "get",
	}

	cacheResp, err := ps.cacheStrategy.Get(ctx, cacheReq)
	if err != nil {
		ps.logger.Warn(ctx, "Cache lookup failed",
			&LogField{key: "error", value: err},
			&LogField{key: "cache_key", value: cacheReq.Path})
	} else if cacheResp.Hit {
		// Cache hit - return cached content
		ps.logger.Debug(ctx, "Cache hit",
			&LogField{key: "cache_key", value: cacheResp.CacheKey},
			&LogField{key: "backend", value: cacheResp.Backend})
		ps.metrics.IncCounter("proxy_cache_hits", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
			"backend":      cacheResp.Backend,
		})
		ps.recordLatency(ctx, "cache_hit", startTime, req.PackageType)

		return &ProxyResponse{
			Content:     cacheResp.Content,
			ContentType: cacheResp.Metadata["content_type"],
			StatusCode:  200,
			Headers:     cacheResp.Metadata,
			Cached:      true,
			CacheKey:    cacheResp.CacheKey,
		}, nil
	}

	// Cache miss - continue to upstream
	ps.metrics.IncCounter("proxy_cache_misses", map[string]string{
		"package_type": req.PackageType,
		"repository":   req.Repository,
	})

	// Step 4: Remote Fetch from Upstream
	upstreamResp, err := ps.fetchFromUpstream(ctx, req)
	if err != nil {
		ps.logger.Error(ctx, "Upstream fetch failed",
			&LogField{key: "error", value: err},
			&LogField{key: "repository", value: req.Repository},
			&LogField{key: "path", value: req.Path})
		ps.metrics.IncCounter("proxy_upstream_failures", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
		})
		return &ProxyResponse{
			StatusCode: 502,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Content:    []byte(`{"error": "upstream fetch failed"}`),
			Error:      err,
		}, nil
	}

	// Step 5: Signature Verification (if enabled)
	// TODO: Add signature verification when SignatureService interface is available

	// Step 6: Cache Write (asynchronous for performance)
	go ps.cacheUpstreamResponse(context.Background(), cacheReq, upstreamResp)

	// Step 7: Record Metrics and Return Response
	ps.recordLatency(ctx, "upstream_fetch", startTime, req.PackageType)
	ps.metrics.IncCounter("proxy_requests_total", map[string]string{
		"package_type": req.PackageType,
		"repository":   req.Repository,
		"method":       req.Method,
		"status":       fmt.Sprintf("%d", upstreamResp.StatusCode),
	})

	ps.logger.Info(ctx, "Proxy request completed",
		&LogField{key: "package_type", value: req.PackageType},
		&LogField{key: "repository", value: req.Repository},
		&LogField{key: "path", value: req.Path},
		&LogField{key: "status", value: upstreamResp.StatusCode},
		&LogField{key: "duration_ms", value: time.Since(startTime).Milliseconds()},
		&LogField{key: "cached", value: false})

	return upstreamResp, nil
}

// GetPackage retrieves a package through proxy
// TODO: Migrate from handlers/proxy/ specific handlers
func (ps *ProxyService) GetPackage(ctx context.Context, req *GetPackageRequest) (*GetPackageResponse, error) {
	// TODO: Implement package retrieval logic
	return &GetPackageResponse{}, nil
}

// GetPackageRequest represents package retrieval request
type GetPackageRequest struct {
	PackageType string `json:"package_type"`
	Repository  string `json:"repository"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	UserID      string `json:"user_id"`
}

// GetPackageResponse represents package retrieval response
type GetPackageResponse struct {
	Content     []byte            `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Headers     map[string]string `json:"headers"`
	Cached      bool              `json:"cached"`
	Metadata    map[string]string `json:"metadata"`
}

// ListPackages lists packages in repository
// TODO: Implement package listing logic
func (ps *ProxyService) ListPackages(ctx context.Context, req *ListPackagesRequest) (*ListPackagesResponse, error) {
	// TODO: Implement package listing logic
	return &ListPackagesResponse{}, nil
}

// ListPackagesRequest represents package listing request
type ListPackagesRequest struct {
	PackageType string `json:"package_type"`
	Repository  string `json:"repository"`
	Search      string `json:"search,omitempty"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	UserID      string `json:"user_id"`
}

// ListPackagesResponse represents package listing response
type ListPackagesResponse struct {
	Packages []PackageInfo `json:"packages"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// PackageInfo represents package information
type PackageInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata"`
}

// HandleUpload handles package upload
// TODO: Implement package upload logic
func (ps *ProxyService) HandleUpload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	// TODO: Implement upload logic
	return &UploadResponse{}, nil
}

// UploadRequest represents package upload request
type UploadRequest struct {
	PackageType string            `json:"package_type"`
	Repository  string            `json:"repository"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Content     []byte            `json:"-"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	UserID      string            `json:"user_id"`
}

// UploadResponse represents package upload response
type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id"`
}

// GetMetadata retrieves package metadata
// TODO: Implement metadata retrieval logic
func (ps *ProxyService) GetMetadata(ctx context.Context, req *GetMetadataRequest) (*GetMetadataResponse, error) {
	// TODO: Implement metadata retrieval logic
	return &GetMetadataResponse{}, nil
}

// GetMetadataRequest represents metadata request
type GetMetadataRequest struct {
	PackageType string `json:"package_type"`
	Repository  string `json:"repository"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	UserID      string `json:"user_id"`
}

// GetMetadataResponse represents metadata response
type GetMetadataResponse struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Dependencies []string          `json:"dependencies"`
	Metadata     map[string]string `json:"metadata"`
	Size         int64             `json:"size"`
	Checksum     string            `json:"checksum"`
}

// ValidateRequest validates proxy request
func (ps *ProxyService) ValidateRequest(ctx context.Context, req *ProxyRequest) error {
	if req.PackageType == "" {
		return fmt.Errorf("package_type is required")
	}
	if req.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	if req.Method == "" {
		req.Method = "GET" // Default to GET
	}
	return nil
}

// AuthorizeRequest checks if request is authorized (deprecated - use HandleProxyRequest)
func (ps *ProxyService) AuthorizeRequest(ctx context.Context, req *ProxyRequest) error {
	return ps.authenticateAndAuthorize(ctx, req)
}

// RecordMetrics records proxy metrics
func (ps *ProxyService) RecordMetrics(ctx context.Context, req *ProxyRequest, resp *ProxyResponse) {
	ps.metrics.IncCounter("proxy_requests_total", map[string]string{
		"package_type": req.PackageType,
		"repository":   req.Repository,
		"method":       req.Method,
		"status":       fmt.Sprintf("%d", resp.StatusCode),
		"cached":       fmt.Sprintf("%t", resp.Cached),
	})

	if resp.Cached {
		ps.metrics.IncCounter("proxy_cache_hits", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
		})
	} else {
		ps.metrics.IncCounter("proxy_cache_misses", map[string]string{
			"package_type": req.PackageType,
			"repository":   req.Repository,
		})
	}
}

// authenticateAndAuthorize handles authentication and authorization
func (ps *ProxyService) authenticateAndAuthorize(ctx context.Context, req *ProxyRequest) error {
	// Authentication
	if req.Authorization == "" && req.UserID == "" {
		return fmt.Errorf("no authentication provided")
	}

	// Use auth service to validate
	if req.Authorization != "" {
		authReq := &ports.AuthRequest{
			Username: "", // Token-based auth doesn't use username/password
			Password: "",
			Provider: "jwt",
		}

		authResp, err := ps.authService.Authenticate(ctx, authReq)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		req.UserID = authResp.User.ID
	}

	// Authorization check
	authzResp, err := ps.authService.Authorize(ctx, &ports.AuthorizeRequest{
		UserID:   req.UserID,
		Resource: fmt.Sprintf("%s/%s%s", req.PackageType, req.Repository, req.Path),
		Action:   req.Method,
	})
	if err != nil {
		return err
	}
	if !authzResp.Allowed {
		return fmt.Errorf("access denied: %s", authzResp.Reason)
	}
	return nil
}

// checkRateLimit validates rate limiting
func (ps *ProxyService) checkRateLimit(ctx context.Context, req *ProxyRequest) error {
	if ps.rateLimiter == nil {
		return nil // No rate limiting configured
	}

	// Create a rate limit key based on user and resource
	rateLimitKey := fmt.Sprintf("%s:%s:%s", req.UserID, req.PackageType, req.Repository)

	allowed, err := ps.rateLimiter.Allow(ctx, rateLimitKey)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("rate limit exceeded for user %s", req.UserID)
	}

	return nil
}

// fetchFromUpstream retrieves content from upstream package manager
func (ps *ProxyService) fetchFromUpstream(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	pkgReq := &ports.PackageRequest{
		Repository:  req.Repository,
		Name:        "", // Extract from path if needed
		Version:     "", // Extract from path if needed
		Path:        req.Path,
		Headers:     req.Headers,
		QueryParams: req.QueryParams,
	}

	pkgResp, err := ps.packageManager.GetPackage(ctx, pkgReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	// Read content from ReadCloser
	content, err := io.ReadAll(pkgResp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read package content: %w", err)
	}
	defer func() {
		if closeErr := pkgResp.Content.Close(); closeErr != nil {
			// Log error but don't override main error
			_ = closeErr // Placeholder to avoid empty branch warning
		}
	}()

	return &ProxyResponse{
		Content:     content,
		ContentType: pkgResp.ContentType,
		StatusCode:  200, // PackageResponse doesn't have StatusCode
		Headers:     pkgResp.Headers,
		Cached:      false,
	}, nil
}

// cacheUpstreamResponse stores upstream response in cache asynchronously
func (ps *ProxyService) cacheUpstreamResponse(
	ctx context.Context, cacheReq *CacheRequest, upstreamResp *ProxyResponse,
) {
	// Determine cache strategy
	decision, err := ps.cacheStrategy.DetermineStrategy(ctx, cacheReq)
	if err != nil {
		ps.logger.Error(ctx, "Failed to determine cache strategy",
			&LogField{key: "error", value: err})
		return
	}

	if !decision.ShouldCache {
		ps.logger.Debug(ctx, "Cache strategy determined not to cache",
			&LogField{key: "reason", value: decision.Reason})
		return
	}

	// Prepare cache storage request
	cacheSetReq := &CacheRequest{
		PackageType: cacheReq.PackageType,
		Repository:  cacheReq.Repository,
		Path:        cacheReq.Path,
		ContentType: upstreamResp.ContentType,
		Size:        int64(len(upstreamResp.Content)),
		Headers:     upstreamResp.Headers,
		Operation:   "set",
	}

	// Store in cache
	if err := ps.cacheStrategy.Set(ctx, cacheSetReq, upstreamResp.Content); err != nil {
		ps.logger.Error(ctx, "Failed to cache upstream response",
			&LogField{key: "error", value: err},
			&LogField{key: "cache_key", value: decision.CacheKey})
		ps.metrics.IncCounter("proxy_cache_write_failures", map[string]string{
			"package_type": cacheReq.PackageType,
			"repository":   cacheReq.Repository,
			"backend":      decision.Backend,
		})
	} else {
		ps.logger.Debug(ctx, "Successfully cached upstream response",
			&LogField{key: "cache_key", value: decision.CacheKey},
			&LogField{key: "backend", value: decision.Backend})
		ps.metrics.IncCounter("proxy_cache_writes", map[string]string{
			"package_type": cacheReq.PackageType,
			"repository":   cacheReq.Repository,
			"backend":      decision.Backend,
		})
	}
}

// recordLatency records latency metrics for different operations
func (ps *ProxyService) recordLatency(ctx context.Context, operation string, startTime time.Time, packageType string) {
	duration := time.Since(startTime)
	ps.metrics.ObserveHistogram("proxy_operation_duration", float64(duration.Milliseconds()), map[string]string{
		"operation":    operation,
		"package_type": packageType,
	})
}

// GetProxyStats returns proxy statistics
// TODO: Implement statistics gathering
func (ps *ProxyService) GetProxyStats(ctx context.Context) (*ProxyStats, error) {
	// TODO: Implement statistics gathering
	return &ProxyStats{}, nil
}

// ProxyStats represents proxy statistics
type ProxyStats struct {
	TotalRequests   int64            `json:"total_requests"`
	CacheHitRate    float64          `json:"cache_hit_rate"`
	ErrorRate       float64          `json:"error_rate"`
	AvgResponseTime int64            `json:"avg_response_time_ms"`
	PackageTypes    map[string]int64 `json:"package_types"`
	Repositories    map[string]int64 `json:"repositories"`
}
