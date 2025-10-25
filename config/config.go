package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error [%s]: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("config validation failed with %d errors:\n", len(e)))
	for i, err := range e {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Config represents the gateway configuration
type Config struct {
	Gateway GatewayConfig
	Plugins []PluginConfig
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate gateway config
	if err := c.Gateway.Validate(); err != nil {
		if verrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, verrs...)
		} else if verr, ok := err.(*ValidationError); ok {
			errors = append(errors, *verr)
		} else {
			errors = append(errors, ValidationError{
				Field:   "gateway",
				Message: err.Error(),
			})
		}
	}

	// Validate plugins
	if len(c.Plugins) == 0 {
		errors = append(errors, ValidationError{
			Field:   "plugins",
			Message: "at least one plugin must be configured",
		})
	}

	// Check for duplicate plugin names
	pluginNames := make(map[string]bool)
	for i, plugin := range c.Plugins {
		if plugin.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("plugins[%d].name", i),
				Message: "plugin name cannot be empty",
			})
			continue
		}

		if pluginNames[plugin.Name] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("plugins[%d].name", i),
				Message: fmt.Sprintf("duplicate plugin name: %s", plugin.Name),
			})
		}
		pluginNames[plugin.Name] = true

		// Validate plugin auth if present
		if plugin.Auth != nil {
			if err := plugin.Auth.Validate(fmt.Sprintf("plugins[%d].auth", i)); err != nil {
				if verrs, ok := err.(ValidationErrors); ok {
					errors = append(errors, verrs...)
				} else if verr, ok := err.(*ValidationError); ok {
					errors = append(errors, *verr)
				}
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// GatewayConfig represents gateway-specific configuration
type GatewayConfig struct {
	BaseURL string
	Port    int
	TLS     *TLS
}

// Validate validates the gateway configuration
func (g *GatewayConfig) Validate() error {
	var errors ValidationErrors

	// Validate BaseURL
	if g.BaseURL == "" {
		errors = append(errors, ValidationError{
			Field:   "gateway.baseURL",
			Message: "baseURL cannot be empty",
		})
	} else {
		// Parse and validate URL format
		parsedURL, err := url.Parse(g.BaseURL)
		if err != nil {
			errors = append(errors, ValidationError{
				Field:   "gateway.baseURL",
				Message: fmt.Sprintf("invalid URL format: %v", err),
			})
		} else {
			// Check scheme
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				errors = append(errors, ValidationError{
					Field:   "gateway.baseURL",
					Message: fmt.Sprintf("invalid URL scheme '%s': must be http or https", parsedURL.Scheme),
				})
			}
			// Check host
			if parsedURL.Host == "" {
				errors = append(errors, ValidationError{
					Field:   "gateway.baseURL",
					Message: "URL must include a host (e.g., http://localhost:8080)",
				})
			}
		}
	}

	// Validate Port
	if g.Port < 1 || g.Port > 65535 {
		errors = append(errors, ValidationError{
			Field:   "gateway.port",
			Message: fmt.Sprintf("port %d is out of range: must be between 1 and 65535", g.Port),
		})
	}

	// Validate TLS configuration
	if g.TLS != nil && g.TLS.Enabled {
		if g.TLS.CertFile == "" {
			errors = append(errors, ValidationError{
				Field:   "gateway.tls.certFile",
				Message: "certFile is required when TLS is enabled",
			})
		}
		if g.TLS.KeyFile == "" {
			errors = append(errors, ValidationError{
				Field:   "gateway.tls.keyFile",
				Message: "keyFile is required when TLS is enabled",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// TLS represents TLS configuration
type TLS struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// PluginConfig represents plugin-specific configuration
type PluginConfig struct {
	Name string
	// Deprecated: Use Type and BaseEntity instead
	Type       string
	BaseEntity string
	Auth       *AuthConfig
	Config     map[string]any
}

// AuthConfig represents authentication configuration with type-safe config
type AuthConfig struct {
	Type   string // basic, bearer, none
	Basic  *BasicAuth
	Bearer *BearerAuth
}

// Validate validates the authentication configuration
func (a *AuthConfig) Validate(fieldPrefix string) error {
	var errors ValidationErrors

	// Validate Type
	validTypes := map[string]bool{
		"basic":  true,
		"bearer": true,
		"none":   true,
		"":       true, // empty is treated as none
	}

	if !validTypes[strings.ToLower(a.Type)] {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.type", fieldPrefix),
			Message: fmt.Sprintf("invalid auth type '%s': must be 'basic', 'bearer', or 'none'", a.Type),
		})
	}

	// Validate type-specific configuration
	authType := strings.ToLower(a.Type)
	switch authType {
	case "basic":
		if a.Basic == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.basic", fieldPrefix),
				Message: "basic auth configuration is required when type is 'basic'",
			})
		} else {
			if a.Basic.Username == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.basic.username", fieldPrefix),
					Message: "username cannot be empty for basic auth",
				})
			}
			if a.Basic.Password == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.basic.password", fieldPrefix),
					Message: "password cannot be empty for basic auth",
				})
			}
		}
	case "bearer":
		if a.Bearer == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.bearer", fieldPrefix),
				Message: "bearer auth configuration is required when type is 'bearer'",
			})
		} else {
			if a.Bearer.Token == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.bearer.token", fieldPrefix),
					Message: "token cannot be empty for bearer auth",
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// BasicAuth represents basic authentication configuration
type BasicAuth struct {
	Username string
	Password string
}

// BearerAuth represents bearer token authentication configuration
type BearerAuth struct {
	Token string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Gateway: GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8880,
		},
		Plugins: []PluginConfig{
			{
				Name:       "memory",
				Type:       "memory",
				BaseEntity: "default",
			},
		},
	}
}
