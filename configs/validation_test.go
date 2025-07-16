package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
		Value:   "test_value",
	}
	
	expected := "validation failed for field 'test_field': test message (value: test_value)"
	assert.Equal(t, expected, err.Error())
}

func TestAptProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *AptProxyConfig
		expectError bool
		errorField  string
	}{
		{
			name: "Valid config",
			config: &AptProxyConfig{
				Path:    "/proxy/apt",
				Proxies: map[string][]AptProxy{
					"ubuntu": {
						{
							Name: "Ubuntu Main",
							URL:  "https://mirror.example.com/ubuntu",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty proxies",
			config: &AptProxyConfig{
				Path:    "/proxy/apt",
				Proxies: map[string][]AptProxy{},
			},
			expectError: true,
			errorField:  "proxies",
		},
		{
			name: "Empty proxy name",
			config: &AptProxyConfig{
				Path: "/proxy/apt",
				Proxies: map[string][]AptProxy{
					"ubuntu": {
						{
							Name: "",
							URL:  "https://mirror.example.com/ubuntu",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies.ubuntu[0].name",
		},
		{
			name: "Invalid proxy URL",
			config: &AptProxyConfig{
				Path: "/proxy/apt",
				Proxies: map[string][]AptProxy{
					"ubuntu": {
						{
							Name: "Ubuntu Main",
							URL:  "invalid-url",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies.ubuntu[0].url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				t.Logf("Got error: %v (type: %T)", err, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError, got: %T", err)
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMavenProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *MavenProxyConfig
		expectError bool
		errorField  string
	}{
		{
			name: "Valid config",
			config: &MavenProxyConfig{
				Path: "/proxy/maven",
				Proxies: []MavenProxyServer{
					{
						Name: "Maven Central",
						URL:  "https://repo1.maven.org/maven2",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty proxies",
			config: &MavenProxyConfig{
				Path:    "/proxy/maven",
				Proxies: []MavenProxyServer{},
			},
			expectError: true,
			errorField:  "proxies",
		},
		{
			name: "Empty proxy name",
			config: &MavenProxyConfig{
				Path: "/proxy/maven",
				Proxies: []MavenProxyServer{
					{
						Name: "",
						URL:  "https://repo1.maven.org/maven2",
					},
				},
			},
			expectError: true,
			errorField:  "proxies[0].name",
		},
		{
			name: "Invalid proxy URL",
			config: &MavenProxyConfig{
				Path: "/proxy/maven",
				Proxies: []MavenProxyServer{
					{
						Name: "Maven Central",
						URL:  "invalid-url",
					},
				},
			},
			expectError: true,
			errorField:  "proxies[0].url",
		},
		{
			name: "Basic auth without password",
			config: &MavenProxyConfig{
				Path: "/proxy/maven",
				Proxies: []MavenProxyServer{
					{
						Name: "Maven Central",
						URL:  "https://repo1.maven.org/maven2",
						BasicAuth: BasicAuth{
							Username: "testuser",
							Password: "",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies[0].basic_auth.password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				t.Logf("Got error: %v (type: %T)", err, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError, got: %T", err)
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNpmProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *NpmProxyConfig
		expectError bool
		errorField  string
	}{
		{
			name: "Valid config",
			config: &NpmProxyConfig{
				Path: "/proxy/npm",
				Proxies: map[string][]NpmProxyServer{
					"default": {
						{
							Name: "NPM Registry",
							URL:  "https://registry.npmjs.org",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty proxies",
			config: &NpmProxyConfig{
				Path:    "/proxy/npm",
				Proxies: map[string][]NpmProxyServer{},
			},
			expectError: true,
			errorField:  "proxies",
		},
		{
			name: "Missing default registry",
			config: &NpmProxyConfig{
				Path: "/proxy/npm",
				Proxies: map[string][]NpmProxyServer{
					"scoped": {
						{
							Name: "Scoped Registry",
							URL:  "https://npm.example.com",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies.default",
		},
		{
			name: "Empty server name",
			config: &NpmProxyConfig{
				Path: "/proxy/npm",
				Proxies: map[string][]NpmProxyServer{
					"default": {
						{
							Name: "",
							URL:  "https://registry.npmjs.org",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies.default[0].name",
		},
		{
			name: "Invalid server URL",
			config: &NpmProxyConfig{
				Path: "/proxy/npm",
				Proxies: map[string][]NpmProxyServer{
					"default": {
						{
							Name: "NPM Registry",
							URL:  "invalid-url",
						},
					},
				},
			},
			expectError: true,
			errorField:  "proxies.default[0].url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				t.Logf("Got error: %v (type: %T)", err, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError, got: %T", err)
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnifiedConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *UnifiedConfig
		expectError bool
		errorField  string
	}{
		{
			name: "Valid config",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
					File: FileCacheConfig{
						Directory: "/tmp/cache",
					},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: false,
		},
		{
			name: "Invalid server port",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 99999,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorField:  "server.port",
		},
		{
			name: "TLS enabled without cert file",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
					TLS: TLSConfig{
						Enabled: true,
						KeyFile: "/path/to/key.pem",
					},
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorField:  "server.tls.cert_file",
		},
		{
			name: "Invalid cache backend",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "invalid",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorField:  "cache.backend",
		},
		{
			name: "S3 cache without bucket",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "s3",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			expectError: true,
			errorField:  "cache.s3.bucket",
		},
		{
			name: "Invalid log level",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			expectError: true,
			errorField:  "logging.level",
		},
		{
			name: "Metrics port same as server port",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Port:    8080,
				},
			},
			expectError: true,
			errorField:  "metrics.port",
		},
		{
			name: "NPM enabled without upstream",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Registries: RegistryConfig{
					NPM: NPMRegistryConfig{
						Enabled:  true,
						Upstream: "",
					},
				},
			},
			expectError: true,
			errorField:  "registries.npm.upstream",
		},
		{
			name: "APT enabled without mirrors",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Registries: RegistryConfig{
					APT: APTRegistryConfig{
						Enabled: true,
						Mirrors: map[string][]APTMirror{},
					},
				},
			},
			expectError: true,
			errorField:  "registries.apt.mirrors",
		},
		{
			name: "Docker enabled without registries",
			config: &UnifiedConfig{
				Server: ServerConfig{
					Port: 8080,
				},
				Cache: CacheConfig{
					Backend: "file",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Registries: RegistryConfig{
					Docker: DockerRegistryConfig{
						Enabled:    true,
						Registries: []DockerRegistryEndpoint{},
					},
				},
			},
			expectError: true,
			errorField:  "registries.docker.registries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				t.Logf("Got error: %v (type: %T)", err, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					require.True(t, ok, "Expected ValidationError, got: %T", err)
					assert.Equal(t, tt.errorField, validationErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}