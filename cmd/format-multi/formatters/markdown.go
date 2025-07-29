package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// MarkdownFormatter 는 Markdown 파일 포맷터
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Name() string {
	return "prettier"
}

func (f *MarkdownFormatter) Language() string {
	return "Markdown"
}

func (f *MarkdownFormatter) IsAvailable() bool {
	_, err := exec.LookPath("prettier")
	return err == nil
}

func (f *MarkdownFormatter) Install() error {
	// JavaScript 포맷터와 동일
	jsFormatter := &JavaScriptFormatter{}
	return jsFormatter.Install()
}

func (f *MarkdownFormatter) Format(filename string, config interface{}) error {
	// prettier with prose wrap
	cmd := exec.Command("prettier", "--write", "--prose-wrap", "always", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// HTMLFormatter 는 HTML 파일 포맷터
type HTMLFormatter struct{}

func (f *HTMLFormatter) Name() string {
	return "prettier"
}

func (f *HTMLFormatter) Language() string {
	return "HTML"
}

func (f *HTMLFormatter) IsAvailable() bool {
	_, err := exec.LookPath("prettier")
	return err == nil
}

func (f *HTMLFormatter) Install() error {
	// JavaScript 포맷터와 동일
	jsFormatter := &JavaScriptFormatter{}
	return jsFormatter.Install()
}

func (f *HTMLFormatter) Format(filename string, config interface{}) error {
	cmd := exec.Command("prettier", "--write", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// CSSFormatter 는 CSS/SCSS/SASS 파일 포맷터
type CSSFormatter struct{}

func (f *CSSFormatter) Name() string {
	return "prettier"
}

func (f *CSSFormatter) Language() string {
	return "CSS"
}

func (f *CSSFormatter) IsAvailable() bool {
	_, err := exec.LookPath("prettier")
	return err == nil
}

func (f *CSSFormatter) Install() error {
	// JavaScript 포맷터와 동일
	jsFormatter := &JavaScriptFormatter{}
	return jsFormatter.Install()
}

func (f *CSSFormatter) Format(filename string, config interface{}) error {
	cmd := exec.Command("prettier", "--write", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prettier failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
