package scimgateway

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/memory"
	"github.com/marcelom97/scimgateway/scim"
)

// mockPlugin is a simple plugin implementation for testing
type mockPlugin struct {
	name string
}

func (p *mockPlugin) Name() string { return p.name }

func (p *mockPlugin) GetUsers(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.User, error) {
	return []*scim.User{}, nil
}

func (p *mockPlugin) CreateUser(ctx context.Context, baseEntity string, user *scim.User) (*scim.User, error) {
	return user, nil
}

func (p *mockPlugin) GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.User, error) {
	return &scim.User{ID: id, UserName: "test"}, nil
}

func (p *mockPlugin) ModifyUser(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return nil
}

func (p *mockPlugin) DeleteUser(ctx context.Context, baseEntity string, id string) error {
	return nil
}

func (p *mockPlugin) GetGroups(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.Group, error) {
	return []*scim.Group{}, nil
}

func (p *mockPlugin) CreateGroup(ctx context.Context, baseEntity string, group *scim.Group) (*scim.Group, error) {
	return group, nil
}

func (p *mockPlugin) GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.Group, error) {
	return &scim.Group{ID: id, DisplayName: "test"}, nil
}

func (p *mockPlugin) ModifyGroup(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return nil
}

func (p *mockPlugin) DeleteGroup(ctx context.Context, baseEntity string, id string) error {
	return nil
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)

	if gw == nil {
		t.Fatal("New() returned nil")
	}

	if gw.config != cfg {
		t.Error("Config not set correctly")
	}

	if gw.pluginManager == nil {
		t.Error("Plugin manager not initialized")
	}
}

func TestNewWithDefaults(t *testing.T) {
	gw := NewWithDefaults()

	if gw == nil {
		t.Fatal("NewWithDefaults() returned nil")
	}

	if gw.config == nil {
		t.Error("Config not set")
	}

	if gw.pluginManager == nil {
		t.Error("Plugin manager not initialized")
	}
}

func TestRegisterPlugin(t *testing.T) {
	gw := NewWithDefaults()
	p := &mockPlugin{name: "test"}

	gw.RegisterPlugin(p) // No config needed for basic test

	// Verify plugin was registered
	plugins := gw.PluginManager().List()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	if plugins[0] != "test" {
		t.Errorf("Expected plugin name 'test', got '%s'", plugins[0])
	}
}

func TestInitialize(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if gw.server == nil {
		t.Error("Server not initialized")
	}

	if gw.handler == nil {
		t.Error("Handler not initialized")
	}
}

