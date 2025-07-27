package maven

import (
	"testing"

	"proxynd/internal/config"
)

func TestDefaultProxyConfig_GetProxies(t *testing.T) {
	// 테스트 데이터 설정
	originalConfig := &config.MavenProxyConfig{
		Proxies: []config.MavenProxyServer{
			{
				Name: "Central",
				URL:  "https://repo1.maven.org/maven2",
			},
			{
				Name: "Google",
				URL:  "https://maven.google.com",
			},
		},
	}

	// DefaultProxyConfig 생성
	proxyConfig := NewDefaultProxyConfig(originalConfig)

	// GetProxies 테스트
	proxies := proxyConfig.GetProxies()

	// 검증
	if len(proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(proxies))
	}

	if proxies[0].Name != "Central" {
		t.Errorf("Expected first proxy name to be 'Central', got '%s'", proxies[0].Name)
	}

	if proxies[1].URL != "https://maven.google.com" {
		t.Errorf("Expected second proxy URL to be 'https://maven.google.com', got '%s'", proxies[1].URL)
	}
}

func TestDefaultProxyConfig_GetCacheConfig(t *testing.T) {
	originalConfig := &config.MavenProxyConfig{}
	proxyConfig := NewDefaultProxyConfig(originalConfig)

	cacheConfig := proxyConfig.GetCacheConfig()

	// 기본값 확인
	if !cacheConfig.Enabled {
		t.Error("Expected cache to be enabled by default")
	}

	if cacheConfig.TTLMinutes != 60 {
		t.Errorf("Expected default TTL to be 60 minutes, got %d", cacheConfig.TTLMinutes)
	}

	if cacheConfig.MaxSizeMB != 500 {
		t.Errorf("Expected default max size to be 500MB, got %d", cacheConfig.MaxSizeMB)
	}
}

func TestDefaultProxyConfig_GetSearchConfig(t *testing.T) {
	originalConfig := &config.MavenProxyConfig{}
	proxyConfig := NewDefaultProxyConfig(originalConfig)

	searchConfig := proxyConfig.GetSearchConfig()

	// 기본값 확인
	if !searchConfig.Enabled {
		t.Error("Expected search to be enabled by default")
	}

	if searchConfig.IndexRebuildHours != 24 {
		t.Errorf("Expected default index rebuild hours to be 24, got %d", searchConfig.IndexRebuildHours)
	}

	if searchConfig.MaxResults != 100 {
		t.Errorf("Expected default max results to be 100, got %d", searchConfig.MaxResults)
	}
}

func TestDefaultProxyConfig_EmptyConfig(t *testing.T) {
	// nil config 테스트
	proxyConfig := NewDefaultProxyConfig(nil)

	proxies := proxyConfig.GetProxies()
	if len(proxies) != 0 {
		t.Errorf("Expected empty proxies for nil config, got %d", len(proxies))
	}

	// 여전히 기본 캐시 및 검색 설정이 작동해야 함
	cacheConfig := proxyConfig.GetCacheConfig()
	if !cacheConfig.Enabled {
		t.Error("Expected cache to be enabled even with nil config")
	}

	searchConfig := proxyConfig.GetSearchConfig()
	if !searchConfig.Enabled {
		t.Error("Expected search to be enabled even with nil config")
	}
}
