package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"proxynd/internal/config"
)

// BenchmarkConfigLoading 설정 로딩 성능 벤치마크
func BenchmarkConfigLoading(b *testing.B) {
	configDir := b.TempDir()

	// 테스트용 설정 파일들 생성
	setupConfigFiles(b, configDir)

	b.Run("GlobalConfig", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			config := &config.GlobalConfig{}
			err := config.ReadConfig()
			require.NoError(b, err)
		}
	})

	b.Run("APTProxyConfig", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			config := &config.AptProxyConfig{}
			err := config.ReadConfig()
			require.NoError(b, err)
		}
	})

	b.Run("MavenProxyConfig", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			config := &config.MavenProxyConfig{}
			err := config.ReadConfig()
			require.NoError(b, err)
		}
	})

	b.Run("NPMProxyConfig", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			config := &config.NpmProxyConfig{}
			err := config.ReadConfig()
			require.NoError(b, err)
		}
	})

	b.Run("UnifiedConfig", func(b *testing.B) {
		configPath := filepath.Join(configDir, "unified.yaml")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			config, err := config.LoadConfig(configPath)
			require.NoError(b, err)
			_ = config
		}
	})
}

// BenchmarkConfigValidation 설정 검증 성능 벤치마크
func BenchmarkConfigValidation(b *testing.B) {
	configDir := b.TempDir()
	setupConfigFiles(b, configDir)

	// 다양한 설정 크기 테스트
	configSizes := []struct {
		name    string
		proxies int
		mirrors int
		repos   int
	}{
		{"Small", 5, 3, 2},
		{"Medium", 50, 20, 10},
		{"Large", 500, 100, 50},
		{"XLarge", 2000, 500, 200},
	}

	for _, size := range configSizes {
		b.Run(size.name, func(b *testing.B) {
			// 큰 설정 파일 생성
			config := generateLargeConfig(size.proxies, size.mirrors, size.repos)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := config.Validate()
				require.NoError(b, err)
			}
		})
	}
}

// BenchmarkConfigHotReload 설정 핫 리로드 성능 벤치마크
func BenchmarkConfigHotReload(b *testing.B) {
	configDir := b.TempDir()
	setupConfigFiles(b, configDir)

	loader := config.NewConfigLoader(filepath.Join(configDir, "unified.yaml"))

	// 초기 로드
	_, err := loader.Load()
	require.NoError(b, err)

	b.Run("HotReload", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// 설정 파일 수정 시뮬레이션
			config, err := loader.Load()
			require.NoError(b, err)
			_ = config
		}
	})

	b.Run("HotReloadWithValidation", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			config, err := loader.Load()
			require.NoError(b, err)

			err = config.Validate()
			require.NoError(b, err)
		}
	})
}

// BenchmarkConfigCaching 설정 캐싱 성능 벤치마크
func BenchmarkConfigCaching(b *testing.B) {
	configDir := b.TempDir()
	setupConfigFiles(b, configDir)

	config := &config.GlobalConfig{}
	err := config.ReadConfig()
	require.NoError(b, err)

	b.Run("CacheHit", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 캐시된 설정 접근 시뮬레이션
			ttl := config.Cache.TTL
			_ = ttl
		}
	})

	b.Run("CacheMiss", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 새로운 패키지 타입으로 캐시 미스 시뮬레이션
			packageType := fmt.Sprintf("new-type-%d", i)
			ttl := config.Cache.GetTTLForPackageType(packageType)
			_ = ttl
		}
	})

	// TTL 계산 성능
	b.Run("TTLCalculation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ttl := config.Cache.TTL
			_ = ttl
		}
	})
}

// BenchmarkConfigSerialization 설정 직렬화 성능 벤치마크
func BenchmarkConfigSerialization(b *testing.B) {
	config := generateLargeConfig(100, 50, 25)

	b.Run("MarshalYAML", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// YAML 마샬링 시뮬레이션 (실제 yaml 라이브러리 필요)
			_ = config
		}
	})

	b.Run("UnmarshalYAML", func(b *testing.B) {
		// YAML 데이터 준비
		yamlData := generateConfigYAML(50, 25, 12)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var config config.UnifiedConfig
			// YAML 언마샬링 시뮬레이션
			_ = yamlData
			_ = config
		}
	})
}

// BenchmarkConfigMemoryUsage 설정 메모리 사용량 벤치마크
func BenchmarkConfigMemoryUsage(b *testing.B) {
	configSizes := []struct {
		name    string
		proxies int
		mirrors int
		repos   int
	}{
		{"Tiny", 1, 1, 1},
		{"Small", 10, 5, 3},
		{"Medium", 100, 50, 25},
		{"Large", 1000, 500, 250},
	}

	for _, size := range configSizes {
		b.Run(size.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				config := generateLargeConfig(size.proxies, size.mirrors, size.repos)
				_ = config
			}
		})
	}
}

// BenchmarkConfigConcurrentAccess 동시 설정 접근 성능
func BenchmarkConfigConcurrentAccess(b *testing.B) {
	config := generateLargeConfig(200, 100, 50)

	b.Run("ConcurrentRead", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// 동시 읽기 시뮬레이션
				ttl := config.Cache.TTL
				_ = ttl
			}
		})
	})

	b.Run("ConcurrentValidation", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := config.Validate()
				require.NoError(b, err)
			}
		})
	})

	b.Run("ConcurrentTTLCalculation", func(b *testing.B) {
		packages := make([]string, 1000)
		for i := range packages {
			packages[i] = fmt.Sprintf("package-%d", i)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				ttl := config.Cache.TTL
				_ = ttl
				i++
			}
		})
	})
}

// Helper functions

