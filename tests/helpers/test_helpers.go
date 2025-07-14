package helpers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TestEnvironment 테스트 환경 설정 구조체
type TestEnvironment struct {
	TempDir    string
	ConfigDir  string
	StorageDir string
	CacheDir   string
	LogDir     string
	Cleanup    func()
}

// SetupTestEnvironment 테스트 환경 설정
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "proxynd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 테스트용 하위 디렉토리 생성
	configDir := filepath.Join(tempDir, "config")
	storageDir := filepath.Join(tempDir, "storage")
	cacheDir := filepath.Join(tempDir, "cache")
	logDir := filepath.Join(tempDir, "logs")

	dirs := []string{configDir, storageDir, cacheDir, logDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir) // 정리
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// 환경 변수 설정
	originalEnv := backupEnv([]string{
		"CONFIG_DIR",
		"STORAGE_DIR",
		"CACHE_DIR",
		"LOG_DIR",
		"SERVER_PORT",
		"LOG_LEVEL",
	})

	os.Setenv("CONFIG_DIR", configDir)
	os.Setenv("STORAGE_DIR", storageDir)
	os.Setenv("CACHE_DIR", cacheDir)
	os.Setenv("LOG_DIR", logDir)
	os.Setenv("SERVER_PORT", "0") // 임의 포트 사용
	os.Setenv("LOG_LEVEL", "error") // 테스트 시 로그 최소화

	env := &TestEnvironment{
		TempDir:    tempDir,
		ConfigDir:  configDir,
		StorageDir: storageDir,
		CacheDir:   cacheDir,
		LogDir:     logDir,
		Cleanup: func() {
			restoreEnv(originalEnv)
			os.RemoveAll(tempDir)
		},
	}

	// 테스트 종료 시 정리
	t.Cleanup(env.Cleanup)

	return env
}

// backupEnv 환경 변수 백업
func backupEnv(keys []string) map[string]string {
	backup := make(map[string]string)
	for _, key := range keys {
		backup[key] = os.Getenv(key)
	}
	return backup
}

// restoreEnv 환경 변수 복원
func restoreEnv(backup map[string]string) {
	for key, value := range backup {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// CreateTestConfig 테스트용 설정 파일 생성
func CreateTestConfig(env *TestEnvironment, configName string, content string) error {
	configPath := filepath.Join(env.ConfigDir, configName)
	return os.WriteFile(configPath, []byte(content), 0644)
}

// CreateTestFile 테스트용 파일 생성
func CreateTestFile(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

// MockHTTPServer 테스트용 HTTP 서버 생성
func MockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// CreateTestFiberApp 테스트용 Fiber 앱 생성
func CreateTestFiberApp() *fiber.App {
	return fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
}

// AssertFileExists 파일 존재 확인
func AssertFileExists(t *testing.T, filePath string) {
	t.Helper()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", filePath)
	}
}

// AssertFileNotExists 파일 비존재 확인
func AssertFileNotExists(t *testing.T, filePath string) {
	t.Helper()
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected file %s to not exist, but it does", filePath)
	}
}

// AssertDirectoryExists 디렉토리 존재 확인
func AssertDirectoryExists(t *testing.T, dirPath string) {
	t.Helper()
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist, but it doesn't", dirPath)
		return
	}
	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory, but it's not", dirPath)
	}
}

// AssertFileContent 파일 내용 확인
func AssertFileContent(t *testing.T, filePath, expectedContent string) {
	t.Helper()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", filePath, err)
		return
	}
	if string(content) != expectedContent {
		t.Errorf("File content mismatch in %s.\nExpected: %s\nGot: %s", filePath, expectedContent, string(content))
	}
}

// WaitForCondition 조건이 만족될 때까지 대기
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timer.C:
			t.Fatalf("Timeout waiting for condition: %s", message)
		}
	}
}

// CaptureLogs 로그 캡처 (테스트용)
func CaptureLogs(t *testing.T, fn func()) string {
	t.Helper()
	
	// 임시 파일로 로그 리다이렉트
	tempFile, err := os.CreateTemp("", "test-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 원본 stdout 백업
	originalStdout := os.Stdout
	os.Stdout = tempFile

	// 테스트 함수 실행
	fn()

	// stdout 복원
	os.Stdout = originalStdout

	// 로그 내용 읽기
	tempFile.Seek(0, 0)
	logContent, err := io.ReadAll(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log content: %v", err)
	}

	return string(logContent)
}

// MockProxyUpstream 테스트용 업스트림 서버 모킹
func MockProxyUpstream(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	
	return MockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if response, exists := responses[path]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, response)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not found")
		}
	})
}

// AssertHTTPStatus HTTP 상태 코드 확인
func AssertHTTPStatus(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected HTTP status %d, got %d", expectedStatus, resp.StatusCode)
	}
}

// AssertHTTPHeader HTTP 헤더 확인
func AssertHTTPHeader(t *testing.T, resp *http.Response, headerName, expectedValue string) {
	t.Helper()
	actualValue := resp.Header.Get(headerName)
	if actualValue != expectedValue {
		t.Errorf("Expected header %s to be %s, got %s", headerName, expectedValue, actualValue)
	}
}

// RunWithTimeout 타임아웃이 있는 함수 실행
func RunWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	
	done := make(chan bool, 1)
	
	go func() {
		fn()
		done <- true
	}()
	
	select {
	case <-done:
		// 정상 완료
	case <-time.After(timeout):
		t.Fatalf("Function execution timed out after %v", timeout)
	}
}

// GetFreePort 사용 가능한 포트 찾기
func GetFreePort() int {
	// 간단한 구현 - 실제로는 더 정교한 방법 사용 권장
	return 0 // OS가 자동으로 할당
}