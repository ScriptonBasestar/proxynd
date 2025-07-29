package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// GoFormatter 는 Go 언어 포맷터
type GoFormatter struct{}

// Name 은 포맷터 이름 반환
func (f *GoFormatter) Name() string {
	return "gofumpt + goimports"
}

// Language 는 언어 이름 반환
func (f *GoFormatter) Language() string {
	return "Go"
}

// IsAvailable 은 필요한 도구들이 설치되어 있는지 확인
func (f *GoFormatter) IsAvailable() bool {
	// gofumpt 확인
	if _, err := exec.LookPath("gofumpt"); err != nil {
		return false
	}
	// goimports 확인
	if _, err := exec.LookPath("goimports"); err != nil {
		return false
	}
	return true
}

// Install 은 Go 포맷터들을 설치
func (f *GoFormatter) Install() error {
	// gofumpt 설치
	cmd := exec.Command("go", "install", "mvdan.cc/gofumpt@latest")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install gofumpt: %v\n%s", err, output)
	}

	// goimports 설치
	cmd = exec.Command("go", "install", "golang.org/x/tools/cmd/goimports@latest")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install goimports: %v\n%s", err, output)
	}

	return nil
}

// Format 은 Go 파일을 포맷팅
func (f *GoFormatter) Format(filename string, config interface{}) error {
	// config에서 localImport 추출
	localImport := "proxynd" // 기본값
	if cfg, ok := config.(map[string]interface{}); ok {
		if li, ok := cfg["local_import"].(string); ok {
			localImport = li
		}
	}

	// 1. gofumpt 실행
	cmd := exec.Command("gofumpt", "-w", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gofumpt failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	// 2. goimports 실행
	cmd = exec.Command("goimports", "-w", "-local", localImport, filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("goimports failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	return nil
}
