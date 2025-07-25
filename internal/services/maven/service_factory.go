package maven

import (
	"proxynd/internal/domain/maven"
	"proxynd/logging"
)

// ServiceFactory Maven 서비스 팩토리
type ServiceFactory struct {
	config maven.ProxyConfig
	logger logging.Logger
}

// NewServiceFactory ServiceFactory 생성자
func NewServiceFactory(config maven.ProxyConfig, logger logging.Logger) *ServiceFactory {
	return &ServiceFactory{
		config: config,
		logger: logger,
	}
}

// CreateDirectoryCollector DirectoryCollector 생성
func (f *ServiceFactory) CreateDirectoryCollector() maven.DirectoryCollector {
	return NewDirectoryCollector(f.config, f.logger)
}

// CreateSearchService SearchService 생성
func (f *ServiceFactory) CreateSearchService(collector maven.DirectoryCollector) maven.SearchService {
	return NewSearchService(f.config, f.logger, collector)
}

// CreateCacheManager CacheManager 생성
func (f *ServiceFactory) CreateCacheManager() maven.CacheManager {
	return NewCacheManager(f.config, f.logger)
}

// CreatePathAnalyzer PathAnalyzer 생성
func (f *ServiceFactory) CreatePathAnalyzer() maven.PathAnalyzer {
	return NewPathAnalyzer(f.logger)
}

// CreateAllServices 모든 서비스 생성 (의존성 자동 연결)
func (f *ServiceFactory) CreateAllServices() (
	maven.DirectoryCollector,
	maven.SearchService,
	maven.CacheManager,
	maven.PathAnalyzer,
) {
	collector := f.CreateDirectoryCollector()
	searchService := f.CreateSearchService(collector)
	cacheManager := f.CreateCacheManager()
	pathAnalyzer := f.CreatePathAnalyzer()

	return collector, searchService, cacheManager, pathAnalyzer
}
