// Package middlewares provides HTTP middleware components for the ProxyND server
package middlewares

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AccessLogEntry 액세스 로그 항목 구조체
type AccessLogEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	ClientIP      string    `json:"client_ip"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	ResponseTime  int64     `json:"response_time_ms"`
	CacheStatus   string    `json:"cache_status"`
	ProxyType     string    `json:"proxy_type,omitempty"`
	PackagePath   string    `json:"package_path,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	ContentLength int64     `json:"content_length"`
	Username      string    `json:"username,omitempty"`
	HashVerified  bool      `json:"hash_verified,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// AccessLogConfig 액세스 로그 설정
type AccessLogConfig struct {
	LogDir        string // 로그 디렉토리
	LogFile       string // 로그 파일명
	Format        string // 로그 포맷 (json, text)
	RotateDaily   bool   // 일별 로그 파일 분할
	ConsoleOutput bool   // 콘솔 출력 여부
}

// AccessLogger 액세스 로거
type AccessLogger struct {
	config   AccessLogConfig
	logFile  *os.File
	mu       sync.Mutex
	lastDate string
}

// NewAccessLogger 새 액세스 로거 생성
func NewAccessLogger(config AccessLogConfig) *AccessLogger {
	// 기본값 설정
	if config.LogDir == "" {
		config.LogDir = "./logs"
	}
	if config.LogFile == "" {
		config.LogFile = "access.log"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	logger := &AccessLogger{
		config: config,
	}

	// 로그 파일 초기화
	logger.initLogFile()

	return logger
}

// initLogFile 로그 파일 초기화
func (l *AccessLogger) initLogFile() {
	// 로그 디렉토리 생성
	if err := os.MkdirAll(l.config.LogDir, 0750); err != nil {
		// 디렉토리 생성 실패 시 에러 무시 (파일 로깅 비활성화)
		return
	}

	// 로그 파일명 결정
	filename := l.config.LogFile
	if l.config.RotateDaily {
		today := time.Now().Format("2006-01-02")
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		filename = fmt.Sprintf("%s-%s%s", base, today, ext)
		l.lastDate = today
	}

	// 로그 파일 열기
	logPath := filepath.Join(l.config.LogDir, filename)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		// 로그 파일 열기 실패 시 stderr에 에러 출력
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open access log file: %v\n", err)
		return
	}

	// 기존 파일 닫기
	if l.logFile != nil {
		_ = l.logFile.Close()
	}

	l.logFile = file
}

// checkRotate 일별 로테이션 확인
func (l *AccessLogger) checkRotate() {
	if !l.config.RotateDaily {
		return
	}

	today := time.Now().Format("2006-01-02")
	if l.lastDate != today {
		l.initLogFile()
	}
}

// Log 액세스 로그 기록
func (l *AccessLogger) Log(entry AccessLogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 로테이션 확인
	l.checkRotate()

	var logLine string

	// 로그 포맷에 따라 출력 생성
	switch l.config.Format {
	case "json":
		data, err := json.Marshal(entry)
		if err != nil {
			data = []byte(fmt.Sprintf(`{"error": "Failed to marshal log entry: %v"}`, err))
		}
		logLine = string(data) + "\n"
	default: // text
		logLine = fmt.Sprintf("[%s] %s %s %s %d %dms %s %s %s\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.ClientIP,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.ResponseTime,
			entry.CacheStatus,
			entry.ProxyType,
			entry.Username,
		)
	}

	// 콘솔 출력
	if l.config.ConsoleOutput {
		_, _ = os.Stdout.WriteString(logLine)
	}

	// 파일 기록
	if l.logFile != nil {
		if _, err := l.logFile.WriteString(logLine); err != nil {
			// 로그 쓰기 실패 시 에러 무시 (로그가 중요하지 않은 경우)
			_ = err // 에러를 명시적으로 무시
		}
		if err := l.logFile.Sync(); err != nil {
			// 파일 동기화 실패 시 에러 무시
			_ = err // 에러를 명시적으로 무시
		}
	}
}

// Close 로거 종료
func (l *AccessLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		_ = l.logFile.Close()
		l.logFile = nil
	}
}

// AccessLogMiddleware 액세스 로그 미들웨어
func AccessLogMiddleware(config AccessLogConfig) fiber.Handler {
	logger := NewAccessLogger(config)

	return func(c *fiber.Ctx) error {
		// 요청 시작 시간
		start := time.Now()

		// 다음 핸들러 실행
		err := c.Next()

		// 로그 항목 생성
		entry := AccessLogEntry{
			Timestamp:     start,
			ClientIP:      c.IP(),
			Method:        c.Method(),
			Path:          c.Path(),
			StatusCode:    c.Response().StatusCode(),
			ResponseTime:  time.Since(start).Milliseconds(),
			ContentLength: int64(len(c.Response().Body())),
			UserAgent:     c.Get("User-Agent"),
		}

		// 캐시 정보
		if cacheHit, ok := c.Locals("cache_hit").(bool); ok && cacheHit {
			entry.CacheStatus = "HIT"
		} else {
			entry.CacheStatus = "MISS"
		}

		// 프록시 정보
		if proxyType := c.Params("type"); proxyType != "" {
			entry.ProxyType = proxyType
		}
		if packagePath := c.Params("*"); packagePath != "" {
			entry.PackagePath = packagePath
		}

		// 인증 정보
		if username, ok := c.Locals("username").(string); ok {
			entry.Username = username
		}

		// 해시 검증 정보
		if hashVerified, ok := c.Locals("hashVerified").(bool); ok {
			entry.HashVerified = hashVerified
		}

		// 오류 정보
		if err != nil {
			entry.ErrorMessage = err.Error()
		}

		// 로그 기록
		logger.Log(entry)

		return err
	}
}

// DefaultAccessLogMiddleware 기본 액세스 로그 미들웨어 (기존 함수 호환성 유지)
func DefaultAccessLogMiddleware() fiber.Handler {
	return AccessLogMiddleware(AccessLogConfig{
		Format:        "text",
		ConsoleOutput: true,
	})
}
