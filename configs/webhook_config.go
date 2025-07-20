package configs

import (
	"path"

	"proxynd/helpers"
)

// WebhookConfig 웹훅 알림 시스템 전체 설정
type WebhookConfig struct {
	// 웹훅 시스템 활성화 여부
	Enabled bool `yaml:"enabled" json:"enabled"`
	// 웹훅 엔드포인트 목록
	Endpoints      []WebhookEndpointConfig     `yaml:"endpoints" json:"endpoints" validate:"dive"`
	RateLimit      WebhookRateLimitConfig      `yaml:"rate_limit" json:"rate_limit" validate:"dive"`           // 속도 제한 설정
	Retry          WebhookRetryConfig          `yaml:"retry" json:"retry" validate:"dive"`                     // 재시도 설정
	FailureStorage WebhookFailureStorageConfig `yaml:"failure_storage" json:"failure_storage" validate:"dive"` // 실패 저장소 설정
	EventFilter    WebhookEventFilter          `yaml:"event_filter" json:"event_filter" validate:"dive"`       // 이벤트 필터링
	Buffering      WebhookBufferingConfig      `yaml:"buffering" json:"buffering" validate:"dive"`             // 버퍼링 설정
	Batching       WebhookBatchingConfig       `yaml:"batching" json:"batching" validate:"dive"`               // 배치 전송 설정
	Security       WebhookSecurityConfig       `yaml:"security" json:"security" validate:"dive"`               // 보안 설정
	Monitoring     WebhookMonitoringConfig     `yaml:"monitoring" json:"monitoring" validate:"dive"`           // 모니터링 설정
}

// WebhookEndpointConfig 개별 웹훅 엔드포인트 설정
type WebhookEndpointConfig struct {
	// 엔드포인트 이름
	Name string `yaml:"name" json:"name" validate:"required,min=1,max=100"`
	// 웹훅 URL
	URL string `yaml:"url" json:"url" validate:"required,url"`
	// 엔드포인트 활성화 여부
	Enabled bool `yaml:"enabled" json:"enabled"`
	// HTTP 메서드 (GET, POST, PUT)
	Method string `yaml:"method" json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE"`
	// 추가 HTTP 헤더
	Headers map[string]string `yaml:"headers" json:"headers"`
	// 요청 타임아웃 (예: "30s")
	Timeout string `yaml:"timeout" json:"timeout" validate:"required,duration"`
	// 구독할 이벤트 타입
	EventTypes []string `yaml:"event_types" json:"event_types" validate:"required,min=1"`
	// 메시지 포맷 (json, slack, discord, teams)
	Format string `yaml:"format" json:"format" validate:"required,oneof=json slack discord teams"`
	// 커스텀 메시지 템플릿
	Template string `yaml:"template" json:"template"`
	// 인증 정보
	Credentials WebhookCredentials `yaml:"credentials" json:"credentials" validate:"dive"`
	// 엔드포인트별 필터
	Filters WebhookEndpointFilters `yaml:"filters" json:"filters" validate:"dive"`
}

// WebhookCredentials 웹훅 인증 정보
type WebhookCredentials struct {
	// none, basic, bearer, hmac, oauth2
	Type         string            `yaml:"type" json:"type"`
	Username     string            `yaml:"username" json:"username,omitempty"`           // Basic 인증용
	Password     string            `yaml:"password" json:"password,omitempty"`           // Basic 인증용
	Token        string            `yaml:"token" json:"token,omitempty"`                 // Bearer 토큰
	Secret       string            `yaml:"secret" json:"secret,omitempty"`               // HMAC 시크릿
	Algorithm    string            `yaml:"algorithm" json:"algorithm,omitempty"`         // HMAC 알고리즘 (sha256, sha512)
	ClientID     string            `yaml:"client_id" json:"client_id,omitempty"`         // OAuth2 클라이언트 ID
	ClientSecret string            `yaml:"client_secret" json:"client_secret,omitempty"` // OAuth2 클라이언트 시크릿
	TokenURL     string            `yaml:"token_url" json:"token_url,omitempty"`         // OAuth2 토큰 URL
	Scope        string            `yaml:"scope" json:"scope,omitempty"`                 // OAuth2 스코프
	ExtraHeaders map[string]string `yaml:"extra_headers" json:"extra_headers,omitempty"` // 추가 인증 헤더
}

