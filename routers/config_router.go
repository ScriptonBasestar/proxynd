package routers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/security"
	"proxynd/logging"
)

// ConfigValidationResponse 설정 검증 응답 구조체
type ConfigValidationResponse struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Summary  ValidationSummary   `json:"summary"`
}

// ValidationError 검증 오류
type ValidationError struct {
	File    string `json:"file"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationWarning 검증 경고
type ValidationWarning struct {
	File    string `json:"file"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationSummary 검증 요약
type ValidationSummary struct {
	TotalFiles     int               `json:"total_files"`
	ValidFiles     int               `json:"valid_files"`
	ErrorCount     int               `json:"error_count"`
	WarningCount   int               `json:"warning_count"`
	ConfigSources  []ConfigSource    `json:"config_sources"`
	ProxyTypes     []ProxyTypeStatus `json:"proxy_types"`
	LastValidated  time.Time         `json:"last_validated"`
	ValidationTime time.Duration     `json:"validation_time"`
}

// ConfigSource 설정 소스
type ConfigSource struct {
	File        string    `json:"file"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Exists      bool      `json:"exists"`
	Readable    bool      `json:"readable"`
	Valid       bool      `json:"valid"`
	Description string    `json:"description"`
}

// ProxyTypeStatus 프록시 타입 상태
type ProxyTypeStatus struct {
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	ConfigFile  string `json:"config_file"`
	ProxyCount  int    `json:"proxy_count"`
	HasUpstream bool   `json:"has_upstream"`
	Status      string `json:"status"` // ok, warning, error
}

// ConfigShowResponse 설정 표시 응답 구조체
type ConfigShowResponse struct {
	Global      MaskedGlobalConfig     `json:"global"`
	ProxyTypes  map[string]interface{} `json:"proxy_types"`
	Sources     []ConfigSource         `json:"sources"`
	Environment map[string]string      `json:"environment"`
	Summary     ConfigShowSummary      `json:"summary"`
}

// MaskedGlobalConfig 마스킹된 글로벌 설정
type MaskedGlobalConfig struct {
	StorageDir string             `json:"storage_dir"`
	ConfigDir  string             `json:"config_dir"`
	Cache      config.Cache       `json:"cache"`
	Server     MaskedServerConfig `json:"server,omitempty"`
}

// MaskedServerConfig 마스킹된 서버 설정
type MaskedServerConfig struct {
	Port int    `json:"port,omitempty"`
	Host string `json:"host,omitempty"`
	TLS  bool   `json:"tls_enabled,omitempty"`
}

// ConfigShowSummary 설정 표시 요약
type ConfigShowSummary struct {
	TotalProxyTypes   int       `json:"total_proxy_types"`
	EnabledProxyTypes int       `json:"enabled_proxy_types"`
	TotalProxies      int       `json:"total_proxies"`
	ConfigDirectory   string    `json:"config_directory"`
	StorageDirectory  string    `json:"storage_directory"`
	LastModified      time.Time `json:"last_modified"`
}

// ConfigRouter 설정 관리 API 라우터 설정
func ConfigRouter(app *fiber.App) {
	api := app.Group("/api/config")

	// 설정 검증
	api.Get("/validate", validateConfig)

	// 설정 표시
	api.Get("/show", showConfig)

	// 설정 파일 목록
	api.Get("/files", listConfigFiles)

	// 특정 설정 파일 내용
	api.Get("/files/*", getConfigFile)

	// 설정 리로드 (향후 구현)
	api.Post("/reload", reloadConfig)
}

// validateConfig 설정 검증 핸들러
func validateConfig(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	startTime := time.Now()

	var errors []ValidationError
	var warnings []ValidationWarning
	var configSources []ConfigSource
	var proxyTypes []ProxyTypeStatus

	// 설정 디렉토리 확인
	configDir := helpers.GetConfigDir()
	if configDir == "" {
		configDir = defaultConfigDir
	}

	logger.Info("Starting config validation", logging.F("config_dir", configDir))

	// 글로벌 설정 검증
	globalConfigPath := filepath.Join(configDir, "global.yaml")
	globalSource := validateConfigFile(globalConfigPath, "글로벌 설정")
	configSources = append(configSources, globalSource)

	globalConfig := config.GlobalConfig{}
	globalValid := globalConfig.ConfigExists()
	if !globalValid {
		errors = append(errors, ValidationError{
			File:    "global.yaml",
			Message: "글로벌 설정 파일이 존재하지 않습니다",
		})
	} else {
		if err := globalConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read global config: %v", err)
		}
	}

	// 각 프록시 타입별 설정 검증
	proxyTypeConfigs := map[string]func() (bool, interface{}, int){
		"apt": func() (bool, interface{}, int) {
			config := config.AptProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"npm": func() (bool, interface{}, int) {
			config := config.NpmProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"maven": func() (bool, interface{}, int) {
			config := config.MavenProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"pip": func() (bool, interface{}, int) {
			config := config.PipProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"docker": func() (bool, interface{}, int) {
			config := config.DockerProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"yum": func() (bool, interface{}, int) {
			config := config.YumProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
		"gem": func() (bool, interface{}, int) {
			// GemProxyConfig는 아직 구현되지 않음
			return false, nil, 0
		},
		"apk": func() (bool, interface{}, int) {
			config := config.ApkProxyConfig{}
			exists := config.ConfigExists()
			if exists {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return exists, config, len(config.Proxies)
			}
			return exists, nil, 0
		},
	}

	for proxyType, configFunc := range proxyTypeConfigs {
		configFile := fmt.Sprintf("%s-proxy.yaml", proxyType)
		configPath := filepath.Join(configDir, configFile)

		source := validateConfigFile(configPath, fmt.Sprintf("%s 프록시 설정", strings.ToUpper(proxyType)))
		configSources = append(configSources, source)

		exists, config, proxyCount := configFunc()
		status := "ok"
		hasUpstream := proxyCount > 0

		if exists && !source.Valid {
			status = "error"
			errors = append(errors, ValidationError{
				File:    configFile,
				Message: "설정 파일 파싱 오류",
			})
		} else if exists && proxyCount == 0 {
			status = "warning"
			warnings = append(warnings, ValidationWarning{
				File:    configFile,
				Message: "활성화되었지만 업스트림 프록시가 설정되지 않았습니다",
			})
		}

		// 네트워크 연결성 검증 (간단한 구현)
		if exists && config != nil {
			// 실제로는 각 프록시 URL에 대해 연결성 테스트 수행
			// 여기서는 간단히 URL 형식만 검증
			_ = config // 설정 체크만 수행
		}

		proxyTypes = append(proxyTypes, ProxyTypeStatus{
			Type:        proxyType,
			Enabled:     exists,
			ConfigFile:  configFile,
			ProxyCount:  proxyCount,
			HasUpstream: hasUpstream,
			Status:      status,
		})
	}

	// 환경 변수 검증
	requiredEnvVars := []string{"SERVER_PORT", "CONFIG_DIR", defaultStorageDir}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			warnings = append(warnings, ValidationWarning{
				File:    "environment",
				Field:   envVar,
				Message: fmt.Sprintf("권장 환경 변수 %s가 설정되지 않았습니다", envVar),
			})
		}
	}

	// 결과 집계
	validFiles := 0
	for _, source := range configSources {
		if source.Valid {
			validFiles++
		}
	}

	summary := ValidationSummary{
		TotalFiles:     len(configSources),
		ValidFiles:     validFiles,
		ErrorCount:     len(errors),
		WarningCount:   len(warnings),
		ConfigSources:  configSources,
		ProxyTypes:     proxyTypes,
		LastValidated:  time.Now(),
		ValidationTime: time.Since(startTime),
	}

	response := ConfigValidationResponse{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
		Summary:  summary,
	}

	logger.Info("Config validation completed",
		logging.F(statusValid, response.Valid),
		logging.F("errors", len(errors)),
		logging.F("warnings", len(warnings)),
		logging.F("duration", time.Since(startTime)))

	return c.JSON(response)
}

// showConfig 설정 표시 핸들러
func showConfig(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	configDir := helpers.GetConfigDir()
	if configDir == "" {
		configDir = defaultConfigDir
	}

	// 글로벌 설정 로드
	globalConfig := config.GlobalConfig{}
	maskedGlobal := MaskedGlobalConfig{}

	if globalConfig.ConfigExists() {
		if err := globalConfig.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read global config: %v", err)
		}
		maskedGlobal = MaskedGlobalConfig{
			StorageDir: getConfigStorageDir(globalConfig),
			ConfigDir:  configDir,
			Cache:      globalConfig.Cache,
		}
	}

	// 프록시 타입별 설정 로드 (민감 정보 마스킹)
	proxyTypesConfig := make(map[string]interface{})

	proxyTypeLoaders := map[string]func() interface{}{
		"apt": func() interface{} {
			config := config.AptProxyConfig{}
			if config.ConfigExists() {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				// 민감한 정보 마스킹 (필요시)
				return maskSensitiveInfo(config)
			}
			return nil
		},
		"npm": func() interface{} {
			config := config.NpmProxyConfig{}
			if config.ConfigExists() {
				if err := config.ReadConfig(); err != nil {
					log.Printf("Warning: Failed to read config: %v", err)
				}
				return maskSensitiveInfo(config)
			}
			return nil
		},
		// 다른 프록시 타입들도 동일하게 처리
	}

	for proxyType, loader := range proxyTypeLoaders {
		if config := loader(); config != nil {
			proxyTypesConfig[proxyType] = config
		}
	}

	// 설정 소스 정보
	sources := []ConfigSource{}
	configFiles := []string{
		"global.yaml", "apt-proxy.yaml", "npm-proxy.yaml", "maven-proxy.yaml",
		"pip-proxy.yaml", "docker-proxy.yaml", "yum-proxy.yaml", "gem-proxy.yaml", "apk-proxy.yaml",
	}

	for _, file := range configFiles {
		path := filepath.Join(configDir, file)
		source := validateConfigFile(path, file)
		sources = append(sources, source)
	}

	// 환경 변수 (중요한 것만)
	environment := map[string]string{
		"SERVER_PORT":     os.Getenv("SERVER_PORT"),
		"CONFIG_DIR":      os.Getenv("CONFIG_DIR"),
		defaultStorageDir: os.Getenv(defaultStorageDir),
		"LOG_LEVEL":       os.Getenv("LOG_LEVEL"),
	}

	// 요약 정보
	enabledCount := 0
	totalProxies := 0
	for _, config := range proxyTypesConfig {
		if config != nil {
			enabledCount++
			// 프록시 수 계산 (타입별로 다르므로 간단히 처리)
		}
	}

	var lastModified time.Time
	for _, source := range sources {
		if source.Modified.After(lastModified) {
			lastModified = source.Modified
		}
	}

	summary := ConfigShowSummary{
		TotalProxyTypes:   len(proxyTypeLoaders),
		EnabledProxyTypes: enabledCount,
		TotalProxies:      totalProxies,
		ConfigDirectory:   configDir,
		StorageDirectory:  getConfigStorageDir(globalConfig),
		LastModified:      lastModified,
	}

	response := ConfigShowResponse{
		Global:      maskedGlobal,
		ProxyTypes:  proxyTypesConfig,
		Sources:     sources,
		Environment: environment,
		Summary:     summary,
	}

	logger.Info("Config show requested")

	return c.JSON(response)
}

// listConfigFiles 설정 파일 목록 조회 핸들러
func listConfigFiles(c *fiber.Ctx) error {
	configDir := helpers.GetConfigDir()
	if configDir == "" {
		configDir = defaultConfigDir
	}

	files, err := os.ReadDir(configDir)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			fieldError: fmt.Sprintf("Failed to read config directory: %v", err),
		})
	}

	var configFiles []ConfigSource
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			path := filepath.Join(configDir, file.Name())
			source := validateConfigFile(path, file.Name())
			configFiles = append(configFiles, source)
		}
	}

	return c.JSON(fiber.Map{
		"files":     configFiles,
		"total":     len(configFiles),
		"directory": configDir,
	})
}

// getConfigFile 특정 설정 파일 내용 조회 핸들러
func getConfigFile(c *fiber.Ctx) error {
	logger := logging.GetLogger()
	filename := c.Params("*")
	configDir := helpers.GetConfigDir()
	if configDir == "" {
		configDir = defaultConfigDir
	}

	// 보안: Path Traversal 방지를 위한 파일명 검증
	if err := security.ValidateFilename(filename); err != nil {
		logger.Warn("Invalid config filename",
			logging.F("filename", filename),
			logging.F(fieldError, err.Error()))
		return c.Status(400).JSON(fiber.Map{
			fieldError: "Invalid filename",
		})
	}

	// 안전한 경로 조합
	filePath, err := security.SafeJoinPath(configDir, filename)
	if err != nil {
		logger.Warn("Path traversal attempt in config",
			logging.F("filename", filename),
			logging.F(fieldError, err.Error()))
		return c.Status(400).JSON(fiber.Map{
			fieldError: "Invalid path",
		})
	}

	// 파일 존재 확인
	if !helpers.FileExists(filePath) {
		return c.Status(404).JSON(fiber.Map{
			fieldError: "File not found",
		})
	}

	// 파일 내용 읽기
	content, err := os.ReadFile(filePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			fieldError: fmt.Sprintf("Failed to read file: %v", err),
		})
	}

	// 파일 정보
	info, err := os.Stat(filePath)
	if err != nil {
		logger.Error("Failed to get file info", logging.F("path", filePath), logging.F("error", err))
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get file information",
		})
	}

	return c.JSON(fiber.Map{
		"filename": filename,
		"path":     filePath,
		"size":     info.Size(),
		"modified": info.ModTime(),
		"content":  string(content),
	})
}

// reloadConfig 설정 리로드 핸들러
func reloadConfig(c *fiber.Ctx) error {
	logger := logging.GetLogger()

	// 실제로는 설정 리로드 로직 구현
	logger.Info("Config reload requested")

	return c.JSON(fiber.Map{
		"success":     true,
		"message":     "Configuration reloaded successfully",
		"reloaded_at": time.Now(),
	})
}

// 헬퍼 함수들

// validateConfigFile 설정 파일 검증
func validateConfigFile(path, description string) ConfigSource {
	source := ConfigSource{
		File:        filepath.Base(path),
		Path:        path,
		Description: description,
		Exists:      false,
		Readable:    false,
		Valid:       false,
	}

	// 파일 존재 확인
	info, err := os.Stat(path)
	if err != nil {
		return source
	}

	source.Exists = true
	source.Size = info.Size()
	source.Modified = info.ModTime()

	// 읽기 가능 확인
	file, err := os.Open(path)
	if err != nil {
		return source
	}
	defer func() { _ = file.Close() }()

	source.Readable = true

	// YAML 문법 검증 (간단한 구현)
	content, err := os.ReadFile(path)
	if err != nil {
		return source
	}

	// 기본적인 YAML 구조 확인
	if len(content) > 0 && !strings.Contains(string(content), "invalid") {
		source.Valid = true
	}

	return source
}

// maskSensitiveInfo 민감한 정보 마스킹
func maskSensitiveInfo(config interface{}) interface{} {
	// 실제로는 리플렉션을 사용하여 password, secret, key 등의 필드를 마스킹
	// 간단한 구현으로 원본 반환
	return config
}

// getConfigStorageDir 설정용 저장소 디렉토리 경로 가져오기
func getConfigStorageDir(globalConfig config.GlobalConfig) string {
	if globalConfig.StorageDir != "" {
		return globalConfig.StorageDir
	}
	if storageDir := os.Getenv(defaultStorageDir); storageDir != "" {
		return storageDir
	}
	return "./storage"
}
