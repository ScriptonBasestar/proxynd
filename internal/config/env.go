// Package config provides configuration management utilities
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

// Environment constants
const (
	// EnvProduction is a const that env production
	EnvProduction = "production"
)

// Env holds all environment variables
type Env struct {
	// Server
	ServerHost string
	ServerPort string
	ServerEnv  string
	ConfigDir  string
	StorageDir string

	// Database
	Database DatabaseConfig

	// Redis
	Redis RedisConfig

	// JWT
	JWT JWTConfig

	// OAuth
	OAuth OAuthConfig

	// Registries
	Registries RegistryConfig

	// API Keys
	APIKeys map[string]string

	// Security
	Security SecurityConfig

	// Cache
	Cache CacheConfig

	// Logging
	Logging LogConfig

	// Metrics
	Metrics MetricsConfig

	// Development
	Development DevConfig
}

// DatabaseConfig represents the configuration for database settings
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// RedisConfig represents the configuration for redis settings
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
}

// JWTConfig represents the configuration for jwt settings
type JWTConfig struct {
	Secret            string
	ExpiryHours       int
	RefreshExpireDays int
}

// OAuthConfig represents the configuration for oauth settings
type OAuthConfig struct {
	Github OAuthProvider
	Gitlab OAuthProvider
	Google OAuthProvider
}

// OAuthProvider represents a oauth provider
type OAuthProvider struct {
	ClientID     string
	ClientSecret string
}

// RegistryConfig represents the configuration for registry settings
type RegistryConfig struct {
	Maven  RegistryCredentials
	NPM    RegistryCredentials
	Docker RegistryCredentials
}

// RegistryCredentials represents a registry credentials
type RegistryCredentials struct {
	Username string
	Password string
	Token    string
}

// SecurityConfig represents the configuration for security settings
type SecurityConfig struct {
	EncryptionKey string
	SigningKey    string
}

// CacheConfig represents the configuration for cache settings
type CacheConfig struct {
	Type    string
	TTL     int
	MaxSize int
}

// LogConfig represents the configuration for log settings
type LogConfig struct {
	Level  string
	Format string
}

// MetricsConfig represents the configuration for metrics settings
type MetricsConfig struct {
	Enabled bool
	Port    string
}

// DevConfig represents the configuration for dev settings
type DevConfig struct {
	Debug     bool
	HotReload bool
}

var (
	env  *Env
	once sync.Once
)

// Load loads environment variables
func Load() (*Env, error) {
	var err error
	once.Do(func() {
		// Load .env file if exists (ignore errors - file may not exist)
		if err := godotenv.Load(); err != nil {
			// Log warning only if the error is not "file not found"
			log.Printf("Note: .env file not loaded (this is normal if not using .env): %v", err)
		}

		env = &Env{
			ServerHost: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
			ServerEnv:  getEnvOrDefault("SERVER_ENV", "development"),
			ConfigDir:  getEnvOrDefault("CONFIG_DIR", "./tmp/config"),
			StorageDir: getEnvOrDefault("STORAGE_DIR", "./tmp/storage"),

			Database: DatabaseConfig{
				Host:     getEnvOrDefault("DB_HOST", "localhost"),
				Port:     getEnvOrDefault("DB_PORT", "5432"),
				User:     getEnvOrDefault("DB_USER", "proxynd"),
				Password: os.Getenv("DB_PASSWORD"), // Optional
				Name:     getEnvOrDefault("DB_NAME", "proxynd_db"),
				SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
			},

			Redis: RedisConfig{
				Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
				Port:     getEnvOrDefault("REDIS_PORT", "6379"),
				Password: os.Getenv("REDIS_PASSWORD"),
				DB:       getEnvAsInt("REDIS_DB", 0),
				URL:      getEnvOrDefault("REDIS_URL", "redis://localhost:6379"),
			},

			JWT: JWTConfig{
				Secret:            getJWTSecret(),
				ExpiryHours:       getEnvAsInt("JWT_EXPIRY_HOURS", 24),
				RefreshExpireDays: getEnvAsInt("JWT_REFRESH_EXPIRE_DAYS", 7),
			},

			OAuth: OAuthConfig{
				Github: OAuthProvider{
					ClientID:     os.Getenv("OAUTH_GITHUB_CLIENT_ID"),
					ClientSecret: os.Getenv("OAUTH_GITHUB_CLIENT_SECRET"),
				},
				Gitlab: OAuthProvider{
					ClientID:     os.Getenv("OAUTH_GITLAB_CLIENT_ID"),
					ClientSecret: os.Getenv("OAUTH_GITLAB_CLIENT_SECRET"),
				},
				Google: OAuthProvider{
					ClientID:     os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
					ClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
				},
			},

			Registries: RegistryConfig{
				Maven: RegistryCredentials{
					Username: os.Getenv("REGISTRY_MAVEN_USERNAME"),
					Password: os.Getenv("REGISTRY_MAVEN_PASSWORD"),
				},
				NPM: RegistryCredentials{
					Token: os.Getenv("REGISTRY_NPM_TOKEN"),
				},
				Docker: RegistryCredentials{
					Username: os.Getenv("REGISTRY_DOCKER_USERNAME"),
					Password: os.Getenv("REGISTRY_DOCKER_PASSWORD"),
				},
			},

			APIKeys: map[string]string{
				"nexus":       os.Getenv("API_KEY_NEXUS"),
				"artifactory": os.Getenv("API_KEY_ARTIFACTORY"),
				"harbor":      os.Getenv("API_KEY_HARBOR"),
			},

			Security: SecurityConfig{
				EncryptionKey: getSecurityKey("ENCRYPTION_KEY"),
				SigningKey:    getSecurityKey("SIGNING_KEY"),
			},

			Cache: CacheConfig{
				Type:    getEnvOrDefault("CACHE_TYPE", "filesystem"),
				TTL:     getEnvAsInt("CACHE_TTL", 3600),
				MaxSize: getEnvAsInt("CACHE_MAX_SIZE", 1024),
			},

			Logging: LogConfig{
				Level:  getEnvOrDefault("LOG_LEVEL", "info"),
				Format: getEnvOrDefault("LOG_FORMAT", "json"),
			},

			Metrics: MetricsConfig{
				Enabled: getEnvAsBool("METRICS_ENABLED", true),
				Port:    getEnvOrDefault("METRICS_PORT", "9090"),
			},

			Development: DevConfig{
				Debug:     getEnvAsBool("DEBUG", false),
				HotReload: getEnvAsBool("HOT_RELOAD", true),
			},
		}

		// Validate required fields in production
		if env.ServerEnv == EnvProduction {
			err = env.validateProduction()
		}
	})

	return env, err
}

