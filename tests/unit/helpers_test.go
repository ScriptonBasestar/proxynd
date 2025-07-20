package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/helpers"
)

// TestYAMLHelper YAML 헬퍼 함수 테스트
func TestYAMLHelper(t *testing.T) {
	// 테스트용 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "proxynd-yaml-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("Write and Read YAML", func(t *testing.T) {
		// 테스트 데이터 구조체
		type TestConfig struct {
			Name     string            `yaml:"name"`
			Version  string            `yaml:"version"`
			Settings map[string]string `yaml:"settings"`
		}

		testData := TestConfig{
			Name:    "test-config",
			Version: "1.0.0",
			Settings: map[string]string{
				"debug":   "true",
				"timeout": "30s",
			},
		}

		// YAML 파일 경로
		yamlPath := filepath.Join(tempDir, "test-config.yaml")

		// YAML 파일 쓰기
		err := helpers.WriteYaml(yamlPath, testData)
		require.NoError(t, err)

		// 파일이 생성되었는지 확인
		_, err = os.Stat(yamlPath)
		require.NoError(t, err)

		// YAML 파일 읽기
		var readData TestConfig
		err = helpers.ReadYamlSafe(yamlPath, &readData)
		require.NoError(t, err)

		// 데이터 비교
		assert.Equal(t, testData.Name, readData.Name)
		assert.Equal(t, testData.Version, readData.Version)
		assert.Equal(t, testData.Settings, readData.Settings)
	})

	t.Run("Read Non-existent File", func(t *testing.T) {
		var data map[string]interface{}
		err := helpers.ReadYamlSafe("non-existent.yaml", &data)
		require.Error(t, err)
	})

	t.Run("Read Invalid YAML", func(t *testing.T) {
		// 잘못된 YAML 파일 생성
		invalidYAMLPath := filepath.Join(tempDir, "invalid.yaml")
		invalidYAML := `
name: test
  invalid: yaml structure
    - missing proper indentation
`
		err := os.WriteFile(invalidYAMLPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		var data map[string]interface{}
		err = helpers.ReadYamlSafe(invalidYAMLPath, &data)
		require.Error(t, err)
	})

	t.Run("Write to Read-only Directory", func(t *testing.T) {
		// 읽기 전용 디렉토리 생성
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0444) // 읽기 전용
		require.NoError(t, err)

		testData := map[string]string{"test": "data"}
		yamlPath := filepath.Join(readOnlyDir, "test.yaml")

		err = helpers.WriteYaml(yamlPath, testData)
		require.Error(t, err)
	})

	t.Run("Complex Data Structure", func(t *testing.T) {
		// 복잡한 데이터 구조체
		type ComplexConfig struct {
			Server struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
				TLS  struct {
					Enabled  bool   `yaml:"enabled"`
					CertFile string `yaml:"cert_file"`
					KeyFile  string `yaml:"key_file"`
				} `yaml:"tls"`
			} `yaml:"server"`
			Database struct {
				Type       string                 `yaml:"type"`
				Host       string                 `yaml:"host"`
				Port       int                    `yaml:"port"`
				Name       string                 `yaml:"name"`
				Users      []string               `yaml:"users"`
				Parameters map[string]interface{} `yaml:"parameters"`
			} `yaml:"database"`
		}

		complexData := ComplexConfig{}
		complexData.Server.Host = "localhost"
		complexData.Server.Port = 8080
		complexData.Server.TLS.Enabled = true
		complexData.Server.TLS.CertFile = "/path/to/cert.pem"
		complexData.Server.TLS.KeyFile = "/path/to/key.pem"
		complexData.Database.Type = "postgresql"
		complexData.Database.Host = "db.example.com"
		complexData.Database.Port = 5432
		complexData.Database.Name = "proxynd"
		complexData.Database.Users = []string{"admin", "readonly"}
		complexData.Database.Parameters = map[string]interface{}{
			"max_connections": 100,
			"timeout":         "30s",
			"ssl_mode":        "require",
		}

		yamlPath := filepath.Join(tempDir, "complex-config.yaml")

		// 쓰기
		err := helpers.WriteYaml(yamlPath, complexData)
		require.NoError(t, err)

		// 읽기
		var readData ComplexConfig
		err = helpers.ReadYamlSafe(yamlPath, &readData)
		require.NoError(t, err)

		// 비교
		assert.Equal(t, complexData.Server.Host, readData.Server.Host)
		assert.Equal(t, complexData.Server.Port, readData.Server.Port)
		assert.Equal(t, complexData.Server.TLS.Enabled, readData.Server.TLS.Enabled)
		assert.Equal(t, complexData.Database.Type, readData.Database.Type)
		assert.Equal(t, complexData.Database.Users, readData.Database.Users)
		assert.Equal(t, complexData.Database.Parameters["max_connections"], readData.Database.Parameters["max_connections"])
	})
}

