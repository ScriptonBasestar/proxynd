package container

import (
	"proxynd/internal/config"
)

// ConfigProvider 설정 제공자 인터페이스 (순환 참조 방지)
type ConfigProvider interface {
	GetAptProxyConfig() (*config.AptProxyConfig, error)
	GetMavenProxyConfig() (*config.MavenProxySettings, error)
	GetNpmProxyConfig() (*config.NpmProxySettings, error)
	GetDockerProxyConfig() (*config.DockerProxySettings, error)
	GetPipProxyConfig() (*config.PipProxySettings, error)
	GetYumProxyConfig() (*config.YumProxySettings, error)
	GetApkProxyConfig() (*config.ApkProxySettings, error)
}

// StorageProvider 스토리지 제공자 인터페이스
type StorageProvider interface {
	GetStorageDir() string
	GetConfigDir() string
}

// ContainerProvider Container 서비스 제공자 인터페이스
type ContainerProvider interface {
	ConfigProvider
	StorageProvider
}
