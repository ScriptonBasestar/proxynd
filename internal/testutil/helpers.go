package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateTempDir 테스트용 임시 디렉토리 생성
func CreateTempDir(t *testing.T, prefix string) string {
	dir, err := os.MkdirTemp("", prefix)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// CreateTestFile 테스트 파일 생성
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	err := os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, err)
	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

// CreateMockServer Mock HTTP 서버 생성
func CreateMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateMockServerWithResponses 다중 응답을 가진 Mock 서버 생성
func CreateMockServerWithResponses(responses map[string]MockResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)
	}))
}

// MockResponse Mock 응답 구조체
type MockResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// NewMockResponse Mock 응답 생성 헬퍼
func NewMockResponse(statusCode int, contentType string, body []byte) MockResponse {
	return MockResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body: body,
	}
}

// ReadCloserFromString 문자열로부터 io.ReadCloser 생성
func ReadCloserFromString(s string) io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(s))
}

// ReadCloserFromBytes 바이트 배열로부터 io.ReadCloser 생성
func ReadCloserFromBytes(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewBuffer(b))
}

// AssertFileContents 파일 내용 검증
func AssertFileContents(t *testing.T, path, expected string) {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, expected, string(content))
}

// AssertFileExists 파일 존재 검증
func AssertFileExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.NoError(t, err)
}

// AssertFileNotExists 파일 부재 검증
func AssertFileNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

// GenerateTestData 테스트 데이터 생성
func GenerateTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// CompareReaders 두 Reader의 내용 비교
func CompareReaders(t *testing.T, r1, r2 io.Reader) {
	b1, err := io.ReadAll(r1)
	require.NoError(t, err)
	b2, err := io.ReadAll(r2)
	require.NoError(t, err)
	require.Equal(t, b1, b2)
}