// Get returns the singleton environment instance
func Get() *Env {
	if env == nil {
		// 개발 환경에서는 기본값으로 로드 시도
		if os.Getenv("SERVER_ENV") != EnvProduction {
			if _, err := Load(); err != nil {
				log.Printf("Warning: Failed to load environment variables: %v", err)
			}
		}
		if env == nil {
			// 여전히 nil인 경우 에러 로그 후 기본 환경 반환
			fmt.Printf("ERROR: environment not loaded, using defaults. Call config.Load() first.\n")
			return &Env{
				ServerEnv:  "development",
				ServerPort: "8080",
				ServerHost: "localhost",
			}
		}
	}
	return env
}

// IsProduction checks if running in production environment
func (e *Env) IsProduction() bool {
	return e.ServerEnv == EnvProduction
}

// IsDevelopment checks if running in development environment
func (e *Env) IsDevelopment() bool {
	return e.ServerEnv == "development"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getJWTSecret returns JWT secret with validation
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a warning but use default for development
		if os.Getenv("SERVER_ENV") == EnvProduction {
			fmt.Printf("ERROR: JWT_SECRET is required in production\n")
			return ""
		}
		fmt.Printf("WARNING: Using default JWT secret for development. Set JWT_SECRET in production.\n")
		return "default-jwt-secret-for-development-only"
	}

	// Validate minimum length
	if len(secret) < 32 {
		if os.Getenv("SERVER_ENV") == EnvProduction {
			fmt.Printf("ERROR: JWT_SECRET must be at least 32 characters long\n")
			return ""
		}
		fmt.Printf("WARNING: JWT_SECRET should be at least 32 characters long\n")
	}

	return secret
}

// getSecurityKey returns security key with validation
func getSecurityKey(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// Generate a warning but use default for development
		if os.Getenv("SERVER_ENV") == EnvProduction {
			fmt.Printf("ERROR: %s is required in production\n", key)
			return ""
		}
		fmt.Printf("WARNING: Using default %s for development. Set %s in production.\n", key, key)
		return "default-key-for-development-only"
	}

	// Validate minimum length
	if len(value) < 32 {
		if os.Getenv("SERVER_ENV") == EnvProduction {
			fmt.Printf("ERROR: %s must be at least 32 characters long\n", key)
			return ""
		}
		fmt.Printf("WARNING: %s should be at least 32 characters long\n", key)
	}

	return value
}

func (e *Env) validateProduction() error {
	// Check for insecure default values in production
	insecureDefaults := []string{
		"your-jwt-secret-key-here-minimum-32-chars-change-this-in-production",
		"your-encryption-key-here-minimum-32-chars",
		"your-signing-key-here-minimum-32-chars",
		"default-jwt-secret-for-development-only",
		"default-key-for-development-only",
	}

	for _, insecure := range insecureDefaults {
		if e.JWT.Secret == insecure ||
			e.Security.EncryptionKey == insecure ||
			e.Security.SigningKey == insecure {
			return fmt.Errorf("insecure default values detected in production environment")
		}
	}

	return nil
}
