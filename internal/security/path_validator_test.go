package security

import (
	"testing"
)

func TestSafeJoinPath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		userInput string
		wantErr   bool
		expected  string
	}{
		{"valid path", "/base", "file.txt", false, "/base/file.txt"},
		{"valid nested", "/base", "dir/file.txt", false, "/base/dir/file.txt"},
		{"traversal attempt", "/base", "../file.txt", true, ""},
		{"absolute path", "/base", "/etc/passwd", true, ""},
		{"dot prefix", "/base", ".hidden", true, ""},
		{"empty input", "/base", "", true, ""},
		{"current dir", "/base", ".", true, ""},
		{"parent dir", "/base", "..", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeJoinPath(tt.base, tt.userInput)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeJoinPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("SafeJoinPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"valid filename", "file.txt", false},
		{"valid with numbers", "file123.txt", false},
		{"valid with dashes", "my-file.txt", false},
		{"invalid with slash", "dir/file.txt", true},
		{"invalid with backslash", "dir\\file.txt", true},
		{"invalid current dir", ".", true},
		{"invalid parent dir", "..", true},
		{"empty filename", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