// WebhookEndpointFilters 엔드포인트별 필터 설정
type WebhookEndpointFilters struct {
	MinLevel     string   `yaml:"min_level" json:"min_level"`         // 최소 알림 레벨 (INFO, WARNING, ERROR, CRITICAL)
	ExcludeTypes []string `yaml:"exclude_types" json:"exclude_types"` // 제외할 이벤트 타입
	IncludeTypes []string `yaml:"include_types" json:"include_types"` // 포함할 이벤트 타입
	Sources      []string `yaml:"sources" json:"sources"`             // 특정 소스만 필터링
	UserAgent    []string `yaml:"user_agent" json:"user_agent"`       // 특정 User-Agent 필터링
	ClientIP     []string `yaml:"client_ip" json:"client_ip"`         // 특정 클라이언트 IP 필터링
	PackageTypes []string `yaml:"package_types" json:"package_types"` // 특정 패키지 타입 필터링 (npm, pip, apt, docker)
}

// WebhookRateLimitConfig 웹훅 속도 제한 설정
type WebhookRateLimitConfig struct {
	// 속도 제한 활성화
	Enabled bool `yaml:"enabled" json:"enabled"`
	// 초당 최대 요청 수
	MaxPerSecond int `yaml:"max_per_second" json:"max_per_second" validate:"min=0,max=1000"`
	// 분당 최대 요청 수
	MaxPerMinute int `yaml:"max_per_minute" json:"max_per_minute" validate:"min=0,max=10000"`
	// 시간당 최대 요청 수
	MaxPerHour int `yaml:"max_per_hour" json:"max_per_hour" validate:"min=0,max=100000"`
	// 버스트 크기
	BurstSize int `yaml:"burst_size" json:"burst_size" validate:"min=0,max=1000"`
	// 슬라이딩 윈도우 기간
	WindowDuration string `yaml:"window_duration" json:"window_duration" validate:"omitempty,duration"`
	// 백오프 전략 (linear, exponential, fixed)
	BackoffStrategy string `yaml:"backoff_strategy" json:"backoff_strategy" validate:"oneof=linear exponential fixed"`
	// 최대 백오프 지연
	MaxBackoffDelay string `yaml:"max_backoff_delay" json:"max_backoff_delay" validate:"omitempty,duration"`
}

// WebhookRetryConfig 웹훅 재시도 설정
type WebhookRetryConfig struct {
	Enabled         bool    `yaml:"enabled" json:"enabled"`                     // 재시도 활성화
	MaxAttempts     int     `yaml:"max_attempts" json:"max_attempts"`           // 최대 재시도 횟수
	InitialDelay    string  `yaml:"initial_delay" json:"initial_delay"`         // 초기 지연 시간
	MaxDelay        string  `yaml:"max_delay" json:"max_delay"`                 // 최대 지연 시간
	BackoffFactor   float64 `yaml:"backoff_factor" json:"backoff_factor"`       // 백오프 배수
	RetryableStatus []int   `yaml:"retryable_status" json:"retryable_status"`   // 재시도 가능한 HTTP 상태 코드
	DeadLetterQueue string  `yaml:"dead_letter_queue" json:"dead_letter_queue"` // 실패한 이벤트 저장 위치
	PersistFailures bool    `yaml:"persist_failures" json:"persist_failures"`   // 실패한 이벤트 로컬 저장 여부
}

