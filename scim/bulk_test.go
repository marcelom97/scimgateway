package scim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestServer_BulkOperations(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	bulkJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "user1",
				"data": {
					"userName": "alice",
					"active": true
				}
			},
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "user2",
				"data": {
					"userName": "bob",
					"active": true
				}
			}
		]
	}`

	req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBufferString(bulkJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Operations) != 2 {
		t.Errorf("Operations = %d, want 2", len(resp.Operations))
	}

	for i, op := range resp.Operations {
		if op.Status != "201" {
			t.Errorf("Operation %d status = %v, want 201", i, op.Status)
		}

		if op.Location == "" {
			t.Errorf("Operation %d should have location", i)
		}
	}

	// Verify users were created
	if len(plugin.users) != 2 {
		t.Errorf("Expected 2 users created, got %d", len(plugin.users))
	}
}

func TestServer_BulkWithBulkId(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	bulkJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "user1",
				"data": {
					"userName": "alice",
					"active": true
				}
			},
			{
				"method": "POST",
				"path": "/Groups",
				"bulkId": "group1",
				"data": {
					"displayName": "Admins",
					"members": [
						{"value": "bulkId:user1"}
					]
				}
			}
		]
	}`

	req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBufferString(bulkJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Operations) != 2 {
		t.Fatalf("Operations = %d, want 2", len(resp.Operations))
	}

	// Both operations should succeed
	for i, op := range resp.Operations {
		if op.Status != "201" {
			t.Errorf("Operation %d status = %v, want 201", i, op.Status)
		}
	}
}

func TestServer_BulkFailOnErrors(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	bulkJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"failOnErrors": 1,
		"Operations": [
			{
				"method": "DELETE",
				"path": "/Users/nonexistent"
			},
			{
				"method": "POST",
				"path": "/Users",
				"data": {
					"userName": "alice"
				}
			}
		]
	}`

	req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBufferString(bulkJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should stop after first error
	if len(resp.Operations) != 1 {
		t.Errorf("Should stop after 1 error, got %d operations", len(resp.Operations))
	}
}

func TestServer_BulkPatch(t *testing.T) {
	plugin := newMockPlugin()
	userID := uuid.New().String()
	plugin.users[userID] = &User{
		ID:       userID,
		UserName: "alice",
		Active:   Bool(true),
	}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	bulkJSON := fmt.Sprintf(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "PATCH",
				"path": "/Users/%s",
				"data": {
					"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
					"Operations": [{
						"op": "replace",
						"path": "active",
						"value": false
					}]
				}
			}
		]
	}`, userID)

	req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBufferString(bulkJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Operations) != 1 {
		t.Fatalf("Operations = %d, want 1", len(resp.Operations))
	}

	if resp.Operations[0].Status != "204" {
		t.Errorf("Status = %v, want 204", resp.Operations[0].Status)
	}

	// Verify user was patched
	if plugin.users[userID].Active != nil && *plugin.users[userID].Active {
		t.Error("User should be inactive")
	}
}

func TestServer_BulkDelete(t *testing.T) {
	plugin := newMockPlugin()
	userIDAlice := uuid.New().String()
	userIDBob := uuid.New().String()
	plugin.users[userIDAlice] = &User{ID: userIDAlice, UserName: "alice"}
	plugin.users[userIDBob] = &User{ID: userIDBob, UserName: "bob"}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	bulkJSON := fmt.Sprintf(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "DELETE",
				"path": "/Users/%s"
			},
			{
				"method": "DELETE",
				"path": "/Users/%s"
			}
		]
	}`, userIDAlice, userIDBob)

	req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBufferString(bulkJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	for i, op := range resp.Operations {
		if op.Status != "204" {
			t.Errorf("Operation %d status = %v, want 204", i, op.Status)
		}
	}

	// Verify users were deleted
	if len(plugin.users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(plugin.users))
	}
}

// TestBulkCircularReferences tests RFC 7644 Section 3.7.3 requirement
// for detecting circular bulkId dependencies
func TestBulkCircularReferences(t *testing.T) {
	tests := []struct {
		name          string
		operations    []BulkOperation
		expectError   bool
		errorContains string
	}{
		{
			name: "direct circular reference (A→B, B→A)",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
						"manager":  map[string]any{"value": "bulkId:user2"},
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user2",
					Data: map[string]any{
						"userName": "bob",
						"manager":  map[string]any{"value": "bulkId:user1"},
					},
				},
			},
			expectError:   true,
			errorContains: "circular bulkId reference",
		},
		{
			name: "indirect circular reference (A→B→C→A)",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
						"manager":  map[string]any{"value": "bulkId:user2"},
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user2",
					Data: map[string]any{
						"userName": "bob",
						"manager":  map[string]any{"value": "bulkId:user3"},
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user3",
					Data: map[string]any{
						"userName": "charlie",
						"manager":  map[string]any{"value": "bulkId:user1"},
					},
				},
			},
			expectError:   true,
			errorContains: "circular bulkId reference",
		},
		{
			name: "self-reference (A→A)",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
						"manager":  map[string]any{"value": "bulkId:user1"},
					},
				},
			},
			expectError:   true,
			errorContains: "circular bulkId reference",
		},
		{
			name: "valid dependency chain (C→B→A, no cycle)",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user2",
					Data: map[string]any{
						"userName": "bob",
						"manager":  map[string]any{"value": "bulkId:user1"},
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user3",
					Data: map[string]any{
						"userName": "charlie",
						"manager":  map[string]any{"value": "bulkId:user2"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "no dependencies",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user2",
					Data: map[string]any{
						"userName": "bob",
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple independent chains",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user2",
					Data: map[string]any{
						"userName": "bob",
						"manager":  map[string]any{"value": "bulkId:user1"},
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user3",
					Data: map[string]any{
						"userName": "charlie",
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user4",
					Data: map[string]any{
						"userName": "dave",
						"manager":  map[string]any{"value": "bulkId:user3"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate bulkId",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
					},
				},
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1", // Duplicate
					Data: map[string]any{
						"userName": "bob",
					},
				},
			},
			expectError:   true,
			errorContains: "duplicate bulkId",
		},
		{
			name: "reference to non-existent bulkId (should pass validation)",
			operations: []BulkOperation{
				{
					Method: "POST",
					Path:   "/Users",
					BulkID: "user1",
					Data: map[string]any{
						"userName": "alice",
						"manager":  map[string]any{"value": "bulkId:nonexistent"},
					},
				},
			},
			expectError: false, // Validation passes, execution would fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBulkOperations(tt.operations)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