// TestURLHelper URL 헬퍼 함수 테스트
func TestURLHelper(t *testing.T) {
	t.Run("Join URL Paths", func(t *testing.T) {
		testCases := []struct {
			name     string
			base     string
			path     string
			expected string
		}{
			{
				name:     "Simple join",
				base:     "https://registry.npmjs.org",
				path:     "express",
				expected: "https://registry.npmjs.org/express",
			},
			{
				name:     "Base with trailing slash",
				base:     "https://registry.npmjs.org/",
				path:     "express",
				expected: "https://registry.npmjs.org/express",
			},
			{
				name:     "Path with leading slash",
				base:     "https://registry.npmjs.org",
				path:     "/express",
				expected: "https://registry.npmjs.org/express",
			},
			{
				name:     "Both with slashes",
				base:     "https://registry.npmjs.org/",
				path:     "/express",
				expected: "https://registry.npmjs.org/express",
			},
			{
				name:     "Empty path",
				base:     "https://registry.npmjs.org",
				path:     "",
				expected: "https://registry.npmjs.org",
			},
			{
				name:     "Complex path",
				base:     "https://pypi.org",
				path:     "simple/requests/requests-2.28.2.tar.gz",
				expected: "https://pypi.org/simple/requests/requests-2.28.2.tar.gz",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := helpers.JoinURL(tc.base, tc.path)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Clean URL Path", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "Double slashes",
				input:    "/npm//express///4.18.2",
				expected: "/npm/express/4.18.2",
			},
			{
				name:     "Dot segments",
				input:    "/npm/./express/../lodash",
				expected: "/npm/lodash",
			},
			{
				name:     "Mixed issues",
				input:    "//npm/./express///../lodash/",
				expected: "/npm/lodash/",
			},
			{
				name:     "Already clean",
				input:    "/npm/express/4.18.2",
				expected: "/npm/express/4.18.2",
			},
			{
				name:     "Root path",
				input:    "/",
				expected: "/",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := helpers.CleanPath(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Parse Package Name", func(t *testing.T) {
		testCases := []struct {
			name         string
			packagePath  string
			expectedName string
			expectedVer  string
		}{
			{
				name:         "Simple package",
				packagePath:  "express",
				expectedName: "express",
				expectedVer:  "",
			},
			{
				name:         "Package with version",
				packagePath:  "express/4.18.2",
				expectedName: "express",
				expectedVer:  "4.18.2",
			},
			{
				name:         "Scoped package",
				packagePath:  "@types/node",
				expectedName: "@types/node",
				expectedVer:  "",
			},
			{
				name:         "Scoped package with version",
				packagePath:  "@types/node/18.15.0",
				expectedName: "@types/node",
				expectedVer:  "18.15.0",
			},
			{
				name:         "Complex path",
				packagePath:  "@babel/core/7.21.0/babel-core-7.21.0.tgz",
				expectedName: "@babel/core",
				expectedVer:  "7.21.0",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				name, version := helpers.ParsePackageName(tc.packagePath)
				assert.Equal(t, tc.expectedName, name)
				assert.Equal(t, tc.expectedVer, version)
			})
		}
	})

	t.Run("Validate URL", func(t *testing.T) {
		validURLs := []string{
			"https://registry.npmjs.org",
			"http://archive.ubuntu.com",
			"https://pypi.org/simple",
			"https://registry-1.docker.io",
		}

		invalidURLs := []string{
			"not-a-url",
			"ftp://invalid.scheme",
			"",
			"https://",
			"http://.com",
		}

		for _, url := range validURLs {
			t.Run("Valid: "+url, func(t *testing.T) {
				assert.True(t, helpers.IsValidURL(url))
			})
		}

		for _, url := range invalidURLs {
			t.Run("Invalid: "+url, func(t *testing.T) {
				assert.False(t, helpers.IsValidURL(url))
			})
		}
	})

	t.Run("URL Encoding", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "Scoped package",
				input:    "@types/node",
				expected: "%40types%2Fnode",
			},
			{
				name:     "Special characters",
				input:    "package-name_v1.0.0",
				expected: "package-name_v1.0.0",
			},
			{
				name:     "Spaces",
				input:    "package name",
				expected: "package%20name",
			},
			{
				name:     "Already encoded",
				input:    "%40types%2Fnode",
				expected: "%2540types%252Fnode",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := helpers.URLEncode(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}
