package config

import (
	"context"
	"fmt"
	"log"
	"sync"

	internalconfig "proxynd/internal/config"
)

// ViperConfigService Viper 기반 설정 서비스 구현
type ViperConfigService struct {
	mu            sync.RWMutex
	loader        *internalconfig.ViperConfigLoader
	unifiedConfig *internalconfig.UnifiedConfig
	globalConfig  *internalconfig.GlobalConfig

	// 개별 프록시 설정 캐시
	aptConfig    *internalconfig.AptProxyConfig
	mavenConfig  *internalconfig.MavenProxySettings
	npmConfig    *internalconfig.NpmProxySettings
	pipConfig    *internalconfig.PipProxySettings
	yumConfig    *internalconfig.YumProxySettings
	apkConfig    *internalconfig.ApkProxySettings
	dockerConfig *internalconfig.DockerProxySettings
}

// NewViperConfigService 새 Viper 기반 설정 서비스 생성
func NewViperConfigService(configPath string) (*ViperConfigService, error) {
	loader := internalconfig.NewViperConfigLoader()
	loader.SetConfigPath(configPath)

	// 통합 설정 로드
	unifiedConfig, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	service := &ViperConfigService{
		loader:        loader,
		unifiedConfig: unifiedConfig,
	}

	// 개별 설정 파일도 로드 (레거시 지원)
	if err := service.loadLegacyConfigs(); err != nil {
		// 레거시 설정 로드 실패는 경고로 처리
		// 통합 설정이 있으면 계속 진행
		fmt.Printf("Warning: failed to load legacy configs: %v\n", err)
	}

	return service, nil
}

// GetGlobalConfig 전역 설정 반환
func (s *ViperConfigService) GetGlobalConfig(_ context.Context) (*internalconfig.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.globalConfig != nil {
		return s.globalConfig, nil
	}

	// 통합 설정에서 전역 설정 추출
	return s.extractGlobalConfig(), nil
}

// GetMavenConfig Maven 프록시 설정 반환
func (s *ViperConfigService) GetMavenConfig(_ context.Context) (*internalconfig.MavenProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.mavenConfig != nil {
		return s.mavenConfig, nil
	}

	// 통합 설정에서 Maven 설정 추출
	return s.extractMavenConfig(), nil
}

// GetAptConfig APT 프록시 설정 반환
func (s *ViperConfigService) GetAptConfig(_ context.Context) (*internalconfig.AptProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.aptConfig != nil {
		return s.aptConfig, nil
	}

	// 통합 설정에서 APT 설정 추출
	return s.extractAptConfig(), nil
}

// GetNpmConfig NPM 프록시 설정 반환
func (s *ViperConfigService) GetNpmConfig(_ context.Context) (*internalconfig.NpmProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.npmConfig != nil {
		return s.npmConfig, nil
	}

	// 통합 설정에서 NPM 설정 추출
	return s.extractNpmConfig(), nil
}

// GetDockerConfig Docker 프록시 설정 반환
func (s *ViperConfigService) GetDockerConfig(_ context.Context) (*internalconfig.DockerProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dockerConfig != nil {
		return s.dockerConfig, nil
	}

	// 통합 설정에서 Docker 설정 추출
	return s.extractDockerConfig(), nil
}

// GetPipConfig PIP 프록시 설정 반환
func (s *ViperConfigService) GetPipConfig(_ context.Context) (*internalconfig.PipProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.pipConfig != nil {
		return s.pipConfig, nil
	}

	// 통합 설정에서 PIP 설정 추출
	return s.extractPipConfig(), nil
}

// GetYumConfig YUM 프록시 설정 반환
func (s *ViperConfigService) GetYumConfig(_ context.Context) (*internalconfig.YumProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.yumConfig != nil {
		return s.yumConfig, nil
	}

	// 통합 설정에서 YUM 설정 추출
	return s.extractYumConfig(), nil
}

// GetApkConfig APK 프록시 설정 반환
func (s *ViperConfigService) GetApkConfig(_ context.Context) (*internalconfig.ApkProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.apkConfig != nil {
		return s.apkConfig, nil
	}

	// 통합 설정에서 APK 설정 추출
	return s.extractApkConfig(), nil
}

