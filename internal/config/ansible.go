package config

import "time"

// AnsibleConfig represents the Ansible configuration
type AnsibleConfig struct {
	Repositories []*AnsibleRepositoryConfig `yaml:"repositories" json:"repositories"`
}

// AnsibleRepositoryConfig represents a single Ansible repository configuration
type AnsibleRepositoryConfig struct {
	Name    string         `yaml:"name" json:"name"`
	Type    RepositoryType `yaml:"type" json:"type"`
	Enabled bool           `yaml:"enabled" json:"enabled"`

	// Proxy settings (for proxy type)
	Upstream  string        `yaml:"upstream" json:"upstream"`
	CacheTTL  time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	CacheEnabled bool       `yaml:"cache_enabled" json:"cache_enabled"`

	// Hosted settings (for hosted type)
	MaxUploadSize      int64    `yaml:"max_upload_size" json:"max_upload_size"`           // in bytes
	AllowedNamespaces  []string `yaml:"allowed_namespaces" json:"allowed_namespaces"`
	RequireAuth        bool     `yaml:"require_auth" json:"require_auth"`
	AdminUsers         []string `yaml:"admin_users" json:"admin_users"`
}

// RepositoryType represents the type of repository
type RepositoryType string

const (
	RepositoryTypeProxy  RepositoryType = "proxy"
	RepositoryTypeHosted RepositoryType = "hosted"
)

// IsProxy returns true if the repository is a proxy
func (c *AnsibleRepositoryConfig) IsProxy() bool {
	return c.Type == RepositoryTypeProxy
}

// IsHosted returns true if the repository is a hosted repository
func (c *AnsibleRepositoryConfig) IsHosted() bool {
	return c.Type == RepositoryTypeHosted
}

// IsNamespaceAllowed checks if a namespace is allowed in this repository
func (c *AnsibleRepositoryConfig) IsNamespaceAllowed(namespace string) bool {
	// If no allowed namespaces specified, all are allowed
	if len(c.AllowedNamespaces) == 0 {
		return true
	}

	for _, allowed := range c.AllowedNamespaces {
		// Support wildcard "*" to allow all namespaces
		if allowed == "*" {
			return true
		}
		if allowed == namespace {
			return true
		}
	}

	return false
}

// IsAdminUser checks if a user is an admin
func (c *AnsibleRepositoryConfig) IsAdminUser(username string) bool {
	for _, admin := range c.AdminUsers {
		if admin == username {
			return true
		}
	}
	return false
}

// Validate validates the repository configuration
func (c *AnsibleRepositoryConfig) Validate() error {
	if c.Name == "" {
		return ErrInvalidConfig("repository name is required")
	}

	if c.Type != RepositoryTypeProxy && c.Type != RepositoryTypeHosted {
		return ErrInvalidConfig("repository type must be 'proxy' or 'hosted'")
	}

	if c.IsProxy() {
		if c.Upstream == "" {
			return ErrInvalidConfig("upstream is required for proxy repositories")
		}
	}

	if c.IsHosted() {
		if c.MaxUploadSize <= 0 {
			// Default to 10MB
			c.MaxUploadSize = 10 * 1024 * 1024
		}
	}

	return nil
}

// ErrInvalidConfig represents a configuration error
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func ErrInvalidConfig(msg string) error {
	return &ConfigError{Message: msg}
}
