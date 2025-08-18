package testutil

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/adapters/pm/common"
	"proxynd/internal/config"
	"proxynd/internal/container"
)

// MockContainerProvider는 테스트용 ContainerProvider 구현체입니다
type MockContainerProvider struct {
	mock.Mock
	storageDir string
	configDir  string

	// 설정 캐시
	aptConfig    *config.AptProxyConfig
	mavenConfig  *config.MavenProxySettings
	npmConfig    *config.NpmProxySettings
	dockerConfig *config.DockerProxySettings
	pipConfig    *config.PipProxySettings
	yumConfig    *config.YumProxySettings
	apkConfig    *config.ApkProxySettings
}

// NewMockContainerProvider 새로운 MockContainerProvider 생성
func NewMockContainerProvider(t *testing.T) *MockContainerProvider {
	tempDir := t.TempDir()

	mock := &MockContainerProvider{
		storageDir: filepath.Join(tempDir, "storage"),
		configDir:  filepath.Join(tempDir, "config"),
	}

	// 기본 설정값들 초기화
	mock.initDefaultConfigs()

	return mock
}

// NewMockContainerProviderWithDirs 특정 디렉토리로 MockContainerProvider 생성
func NewMockContainerProviderWithDirs(storageDir, configDir string) *MockContainerProvider {
	mock := &MockContainerProvider{
		storageDir: storageDir,
		configDir:  configDir,
	}

	mock.initDefaultConfigs()
	return mock
}

// ContainerProvider 인터페이스 구현

// GetStorageDir 스토리지 디렉토리 반환
func (m *MockContainerProvider) GetStorageDir() string {
	return m.storageDir
}

// GetConfigDir 설정 디렉토리 반환
func (m *MockContainerProvider) GetConfigDir() string {
	return m.configDir
}

// GetAptProxyConfig APT 프록시 설정 반환
func (m *MockContainerProvider) GetAptProxyConfig() (*config.AptProxyConfig, error) {
	if m.aptConfig != nil {
		return m.aptConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.AptProxyConfig), nil
}

// GetMavenProxyConfig Maven 프록시 설정 반환
func (m *MockContainerProvider) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
	if m.mavenConfig != nil {
		return m.mavenConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.MavenProxySettings), nil
}

// GetNpmProxyConfig NPM 프록시 설정 반환
func (m *MockContainerProvider) GetNpmProxyConfig() (*config.NpmProxySettings, error) {
	if m.npmConfig != nil {
		return m.npmConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.NpmProxySettings), nil
}

// GetDockerProxyConfig Docker 프록시 설정 반환
func (m *MockContainerProvider) GetDockerProxyConfig() (*config.DockerProxySettings, error) {
	if m.dockerConfig != nil {
		return m.dockerConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.DockerProxySettings), nil
}

// GetPipProxyConfig PIP 프록시 설정 반환
func (m *MockContainerProvider) GetPipProxyConfig() (*config.PipProxySettings, error) {
	if m.pipConfig != nil {
		return m.pipConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.PipProxySettings), nil
}

// GetYumProxyConfig YUM 프록시 설정 반환
func (m *MockContainerProvider) GetYumProxyConfig() (*config.YumProxySettings, error) {
	if m.yumConfig != nil {
		return m.yumConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.YumProxySettings), nil
}

// GetApkProxyConfig APK 프록시 설정 반환
func (m *MockContainerProvider) GetApkProxyConfig() (*config.ApkProxySettings, error) {
	if m.apkConfig != nil {
		return m.apkConfig, nil
	}

	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.ApkProxySettings), nil
}

// 테스트용 헬퍼 메서드들

// SetStorageDir 스토리지 디렉토리 설정
func (m *MockContainerProvider) SetStorageDir(dir string) {
	m.storageDir = dir
}

// SetConfigDir 설정 디렉토리 설정
func (m *MockContainerProvider) SetConfigDir(dir string) {
	m.configDir = dir
}

// SetAptConfig APT 설정 직접 설정
func (m *MockContainerProvider) SetAptConfig(cfg *config.AptProxyConfig) {
	m.aptConfig = cfg
}

// SetMavenConfig Maven 설정 직접 설정
func (m *MockContainerProvider) SetMavenConfig(cfg *config.MavenProxySettings) {
	m.mavenConfig = cfg
}

// SetNpmConfig NPM 설정 직접 설정
func (m *MockContainerProvider) SetNpmConfig(cfg *config.NpmProxySettings) {
	m.npmConfig = cfg
}

// SetDockerConfig Docker 설정 직접 설정
func (m *MockContainerProvider) SetDockerConfig(cfg *config.DockerProxySettings) {
	m.dockerConfig = cfg
}

// SetPipConfig PIP 설정 직접 설정
func (m *MockContainerProvider) SetPipConfig(cfg *config.PipProxySettings) {
	m.pipConfig = cfg
}