// ValidateAll 모든 설정 검증
func (s *ViperConfigService) ValidateAll(_ context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 통합 설정 검증
	if s.unifiedConfig != nil {
		if err := s.unifiedConfig.Validate(); err != nil {
			return fmt.Errorf("unified config validation failed: %w", err)
		}
	}

	// 개별 설정 검증
	validators := []struct {
		name   string
		config interface{ Validate() error }
	}{
		{"global", s.globalConfig},
		{"apt", s.aptConfig},
		{"maven", s.mavenConfig},
		{"npm", s.npmConfig},
		{"pip", s.pipConfig},
		{"yum", s.yumConfig},
		{"apk", s.apkConfig},
		{"docker", s.dockerConfig},
	}

	for _, v := range validators {
		if v.config != nil {
			if err := v.config.Validate(); err != nil {
				return fmt.Errorf("%s config validation failed: %w", v.name, err)
			}
		}
	}

	return nil
}

// Reload 설정 재로드
func (s *ViperConfigService) Reload(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 통합 설정 재로드
	unifiedConfig, err := s.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	s.unifiedConfig = unifiedConfig

	// 레거시 설정 재로드
	if err := s.loadLegacyConfigs(); err != nil {
		log.Printf("Warning: Failed to load legacy configs: %v", err)
	}

	return nil
}

// GetUnifiedConfig 통합 설정 반환
func (s *ViperConfigService) GetUnifiedConfig() *internalconfig.UnifiedConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unifiedConfig
}

// GetString 문자열 설정값 조회
func (s *ViperConfigService) GetString(key string) string {
	return s.loader.GetString(key)
}

// GetInt 정수 설정값 조회
func (s *ViperConfigService) GetInt(key string) int {
	return s.loader.GetInt(key)
}

// GetBool 불린 설정값 조회
func (s *ViperConfigService) GetBool(key string) bool {
	return s.loader.GetBool(key)
}

// WatchConfig 설정 변경 감시
func (s *ViperConfigService) WatchConfig(callback func()) {
	s.loader.WatchConfig(func(config *internalconfig.UnifiedConfig) {
		s.mu.Lock()
		s.unifiedConfig = config
		s.mu.Unlock()

		callback()
	})
}

// loadLegacyConfigs 레거시 개별 설정 파일 로드
func (s *ViperConfigService) loadLegacyConfigs() error {
	// 전역 설정
	globalConfig := &internalconfig.GlobalConfig{}
	if globalConfig.ConfigExists() {
		if err := globalConfig.ReadConfig(); err == nil {
			s.globalConfig = globalConfig
		}
	}

	// APT 설정
	aptConfig := &internalconfig.AptProxyConfig{}
	if aptConfig.ConfigExists() {
		if err := aptConfig.ReadConfig(); err == nil {
			s.aptConfig = aptConfig
		}
	}

	// Maven 설정
	mavenConfig := &internalconfig.MavenProxySettings{}
	if mavenConfig.ConfigExists() {
		if err := mavenConfig.ReadConfig(); err == nil {
			s.mavenConfig = mavenConfig
		}
	}

	// NPM 설정
	npmConfig := &internalconfig.NpmProxySettings{}
	if npmConfig.ConfigExists() {
		if err := npmConfig.ReadConfig(); err == nil {
			s.npmConfig = npmConfig
		}
	}

	// PIP 설정
	pipConfig := &internalconfig.PipProxySettings{}
	if pipConfig.ConfigExists() {
		if err := pipConfig.ReadConfig(); err == nil {
			s.pipConfig = pipConfig
		}
	}

	// YUM 설정
	yumConfig := &internalconfig.YumProxySettings{}
	if yumConfig.ConfigExists() {
		if err := yumConfig.ReadConfig(); err == nil {
			s.yumConfig = yumConfig
		}
	}

	// APK 설정
	apkConfig := &internalconfig.ApkProxySettings{}
	if apkConfig.ConfigExists() {
		if err := apkConfig.ReadConfig(); err == nil {
			s.apkConfig = apkConfig
		}
	}

	// Docker 설정
	dockerConfig := &internalconfig.DockerProxySettings{}
	if dockerConfig.ConfigExists() {
		if err := dockerConfig.ReadConfig(); err == nil {
			s.dockerConfig = dockerConfig
		}
	}

	return nil
}

// extractGlobalConfig 통합 설정에서 전역 설정 추출
func (s *ViperConfigService) extractGlobalConfig() *internalconfig.GlobalConfig {
	if s.unifiedConfig == nil {
		return nil
	}

	return &internalconfig.GlobalConfig{
		StorageDir: s.unifiedConfig.Cache.File.Directory,
		ConfigDir:  s.loader.GetString("config_dir"),
		Cache: internalconfig.Cache{
			TTL: int(s.unifiedConfig.Cache.TTL.Seconds()),
		},
	}
}

