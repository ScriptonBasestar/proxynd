package commands

import (
	"testing"
)

func TestBoolIcon(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "✅"},
		{false, "❌"},
	}

	for _, test := range tests {
		result := boolIcon(test.input)
		if result != test.expected {
			t.Errorf("boolIcon(%t) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestConfigValidationStructures(t *testing.T) {
	// 구조체 초기화 테스트
	response := ConfigValidationResponse{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		Summary: ValidationSummary{
			TotalFiles:   5,
			ValidFiles:   4,
			ErrorCount:   0,
			WarningCount: 1,
		},
	}

	if !response.Valid {
		t.Error("Expected response to be valid")
	}

	if response.Summary.TotalFiles != 5 {
		t.Errorf("Expected TotalFiles to be 5, got %d", response.Summary.TotalFiles)
	}

	if response.Summary.ValidFiles != 4 {
		t.Errorf("Expected ValidFiles to be 4, got %d", response.Summary.ValidFiles)
	}
}

func TestProxyTypeStatus(t *testing.T) {
	status := ProxyTypeStatus{
		Type:        "apt",
		Enabled:     true,
		ConfigFile:  "apt-proxy.yaml",
		ProxyCount:  3,
		HasUpstream: true,
		Status:      "ok",
	}

	if status.Type != "apt" {
		t.Errorf("Expected Type to be 'apt', got %s", status.Type)
	}

	if !status.Enabled {
		t.Error("Expected status to be enabled")
	}

	if status.ProxyCount != 3 {
		t.Errorf("Expected ProxyCount to be 3, got %d", status.ProxyCount)
	}
}
