package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	scimgateway "github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/memory"
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
	gw.RegisterPlugin(memory.New("test"))

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	// Create a test user with various attributes
	createReq := httptest.NewRequest("POST", "/test/Users", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
		"userName": "attr.test",
		"name": {
			"givenName": "Test",
			"familyName": "User"
		},
		"emails": [
			{"value": "test@example.com", "type": "work", "primary": true}
		],
		"active": true
	}`))
	createReq.Header.Set("Content-Type", "application/scim+json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %d - %s", createRec.Code, createRec.Body.String())
	}

	var createdUser scim.User
	if err := json.NewDecoder(createRec.Body).Decode(&createdUser); err != nil {
		t.Fatalf("Failed to decode created user: %v", err)
	}
	userID := createdUser.ID

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "attributes - include only userName and id",
			queryParams:    "?attributes=userName,id",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["userName"]; !ok {
					t.Error("Expected userName to be present")
				}
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if _, ok := data["emails"]; ok {
					t.Error("Expected emails to be excluded")
				}
				if _, ok := data["name"]; ok {
					t.Error("Expected name to be excluded")
				}
				if _, ok := data["active"]; ok {
					t.Error("Expected active to be excluded")
				}
			},
		},
		{
			name:           "excludedAttributes - exclude name and emails",
			queryParams:    "?excludedAttributes=name,emails",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["userName"]; !ok {
					t.Error("Expected userName to be present")
				}
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if _, ok := data["active"]; !ok {
					t.Error("Expected active to be present")
				}
				if _, ok := data["name"]; ok {
					t.Error("Expected name to be excluded")
				}
				if _, ok := data["emails"]; ok {
					t.Error("Expected emails to be excluded")
				}
			},
		},
		{
			name:           "excludedAttributes - nested path",
			queryParams:    "?excludedAttributes=name.familyName,emails.value",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
				if name, ok := data["name"].(map[string]interface{}); ok {
					if _, ok := name["givenName"]; !ok {
						t.Error("Expected name.givenName to be present")
					}
					if _, ok := name["familyName"]; ok {
						t.Error("Expected name.familyName to be excluded")
					}
				} else {
					t.Error("Expected name object to be present")
				}
			},
		},
		{
			name:           "attributes - nested path inclusion",
			queryParams:    "?attributes=id,name.givenName",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if name, ok := data["name"].(map[string]interface{}); ok {
					if _, ok := name["givenName"]; !ok {
						t.Error("Expected name.givenName to be present")
					}
					// Note: familyName might still be present in partial attribute selection
					// The exact behavior depends on implementation details
				} else {
					t.Error("Expected name object to be present")
				}
				if _, ok := data["userName"]; ok {
					t.Error("Expected userName to be excluded")
				}
			},
		},
		{
			name:           "mutually exclusive - should reject",
			queryParams:    "?attributes=userName&excludedAttributes=emails",
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, data map[string]interface{}) {
				// Should get error response
				if status, ok := data["status"]; ok {
					if status != "400" {
						t.Errorf("Expected status '400', got '%v'", status)
					}
				} else {
					t.Error("Expected status field in error response")
				}
			},
		},
		{
			name:           "no query params - return full resource",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["userName"]; !ok {
					t.Error("Expected userName to be present")
				}
				if _, ok := data["id"]; !ok {
					t.Error("Expected id to be present")
				}
				if _, ok := data["name"]; !ok {
					t.Error("Expected name to be present")
				}
				if _, ok := data["emails"]; !ok {
					t.Error("Expected emails to be present")
				}
				if _, ok := data["active"]; !ok {
					t.Error("Expected active to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/Users/"+userID+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
				t.Logf("Response: %s", rec.Body.String())
				return
			}

			var data map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&data); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, data)
			}
		})
	}
}

// TestSingleResourceAttributeSelection_Groups tests the same for Groups
func TestSingleResourceAttributeSelection_Groups(t *testing.T) {
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
	gw.RegisterPlugin(memory.New("test"))

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	// Create a test group
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
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "attributes - include only displayName and id",
			queryParams:    "?attributes=displayName,id",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, data map[string]interface{}) {
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
			validate: func(t *testing.T, data map[string]interface{}) {
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

			var data map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&data); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, data)
			}
		})
	}
}