// SetYumConfig YUM 설정 직접 설정
func (m *MockContainerProvider) SetYumConfig(cfg *config.YumProxySettings) {
	m.yumConfig = cfg
}

// SetApkConfig APK 설정 직접 설정
func (m *MockContainerProvider) SetApkConfig(cfg *config.ApkProxySettings) {
	m.apkConfig = cfg
}

// 모든 설정을 오류 반환하도록 설정 (실패 시나리오 테스트용)
func (m *MockContainerProvider) SetAllConfigsToError(err error) {
	m.aptConfig = nil
	m.mavenConfig = nil
	m.npmConfig = nil
	m.dockerConfig = nil
	m.pipConfig = nil
	m.yumConfig = nil
	m.apkConfig = nil

	m.On("GetAptProxyConfig").Return((*config.AptProxyConfig)(nil), err)
	m.On("GetMavenProxyConfig").Return((*config.MavenProxySettings)(nil), err)
	m.On("GetNpmProxyConfig").Return((*config.NpmProxySettings)(nil), err)
	m.On("GetDockerProxyConfig").Return((*config.DockerProxySettings)(nil), err)
	m.On("GetPipProxyConfig").Return((*config.PipProxySettings)(nil), err)
	m.On("GetYumProxyConfig").Return((*config.YumProxySettings)(nil), err)
	m.On("GetApkProxyConfig").Return((*config.ApkProxySettings)(nil), err)
}

// initDefaultConfigs 기본 설정값들 초기화
func (m *MockContainerProvider) initDefaultConfigs() {
	// APT 기본 설정
	m.aptConfig = &config.AptProxyConfig{
		Path:     "apt",
		UseCache: true,
		Proxies: map[string][]config.AptProxy{
			"ubuntu": {
				{
					Name: "ubuntu-main",
					URL:  "http://archive.ubuntu.com/ubuntu",
				},
			},
		},
	}

	// Maven 기본 설정
	m.mavenConfig = &config.MavenProxySettings{
		Path:     "maven",
		UseCache: true,
		Proxies: []config.MavenProxyServer{
			{
				Name: "central",
				URL:  "https://repo1.maven.org/maven2",
			},
		},
	}

	// NPM 기본 설정
	m.npmConfig = &config.NpmProxySettings{
		Path:     "npm",
		UseCache: true,
		Proxies: map[string][]config.NpmProxyServer{
			"default": {
				{
					Name: "npmjs",
					URL:  "https://registry.npmjs.org",
				},
			},
		},
	}

	// Docker 기본 설정
	m.dockerConfig = &config.DockerProxySettings{
		Path:     "docker",
		UseCache: true,
		Proxies: []config.DockerProxyServer{
			{
				Name: "docker-hub",
				URL:  "https://registry-1.docker.io",
			},
		},
	}

	// PIP 기본 설정
	m.pipConfig = &config.PipProxySettings{
		Path:     "pip",
		UseCache: true,
		Proxies: []config.PipProxyServer{
			{
				Name: "pypi",
				URL:  "https://pypi.org",
			},
		},
	}

	// YUM 기본 설정
	m.yumConfig = &config.YumProxySettings{
		Path:     "yum",
		UseCache: true,
		Proxies: []config.YumProxy{
			{
				Name: "centos",
				URL:  "http://mirror.centos.org/centos",
			},
		},
	}

	// APK 기본 설정
	m.apkConfig = &config.ApkProxySettings{
		Path:     "apk",
		UseCache: true,
		Proxies: []config.ApkProxy{
			{
				Name: "alpine",
				URL:  "https://dl-cdn.alpinelinux.org/alpine",
			},
		},
	}
}

// AsContainerProvider ContainerProvider 인터페이스로 반환
func (m *MockContainerProvider) AsContainerProvider() container.ContainerProvider {
	return m
}

// WithError 특정 설정 메서드가 에러를 반환하도록 설정
func (m *MockContainerProvider) WithError(method string, err error) *MockContainerProvider {
	switch method {
	case "GetAptProxyConfig":
		m.aptConfig = nil
		m.On("GetAptProxyConfig").Return((*config.AptProxyConfig)(nil), err)
	case "GetMavenProxyConfig":
		m.mavenConfig = nil
		m.On("GetMavenProxyConfig").Return((*config.MavenProxySettings)(nil), err)
	case "GetNpmProxyConfig":
		m.npmConfig = nil
		m.On("GetNpmProxyConfig").Return((*config.NpmProxySettings)(nil), err)
	case "GetDockerProxyConfig":
		m.dockerConfig = nil
		m.On("GetDockerProxyConfig").Return((*config.DockerProxySettings)(nil), err)
	case "GetPipProxyConfig":
		m.pipConfig = nil
		m.On("GetPipProxyConfig").Return((*config.PipProxySettings)(nil), err)
	case "GetYumProxyConfig":
		m.yumConfig = nil
		m.On("GetYumProxyConfig").Return((*config.YumProxySettings)(nil), err)
	case "GetApkProxyConfig":
		m.apkConfig = nil
		m.On("GetApkProxyConfig").Return((*config.ApkProxySettings)(nil), err)
	}
	return m
}

