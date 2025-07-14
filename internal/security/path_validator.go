package security

import (
	"errors"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidPath = errors.New("invalid path: potential directory traversal attack")
	ErrPathOutsideRoot = errors.New("path is outside allowed root directory")
)

// SafeJoinPath safely joins path components and validates against directory traversal
func SafeJoinPath(basePath, userPath string) (string, error) {
	// Remove any leading/trailing spaces
	userPath = strings.TrimSpace(userPath)
	
	// Reject obvious traversal patterns
	if ContainsTraversalPattern(userPath) {
		return "", ErrInvalidPath
	}
	
	// Clean the user path
	cleanedPath := filepath.Clean(userPath)
	
	// Join with base path
	fullPath := filepath.Join(basePath, cleanedPath)
	
	// Get absolute paths for comparison
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", err
	}
	
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	
	// Ensure the resolved path is within the base path
	if !strings.HasPrefix(absPath, absBase) {
		return "", ErrPathOutsideRoot
	}
	
	return fullPath, nil
}

// ContainsTraversalPattern checks for common directory traversal patterns (exported)
func ContainsTraversalPattern(path string) bool {
	patterns := []string{
		"..",
		"..\\",
		"../",
		"..%2F",
		"..%2f",
		"..%5C",
		"..%5c",
		"%2e%2e",
		"%252e%252e",
	}
	
	for _, pattern := range patterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	
	return false
}

// ValidateFilename ensures filename doesn't contain dangerous characters
func ValidateFilename(filename string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}
	
	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		return errors.New("filename contains null bytes")
	}
	
	// Check for path separators
	if strings.ContainsAny(filename, "/\\") {
		return errors.New("filename cannot contain path separators")
	}
	
	return nil
}
