package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// YAMLFormatter 는 YAML 파일 포맷터
type YAMLFormatter struct{}

func (f *YAMLFormatter) Name() string {
	return "prettier"
}

func (f *YAMLFormatter) Language() string {
	return "YAML"
}

func (f *YAMLFormatter) IsAvailable() bool {
	_, err := exec.LookPath("prettier")
	return err == nil
}

func (f *YAMLFormatter) Install() error {
	// JavaScript 포맷터와 동일
	jsFormatter := &JavaScriptFormatter{}
	return jsFormatter.Install()
}

func (f *YAMLFormatter) Format(filename string, config interface{}) error {
	cmd := exec.Command("prettier", "--write", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// JSONFormatter 는 JSON 파일 포맷터
type JSONFormatter struct{}

func (f *JSONFormatter) Name() string {
	return "prettier"
}

func (f *JSONFormatter) Language() string {
	return "JSON"
}

func (f *JSONFormatter) IsAvailable() bool {
	_, err := exec.LookPath("prettier")
	return err == nil
}

func (f *JSONFormatter) Install() error {
	// JavaScript 포맷터와 동일
	jsFormatter := &JavaScriptFormatter{}
	return jsFormatter.Install()
}

func (f *JSONFormatter) Format(filename string, config interface{}) error {
	cmd := exec.Command("prettier", "--write", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