// WebhookFailureStorageConfig 웹훅 실패 저장소 설정
type WebhookFailureStorageConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`                   // 실패 저장소 활성화
	StorageDir      string `yaml:"storage_dir" json:"storage_dir"`           // 실패 이벤트 저장 디렉토리
	RetentionHours  int    `yaml:"retention_hours" json:"retention_hours"`   // 실패 이벤트 보관 시간 (시간)
	MaxFileSize     int64  `yaml:"max_file_size" json:"max_file_size"`       // 최대 파일 크기 (바이트)
	MaxFiles        int    `yaml:"max_files" json:"max_files"`               // 최대 파일 수
	CleanupInterval string `yaml:"cleanup_interval" json:"cleanup_interval"` // 정리 작업 간격
	CompressOld     bool   `yaml:"compress_old" json:"compress_old"`         // 오래된 파일 압축 여부
	EncryptStorage  bool   `yaml:"encrypt_storage" json:"encrypt_storage"`   // 저장소 암호화 여부
}

// WebhookEventFilter 이벤트 필터링 설정
type WebhookEventFilter struct {
	Enabled        bool                       `yaml:"enabled" json:"enabled"`                 // 필터링 활성화
	DefaultLevel   string                     `yaml:"default_level" json:"default_level"`     // 기본 알림 레벨
	LevelOverrides map[string]string          `yaml:"level_overrides" json:"level_overrides"` // 이벤트 타입별 레벨 오버라이드
	EventRules     []WebhookEventRule         `yaml:"event_rules" json:"event_rules"`         // 이벤트 규칙
	Aggregation    WebhookAggregationConfig   `yaml:"aggregation" json:"aggregation"`         // 이벤트 집계 설정
	Deduplication  WebhookDeduplicationConfig `yaml:"deduplication" json:"deduplication"`     // 중복 제거 설정
}

// WebhookEventRule 이벤트 처리 규칙
type WebhookEventRule struct {
	Name          string                 `yaml:"name" json:"name"`                   // 규칙 이름
	Enabled       bool                   `yaml:"enabled" json:"enabled"`             // 규칙 활성화
	Priority      int                    `yaml:"priority" json:"priority"`           // 우선순위 (낮을수록 먼저 처리)
	Conditions    map[string]interface{} `yaml:"conditions" json:"conditions"`       // 조건 (이벤트 타입, 레벨, 메타데이터 등)
	Actions       []string               `yaml:"actions" json:"actions"`             // 액션 (allow, deny, modify, route)
	Target        string                 `yaml:"target" json:"target"`               // 대상 엔드포인트 (route 액션용)
	Modifications map[string]interface{} `yaml:"modifications" json:"modifications"` // 이벤트 수정 사항
}

// WebhookAggregationConfig 이벤트 집계 설정
type WebhookAggregationConfig struct {
	Enabled       bool              `yaml:"enabled" json:"enabled"`               // 집계 활성화
	WindowSize    string            `yaml:"window_size" json:"window_size"`       // 집계 윈도우 크기 (예: "5m")
	MaxEvents     int               `yaml:"max_events" json:"max_events"`         // 윈도우당 최대 이벤트 수
	GroupBy       []string          `yaml:"group_by" json:"group_by"`             // 그룹화 기준 (type, level, source 등)
	Strategies    map[string]string `yaml:"strategies" json:"strategies"`         // 집계 전략 (count, sample, merge)
	FlushTriggers []string          `yaml:"flush_triggers" json:"flush_triggers"` // 플러시 트리거 (time, count, critical)
}

// WebhookDeduplicationConfig 중복 제거 설정
type WebhookDeduplicationConfig struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`         // 중복 제거 활성화
	WindowSize string   `yaml:"window_size" json:"window_size"` // 중복 제거 윈도우 크기
	KeyFields  []string `yaml:"key_fields" json:"key_fields"`   // 중복 검사 키 필드
	Strategy   string   `yaml:"strategy" json:"strategy"`       // 중복 제거 전략 (first, last, merge, count)
	MaxCount   int      `yaml:"max_count" json:"max_count"`     // 최대 중복 허용 횟수
}

// WebhookBatchingConfig 배치 전송 설정
type WebhookBatchingConfig struct {
	Enabled           bool   `yaml:"enabled" json:"enabled"`                       // 배치 전송 활성화
	MaxSize           int    `yaml:"max_size" json:"max_size"`                     // 배치 최대 크기
	MaxWaitTime       string `yaml:"max_wait_time" json:"max_wait_time"`           // 최대 대기 시간
	FlushInterval     string `yaml:"flush_interval" json:"flush_interval"`         // 강제 플러시 간격
	GroupBy           string `yaml:"group_by" json:"group_by"`                     // 그룹화 기준 (endpoint, type, level)
	CompressionFormat string `yaml:"compression_format" json:"compression_format"` // 압축 포맷 (none, gzip, zstd)
	RetryFailedBatch  bool   `yaml:"retry_failed_batch" json:"retry_failed_batch"` // 실패한 배치 재시도
}