func TestHandler(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}
	if handler == nil {
		t.Error("Handler() returned nil")
	}

	// Test that handler works
	req := httptest.NewRequest("GET", "/test/Users", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandlerNotInitialized(t *testing.T) {
	gw := NewWithDefaults()

	handler, err := gw.Handler()
	if err == nil {
		t.Error("Handler() should return error if not initialized")
	}
	if handler != nil {
		t.Error("Handler() should return nil handler when not initialized")
	}

	expectedErrMsg := "gateway not initialized"
	if err != nil && len(err.Error()) < len(expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestConfig(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)

	if gw.Config() != cfg {
		t.Error("Config() returned wrong config")
	}
}

func TestPluginManager(t *testing.T) {
	gw := NewWithDefaults()

	pm := gw.PluginManager()
	if pm == nil {
		t.Error("PluginManager() returned nil")
	}
}

func TestMultiplePlugins(t *testing.T) {
	gw := NewWithDefaults()

	p1 := &mockPlugin{name: "plugin1"}
	p2 := &mockPlugin{name: "plugin2"}

	gw.RegisterPlugin(p1)
	gw.RegisterPlugin(p2)

	plugins := gw.PluginManager().List()
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}
}

func TestGatewayIntegration(t *testing.T) {
	// Full integration test
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "test",
				Auth: &config.AuthConfig{
					Type: "basic",
					Basic: &config.BasicAuth{
						Username: "admin",
						Password: "secret",
					},
				},
			},
		},
	}

	gw := New(cfg)
	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	// Test GET /test/Users without auth (should fail)
	req := httptest.NewRequest("GET", "/test/Users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Test GET /test/Users with auth (should succeed)
	req = httptest.NewRequest("GET", "/test/Users", nil)
	req.SetBasicAuth("admin", "secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAttributeSelectionMutualExclusivity tests RFC 7644 Section 3.9 requirement
// that attributes and excludedAttributes parameters are mutually exclusive
func TestAttributeSelectionMutualExclusivity(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8880",
			Port:    8880,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	plugin := &mockPlugin{name: "test"}
	gw.RegisterPlugin(plugin)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	tests := []struct {
		name           string
		endpoint       string
		queryParams    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "attributes only on Users - should succeed",
			endpoint:       "/test/Users",
			queryParams:    "?attributes=userName,emails",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "excludedAttributes only on Users - should succeed",
			endpoint:       "/test/Users",
			queryParams:    "?excludedAttributes=groups,meta",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "both parameters on Users - should fail",
			endpoint:       "/test/Users",
			queryParams:    "?attributes=userName&excludedAttributes=groups",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
		},
		{
			name:           "neither parameter on Users - should succeed",
			endpoint:       "/test/Users",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "attributes only on Groups - should succeed",
			endpoint:       "/test/Groups",
			queryParams:    "?attributes=displayName,members",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "excludedAttributes only on Groups - should succeed",
			endpoint:       "/test/Groups",
			queryParams:    "?excludedAttributes=members",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "both parameters on Groups - should fail",
			endpoint:       "/test/Groups",
			queryParams:    "?attributes=displayName&excludedAttributes=members",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
		},
		{
			name:           "both parameters on single user GET - should fail",
			endpoint:       "/test/Users/123",
			queryParams:    "?attributes=userName&excludedAttributes=groups",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
		},
		{
			name:           "both parameters on single group GET - should fail",
			endpoint:       "/test/Groups/123",
			queryParams:    "?attributes=displayName&excludedAttributes=members",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.endpoint + tt.queryParams
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler, err := gw.Handler()
			if err != nil {
				t.Fatalf("Handler() error = %v", err)
			}
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.shouldContain != "" {
				body := w.Body.String()
				if !containsString(body, tt.shouldContain) {
					t.Errorf("Expected response body to contain %q, got: %s", tt.shouldContain, body)
				}

				// Verify SCIM error format
				if !containsString(body, "invalidFilter") {
					t.Errorf("Expected scimType 'invalidFilter' in error response, got: %s", body)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSetLogger tests setting and using a logger
func TestSetLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	gw := NewWithDefaults()

	// Logger should never be nil (defaults to discard logger)
	if gw.logger == nil {
		t.Error("Logger should not be nil (should default to discard logger)")
	}

	gw.SetLogger(logger)

	// After setting, logger should still not be nil
	if gw.logger == nil {
		t.Error("Logger should not be nil after SetLogger")
	}

	// Setting nil should use discard logger, not actually set to nil
	gw.SetLogger(nil)
	if gw.logger == nil {
		t.Error("Logger should not be nil even when SetLogger(nil) is called")
	}
}

// TestInitializeWithLogger tests that initialization logs appropriately
func TestInitializeWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "testplugin"},
		},
	}

	gw := New(cfg)
	gw.SetLogger(logger)
	p := &mockPlugin{name: "testplugin"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Check that initialization was logged
	logOutput := buf.String()
	if !strings.Contains(logOutput, "initializing SCIM gateway") {
		t.Error("Expected initialization log message")
	}
	if !strings.Contains(logOutput, "gateway initialized successfully") {
		t.Error("Expected successful initialization log message")
	}
	if !strings.Contains(logOutput, "testplugin") {
		t.Error("Expected plugin name in logs")
	}
}

// TestInitializeWithoutLogger tests that initialization works without a logger
func TestInitializeWithoutLogger(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	// Don't set a logger - should work fine
	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if gw.server == nil {
		t.Error("Server not initialized")
	}
}

// TestPluginNotFoundLogging tests that plugin not found errors are logged
func TestPluginNotFoundLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	gw.SetLogger(logger)
	p := &mockPlugin{name: "existingplugin"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	// Try to access a non-existent plugin
	req := httptest.NewRequest("GET", "/nonexistent/Users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Check that the plugin not found was logged
	logOutput := buf.String()
	if !strings.Contains(logOutput, "plugin not found") {
		t.Errorf("Expected 'plugin not found' in logs, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "nonexistent") {
		t.Errorf("Expected plugin name 'nonexistent' in logs, got: %s", logOutput)
	}
}

// TestLoggerNilSafety tests that calling SetLogger(nil) doesn't cause panics
// (internally it uses a discard logger, so logger field is never actually nil)
func TestLoggerNilSafety(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	// Call SetLogger(nil) - should use discard logger internally
	gw.SetLogger(nil)

	// Verify logger is not actually nil
	if gw.logger == nil {
		t.Error("Logger should not be nil after SetLogger(nil)")
	}

	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	// Try to access a non-existent plugin - should not panic
	req := httptest.NewRequest("GET", "/nonexistent/Users", nil)
	w := httptest.NewRecorder()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred: %v", r)
		}
	}()

	handler.ServeHTTP(w, req)
}

// TestInitializeWithInvalidConfig tests that Initialize fails with invalid configuration
func TestInitializeWithInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		errContains string
	}{
		{
			name: "invalid port",
			config: &config.Config{
				Gateway: config.GatewayConfig{
					BaseURL: "http://localhost:8080",
					Port:    99999, // Invalid port
				},
				Plugins: []config.PluginConfig{
					{Name: "test"},
				},
			},
			errContains: "port",
		},
		{
			name: "empty baseURL",
			config: &config.Config{
				Gateway: config.GatewayConfig{
					BaseURL: "", // Invalid
					Port:    8080,
				},
				Plugins: []config.PluginConfig{
					{Name: "test"},
				},
			},
			errContains: "baseURL",
		},
		{
			name: "no plugins",
			config: &config.Config{
				Gateway: config.GatewayConfig{
					BaseURL: "http://localhost:8080",
					Port:    8080,
				},
				Plugins: []config.PluginConfig{}, // No plugins
			},
			errContains: "at least one plugin",
		},
		{
			name: "TLS enabled without cert",
			config: &config.Config{
				Gateway: config.GatewayConfig{
					BaseURL: "https://localhost:8443",
					Port:    8443,
					TLS: &config.TLS{
						Enabled:  true,
						CertFile: "", // Missing
						KeyFile:  "/path/to/key.pem",
					},
				},
				Plugins: []config.PluginConfig{
					{Name: "test"},
				},
			},
			errContains: "certFile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := New(tt.config)
			gw.RegisterPlugin(&mockPlugin{name: "test"})

			err := gw.Initialize()
			if err == nil {
				t.Fatal("Initialize() should fail with invalid config")
			}

			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Initialize() error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}

// TestInitializeValidatesConfigBeforeStarting tests that validation happens before any initialization
func TestInitializeValidatesConfigBeforeStarting(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "", // Invalid
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	gw.RegisterPlugin(&mockPlugin{name: "test"})

	err := gw.Initialize()
	if err == nil {
		t.Fatal("Initialize() should fail with invalid config")
	}

	// Server should not be initialized
	if gw.server != nil {
		t.Error("Server should not be initialized when config is invalid")
	}

	// Handler should not be set
	if gw.handler != nil {
		t.Error("Handler should not be set when config is invalid")
	}
}

// TestInitializeWithNoPluginsRegistered tests that Initialize fails when no plugins are registered
func TestInitializeWithNoPluginsRegistered(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	// Note: NOT calling gw.RegisterPlugin() - this should cause an error

	err := gw.Initialize()
	if err == nil {
		t.Fatal("Initialize() should fail when no plugins are registered")
	}

	if !strings.Contains(err.Error(), "no plugins registered") {
		t.Errorf("Initialize() error = %v, should contain 'no plugins registered'", err)
	}

	// Server should not be initialized
	if gw.server != nil {
		t.Error("Server should not be initialized when no plugins are registered")
	}
}

// TestRequestLoggingIntegration tests that HTTP requests are logged through the middleware
func TestRequestLoggingIntegration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := New(cfg)
	gw.SetLogger(logger)

	p := &mockPlugin{name: "test"}
	gw.RegisterPlugin(p)

	err := gw.Initialize()
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	// Make a request
	req := httptest.NewRequest("GET", "/test/Users", nil)
	req.Header.Set("User-Agent", "test-client")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that request was logged
	logOutput := buf.String()

	expectedStrings := []string{
		"HTTP request",
		"GET",
		"/test/Users",
		"status",
		"200",
		"duration_ms",
		"remote_addr",
		"user_agent",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(logOutput, expected) {
			t.Errorf("Expected log to contain %q, got: %s", expected, logOutput)
		}
	}
}

// ============================================================================
// Per-Plugin Authentication Tests
// ============================================================================

func TestGateway_PerPluginAuth_SinglePlugin_BearerAuth(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "test",
				Auth: &config.AuthConfig{
					Type: "bearer",
					Bearer: &config.BearerAuth{
						Token: "secret-token",
					},
				},
			},
		},
	}

	gw := New(cfg)
	plugin := memory.New("test")
	gw.RegisterPlugin(plugin)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid bearer token",
			authHeader:     "Bearer secret-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid bearer token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Wrong auth type (basic instead of bearer)",
			authHeader:     "Basic YWRtaW46cGFzcw==", // admin:pass
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/Users", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGateway_PerPluginAuth_SinglePlugin_BasicAuth(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "test",
				Auth: &config.AuthConfig{
					Type: "basic",
					Basic: &config.BasicAuth{
						Username: "admin",
						Password: "password",
					},
				},
			},
		},
	}

	gw := New(cfg)
	plugin := memory.New("test")
	gw.RegisterPlugin(plugin)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	tests := []struct {
		name           string
		username       string
		password       string
		setAuth        bool
		expectedStatus int
	}{
		{
			name:           "Valid credentials",
			username:       "admin",
			password:       "password",
			setAuth:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid username",
			username:       "wrong",
			password:       "password",
			setAuth:        true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid password",
			username:       "admin",
			password:       "wrong",
			setAuth:        true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No credentials",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/Users", nil)
			if tt.setAuth {
				req.SetBasicAuth(tt.username, tt.password)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGateway_PerPluginAuth_MultiplePlugins_DifferentAuth(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "plugin1",
				Auth: &config.AuthConfig{
					Type: "bearer",
					Bearer: &config.BearerAuth{
						Token: "token-1",
					},
				},
			},
			{
				Name: "plugin2",
				Auth: &config.AuthConfig{
					Type: "basic",
					Basic: &config.BasicAuth{
						Username: "user2",
						Password: "pass2",
					},
				},
			},
			{
				Name: "plugin3",
				// No auth for plugin3
			},
		},
	}

	gw := New(cfg)

	// Register all plugins with their configs
	plugin1 := memory.New("plugin1")
	gw.RegisterPlugin(plugin1)

	plugin2 := memory.New("plugin2")
	gw.RegisterPlugin(plugin2)

	plugin3 := memory.New("plugin3")
	gw.RegisterPlugin(plugin3)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		setupAuth      func(*http.Request)
		expectedStatus int
		description    string
	}{
		// Plugin 1 tests
		{
			name: "Plugin1 with valid token",
			path: "/plugin1/Users",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token-1")
			},
			expectedStatus: http.StatusOK,
			description:    "Plugin1 should accept valid bearer token",
		},
		{
			name: "Plugin1 with invalid token",
			path: "/plugin1/Users",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer wrong-token")
			},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin1 should reject invalid bearer token",
		},
		{
			name:           "Plugin1 without auth",
			path:           "/plugin1/Users",
			setupAuth:      func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin1 should reject requests without auth",
		},
		{
			name: "Plugin1 with plugin2's basic auth",
			path: "/plugin1/Users",
			setupAuth: func(r *http.Request) {
				r.SetBasicAuth("user2", "pass2")
			},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin1 should reject basic auth (expects bearer)",
		},

		// Plugin 2 tests
		{
			name: "Plugin2 with valid basic auth",
			path: "/plugin2/Groups",
			setupAuth: func(r *http.Request) {
				r.SetBasicAuth("user2", "pass2")
			},
			expectedStatus: http.StatusOK,
			description:    "Plugin2 should accept valid basic auth",
		},
		{
			name: "Plugin2 with invalid basic auth",
			path: "/plugin2/Groups",
			setupAuth: func(r *http.Request) {
				r.SetBasicAuth("user2", "wrong")
			},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin2 should reject invalid basic auth",
		},
		{
			name:           "Plugin2 without auth",
			path:           "/plugin2/Groups",
			setupAuth:      func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin2 should reject requests without auth",
		},
		{
			name: "Plugin2 with plugin1's bearer token",
			path: "/plugin2/Groups",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token-1")
			},
			expectedStatus: http.StatusUnauthorized,
			description:    "Plugin2 should reject bearer token (expects basic)",
		},

		// Plugin 3 tests (no auth)
		{
			name:           "Plugin3 without auth",
			path:           "/plugin3/Users",
			setupAuth:      func(r *http.Request) {},
			expectedStatus: http.StatusOK,
			description:    "Plugin3 should allow requests without auth",
		},
		{
			name: "Plugin3 with bearer token (not required)",
			path: "/plugin3/Users",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer any-token")
			},
			expectedStatus: http.StatusOK,
			description:    "Plugin3 should allow requests even with auth header",
		},
		{
			name: "Plugin3 with basic auth (not required)",
			path: "/plugin3/Groups",
			setupAuth: func(r *http.Request) {
				r.SetBasicAuth("any", "creds")
			},
			expectedStatus: http.StatusOK,
			description:    "Plugin3 should allow requests even with auth header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			tt.setupAuth(req)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGateway_PerPluginAuth_DifferentEndpoints(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "protected",
				Auth: &config.AuthConfig{
					Type: "bearer",
					Bearer: &config.BearerAuth{
						Token: "valid-token",
					},
				},
			},
		},
	}

	gw := New(cfg)
	plugin := memory.New("protected")
	gw.RegisterPlugin(plugin)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	endpoints := []string{
		"/protected/Users",
		"/protected/Groups",
		"/protected/ServiceProviderConfig",
		"/protected/ResourceTypes",
		"/protected/Schemas",
		"/protected/Bulk",
		"/protected/.search",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			// Test with valid auth
			req := httptest.NewRequest("GET", endpoint, nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Endpoint %s with valid auth: Expected status 200 or 405, got %d", endpoint, w.Code)
			}

			// Test without auth
			req = httptest.NewRequest("GET", endpoint, nil)
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Endpoint %s without auth: Expected status 401, got %d", endpoint, w.Code)
			}
		})
	}
}

