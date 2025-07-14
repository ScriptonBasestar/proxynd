package config

import (
	"os"
	"sync"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear any existing environment
	os.Clearenv()
	
	// Reset the singleton for testing
	env = nil
	once = sync.Once{}
	
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "Development environment with defaults",
			envVars: map[string]string{
				"SERVER_ENV": "development",
			},
			wantErr: false,
		},
		{
			name: "Production environment with proper secrets",
			envVars: map[string]string{
				"SERVER_ENV":     "production",
				"JWT_SECRET":     "this-is-a-very-secure-jwt-secret-key-that-is-long-enough",
				"ENCRYPTION_KEY": "this-is-a-very-secure-encryption-key-that-is-long-enough",
				"SIGNING_KEY":    "this-is-a-very-secure-signing-key-that-is-long-enough",
			},
			wantErr: false,
		},
		{
			name: "Production environment with insecure defaults",
			envVars: map[string]string{
				"SERVER_ENV":     "production",
				"JWT_SECRET":     "your-jwt-secret-key-here-minimum-32-chars-change-this-in-production",
				"ENCRYPTION_KEY": "your-encryption-key-here-minimum-32-chars",
				"SIGNING_KEY":    "your-signing-key-here-minimum-32-chars",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset environment and singleton
			os.Clearenv()
			env = nil
			once = sync.Once{}
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			
			// Test Load function
			loadedEnv, err := Load()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if loadedEnv == nil {
					t.Error("Load() returned nil environment")
					return
				}
				
				// Test basic fields
				if loadedEnv.ServerHost == "" {
					t.Error("ServerHost should not be empty")
				}
				
				if loadedEnv.ServerPort == "" {
					t.Error("ServerPort should not be empty")
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	// Reset environment
	os.Clearenv()
	env = nil
	once = sync.Once{}
	
	// Test panic when not loaded
	defer func() {
		if r := recover(); r == nil {
			t.Error("Get() should panic when environment not loaded")
		}
	}()
	
	Get()
}

func TestGetAfterLoad(t *testing.T) {
	// Reset environment
	os.Clearenv()
	env = nil
	once = sync.Once{}
	
	// Set development environment
	os.Setenv("SERVER_ENV", "development")
	
	// Load environment
	_, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	// Test Get function
	loadedEnv := Get()
	if loadedEnv == nil {
		t.Error("Get() returned nil after Load()")
	}
}

func TestEnvironmentChecks(t *testing.T) {
	// Reset environment
	os.Clearenv()
	env = nil
	once = sync.Once{}
	
	tests := []struct {
		name        string
		serverEnv   string
		wantProd    bool
		wantDev     bool
	}{
		{
			name:      "Production environment",
			serverEnv: "production",
			wantProd:  true,
			wantDev:   false,
		},
		{
			name:      "Development environment",
			serverEnv: "development",
			wantProd:  false,
			wantDev:   true,
		},
		{
			name:      "Other environment",
			serverEnv: "staging",
			wantProd:  false,
			wantDev:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset
			os.Clearenv()
			env = nil
			once = sync.Once{}
			
			os.Setenv("SERVER_ENV", tt.serverEnv)
			
			// Set required production secrets if needed
			if tt.wantProd {
				os.Setenv("JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-that-is-long-enough")
				os.Setenv("ENCRYPTION_KEY", "this-is-a-very-secure-encryption-key-that-is-long-enough")
				os.Setenv("SIGNING_KEY", "this-is-a-very-secure-signing-key-that-is-long-enough")
			}
			
			loadedEnv, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}
			
			if loadedEnv.IsProduction() != tt.wantProd {
				t.Errorf("IsProduction() = %v, want %v", loadedEnv.IsProduction(), tt.wantProd)
			}
			
			if loadedEnv.IsDevelopment() != tt.wantDev {
				t.Errorf("IsDevelopment() = %v, want %v", loadedEnv.IsDevelopment(), tt.wantDev)
			}
		})
	}
}

func TestJWTSecretValidation(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		serverEnv string
		wantPanic bool
	}{
		{
			name:      "Valid long secret",
			secret:    "this-is-a-very-secure-jwt-secret-key-that-is-long-enough",
			serverEnv: "production",
			wantPanic: false,
		},
		{
			name:      "Short secret in production",
			secret:    "short",
			serverEnv: "production",
			wantPanic: true,
		},
		{
			name:      "No secret in production",
			secret:    "",
			serverEnv: "production",
			wantPanic: true,
		},
		{
			name:      "No secret in development",
			secret:    "",
			serverEnv: "development",
			wantPanic: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("SERVER_ENV", tt.serverEnv)
			if tt.secret != "" {
				os.Setenv("JWT_SECRET", tt.secret)
			}
			
			defer func() {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("getJWTSecret() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()
			
			getJWTSecret()
		})
	}
}