package security

import (
	"testing"
)

func TestSafeJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		userPath string
		wantErr  bool
	}{
		// Valid cases
		{"Simple file", "/var/cache", "package.tar.gz", false},
		{"Nested path", "/var/cache", "ubuntu/focal/package.deb", false},
		{"With dot", "/var/cache", "./file.txt", false},
		
		// Attack attempts
		{"Simple traversal", "/var/cache", "../etc/passwd", true},
		{"Double traversal", "/var/cache", "../../etc/passwd", true},
		{"URL encoded traversal", "/var/cache", "..%2Fetc%2Fpasswd", true},
		{"Windows traversal", "/var/cache", "..\\windows\\system32", true},
		{"Mixed traversal", "/var/cache", "valid/../../../etc/passwd", true},
		{"Hidden traversal", "/var/cache", ".../.../etc/passwd", true},
		{"Unicode traversal", "/var/cache", "..%c0%af../etc/passwd", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SafeJoinPath(tt.basePath, tt.userPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeJoinPath() error = %v, wantErr %v", err, tt.wantErr)
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
		{"Valid filename", "package-1.0.0.tar.gz", false},
		{"Empty filename", "", true},
		{"With forward slash", "path/to/file", true},
		{"With backslash", "path\\to\\file", true},
		{"With null byte", "file\x00.txt", true},
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
