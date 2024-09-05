package helpers

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Int64ToString function convert a float number to a string
func Int64ToString(inputNum int64) string {
	return strconv.FormatInt(inputNum, 10)
}

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

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

func FileExists(join string) bool {
	if _, err := os.Stat(join); os.IsNotExist(err) {
		return false
	}
	return true
}
