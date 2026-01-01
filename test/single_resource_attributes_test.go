package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/internal/testutil"
	"github.com/marcelom97/scimgateway/scim"
)

// TestSingleResourceAttributeSelection tests attribute selection and exclusion
// for single resource GET operations (GET /Users/{id} and GET /Groups/{id})
func TestSingleResourceAttributeSelection(t *testing.T) {
	// Setup gateway
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{Name: "test"},
		},
	}

	gw := scimgateway.New(cfg)
	gw.RegisterPlugin(testutil.NewMemoryPlugin("test"))

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	// Create a test group with various attributes
	createReq := httptest.NewRequest("POST", "/test/Groups", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
		"displayName": "Test Group",
		"members": []
	}`))
	createReq.Header.Set("Content-Type", "application/scim+json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("Failed to create group: %d - %s", createRec.Code, createRec.Body.String())
	}

	var createdGroup scim.Group
	if err := json.NewDecoder(createRec.Body).Decode(&createdGroup); err != nil {
		t.Fatalf("Failed to decode created group: %v", err)
	}
	groupID := createdGroup.ID

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validate       func(*testing.T, map[string]any)
	}{
		{
			name:           "attributes - include only displayName and id",
			queryParams:    "?attributes=displayName,id",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]any) {
				if _, ok := data["displayName"]; !ok {
					t.Error("Expected displayName to be present")
				}
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if _, ok := data["members"]; ok {
					t.Error("Expected members to be excluded")
				}
			},
		},
		{
			name:           "excludedAttributes - exclude members",
			queryParams:    "?excludedAttributes=members",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]any) {
				if _, ok := data["displayName"]; !ok {
					t.Error("Expected displayName to be present")
				}
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if _, ok := data["members"]; ok {
					t.Error("Expected members to be excluded")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/Groups/"+groupID+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
				t.Logf("Response: %s", rec.Body.String())
				return
			}

			var data map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&data); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, data)
			}
		})
	}
}