// extractAptConfig 통합 설정에서 APT 설정 추출
func (s *ViperConfigService) extractAptConfig() *internalconfig.AptProxyConfig {
	if s.unifiedConfig == nil || !s.unifiedConfig.Registries.APT.Enabled {
		return nil
	}

	config := &internalconfig.AptProxyConfig{
		Path:     "/apt",
		UseCache: s.unifiedConfig.Registries.APT.UserCache,
		Proxies:  make(map[string][]internalconfig.AptProxy),
	}

	// 미러 설정을 프록시로 변환
	for distribution, mirrors := range s.unifiedConfig.Registries.APT.Mirrors {
		proxyList := make([]internalconfig.AptProxy, 0, len(mirrors))
		for _, mirror := range mirrors {
			proxyList = append(proxyList, internalconfig.AptProxy{
				Name: mirror.Name,
				URL:  mirror.URL,
			})
		}
		config.Proxies[distribution] = proxyList
	}

	return config
}

// extractMavenConfig 통합 설정에서 Maven 설정 추출
func (s *ViperConfigService) extractMavenConfig() *internalconfig.MavenProxySettings {
	if s.unifiedConfig == nil || !s.unifiedConfig.Registries.Maven.Enabled {
		return nil
	}

	config := &internalconfig.MavenProxySettings{
		Path:     "/maven",
		UseCache: true,
		Proxies:  []internalconfig.MavenProxyServer{},
	}

	// Maven 레지스트리 설정을 프록시 서버로 변환
	for _, repo := range s.unifiedConfig.Registries.Maven.Repositories {
		config.Proxies = append(config.Proxies, internalconfig.MavenProxyServer{
			Name: repo.Name,
			URL:  repo.URL,
		})
	}

	return config
}

// extractNpmConfig 통합 설정에서 NPM 설정 추출
func (s *ViperConfigService) extractNpmConfig() *internalconfig.NpmProxySettings {
	if s.unifiedConfig == nil || !s.unifiedConfig.Registries.NPM.Enabled {
		return nil
	}

	return &internalconfig.NpmProxySettings{
		Path:     "/npm",
		UseCache: s.unifiedConfig.Registries.NPM.UserCache,
		Proxies: map[string][]internalconfig.NpmProxyServer{
			"default": {
				{
					Name: "official",
					URL:  s.unifiedConfig.Registries.NPM.Upstream,
				},
			},
		},
	}
}

// extractPipConfig 통합 설정에서 PIP 설정 추출
func (s *ViperConfigService) extractPipConfig() *internalconfig.PipProxySettings {
	if s.unifiedConfig == nil || !s.unifiedConfig.Registries.PyPI.Enabled {
		return nil
	}

	return &internalconfig.PipProxySettings{
		Path:     "/pypi",
		UseCache: s.unifiedConfig.Registries.PyPI.UserCache,
		Proxies: []internalconfig.PipProxyServer{
			{
				Name: "pypi",
				URL:  s.unifiedConfig.Registries.PyPI.Upstream,
			},
		},
	}
}

// extractYumConfig 통합 설정에서 YUM 설정 추출
func (s *ViperConfigService) extractYumConfig() *internalconfig.YumProxySettings {
	// 기본 YUM 설정 반환
	return &internalconfig.YumProxySettings{
		Path:     "/yum",
		UseCache: true,
		Proxies:  []internalconfig.YumProxy{},
	}
}

// extractApkConfig 통합 설정에서 APK 설정 추출
func (s *ViperConfigService) extractApkConfig() *internalconfig.ApkProxySettings {
	// 기본 APK 설정 반환
	return &internalconfig.ApkProxySettings{
		Path:     "/apk",
		UseCache: true,
		Proxies:  []internalconfig.ApkProxy{},
	}
}

// extractDockerConfig 통합 설정에서 Docker 설정 추출
func (s *ViperConfigService) extractDockerConfig() *internalconfig.DockerProxySettings {
	if s.unifiedConfig == nil || !s.unifiedConfig.Registries.Docker.Enabled {
		return nil
	}

	config := &internalconfig.DockerProxySettings{
		Path:     "/docker",
		UseCache: s.unifiedConfig.Registries.Docker.UseCache,
		Proxies:  []internalconfig.DockerProxyServer{},
	}

	for _, reg := range s.unifiedConfig.Registries.Docker.Registries {
		config.Proxies = append(config.Proxies, internalconfig.DockerProxyServer{
			Name: reg.Name,
			URL:  reg.URL,
			Auth: internalconfig.DockerAuth{
				Username: reg.Username,
				Password: reg.Password,
			},
		})
	}

	return config
}

// Ensure ViperConfigService implements the Service interface
var _ Service = (*ViperConfigService)(nil)