func TestGateway_PerPluginAuth_ConfigBasedAuth(t *testing.T) {
	// Test that auth is correctly applied from config
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "test",
				Auth: &config.AuthConfig{
					Type: "bearer",
					Bearer: &config.BearerAuth{
						Token: "token",
					},
				},
			},
		},
	}

	gw := New(cfg)

	// Register plugin - auth should come from config
	plugin := memory.New("test")
	gw.RegisterPlugin(plugin)

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	// Should require auth from config
	req := httptest.NewRequest("GET", "/test/Users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 without auth, got %d", w.Code)
	}

	// Should allow with valid token from config
	req = httptest.NewRequest("GET", "/test/Users", nil)
	req.Header.Set("Authorization", "Bearer token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid token, got %d", w.Code)
	}
}

func TestGateway_PerPluginAuth_NoPlugins(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "dummy"},
		},
	}

	gw := New(cfg)

	// Don't register any plugins - Initialize should fail
	err := gw.Initialize()
	if err == nil {
		t.Error("Expected Initialize to fail with no plugins registered")
	}
}

// ============================================================================
// Logging Middleware Tests
// ============================================================================

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
		path          string
		method        string
		shouldContain []string
	}{
		{
			name:          "successful request logs at INFO level",
			statusCode:    http.StatusOK,
			expectedLevel: "INFO",
			path:          "/test/Users",
			method:        "GET",
			shouldContain: []string{"HTTP request", "GET", "/test/Users", "200"},
		},
		{
			name:          "client error logs at WARN level",
			statusCode:    http.StatusBadRequest,
			expectedLevel: "WARN",
			path:          "/test/Users",
			method:        "POST",
			shouldContain: []string{"HTTP request", "POST", "/test/Users", "400"},
		},
		{
			name:          "server error logs at ERROR level",
			statusCode:    http.StatusInternalServerError,
			expectedLevel: "ERROR",
			path:          "/test/Users/123",
			method:        "DELETE",
			shouldContain: []string{"HTTP request", "DELETE", "/test/Users/123", "500"},
		},
		{
			name:          "logs include query parameters",
			statusCode:    http.StatusOK,
			expectedLevel: "INFO",
			path:          "/test/Users?filter=userName+eq+john&count=10",
			method:        "GET",
			shouldContain: []string{"HTTP request", "GET", "/test/Users", "filter=userName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture logs
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			// Create a test handler that returns the specified status code
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("test response"))
			})

			// Wrap with logging middleware
			handler := LoggingMiddleware(logger)(testHandler)

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("User-Agent", "test-agent")
			w := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			// Check log output
			logOutput := buf.String()

			if !strings.Contains(logOutput, tt.expectedLevel) {
				t.Errorf("Expected log level %s in output, got: %s", tt.expectedLevel, logOutput)
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(logOutput, expected) {
					t.Errorf("Expected log to contain %q, got: %s", expected, logOutput)
				}
			}

			// Verify duration_ms is logged
			if !strings.Contains(logOutput, "duration_ms") {
				t.Error("Expected log to contain duration_ms")
			}

			// Verify remote_addr is logged
			if !strings.Contains(logOutput, "remote_addr") {
				t.Error("Expected log to contain remote_addr")
			}
		})
	}
}

func TestLoggingMiddlewareWithoutWriteHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Handler that doesn't explicitly call WriteHeader (should default to 200)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	handler := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should log status 200
	logOutput := buf.String()
	if !strings.Contains(logOutput, "\"status\":200") {
		t.Errorf("Expected status 200 in logs, got: %s", logOutput)
	}
}

func TestLoggingMiddlewareWithDiscardLogger(t *testing.T) {
	// Test that discard logger doesn't break anything
	logger := discardLogger()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred with discard logger: %v", r)
		}
	}()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestResponseWriterMultipleWriteHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Handler that tries to call WriteHeader multiple times
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusInternalServerError) // Should be ignored
		w.Write([]byte("OK"))
	})

	handler := LoggingMiddleware(logger)(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should log the first status code (200), not the second attempt (500)
	logOutput := buf.String()
	if !strings.Contains(logOutput, "\"status\":200") {
		t.Errorf("Expected status 200 in logs, got: %s", logOutput)
	}
	if strings.Contains(logOutput, "500") {
		t.Errorf("Should not contain status 500, got: %s", logOutput)
	}
}
