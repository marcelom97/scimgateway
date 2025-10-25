package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcelom97/scimgateway/config"
)

func TestPerPluginAuthMiddleware_NoAuth(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "public"}
	manager.Register(plugin, nil) // No config = no auth

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request to plugin without auth
	req := httptest.NewRequest("GET", "/public/Users", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should be called
	if !handlerCalled {
		t.Error("Expected handler to be called for plugin without auth")
	}

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_WithBearerAuth_Valid(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "protected"}
	pluginCfg := &config.PluginConfig{
		Name: "protected",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "valid-token"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request with valid token
	req := httptest.NewRequest("GET", "/protected/Users", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should be called
	if !handlerCalled {
		t.Error("Expected handler to be called with valid auth")
	}

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_WithBearerAuth_Invalid(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "protected"}
	pluginCfg := &config.PluginConfig{
		Name: "protected",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "valid-token"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request with invalid token
	req := httptest.NewRequest("GET", "/protected/Users", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should NOT be called
	if handlerCalled {
		t.Error("Expected handler to NOT be called with invalid auth")
	}

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_WithBearerAuth_Missing(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "protected"}
	pluginCfg := &config.PluginConfig{
		Name: "protected",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "valid-token"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request without auth header
	req := httptest.NewRequest("GET", "/protected/Users", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should NOT be called
	if handlerCalled {
		t.Error("Expected handler to NOT be called without auth header")
	}

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_WithBasicAuth_Valid(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "protected"}
	pluginCfg := &config.PluginConfig{
		Name: "protected",
		Auth: &config.AuthConfig{
			Type:  "basic",
			Basic: &config.BasicAuth{Username: "admin", Password: "password"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request with valid basic auth
	req := httptest.NewRequest("GET", "/protected/Users", nil)
	req.SetBasicAuth("admin", "password")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should be called
	if !handlerCalled {
		t.Error("Expected handler to be called with valid basic auth")
	}

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_WithBasicAuth_Invalid(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "protected"}
	pluginCfg := &config.PluginConfig{
		Name: "protected",
		Auth: &config.AuthConfig{
			Type:  "basic",
			Basic: &config.BasicAuth{Username: "admin", Password: "password"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware(handler)

	// Test request with invalid credentials
	req := httptest.NewRequest("GET", "/protected/Users", nil)
	req.SetBasicAuth("admin", "wrong-password")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Handler should NOT be called
	if handlerCalled {
		t.Error("Expected handler to NOT be called with invalid basic auth")
	}

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_MultiplePlugins_DifferentAuth(t *testing.T) {
	manager := NewManager()

	// Plugin 1: Bearer auth
	plugin1 := &mockPlugin{name: "plugin1"}
	pluginCfg1 := &config.PluginConfig{
		Name: "plugin1",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token1"},
		},
	}
	manager.Register(plugin1, pluginCfg1)

	// Plugin 2: Basic auth
	plugin2 := &mockPlugin{name: "plugin2"}
	pluginCfg2 := &config.PluginConfig{
		Name: "plugin2",
		Auth: &config.AuthConfig{
			Type:  "basic",
			Basic: &config.BasicAuth{Username: "user", Password: "pass"},
		},
	}
	manager.Register(plugin2, pluginCfg2)

	// Plugin 3: No auth
	plugin3 := &mockPlugin{name: "plugin3"}
	manager.Register(plugin3, nil)

	// Create middleware
	middleware := PerPluginAuthMiddleware(manager)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		path           string
		authHeader     func(*http.Request)
		expectedStatus int
	}{
		{
			name: "Plugin1 with valid bearer token",
			path: "/plugin1/Users",
			authHeader: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token1")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Plugin1 with invalid bearer token",
			path: "/plugin1/Users",
			authHeader: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer wrong-token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Plugin2 with valid basic auth",
			path: "/plugin2/Groups",
			authHeader: func(r *http.Request) {
				r.SetBasicAuth("user", "pass")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Plugin2 with invalid basic auth",
			path: "/plugin2/Groups",
			authHeader: func(r *http.Request) {
				r.SetBasicAuth("user", "wrong")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Plugin3 without auth",
			path:           "/plugin3/Users",
			authHeader:     func(r *http.Request) {},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Plugin1 with bearer token for plugin2 (wrong auth type)",
			path: "/plugin1/Users",
			authHeader: func(r *http.Request) {
				r.SetBasicAuth("user", "pass")
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			tt.authHeader(req)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestPerPluginAuthMiddleware_NestedPaths(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "plugin"}
	pluginCfg := &config.PluginConfig{
		Name: "plugin",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "valid-token"},
		},
	}
	manager.Register(plugin, pluginCfg)

	middleware := PerPluginAuthMiddleware(manager)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := middleware(handler)

	paths := []string{
		"/plugin/Users",
		"/plugin/Users/123",
		"/plugin/Groups",
		"/plugin/Groups/456",
		"/plugin/Bulk",
		"/plugin/.search",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Test with valid auth
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: Expected status 200 with valid auth, got %d", path, w.Code)
			}

			// Test without auth
			req = httptest.NewRequest("GET", path, nil)
			w = httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Path %s: Expected status 401 without auth, got %d", path, w.Code)
			}
		})
	}
}

func TestPerPluginAuthMiddleware_EmptyPath(t *testing.T) {
	manager := NewManager()
	middleware := PerPluginAuthMiddleware(manager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test with empty path
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Should allow request (no plugin to check)
	if !handlerCalled {
		t.Error("Expected handler to be called for empty path")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPerPluginAuthMiddleware_UnknownPlugin(t *testing.T) {
	manager := NewManager()
	middleware := PerPluginAuthMiddleware(manager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	// Test request to unknown plugin
	req := httptest.NewRequest("GET", "/unknown/Users", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Should allow request (plugin not registered, no auth to check)
	if !handlerCalled {
		t.Error("Expected handler to be called for unknown plugin")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
