// Package helpers provides test helper utilities
package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Fixtures 테스트 데이터 관리자
type Fixtures struct {
	baseDir string
}

// NewFixtures Fixtures 인스턴스 생성
func NewFixtures(baseDir string) *Fixtures {
	return &Fixtures{baseDir: baseDir}
}

// LoadJSON JSON 파일에서 테스트 데이터 로드
func (f *Fixtures) LoadJSON(t *testing.T, filename string, dest interface{}) {
	t.Helper()

	filePath := filepath.Join(f.baseDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read fixture file %s: %v", filePath, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("Failed to unmarshal JSON from %s: %v", filePath, err)
	}
}

// LoadString 텍스트 파일에서 문자열 로드
func (f *Fixtures) LoadString(t *testing.T, filename string) string {
	t.Helper()

	filePath := filepath.Join(f.baseDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read fixture file %s: %v", filePath, err)
	}

	return string(data)
}

// LoadBytes 바이너리 파일에서 바이트 로드
func (f *Fixtures) LoadBytes(t *testing.T, filename string) []byte {
	t.Helper()

	filePath := filepath.Join(f.baseDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read fixture file %s: %v", filePath, err)
	}

	return data
}

// SaveJSON 테스트 데이터를 JSON 파일로 저장
func (f *Fixtures) SaveJSON(t *testing.T, filename string, data interface{}) {
	t.Helper()

	if err := os.MkdirAll(f.baseDir, 0o755); err != nil {
		t.Fatalf("Failed to create fixtures directory: %v", err)
	}

	filePath := filepath.Join(f.baseDir, filename)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0o644); err != nil {
		t.Fatalf("Failed to write fixture file %s: %v", filePath, err)
	}
}

// SaveString 문자열을 텍스트 파일로 저장
func (f *Fixtures) SaveString(t *testing.T, filename, content string) {
	t.Helper()

	if err := os.MkdirAll(f.baseDir, 0o755); err != nil {
		t.Fatalf("Failed to create fixtures directory: %v", err)
	}

	filePath := filepath.Join(f.baseDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write fixture file %s: %v", filePath, err)
	}
}

// GetPath 픽스처 파일의 전체 경로 반환
func (f *Fixtures) GetPath(filename string) string {
	return filepath.Join(f.baseDir, filename)
}

// Exists 픽스처 파일 존재 여부 확인
func (f *Fixtures) Exists(filename string) bool {
	filePath := filepath.Join(f.baseDir, filename)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// 표준 픽스처 데이터 생성 함수들

// CreateStandardFixtures 표준 테스트 픽스처 생성
func CreateStandardFixtures(fixturesDir string) error {
	fixtures := NewFixtures(fixturesDir)
	_ = fixtures // 현재는 미사용이지만 향후 확장 예정

	// 샘플 설정 파일들
	configs := map[string]interface{}{
		"global.json": map[string]interface{}{
			"server": map[string]interface{}{
				"port": 8080,
				"host": "0.0.0.0",
			},
			"logging": map[string]interface{}{
				"level":  "info",
				"format": "json",
			},
		},
		"apt-proxy.json": map[string]interface{}{
			"enabled":   true,
			"upstream":  "http://archive.ubuntu.com/ubuntu",
			"cache_ttl": 3600,
		},
		"npm-proxy.json": map[string]interface{}{
			"enabled":   true,
			"upstream":  "https://registry.npmjs.org",
			"cache_ttl": 1800,
		},
	}

	for filename, config := range configs {
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %v", filename, err)
		}

		filePath := filepath.Join(fixturesDir, filename)
		if err := os.WriteFile(filePath, jsonData, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %v", filename, err)
		}
	}

	// 샘플 패키지 메타데이터
	packageData := map[string]string{
		"package.json": `{
  "name": "test-package",
  "version": "1.0.0",
  "description": "Test package for ProxyND",
  "main": "index.js"
}`,
		"deb-package.txt": `Package: test-package
Version: 1.0.0-1
Architecture: amd64
Description: Test Debian package`,
		"release-file.txt": `Origin: Ubuntu
Label: Ubuntu
Suite: jammy
Codename: jammy
Date: Thu, 01 Jan 2024 00:00:00 UTC`,
	}

	for filename, content := range packageData {
		filePath := filepath.Join(fixturesDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %v", filename, err)
		}
	}

	return nil
}

// LoadTestConfig 표준 테스트 설정 로드
func LoadTestConfig(t *testing.T, configType string) map[string]interface{} {
	t.Helper()

	fixturesDir := filepath.Join("testdata", "fixtures")
	fixtures := NewFixtures(fixturesDir)

	var config map[string]interface{}
	fixtures.LoadJSON(t, fmt.Sprintf("%s.json", configType), &config)

	return config
}

// CreateTempFixture 임시 픽스처 파일 생성 (테스트 종료 시 자동 삭제)
func CreateTempFixture(t *testing.T, filename, content string) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "proxynd-fixture-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	filePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp fixture: %v", err)
	}

	return filePath
}
