package commands

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s; expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 15, "this is a ve..."},
		{"exact length", 12, "exact length"},
	}

	for _, test := range tests {
		result := truncateString(test.input, test.length)
		if result != test.expected {
			t.Errorf("truncateString(%s, %d) = %s; expected %s", test.input, test.length, result, test.expected)
		}
	}
}
