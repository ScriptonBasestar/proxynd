package proxy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"proxynd/configs"
)

// TestMavenProxyConfig_Validate tests Maven proxy configuration validation
func TestMavenProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *configs.MavenProxyConfig
		expectError bool
		errorField  string
	}{
		{
			name: "valid config",
			config: &configs.MavenProxyConfig{
				Path:     "proxy/maven",
				UseCache: true,
				Proxies: []configs.MavenProxyServer{
					{
						Name:    "central",
						URL:     "https://repo1.maven.org/maven2",
						Enabled: true,
					},
				},
				Cache: configs.MavenProxyCacheConfig{
					Enabled: true,
				},
			},
			expectError: false,
		},
		{
			name: "empty path",
			config: &configs.MavenProxyConfig{
				Path: "",
				Proxies: []configs.MavenProxyServer{
					{Name: "central", URL: "https://repo1.maven.org/maven2"},
				},
			},
			expectError: true,
			errorField:  "path",
		},
		{
			name: "no proxies",
			config: &configs.MavenProxyConfig{
				Path:    "proxy/maven",
				Proxies: []configs.MavenProxyServer{},
			},
			expectError: true,
			errorField:  "proxies",
		},
		{
			name: "proxy without name",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{Name: "", URL: "https://repo1.maven.org/maven2"},
				},
			},
			expectError: true,
			errorField:  "proxies[0].name",
		},
		{
			name: "proxy without URL",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{Name: "central", URL: ""},
				},
			},
			expectError: true,
			errorField:  "proxies[0].url",
		},
		{
			name: "invalid URL format",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{Name: "central", URL: "invalid-url"},
				},
			},
			expectError: true,
			errorField:  "proxies[0].url",
		},
		{
			name: "username without password",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{
						Name: "central",
						URL:  "https://repo1.maven.org/maven2",
						BasicAuth: configs.BasicAuth{
							Username: "user",
							Password: "",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies[0].basic_auth.password",
		},
		{
			name: "valid basic auth",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{
						Name: "private-repo",
						URL:  "https://private.maven.org/maven2",
						BasicAuth: configs.BasicAuth{
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple valid proxies",
			config: &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{Name: "central", URL: "https://repo1.maven.org/maven2"},
					{Name: "spring", URL: "https://repo.spring.io/release"},
					{Name: "jcenter", URL: "https://jcenter.bintray.com"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					assert.Contains(t, err.Error(), tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMavenProxyServer_Validation tests individual server validation
func TestMavenProxyServer_Validation(t *testing.T) {
	tests := []struct {
		name   string
		server configs.MavenProxyServer
		valid  bool
	}{
		{
			name: "valid server",
			server: configs.MavenProxyServer{
				Name:    "central",
				URL:     "https://repo1.maven.org/maven2",
				Enabled: true,
			},
			valid: true,
		},
		{
			name: "server with description",
			server: configs.MavenProxyServer{
				Name:        "central",
				URL:         "https://repo1.maven.org/maven2",
				Description: "Maven Central Repository",
				Enabled:     true,
			},
			valid: true,
		},
		{
			name: "server with ID",
			server: configs.MavenProxyServer{
				ID:      "central-repo",
				Name:    "central",
				URL:     "https://repo1.maven.org/maven2",
				Enabled: true,
			},
			valid: true,
		},
		{
			name: "disabled server",
			server: configs.MavenProxyServer{
				Name:    "disabled-repo",
				URL:     "https://disabled.maven.org/maven2",
				Enabled: false,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &configs.MavenProxyConfig{
				Path:    "proxy/maven",
				Proxies: []configs.MavenProxyServer{tt.server},
			}

			err := config.Validate()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestBasicAuth_Validation tests basic auth configuration
func TestBasicAuth_Validation(t *testing.T) {
	tests := []struct {
		name        string
		auth        configs.BasicAuth
		expectError bool
	}{
		{
			name:        "no auth",
			auth:        configs.BasicAuth{},
			expectError: false,
		},
		{
			name: "complete auth",
			auth: configs.BasicAuth{
				Username: "user",
				Password: "pass",
			},
			expectError: false,
		},
		{
			name: "username only",
			auth: configs.BasicAuth{
				Username: "user",
				Password: "",
			},
			expectError: true,
		},
		{
			name: "password only (unusual but allowed)",
			auth: configs.BasicAuth{
				Username: "",
				Password: "pass",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &configs.MavenProxyConfig{
				Path: "proxy/maven",
				Proxies: []configs.MavenProxyServer{
					{
						Name:      "test",
						URL:       "https://test.maven.org/maven2",
						BasicAuth: tt.auth,
					},
				},
			}

			err := config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
