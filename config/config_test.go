package config

import (
	"net/http"
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid config",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    8080,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty baseURL",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "",
					Port:    8080,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"gateway.baseURL", "cannot be empty"},
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    0,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"gateway.port", "out of range"},
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    70000,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"gateway.port", "out of range"},
		},
		{
			name: "invalid URL scheme",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "ftp://localhost",
					Port:    8080,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"gateway.baseURL", "invalid URL scheme"},
		},
		{
			name: "no plugins",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    8080,
				},
				Plugins: []PluginConfig{},
			},
			wantErr:     true,
			errContains: []string{"plugins", "at least one plugin"},
		},
		{
			name: "duplicate plugin names",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    8080,
				},
				Plugins: []PluginConfig{
					{Name: "test"},
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"duplicate plugin name"},
		},
		{
			name: "empty plugin name",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "http://localhost",
					Port:    8080,
				},
				Plugins: []PluginConfig{
					{Name: ""},
				},
			},
			wantErr:     true,
			errContains: []string{"plugin name cannot be empty"},
		},
		{
			name: "TLS enabled without cert file",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "https://localhost:8443",
					Port:    8443,
					TLS: &TLS{
						Enabled:  true,
						KeyFile:  "/path/to/key.pem",
						CertFile: "",
					},
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"tls.certFile", "required when TLS is enabled"},
		},
		{
			name: "TLS enabled without key file",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "https://localhost:8443",
					Port:    8443,
					TLS: &TLS{
						Enabled:  true,
						CertFile: "/path/to/cert.pem",
						KeyFile:  "",
					},
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr:     true,
			errContains: []string{"tls.keyFile", "required when TLS is enabled"},
		},
		{
			name: "valid TLS config",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "https://localhost:8443",
					Port:    8443,
					TLS: &TLS{
						Enabled:  true,
						CertFile: "/path/to/cert.pem",
						KeyFile:  "/path/to/key.pem",
					},
				},
				Plugins: []PluginConfig{
					{Name: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple validation errors",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "",
					Port:    99999,
				},
				Plugins: []PluginConfig{},
			},
			wantErr:     true,
			errContains: []string{"gateway.baseURL", "gateway.port", "plugins"},
		},
		{
			name: "valid config with https",
			config: &Config{
				Gateway: GatewayConfig{
					BaseURL: "https://api.example.com",
					Port:    443,
				},
				Plugins: []PluginConfig{
					{Name: "prod"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				errStr := err.Error()
				for _, expected := range tt.errContains {
					if !strings.Contains(errStr, expected) {
						t.Errorf("Config.Validate() error = %v, should contain %q", err, expected)
					}
				}
			}
		})
	}
}

func TestValidationErrorsFormatting(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := &ValidationError{
			Field:   "test.field",
			Message: "test message",
		}

		expected := "config validation error [test.field]: test message"
		if err.Error() != expected {
			t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errors := ValidationErrors{
			ValidationError{Field: "field1", Message: "error 1"},
			ValidationError{Field: "field2", Message: "error 2"},
		}

		errStr := errors.Error()
		if !strings.Contains(errStr, "config validation failed with 2 errors") {
			t.Error("ValidationErrors.Error() should mention error count")
		}
		if !strings.Contains(errStr, "field1") || !strings.Contains(errStr, "field2") {
			t.Error("ValidationErrors.Error() should contain all field names")
		}
	})
}

func TestGatewayConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      GatewayConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: GatewayConfig{
				BaseURL: "http://localhost",
				Port:    8080,
			},
			wantErr: false,
		},
		{
			name: "valid port boundaries - min",
			config: GatewayConfig{
				BaseURL: "http://localhost:1",
				Port:    1,
			},
			wantErr: false,
		},
		{
			name: "valid port boundaries - max",
			config: GatewayConfig{
				BaseURL: "http://localhost:65535",
				Port:    65535,
			},
			wantErr: false,
		},
		{
			name: "URL without host",
			config: GatewayConfig{
				BaseURL: "http://",
				Port:    8080,
			},
			wantErr:     true,
			errContains: "must include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GatewayConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GatewayConfig.Validate() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestAuthConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      AuthConfig
		fieldPrefix string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid basic auth",
			config: AuthConfig{
				Basic: &BasicAuth{
					Username: "user",
					Password: "pass",
				},
			},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name: "valid bearer auth",
			config: AuthConfig{
				Bearer: &BearerAuth{
					Token: "secret-token",
				},
			},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name:        "none auth type",
			config:      AuthConfig{},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name:        "empty auth type (treated as none)",
			config:      AuthConfig{},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name: "case insensitive type",
			config: AuthConfig{
				Basic: &BasicAuth{
					Username: "user",
					Password: "pass",
				},
			},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name: "valid custom auth",
			config: AuthConfig{
				Type: "custom",
				Custom: &CustomAuth{
					Authenticator: &mockAuthenticator{},
				},
			},
			fieldPrefix: "gateway.auth",
			wantErr:     false,
		},
		{
			name: "custom auth without authenticator",
			config: AuthConfig{
				Type:   "custom",
				Custom: nil,
			},
			fieldPrefix: "gateway.auth",
			wantErr:     true,
			errContains: "custom auth configuration is required",
		},
		{
			name: "custom auth with nil authenticator",
			config: AuthConfig{
				Type: "custom",
				Custom: &CustomAuth{
					Authenticator: nil,
				},
			},
			fieldPrefix: "gateway.auth",
			wantErr:     true,
			errContains: "custom auth configuration is required",
		},
		{
			name: "invalid auth type",
			config: AuthConfig{
				Type: "oauth2",
			},
			fieldPrefix: "gateway.auth",
			wantErr:     true,
			errContains: "invalid auth type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.fieldPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AuthConfig.Validate() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()

	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig() is not valid: %v", err)
	}
}

// mockAuthenticator is a test double for auth.Authenticator
type mockAuthenticator struct{}

func (m *mockAuthenticator) Authenticate(r *http.Request) error {
	return nil
}
