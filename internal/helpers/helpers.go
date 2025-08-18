// Package helpers provides utility functions for ProxyND.
// It includes environment variable handling, file operations, and common conversions.
package helpers

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
)

// Int64ToString converts an int64 number to its string representation.
// It uses base 10 for the conversion.
func Int64ToString(inputNum int64) string {
	return strconv.FormatInt(inputNum, 10)
}

// GetEnv retrieves the value of an environment variable.
// If the variable is not set or empty, it returns the fallback value.
func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

// GetConfigDir returns the configuration directory path from CONFIG_DIR environment variable.
// It expands home directory if the path starts with ~ and exits fatally if CONFIG_DIR is not set.
func GetConfigDir() string {
	value := os.Getenv("CONFIG_DIR")
	if len(value) == 0 {
		// end fatal
		log.Fatal("env CONFIG_DIR is not set")
	}
	if strings.HasPrefix(value, "~") {
		value = ExpandHome(value)
	}
	return value
}

// GetStorageDir returns the storage directory path from STORAGE_DIR environment variable.
// It expands home directory if the path starts with ~ and exits fatally if STORAGE_DIR is not set.
func GetStorageDir() string {
	value := os.Getenv("STORAGE_DIR")
	if len(value) == 0 {
		// end fatal
		log.Fatal("env STORAGE_DIR is not set")
	}
	if strings.HasPrefix(value, "~") {
		value = ExpandHome(value)
	}
	return value
}

// ExpandHome expands the tilde (~) in a path to the user's home directory.
// If the path doesn't start with ~/ or home directory cannot be determined, it returns the path unchanged.
func ExpandHome(path string) string {
	if path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return homeDir + path[1:]
	}
	return path
}

// FileExists checks if a file or directory exists at the given path.
// Returns true if the path exists, false otherwise.
func FileExists(join string) bool {
	if _, err := os.Stat(join); os.IsNotExist(err) {
		return false
	}
	return true
}

// NewConfigError creates a new configuration error with the given message.
// The error message is prefixed with "config error: ".
func NewConfigError(message string) error {
	return errors.New("config error: " + message)
}

// NewConfigFieldError creates a new configuration field error.
// It includes the configuration name and specific field error message.
// The error format is: "config error in [configName]: [message]"
func NewConfigFieldError(configName, message string) error {
	return errors.New("config error in " + configName + ": " + message)
}