// WebhookBufferingConfig 이벤트 버퍼링 설정
type WebhookBufferingConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`                 // 버퍼링 활성화
	BufferSize     int    `yaml:"buffer_size" json:"buffer_size"`         // 버퍼 크기
	FlushInterval  string `yaml:"flush_interval" json:"flush_interval"`   // 플러시 간격
	FlushThreshold int    `yaml:"flush_threshold" json:"flush_threshold"` // 플러시 임계값
	BatchSize      int    `yaml:"batch_size" json:"batch_size"`           // 배치 크기
	PersistBuffer  bool   `yaml:"persist_buffer" json:"persist_buffer"`   // 버퍼 영속화 여부
	BufferPath     string `yaml:"buffer_path" json:"buffer_path"`         // 버퍼 저장 경로
}

// WebhookSecurityConfig 웹훅 보안 설정
type WebhookSecurityConfig struct {
	EnableTLS        bool     `yaml:"enable_tls" json:"enable_tls"`               // TLS 활성화
	VerifySSL        bool     `yaml:"verify_ssl" json:"verify_ssl"`               // SSL 인증서 검증
	CACertPath       string   `yaml:"ca_cert_path" json:"ca_cert_path"`           // CA 인증서 경로
	ClientCertPath   string   `yaml:"client_cert_path" json:"client_cert_path"`   // 클라이언트 인증서 경로
	ClientKeyPath    string   `yaml:"client_key_path" json:"client_key_path"`     // 클라이언트 키 경로
	AllowedHosts     []string `yaml:"allowed_hosts" json:"allowed_hosts"`         // 허용된 호스트 목록
	BlockedHosts     []string `yaml:"blocked_hosts" json:"blocked_hosts"`         // 차단된 호스트 목록
	MaxPayloadSize   int64    `yaml:"max_payload_size" json:"max_payload_size"`   // 최대 페이로드 크기
	EncryptPayload   bool     `yaml:"encrypt_payload" json:"encrypt_payload"`     // 페이로드 암호화
	EncryptionKey    string   `yaml:"encryption_key" json:"encryption_key"`       // 암호화 키
	SignPayload      bool     `yaml:"sign_payload" json:"sign_payload"`           // 페이로드 서명
	SigningKey       string   `yaml:"signing_key" json:"signing_key"`             // 서명 키
	SigningAlgorithm string   `yaml:"signing_algorithm" json:"signing_algorithm"` // 서명 알고리즘
}

// WebhookMonitoringConfig 웹훅 모니터링 설정
type WebhookMonitoringConfig struct {
	Enabled             bool      `yaml:"enabled" json:"enabled"`                             // 모니터링 활성화
	MetricsInterval     string    `yaml:"metrics_interval" json:"metrics_interval"`           // 메트릭 수집 간격
	EnableHistogram     bool      `yaml:"enable_histogram" json:"enable_histogram"`           // 히스토그램 메트릭 활성화
	HistogramBuckets    []float64 `yaml:"histogram_buckets" json:"histogram_buckets"`         // 히스토그램 버킷
	LogSuccessful       bool      `yaml:"log_successful" json:"log_successful"`               // 성공한 요청 로깅
	LogFailed           bool      `yaml:"log_failed" json:"log_failed"`                       // 실패한 요청 로깅
	LogLevel            string    `yaml:"log_level" json:"log_level"`                         // 로그 레벨
	StatsRetention      string    `yaml:"stats_retention" json:"stats_retention"`             // 통계 보관 기간
	HealthCheckURL      string    `yaml:"health_check_url" json:"health_check_url"`           // 웹훅 헬스체크 URL
	HealthCheckInterval string    `yaml:"health_check_interval" json:"health_check_interval"` // 헬스체크 간격
	AlertOnFailureRate  float64   `yaml:"alert_on_failure_rate" json:"alert_on_failure_rate"` // 실패율 알림 임계값
}

// ConfigExists 웹훅 설정 파일 존재 여부 확인
func (w *WebhookConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "webhook.yaml"))
}

// ReadConfig 웹훅 설정 파일 읽기
func (w *WebhookConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "webhook.yaml"), w); err != nil {
		return err
	}
	return w.Validate()
}

// Validate validates the webhook configuration
func (w *WebhookConfig) Validate() error {
	// Validate struct tags
	if err := ValidateStruct(w); err != nil {
		return err
	}

	// Additional custom validation
	if w.Enabled && len(w.Endpoints) == 0 {
		return helpers.NewConfigFieldError("webhook", "at least one endpoint must be configured when webhook is enabled")
	}

	// Validate rate limit values
	if w.RateLimit.Enabled {
		if w.RateLimit.MaxPerSecond <= 0 && w.RateLimit.MaxPerMinute <= 0 && w.RateLimit.MaxPerHour <= 0 {
			return helpers.NewConfigFieldError("webhook",
				"at least one rate limit must be greater than 0 when rate limiting is enabled")
		}
	}

	return nil
}

// GetDefaultWebhookConfig 기본 웹훅 설정 반환
func GetDefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		Enabled: false,
		Endpoints: []WebhookEndpointConfig{
			{
				Name:    "default",
				URL:     "",
				Enabled: false,
				Method:  "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"User-Agent":   "ProxyND-Webhook/1.0",
				},
				Timeout:    "30s",
				EventTypes: []string{"*"}, // 모든 이벤트 타입
				Format:     "json",
				Credentials: WebhookCredentials{
					Type: "none",
				},
				Filters: WebhookEndpointFilters{
					MinLevel: "INFO",
				},
			},
		},
		RateLimit: WebhookRateLimitConfig{
			Enabled:         true,
			MaxPerSecond:    10,
			MaxPerMinute:    100,
			MaxPerHour:      1000,
			BurstSize:       20,
			WindowDuration:  "1m",
			BackoffStrategy: "exponential",
			MaxBackoffDelay: "5m",
		},
		Retry: WebhookRetryConfig{
			Enabled:       true,
			MaxAttempts:   3,
			InitialDelay:  "1s",
			MaxDelay:      "30s",
			BackoffFactor: 2.0,
			RetryableStatus: []int{
				408, // Request Timeout
				429, // Too Many Requests
				500, // Internal Server Error
				502, // Bad Gateway
				503, // Service Unavailable
				504, // Gateway Timeout
			},
			PersistFailures: true,
		},
		FailureStorage: WebhookFailureStorageConfig{
			Enabled:         true,
			StorageDir:      "/tmp/proxynd-webhook-failures",
			RetentionHours:  72,      // 3일
			MaxFileSize:     1048576, // 1MB
			MaxFiles:        1000,
			CleanupInterval: "1h",
			CompressOld:     true,
			EncryptStorage:  false,
		},
		EventFilter: WebhookEventFilter{
			Enabled:      true,
			DefaultLevel: "INFO",
			LevelOverrides: map[string]string{
				"security.*": "WARNING",
				"system.*":   "ERROR",
			},
			Aggregation: WebhookAggregationConfig{
				Enabled:    false,
				WindowSize: "5m",
				MaxEvents:  100,
				GroupBy:    []string{"type", "level"},
				Strategies: map[string]string{
					"cache.*":    "count",
					"auth.*":     "sample",
					"security.*": "merge",
				},
				FlushTriggers: []string{"time", "critical"},
			},
			Deduplication: WebhookDeduplicationConfig{
				Enabled:    true,
				WindowSize: "1m",
				KeyFields:  []string{"type", "source", "message"},
				Strategy:   "count",
				MaxCount:   5,
			},
		},
		Buffering: WebhookBufferingConfig{
			Enabled:        true,
			BufferSize:     1000,
			FlushInterval:  "10s",
			FlushThreshold: 100,
			BatchSize:      50,
			PersistBuffer:  true,
			BufferPath:     "/tmp/proxynd-webhook-buffer",
		},
		Batching: WebhookBatchingConfig{
			Enabled:           true,
			MaxSize:           10,
			MaxWaitTime:       "5s",
			FlushInterval:     "30s",
			GroupBy:           "endpoint",
			CompressionFormat: "gzip",
			RetryFailedBatch:  true,
		},
		Security: WebhookSecurityConfig{
			EnableTLS:        true,
			VerifySSL:        true,
			MaxPayloadSize:   1048576, // 1MB
			EncryptPayload:   false,
			SignPayload:      false,
			SigningAlgorithm: "sha256",
		},
		Monitoring: WebhookMonitoringConfig{
			Enabled:         true,
			MetricsInterval: "30s",
			EnableHistogram: true,
			HistogramBuckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
			},
			LogSuccessful:       false,
			LogFailed:           true,
			LogLevel:            "INFO",
			StatsRetention:      "24h",
			HealthCheckInterval: "5m",
			AlertOnFailureRate:  0.1, // 10% 실패율에서 알림
		},
	}
}
