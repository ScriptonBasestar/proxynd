package common

import (
	"encoding/base64"
	"fmt"

	"proxynd/internal/ports"
)

// ApplyAuthentication applies authentication headers based on auth config
// This is a common utility used by all package manager drivers
func ApplyAuthentication(headers map[string]string, auth *ports.AuthConfig) error {
	if auth == nil {
		return nil
	}

	switch auth.Type {
	case "basic":
		// Basic Authentication
		if auth.Username != "" && auth.Password != "" {
			headers["Authorization"] = fmt.Sprintf("Basic %s", EncodeBasicAuth(auth.Username, auth.Password))
		}
	case "bearer", "token":
		// Bearer Token Authentication
		if auth.Token != "" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", auth.Token)
		}
	case "digest":
		// Digest Authentication - add WWW-Authenticate response handling
		// Note: Full digest auth requires challenge-response, implemented in HTTP client
		if auth.Username != "" {
			headers["X-Auth-Username"] = auth.Username
		}
	}

	return nil
}

// EncodeBasicAuth encodes username and password for HTTP Basic Authentication
func EncodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
