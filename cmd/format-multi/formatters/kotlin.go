package formatters

import (
	"fmt"
	"os/exec"
	"strings"
)

// KotlinFormatter 는 Kotlin 언어 포맷터
type KotlinFormatter struct{}

func (f *KotlinFormatter) Name() string {
	return "ktfmt"
}

func (f *KotlinFormatter) Language() string {
	return "Kotlin"
}

func (f *KotlinFormatter) IsAvailable() bool {
	_, err := exec.LookPath("ktfmt")
	return err == nil
}

func (f *KotlinFormatter) Install() error {
	// ktfmt는 수동 설치가 필요함
	return fmt.Errorf("ktfmt must be installed manually. Visit: https://github.com/facebook/ktfmt")
}

func (f *KotlinFormatter) Format(filename string, config interface{}) error {
	cmd := exec.Command("ktfmt", filename)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ktfmt failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