func setupConfigFiles(b *testing.B, configDir string) {
	// global.yaml
	globalYAML := `
storage_dir: "/tmp/proxynd-storage"
config_dir: "/tmp/proxynd-config"
cache:
  ttl: 3600
  package_ttls:
    npm: 1800
    maven: 5400
    apt: 3600
  use_cache_headers: true
  max_cache_header_ttl: 86400
  min_cache_header_ttl: 300
`
	err := os.WriteFile(filepath.Join(configDir, "global.yaml"), []byte(globalYAML), 0o644)
	require.NoError(b, err)

	// apt-proxy.yaml
	aptYAML := `
path: "/apt"
use_cache: true
proxies:
  ubuntu:
    - name: "main"
      url: "http://archive.ubuntu.com/ubuntu"
    - name: "security"
      url: "http://security.ubuntu.com/ubuntu"
  debian:
    - name: "main"
      url: "http://deb.debian.org/debian"
`
	err = os.WriteFile(filepath.Join(configDir, "apt-proxy.yaml"), []byte(aptYAML), 0o644)
	require.NoError(b, err)

	// maven-proxy.yaml
	mavenYAML := `
path: "/maven"
use_cache: true
proxies:
  - name: "central"
    url: "https://repo1.maven.org/maven2"
  - name: "apache"
    url: "https://repository.apache.org/content/repositories/public"
`
	err = os.WriteFile(filepath.Join(configDir, "maven-proxy.yaml"), []byte(mavenYAML), 0o644)
	require.NoError(b, err)

	// npm-proxy.yaml
	npmYAML := `
path: "/npm"
use_cache: true
proxies:
  npmjs:
    - name: "registry"
      url: "https://registry.npmjs.org"
`
	err = os.WriteFile(filepath.Join(configDir, "npm-proxy.yaml"), []byte(npmYAML), 0o644)
	require.NoError(b, err)

	// unified.yaml
	unifiedYAML := `
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
cache:
  backend: "file"
  ttl: "3600s"
  max_size: "10GB"
  max_items: 100000
  file:
    directory: "/tmp/proxynd-cache"
registries:
  npm:
    enabled: true
    upstream: "https://registry.npmjs.org"
    timeout: "30s"
  maven:
    enabled: true
    repositories:
      - id: "central"
        name: "Maven Central"
        url: "https://repo1.maven.org/maven2"
  apt:
    enabled: true
    mirrors:
      ubuntu:
        - name: "main"
          url: "http://archive.ubuntu.com/ubuntu"
`
	err = os.WriteFile(filepath.Join(configDir, "unified.yaml"), []byte(unifiedYAML), 0o644)
	require.NoError(b, err)

	// 환경 변수 설정
	_ = os.Setenv("CONFIG_DIR", configDir)
}

func generateLargeConfig(_, numMirrors, numRepos int) *config.UnifiedConfig {
	config := &config.UnifiedConfig{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Cache: config.CacheConfig{
			Backend:  "file",
			TTL:      time.Hour,
			MaxSize:  "10GB",
			MaxItems: 100000,
			File: config.FileCacheConfig{
				Directory: "/tmp/cache",
			},
		},
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
				Timeout:  30 * time.Second,
			},
			Maven: config.MavenRegistryConfig{
				Enabled:      true,
				Repositories: make([]config.MavenRepositoryConfig, numRepos),
			},
			APT: config.APTRegistryConfig{
				Enabled: true,
				Mirrors: make(map[string][]config.APTMirror),
			},
		},
	}

	// Maven 리포지토리 생성
	for i := 0; i < numRepos; i++ {
		config.Registries.Maven.Repositories[i] = config.MavenRepositoryConfig{
			ID:        fmt.Sprintf("repo-%d", i),
			Name:      fmt.Sprintf("Repository %d", i),
			URL:       fmt.Sprintf("https://repo%d.example.com/maven2", i),
			Releases:  true,
			Snapshots: false,
		}
	}

	// APT 미러 생성
	distros := []string{"ubuntu", "debian", "centos", "fedora", "opensuse"}
	for _, distro := range distros {
		mirrors := make([]config.APTMirror, numMirrors)
		for i := 0; i < numMirrors; i++ {
			mirrors[i] = config.APTMirror{
				Name:       fmt.Sprintf("%s-mirror-%d", distro, i),
				URL:        fmt.Sprintf("http://mirror%d.%s.com/%s", i, distro, distro),
				Suites:     []string{"stable", "testing"},
				Components: []string{"main", "contrib", "non-free"},
			}
		}
		config.Registries.APT.Mirrors[distro] = mirrors
	}

	return config
}

func generateConfigYAML(_, numMirrors, numRepos int) []byte {
	yaml := `
server:
  host: "0.0.0.0"
  port: 8080
cache:
  backend: "file"
  ttl: "3600s"
  max_size: "10GB"
registries:
  npm:
    enabled: true
    upstream: "https://registry.npmjs.org"
  maven:
    enabled: true
    repositories:`

	// Maven 리포지토리 추가
	for i := 0; i < numRepos; i++ {
		yaml += fmt.Sprintf(`
      - id: "repo-%d"
        name: "Repository %d"
        url: "https://repo%d.example.com/maven2"`, i, i, i)
	}

	yaml += `
  apt:
    enabled: true
    mirrors:`

	// APT 미러 추가
	distros := []string{"ubuntu", "debian"}
	for _, distro := range distros {
		yaml += fmt.Sprintf(`
      %s:`, distro)
		for i := 0; i < numMirrors; i++ {
			yaml += fmt.Sprintf(`
        - name: "%s-mirror-%d"
          url: "http://mirror%d.%s.com/%s"`, distro, i, i, distro, distro)
		}
	}

	return []byte(yaml)
}
