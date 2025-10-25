package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/memory"
	"github.com/marcelom97/scimgateway/scim"
)

func TestETagIntegration(t *testing.T) {
	// Setup gateway with memory plugin
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{{Name: "test", Type: "memory"}},
	}
	gw := gateway.New(cfg)
	memPlugin := memory.New("memory")
	gw.RegisterPlugin(memPlugin)
	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	tests := []struct {
		name       string
		setup      func(t *testing.T) (resourceID, etag string)
		method     string
		getPath    func(resourceID string) string
		getBody    func() []byte
		setHeaders func(req *http.Request, etag string)
		wantStatus int
		verify     func(t *testing.T, w *httptest.ResponseRecorder, originalETag string)
	}{
		{
			name: "GET returns ETag header and version",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "testuser",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				if w.Code != http.StatusCreated {
					t.Fatalf("Setup failed: expected 201, got %d", w.Code)
				}

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, ""
			},
			method:     http.MethodGet,
			getPath:    func(id string) string { return "/memory/Users/" + id },
			getBody:    func() []byte { return nil },
			setHeaders: func(req *http.Request, etag string) {},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, w *httptest.ResponseRecorder, _ string) {
				etag := w.Header().Get("ETag")
				if etag == "" {
					t.Error("Expected ETag header to be present")
				}

				var retrieved scim.User
				json.Unmarshal(w.Body.Bytes(), &retrieved)
				if retrieved.Meta == nil || retrieved.Meta.Version == "" {
					t.Error("Expected meta.version to be set")
				}

				// Verify version matches ETag (without W/" and trailing ")
				if len(etag) > 3 {
					expectedVersion := etag[3 : len(etag)-1]
					if retrieved.Meta.Version != expectedVersion {
						t.Errorf("Expected version %s, got %s", expectedVersion, retrieved.Meta.Version)
					}
				}
			},
		},
		{
			name: "If-None-Match returns 304 when not modified",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "etag-test-user",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, w.Header().Get("ETag")
			},
			method:  http.MethodGet,
			getPath: func(id string) string { return "/memory/Users/" + id },
			getBody: func() []byte { return nil },
			setHeaders: func(req *http.Request, etag string) {
				req.Header.Set("If-None-Match", etag)
			},
			wantStatus: http.StatusNotModified,
			verify: func(t *testing.T, w *httptest.ResponseRecorder, _ string) {
				if w.Body.Len() > 0 {
					t.Error("Expected empty body for 304 response")
				}
			},
		},
		{
			name: "If-Match succeeds when ETag matches (PATCH)",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "match-test-user",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, w.Header().Get("ETag")
			},
			method:  http.MethodPatch,
			getPath: func(id string) string { return "/memory/Users/" + id },
			getBody: func() []byte {
				patch := &scim.PatchOp{
					Schemas: []string{scim.SchemaPatchOp},
					Operations: []scim.PatchOperation{
						{Op: "replace", Path: "active", Value: false},
					},
				}
				patchJSON, _ := json.Marshal(patch)
				return patchJSON
			},
			setHeaders: func(req *http.Request, etag string) {
				req.Header.Set("If-Match", etag)
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, w *httptest.ResponseRecorder, originalETag string) {
				newETag := w.Header().Get("ETag")
				if newETag == originalETag {
					t.Error("Expected ETag to change after modification")
				}
			},
		},
		{
			name: "If-Match fails when ETag mismatches",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "mismatch-test-user",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, `W/"wrong-etag"`
			},
			method:  http.MethodPatch,
			getPath: func(id string) string { return "/memory/Users/" + id },
			getBody: func() []byte {
				patch := &scim.PatchOp{
					Schemas: []string{scim.SchemaPatchOp},
					Operations: []scim.PatchOperation{
						{Op: "replace", Path: "active", Value: false},
					},
				}
				patchJSON, _ := json.Marshal(patch)
				return patchJSON
			},
			setHeaders: func(req *http.Request, etag string) {
				req.Header.Set("If-Match", etag)
			},
			wantStatus: http.StatusPreconditionFailed,
			verify:     func(t *testing.T, w *httptest.ResponseRecorder, _ string) {},
		},
		{
			name: "DELETE with If-Match succeeds",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "delete-test-user",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, w.Header().Get("ETag")
			},
			method:  http.MethodDelete,
			getPath: func(id string) string { return "/memory/Users/" + id },
			getBody: func() []byte { return nil },
			setHeaders: func(req *http.Request, etag string) {
				req.Header.Set("If-Match", etag)
			},
			wantStatus: http.StatusNoContent,
			verify:     func(t *testing.T, w *httptest.ResponseRecorder, _ string) {},
		},
		{
			name: "PUT with If-Match succeeds",
			setup: func(t *testing.T) (string, string) {
				user := &scim.User{
					UserName: "put-test-user",
					Active:   scim.Bool(true),
				}
				userJSON, _ := json.Marshal(user)
				req := httptest.NewRequest(http.MethodPost, "/memory/Users", bytes.NewReader(userJSON))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				var created scim.User
				json.Unmarshal(w.Body.Bytes(), &created)
				return created.ID, w.Header().Get("ETag")
			},
			method:  http.MethodPut,
			getPath: func(id string) string { return "/memory/Users/" + id },
			getBody: func() []byte {
				updated := scim.User{
					UserName: "put-test-user-updated",
					Active:   scim.Bool(false),
				}
				updatedJSON, _ := json.Marshal(updated)
				return updatedJSON
			},
			setHeaders: func(req *http.Request, etag string) {
				req.Header.Set("If-Match", etag)
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, w *httptest.ResponseRecorder, originalETag string) {
				newETag := w.Header().Get("ETag")
				if newETag == originalETag {
					t.Error("Expected ETag to change after PUT")
				}
			},
		},
		{
			name: "Groups also support ETags",
			setup: func(t *testing.T) (string, string) {
				return "", "" // No setup needed, will create in test
			},
			method:  http.MethodPost,
			getPath: func(id string) string { return "/memory/Groups" },
			getBody: func() []byte {
				group := &scim.Group{
					DisplayName: "test-group",
				}
				groupJSON, _ := json.Marshal(group)
				return groupJSON
			},
			setHeaders: func(req *http.Request, etag string) {},
			wantStatus: http.StatusCreated,
			verify: func(t *testing.T, w *httptest.ResponseRecorder, _ string) {
				etag := w.Header().Get("ETag")
				if etag == "" {
					t.Error("Expected ETag header to be present for Groups")
				}

				var created scim.Group
				json.Unmarshal(w.Body.Bytes(), &created)
				if created.Meta == nil || created.Meta.Version == "" {
					t.Error("Expected meta.version to be set for Groups")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			resourceID, etag := tt.setup(t)

			// Prepare request
			path := tt.getPath(resourceID)
			body := tt.getBody()
			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(tt.method, path, bytes.NewReader(body))
			} else {
				req = httptest.NewRequest(tt.method, path, nil)
			}
			tt.setHeaders(req, etag)
			w := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(w, req)

			// Assert status code
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			// Custom verification
			tt.verify(t, w, etag)
		})
	}
}
