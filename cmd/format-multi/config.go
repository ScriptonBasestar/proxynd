package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 는 전체 설정 구조체
type Config struct {
	Languages map[string]*LanguageConfig `yaml:"languages"`
}

// LanguageConfig 는 언어별 설정
type LanguageConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Tools       []string          `yaml:"tools"`
	Options     map[string]string `yaml:"options"`
	LocalImport string            `yaml:"local_import"` // Go용
	Profile     string            `yaml:"profile"`      // Python isort용
}

// loadConfig 는 설정 파일을 로드
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// 기본값 설정
	if config.Languages == nil {
		config.Languages = make(map[string]*LanguageConfig)
	}

	return &config, nil
}

// getLanguageConfig 는 특정 언어의 설정을 반환
func getLanguageConfig(config *Config, language string) *LanguageConfig {
	if config == nil || config.Languages == nil {
		return nil
	}
	return config.Languages[language]
}

// 기본 설정
func defaultConfig() *Config {
	return &Config{
		Languages: map[string]*LanguageConfig{
			"Go": {
				Enabled:     true,
				Tools:       []string{"gofumpt", "goimports"},
				LocalImport: "proxynd",
			},
			"Python": {
				Enabled: true,
				Tools:   []string{"black", "isort"},
				Profile: "black",
			},
			"JavaScript": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"TypeScript": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"Kotlin": {
				Enabled: true,
				Tools:   []string{"ktfmt"},
			},
			"Shell": {
				Enabled: true,
				Tools:   []string{"shfmt"},
			},
			"YAML": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"JSON": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"Markdown": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"HTML": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
			"CSS": {
				Enabled: true,
				Tools:   []string{"prettier"},
			},
		},
	}
}
