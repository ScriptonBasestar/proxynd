package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// ShellFormatter 는 Shell 스크립트 포맷터
type ShellFormatter struct{}

func (f *ShellFormatter) Name() string {
	return "shfmt"
}

func (f *ShellFormatter) Language() string {
	return "Shell"
}

func (f *ShellFormatter) IsAvailable() bool {
	_, err := exec.LookPath("shfmt")
	return err == nil
}

func (f *ShellFormatter) Install() error {
	// go install 시도
	cmd := exec.Command("go", "install", "mvdan.cc/sh/v3/cmd/shfmt@latest")
	if output, err := cmd.CombinedOutput(); err != nil {
		// brew 시도 (macOS)
		if _, err := exec.LookPath("brew"); err == nil {
			cmd = exec.Command("brew", "install", "shfmt")
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to install shfmt: %v\n%s", err, output)
			}
			return nil
		}
		return fmt.Errorf("failed to install shfmt via go install: %v\n%s", err, output)
	}
	return nil
}

func (f *ShellFormatter) Format(filename string, config interface{}) error {
	// shfmt with 2 space indent (bash style)
	cmd := exec.Command("shfmt", "-i", "2", "-w", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shfmt failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