// WithConfig 특정 설정을 지정하여 MockContainer를 구성
func (m *MockContainerProvider) WithConfig(proxyType string, cfg interface{}) *MockContainerProvider {
	switch proxyType {
	case common.PMTypeApt:
		if aptCfg, ok := cfg.(*config.AptProxyConfig); ok {
			m.SetAptConfig(aptCfg)
		}
	case common.PMTypeMaven:
		if mavenCfg, ok := cfg.(*config.MavenProxySettings); ok {
			m.SetMavenConfig(mavenCfg)
		}
	case common.PMTypeNpm:
		if npmCfg, ok := cfg.(*config.NpmProxySettings); ok {
			m.SetNpmConfig(npmCfg)
		}
	case common.PMTypeDocker:
		if dockerCfg, ok := cfg.(*config.DockerProxySettings); ok {
			m.SetDockerConfig(dockerCfg)
		}
	case common.PMTypePip:
		if pipCfg, ok := cfg.(*config.PipProxySettings); ok {
			m.SetPipConfig(pipCfg)
		}
	case common.PMTypeYum:
		if yumCfg, ok := cfg.(*config.YumProxySettings); ok {
			m.SetYumConfig(yumCfg)
		}
	case common.PMTypeApk:
		if apkCfg, ok := cfg.(*config.ApkProxySettings); ok {
			m.SetApkConfig(apkCfg)
		}
	}
	return m
}

// 체이닝을 위한 빌더 패턴 메서드들

// WithStorageDir 스토리지 디렉토리 설정 (체이닝 가능)
func (m *MockContainerProvider) WithStorageDir(dir string) *MockContainerProvider {
	m.SetStorageDir(dir)
	return m
}

// WithConfigDir 설정 디렉토리 설정 (체이닝 가능)
func (m *MockContainerProvider) WithConfigDir(dir string) *MockContainerProvider {
	m.SetConfigDir(dir)
	return m
}

// Verify Mock 호출 검증
func (m *MockContainerProvider) Verify(t *testing.T) {
	m.AssertExpectations(t)
}

// Reset Mock 상태 초기화
func (m *MockContainerProvider) Reset() {
	m.Mock = mock.Mock{}
	m.initDefaultConfigs()
}

// CreateWorkingConfig 실제로 동작하는 설정으로 초기화 (통합 테스트용)
func (m *MockContainerProvider) CreateWorkingConfig(proxyType string) error {
	switch proxyType {
	case common.PMTypeApt:
		m.aptConfig = &config.AptProxyConfig{
			Path:     "apt-test",
			UseCache: true,
			Proxies: map[string][]config.AptProxy{
				"test": {
					{
						Name: "ubuntu-test",
						URL:  "https://httpbin.org/json", // 테스트용 엔드포인트
					},
				},
			},
		}
	case common.PMTypeMaven:
		m.mavenConfig = &config.MavenProxySettings{
			Path:     "maven-test",
			UseCache: true,
			Proxies: []config.MavenProxyServer{
				{
					Name: "test-repo",
					URL:  "https://httpbin.org",
				},
			},
		}
	case common.PMTypeNpm:
		m.npmConfig = &config.NpmProxySettings{
			Path:     "npm-test",
			UseCache: true,
			Proxies: map[string][]config.NpmProxyServer{
				"default": {
					{
						Name: "test-npm",
						URL:  "https://httpbin.org",
					},
				},
			},
		}
	case "docker":
		m.dockerConfig = &config.DockerProxySettings{
			Path:     "docker-test",
			UseCache: true,
			Proxies: []config.DockerProxyServer{
				{
					Name: "test-registry",
					URL:  "https://httpbin.org",
				},
			},
		}
	case "pip":
		m.pipConfig = &config.PipProxySettings{
			Path:     "pip-test",
			UseCache: true,
			Proxies: []config.PipProxyServer{
				{
					Name: "test-pypi",
					URL:  "https://httpbin.org",
				},
			},
		}
	case common.PMTypeYum:
		m.yumConfig = &config.YumProxySettings{
			Path:     "yum-test",
			UseCache: true,
			Proxies: []config.YumProxy{
				{
					Name: "test-centos",
					URL:  "https://httpbin.org",
				},
			},
		}
	case common.PMTypeApk:
		m.apkConfig = &config.ApkProxySettings{
			Path:     "apk-test",
			UseCache: true,
			Proxies: []config.ApkProxy{
				{
					Name: "test-alpine",
					URL:  "https://httpbin.org",
				},
			},
		}
	default:
		return fmt.Errorf("unsupported proxy type: %s", proxyType)
	}
	return nil
}
