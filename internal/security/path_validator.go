package security

import (
	"errors"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidPath  = errors.New("invalid path: contains path traversal")
	ErrEmptyPath    = errors.New("path cannot be empty")
	ErrAbsolutePath = errors.New("absolute paths not allowed")
)

// SafeJoinPath safely joins a base path with user input
func SafeJoinPath(base, userInput string) (string, error) {
	if userInput == "" {
		return "", ErrEmptyPath
	}

	if filepath.IsAbs(userInput) {
		return "", ErrAbsolutePath
	}

	// Clean the path and check for traversal attempts
	cleaned := filepath.Clean(userInput)
	if strings.Contains(cleaned, "..") {
		return "", ErrInvalidPath
	}

	// Additional check for hidden files/directories if needed
	if strings.HasPrefix(cleaned, ".") {
		return "", ErrInvalidPath
	}

	return filepath.Join(base, cleaned), nil
}

// ValidateFilename validates individual filename components
func ValidateFilename(filename string) error {
	if filename == "" {
		return ErrEmptyPath
	}

	if strings.ContainsAny(filename, "/\\") {
		return ErrInvalidPath
	}

	if filename == "." || filename == ".." {
		return ErrInvalidPath
	}

	return nil
}
