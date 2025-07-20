package plugins

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// OperationMode 운영 모드 정의
type OperationMode string

const (
	// ProxyMode is exported
	// ProxyMode is exported
	ProxyMode OperationMode = "proxy" // On-Demand: 요청 시점에 업스트림에서 받아오는 방식
	// MirrorMode is a const that mirror mode
	MirrorMode OperationMode = "mirror" // Pre-fetch: 모든 패키지를 미리 받아놓는 방식
)

// CachePolicy 캐시 정책 정의
type CachePolicy struct {
	TTL              time.Duration // 캐시 유지 시간
	MaxSize          int64         // 최대 캐시 크기 (bytes)
	EnableGzipCache  bool          // gzip 압축 캐시 여부
	CacheHeaders     []string      // 캐시할 헤더 목록
	SkipCacheHeaders []string      // 캐시하지 않을 헤더 목록
}

// UpstreamConfig 업스트림 설정
type UpstreamConfig struct {
	URL      string            `json:"url" yaml:"url"`
	Priority int               `json:"priority" yaml:"priority"`
	Timeout  time.Duration     `json:"timeout" yaml:"timeout"`
	Headers  map[string]string `json:"headers" yaml:"headers"`
	Username string            `json:"username,omitempty" yaml:"username,omitempty"`
	Password string            `json:"password,omitempty" yaml:"password,omitempty"`
}

// MirrorConfig 미러 설정
type MirrorConfig struct {
	URL            string            `json:"url" yaml:"url"`
	SyncInterval   time.Duration     `json:"sync_interval" yaml:"sync_interval"`
	Headers        map[string]string `json:"headers" yaml:"headers"`
	Username       string            `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string            `json:"password,omitempty" yaml:"password,omitempty"`
	IncludePattern []string          `json:"include_pattern,omitempty" yaml:"include_pattern,omitempty"`
	ExcludePattern []string          `json:"exclude_pattern,omitempty" yaml:"exclude_pattern,omitempty"`
}

// GroupResult 그룹 요청 결과
type GroupResult struct {
	UpstreamURL string            `json:"upstream_url"`
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	Error       error             `json:"error,omitempty"`
	Duration    time.Duration     `json:"duration"`
}

// PackageHandler 핵심 패키지 핸들러 인터페이스
type PackageHandler interface {
	// 기본 정보
	GetType() string    // 패키지 타입 (npm, maven, apt, etc.)
	GetName() string    // 핸들러 이름
	GetVersion() string // 핸들러 버전

	// 운영 모드 지원
	SupportsMode(mode OperationMode) bool

	// 생명주기 관리
	Initialize(config interface{}) error
	Shutdown(ctx context.Context) error

	// 요청 처리
	HandleRequest(ctx *fiber.Ctx, mode OperationMode) error

	// 캐시 정책
	GetCachePolicy() CachePolicy

	// 헬스체크
	HealthCheck(ctx context.Context) error

	// 설정 관리
	ValidateConfig(config interface{}) error
	GetDefaultConfig() interface{}
}

// GroupManager 그룹 매니저 인터페이스
type GroupManager interface {
	// 프록시 모드: 요청 시점에 그룹에서 데이터 가져오기
	FetchOnDemand(ctx context.Context, path string, upstreams []UpstreamConfig) ([]GroupResult, error)

	// 미러 모드: 미리 그룹에서 모든 데이터 동기화
	SyncFromMirrors(ctx context.Context, mirrors []MirrorConfig) error

	// 단일 미러 동기화
	SyncFromSingleMirror(ctx context.Context, mirror MirrorConfig) error

	// 그룹 헬스체크
	HealthCheckGroup(ctx context.Context, upstreams []UpstreamConfig) (map[string]bool, error)
}

// PluginRegistry 플러그인 레지스트리 인터페이스
type PluginRegistry interface {
	// 핸들러 등록/해제
	Register(handler PackageHandler) error
	Unregister(packageType string) error

	// 핸들러 조회
	GetHandler(packageType string) (PackageHandler, bool)
	ListHandlers() []string

	// 모드별 핸들러 조회
	GetHandlersByMode(mode OperationMode) []PackageHandler

	// 상태 관리
	IsRegistered(packageType string) bool
	GetHandlerCount() int
}

// PluginMetadata 플러그인 메타데이터
type PluginMetadata struct {
	Type         string            `json:"type" yaml:"type"`
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	Description  string            `json:"description" yaml:"description"`
	Author       string            `json:"author" yaml:"author"`
	Homepage     string            `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	License      string            `json:"license,omitempty" yaml:"license,omitempty"`
	Tags         []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Config       map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// Plugin 플러그인 인터페이스
type Plugin interface {
	// 메타데이터
	GetMetadata() PluginMetadata

	// 핸들러 생성
	CreateHandler() PackageHandler

	// 플러그인 생명주기
	Load() error
	Unload() error

	// 의존성 관리
	GetDependencies() []string
	CheckDependencies() error
}
