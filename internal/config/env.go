package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	
	"github.com/joho/godotenv"
)

// Env holds all environment variables
type Env struct {
	mu sync.RWMutex
	
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

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string
}

type JWTConfig struct {
	Secret             string
	ExpiryHours        int
	RefreshExpireDays  int
}

type OAuthConfig struct {
	Github OAuthProvider
	Gitlab OAuthProvider
	Google OAuthProvider
}

type OAuthProvider struct {
	ClientID     string
	ClientSecret string
}

type RegistryConfig struct {
	Maven  RegistryCredentials
	NPM    RegistryCredentials
	Docker RegistryCredentials
}

type RegistryCredentials struct {
	Username string
	Password string
	Token    string
}

type SecurityConfig struct {
	EncryptionKey string
	SigningKey    string
}

type CacheConfig struct {
	Type    string
	TTL     int
	MaxSize int
}

type LogConfig struct {
	Level  string
	Format string
}

type MetricsConfig struct {
	Enabled bool
	Port    string
}

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
		// Load .env file if exists
		_ = godotenv.Load()
		
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
		if env.ServerEnv == "production" {
			err = env.validateProduction()
		}
	})
	
	return env, err
}

// Get returns the singleton environment instance
func Get() *Env {
	if env == nil {
		panic("environment not loaded: call config.Load() first")
	}
	return env
}

// IsProduction checks if running in production environment
func (e *Env) IsProduction() bool {
	return e.ServerEnv == "production"
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

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
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
		if os.Getenv("SERVER_ENV") == "production" {
			panic("JWT_SECRET is required in production")
		}
		return "default-jwt-secret-for-development-only"
	}
	
	// Validate minimum length
	if len(secret) < 32 {
		panic("JWT_SECRET must be at least 32 characters long")
	}
	
	return secret
}

// getSecurityKey returns security key with validation  
func getSecurityKey(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// Generate a warning but use default for development
		if os.Getenv("SERVER_ENV") == "production" {
			panic(fmt.Sprintf("%s is required in production", key))
		}
		return "default-key-for-development-only"
	}
	
	// Validate minimum length
	if len(value) < 32 {
		panic(fmt.Sprintf("%s must be at least 32 characters long", key))
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