package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	toolPrettier = "prettier"
	toolNPM      = "npm"
	toolYarn     = "yarn"
	toolPNPM     = "pnpm"
)

// JavaScriptFormatter 는 JavaScript/TypeScript 언어 포맷터
type JavaScriptFormatter struct{}

// Name 은 포맷터 이름 반환
func (f *JavaScriptFormatter) Name() string {
	return toolPrettier
}

// Language 는 언어 이름 반환
func (f *JavaScriptFormatter) Language() string {
	return "JavaScript/TypeScript"
}

// IsAvailable 은 필요한 도구들이 설치되어 있는지 확인
func (f *JavaScriptFormatter) IsAvailable() bool {
	// prettier 확인
	if _, err := exec.LookPath(toolPrettier); err != nil {
		return false
	}
	return true
}

// Install 은 JavaScript/TypeScript 포맷터를 설치
func (f *JavaScriptFormatter) Install() error {
	// npm이 있는지 확인
	npmCmd := toolNPM
	if _, err := exec.LookPath(toolNPM); err != nil {
		// yarn 시도
		if _, err := exec.LookPath(toolYarn); err != nil {
			// pnpm 시도
			if _, err := exec.LookPath(toolPNPM); err != nil {
				return fmt.Errorf("npm, yarn, or pnpm not found")
			}
			npmCmd = toolPNPM
		} else {
			npmCmd = toolYarn
		}
	}

	// prettier 전역 설치
	var cmd *exec.Cmd
	switch npmCmd {
	case toolNPM:
		cmd = exec.Command(toolNPM, "install", "-g", toolPrettier)
	case toolYarn:
		cmd = exec.Command(toolYarn, "global", "add", toolPrettier)
	case toolPNPM:
		cmd = exec.Command(toolPNPM, "add", "-g", toolPrettier)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install prettier: %v\n%s", err, output)
	}

	// eslint 설치 (선택적)
	switch npmCmd {
	case "npm":
		cmd = exec.Command("npm", "install", "-g", "eslint")
	case "yarn":
		cmd = exec.Command("yarn", "global", "add", "eslint")
	case "pnpm":
		cmd = exec.Command("pnpm", "add", "-g", "eslint")
	}

	if _, err := cmd.CombinedOutput(); err != nil {
		// eslint는 선택적이므로 에러 무시
		fmt.Printf("Warning: failed to install eslint: %v\n", err)
	}

	return nil
}

// Format 은 JavaScript/TypeScript 파일을 포맷팅
func (f *JavaScriptFormatter) Format(filename string, config interface{}) error {
	// 1. prettier 실행
	cmd := exec.Command(toolPrettier, "--write", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	// 2. eslint --fix 실행 (선택적, .eslintrc가 있는 경우만)
	if _, err := exec.LookPath("eslint"); err == nil {
		// eslint 설정 파일이 있는지 확인
		configFiles := []string{".eslintrc", ".eslintrc.js", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml"}
		hasConfig := false
		for _, cf := range configFiles {
			if _, err := exec.Command("test", "-f", cf).Output(); err == nil {
				hasConfig = true
				break
			}
		}

		if hasConfig {
			cmd = exec.Command("eslint", "--fix", filename)
			if output, err := cmd.CombinedOutput(); err != nil {
				// eslint는 선택적이므로 에러를 경고로만 처리
				outputStr := strings.TrimSpace(string(output))
				if !strings.Contains(outputStr, "no issues found") {
					fmt.Printf("Warning: eslint failed on %s: %v\n", filename, err)
				}
			}
		}
	}

	return nil
}
