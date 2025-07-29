package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// PythonFormatter 는 Python 언어 포맷터
type PythonFormatter struct{}

// Name 은 포맷터 이름 반환
func (f *PythonFormatter) Name() string {
	return "black + isort"
}

// Language 는 언어 이름 반환
func (f *PythonFormatter) Language() string {
	return "Python"
}

// IsAvailable 은 필요한 도구들이 설치되어 있는지 확인
func (f *PythonFormatter) IsAvailable() bool {
	// black 확인
	if _, err := exec.LookPath("black"); err != nil {
		return false
	}
	// isort 확인
	if _, err := exec.LookPath("isort"); err != nil {
		return false
	}
	return true
}

// Install 은 Python 포맷터들을 설치
func (f *PythonFormatter) Install() error {
	// pip가 있는지 확인
	pipCmd := "pip"
	if _, err := exec.LookPath("pip"); err != nil {
		// pip3 시도
		if _, err := exec.LookPath("pip3"); err != nil {
			return fmt.Errorf("pip or pip3 not found")
		}
		pipCmd = "pip3"
	}

	// black 설치
	cmd := exec.Command(pipCmd, "install", "black")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install black: %v\n%s", err, output)
	}

	// isort 설치
	cmd = exec.Command(pipCmd, "install", "isort")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install isort: %v\n%s", err, output)
	}

	// autoflake 설치 (선택적)
	cmd = exec.Command(pipCmd, "install", "autoflake")
	if _, err := cmd.CombinedOutput(); err != nil {
		// autoflake는 선택적이므로 에러 무시
		fmt.Printf("Warning: failed to install autoflake: %v\n", err)
	}

	return nil
}

// Format 은 Python 파일을 포맷팅
func (f *PythonFormatter) Format(filename string, config interface{}) error {
	// config에서 profile 추출
	profile := "black" // 기본값
	if cfg, ok := config.(map[string]interface{}); ok {
		if p, ok := cfg["profile"].(string); ok {
			profile = p
		}
	}

	// 1. black 실행
	cmd := exec.Command("black", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		// black은 파일이 이미 포맷되어 있으면 exit code 1을 반환할 수 있음
		outputStr := strings.TrimSpace(string(output))
		if !strings.Contains(outputStr, "already well formatted") {
			return fmt.Errorf("black failed: %v\n%s", err, outputStr)
		}
	}

	// 2. isort 실행
	cmd = exec.Command("isort", filename, "--profile", profile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("isort failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	// 3. autoflake 실행 (선택적)
	if _, err := exec.LookPath("autoflake"); err == nil {
		cmd = exec.Command("autoflake", "--in-place", "--remove-all-unused-imports", "--remove-unused-variables", filename)
		if _, err := cmd.CombinedOutput(); err != nil {
			// autoflake는 선택적이므로 에러를 경고로만 처리
			fmt.Printf("Warning: autoflake failed on %s: %v\n", filename, err)
		}
	}

	return nil
}
