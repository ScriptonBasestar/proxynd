package plugins

import (
	"testing"
	"time"
)

func TestValidateConfig_DuplicateNames(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Core: []PluginConfig{
				{Name: "npm", Priority: 100},
				{Name: "npm", Priority: 110}, // Duplicate!
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for duplicate plugin names, got nil")
	}
	if err != nil && err.Error() != "duplicate plugin name: npm" {
		t.Errorf("Expected 'duplicate plugin name: npm', got: %v", err)
	}
}

func TestValidateConfig_ConflictingPriorities(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Core: []PluginConfig{
				{Name: "npm", Priority: 100},
				{Name: "maven", Priority: 100}, // Same priority!
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for conflicting priorities, got nil")
	}
}

func TestValidateConfig_InvalidPriorityRange_Core(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Core: []PluginConfig{
				{Name: "npm", Priority: 250}, // Invalid for core (should be 100-199)
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid core priority range, got nil")
	}
}

func TestValidateConfig_InvalidPriorityRange_Enterprise(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Enterprise: []PluginConfig{
				{Name: "rbac", Priority: 150}, // Invalid for enterprise (should be 200-299)
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid enterprise priority range, got nil")
	}
}

func TestValidateConfig_InvalidPriorityRange_Cloud(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Cloud: []PluginConfig{
				{Name: "multitenancy", Priority: 250}, // Invalid for cloud (should be 300-399)
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid cloud priority range, got nil")
	}
}

func TestValidateConfig_InvalidLifecycleTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		field   string
	}{
		{"zero init timeout", 0, "initTimeout"},
		{"negative init timeout", -1 * time.Second, "initTimeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled: true,
				Lifecycle: LifecycleConfig{
					InitTimeout:     tt.timeout,
					ReadyTimeout:    10 * time.Second,
					ShutdownTimeout: 30 * time.Second,
					FailurePolicy:   FailurePolicyContinue,
				},
				Registry: RegistryConfig{
					Core: []PluginConfig{},
				},
			}

			err := ValidateConfig(cfg)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestValidateConfig_InvalidFailurePolicy(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   "invalid_policy",
		},
		Registry: RegistryConfig{
			Core: []PluginConfig{},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid failure policy, got nil")
	}
}

func TestValidateConfig_ValidConfiguration(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Core: []PluginConfig{
				{Name: "npm", Priority: 100},
				{Name: "maven", Priority: 110},
			},
			Enterprise: []PluginConfig{
				{Name: "rbac", Priority: 200},
			},
			Cloud: []PluginConfig{
				{Name: "multitenancy", Priority: 300},
			},
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected no error for valid configuration, got: %v", err)
	}
}

func TestValidateConfig_NilConfig(t *testing.T) {
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}
