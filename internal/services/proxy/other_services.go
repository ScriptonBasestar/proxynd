package proxy

import (
	"context"
	"fmt"
)

// YumService handles YUM repository proxy requests
type YumService struct {
	*BaseProxyService
}

// NewYumService creates a new YUM proxy service
func NewYumService(cache CacheService, configService ConfigService,
	upstreamClient UpstreamClient,
) (*YumService, error) {
	base := NewBaseProxyService("yum", cache, configService, upstreamClient)
	return &YumService{BaseProxyService: base}, nil
}

// HandleRequest processes a YUM proxy request
func (s *YumService) HandleRequest(_ context.Context, _ ProxyRequest) (*ProxyResponse, error) {
	return s.HandleError(fmt.Errorf("YUM service not yet implemented"), 501), nil
}

// ApkService handles APK repository proxy requests
type ApkService struct {
	*BaseProxyService
}

// NewApkService creates a new APK proxy service
func NewApkService(cache CacheService, configService ConfigService,
	upstreamClient UpstreamClient,
) (*ApkService, error) {
	base := NewBaseProxyService("apk", cache, configService, upstreamClient)
	return &ApkService{BaseProxyService: base}, nil
}

// HandleRequest processes an APK proxy request
func (s *ApkService) HandleRequest(_ context.Context, _ ProxyRequest) (*ProxyResponse, error) {
	return s.HandleError(fmt.Errorf("APK service not yet implemented"), 501), nil
}
