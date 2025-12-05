package config

import "fmt"

// validTestConfig returns a minimal valid configuration for testing
func validTestConfig() string {
	return `server:
  host: "0.0.0.0"
  port: 8080
logging:
  level: "info"
  format: "json"
cache:
  backend: "file"
  ttl: 3600s
  file:
    base_dir: "/tmp/proxynd-test-cache"
`
}

// validTestConfigWithPort returns a valid config with custom port
func validTestConfigWithPort(port int) string {
	return fmt.Sprintf(`server:
  host: "0.0.0.0"
  port: %d
logging:
  level: "info"
  format: "json"
cache:
  backend: "file"
  ttl: 3600s
  file:
    base_dir: "/tmp/proxynd-test-cache"
`, port)
}
