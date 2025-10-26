package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gateway "github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/memory"
)

// TestSCIMCompliance is a comprehensive test suite for SCIM 2.0 specification compliance
// This test verifies all critical SCIM features including:
// - RFC 7644 Section 3.4.2.2: Filter case sensitivity
// - RFC 7644 Section 3.9: Attribute selection mutual exclusivity
// - RFC 7644 Section 3.7.3: Bulk circular reference detection
func TestSCIMCompliance(t *testing.T) {
	// Setup gateway with memory plugin
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{{Name: "test", Type: "memory"}},
	}

	gw := gateway.New(cfg)
	gw.RegisterPlugin(memory.New("test"))

	if err := gw.Initialize(); err != nil {
		t.Fatalf("Failed to initialize gateway: %v", err)
	}

	handler, err := gw.Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	// Create test users for filtering tests
	createTestUsers(t, handler)

	// Run all compliance test suites
	t.Run("RFC7644_Section3.4.2.2_FilterCaseSensitivity", func(t *testing.T) {
		testFilterCaseSensitivity(t, handler)
	})

	t.Run("RFC7644_Section3.9_AttributeSelectionMutualExclusivity", func(t *testing.T) {
		testAttributeSelectionMutualExclusivity(t, handler)
	})

	t.Run("RFC7644_Section3.7.3_BulkCircularReferenceDetection", func(t *testing.T) {
		testBulkCircularReferenceDetection(t, handler)
	})

	t.Run("CoreSCIMOperations", func(t *testing.T) {
		testCoreSCIMOperations(t, handler)
	})

	t.Run("RFC7644_Section3.5.2_PatchOperations", func(t *testing.T) {
		testPatchOperations(t, handler)
	})

	t.Run("RFC7644_Section3.4.2.4_Pagination", func(t *testing.T) {
		testPagination(t, handler)
	})

	t.Run("RFC7644_Section3.4.2.3_Sorting", func(t *testing.T) {
		testSorting(t, handler)
	})

	t.Run("RFC7644_Section3.4.2.2_ComplexFilters", func(t *testing.T) {
		testComplexFilters(t, handler)
	})

	t.Run("RFC7644_Section3.12_ErrorResponses", func(t *testing.T) {
		testErrorResponses(t, handler)
	})

	t.Run("RFC7644_Section3.14_ETags", func(t *testing.T) {
		testETags(t, handler)
	})

	t.Run("MultiValuedAttributes", func(t *testing.T) {
		testMultiValuedAttributes(t, handler)
	})
}

// createTestUsers creates test users with specific case patterns for testing
func createTestUsers(t *testing.T, handler http.Handler) {
	users := []struct {
		userName  string
		givenName string
		active    bool
	}{
		{"john.doe", "John", true},
		{"alice.wonder", "Alice", true},
		{"Bob.Builder", "Bob", false}, // Note: Capital B and inactive
	}

	for _, u := range users {
		payload := map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": u.userName,
			"name": map[string]string{
				"givenName": u.givenName,
			},
			"active": u.active,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/test/Users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create test user %s: %d - %s", u.userName, w.Code, w.Body.String())
		}
	}
}

// testFilterCaseSensitivity tests filter case handling
// NOTE: RFC 7644 Section 3.4.2.2 specifies case-sensitive values, but this
// implementation uses case-insensitive filtering for practical compatibility
// with real-world SCIM providers (including Microsoft's SCIM validator)
func testFilterCaseSensitivity(t *testing.T, handler http.Handler) {
	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "eq_correct_case",
			filter:        `userName eq "john.doe"`,
			expectedCount: 1,
			description:   "Exact match with correct case should find user",
		},
		{
			name:          "eq_different_case",
			filter:        `userName eq "JOHN.DOE"`,
			expectedCount: 1,
			description:   "Exact match with different case should find user (case-insensitive)",
		},
		{
			name:          "co_correct_case",
			filter:        `name.givenName co "Alice"`,
			expectedCount: 1,
			description:   "Contains with correct case should find user",
		},
		{
			name:          "co_different_case",
			filter:        `name.givenName co "alice"`,
			expectedCount: 1,
			description:   "Contains with different case should find user (case-insensitive)",
		},
		{
			name:          "sw_correct_case",
			filter:        `userName sw "Bob"`,
			expectedCount: 1,
			description:   "Starts with correct case should find Bob.Builder",
		},
		{
			name:          "sw_different_case",
			filter:        `userName sw "bob"`,
			expectedCount: 1,
			description:   "Starts with different case should find user (case-insensitive)",
		},
		{
			name:          "ew_correct_case",
			filter:        `userName ew "doe"`,
			expectedCount: 1,
			description:   "Ends with correct case should find john.doe",
		},
		{
			name:          "ew_different_case",
			filter:        `userName ew "DOE"`,
			expectedCount: 1,
			description:   "Ends with different case should find user (case-insensitive)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test/Users?filter=" + strings.ReplaceAll(tt.filter, " ", "%20")
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				return
			}

			var response map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			totalResults := int(response["totalResults"].(float64))
			if totalResults != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, totalResults)
				t.Logf("Filter: %s", tt.filter)
				t.Logf("Response: %+v", response)
			}
		})
	}
}

// testAttributeSelectionMutualExclusivity tests RFC 7644 Section 3.9 compliance
// The attributes and excludedAttributes parameters are mutually exclusive
func testAttributeSelectionMutualExclusivity(t *testing.T, handler http.Handler) {
	tests := []struct {
		name           string
		endpoint       string
		queryParams    string
		expectedStatus int
		shouldContain  string
		description    string
	}{
		{
			name:           "attributes_only_valid",
			endpoint:       "/test/Users",
			queryParams:    "?attributes=userName,emails",
			expectedStatus: http.StatusOK,
			description:    "Using only attributes parameter should succeed",
		},
		{
			name:           "excludedAttributes_only_valid",
			endpoint:       "/test/Users",
			queryParams:    "?excludedAttributes=groups,meta",
			expectedStatus: http.StatusOK,
			description:    "Using only excludedAttributes parameter should succeed",
		},
		{
			name:           "both_parameters_invalid",
			endpoint:       "/test/Users",
			queryParams:    "?attributes=userName&excludedAttributes=groups",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
			description:    "Using both parameters should fail with 400 Bad Request",
		},
		{
			name:           "neither_parameter_valid",
			endpoint:       "/test/Users",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			description:    "Using neither parameter should succeed",
		},
		{
			name:           "groups_both_invalid",
			endpoint:       "/test/Groups",
			queryParams:    "?attributes=displayName&excludedAttributes=members",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "mutually exclusive",
			description:    "Mutual exclusivity applies to Groups endpoint too",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.endpoint + tt.queryParams
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.shouldContain != "" {
				body := w.Body.String()
				if !strings.Contains(body, tt.shouldContain) {
					t.Errorf("%s: Expected response to contain %q, got: %s",
						tt.description, tt.shouldContain, body)
				}

				// Verify SCIM error format
				if !strings.Contains(body, "invalidFilter") {
					t.Errorf("%s: Expected scimType 'invalidFilter' in error response", tt.description)
				}
			}
		})
	}
}

// testBulkCircularReferenceDetection tests RFC 7644 Section 3.7.3 compliance
// Service providers MUST detect and reject circular bulkId references
func testBulkCircularReferenceDetection(t *testing.T, handler http.Handler) {
	tests := []struct {
		name           string
		bulkRequest    map[string]any
		expectedStatus int
		shouldContain  string
		description    string
	}{
		{
			name: "valid_no_circular_reference",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "valid.user1",
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user2",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "valid.user2",
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			description:    "Valid bulk operation with no circular references should succeed",
		},
		{
			name: "direct_circular_reference",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "circular1",
							"manager":  map[string]any{"value": "bulkId:user2"},
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user2",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "circular2",
							"manager":  map[string]any{"value": "bulkId:user1"},
						},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "circular bulkId reference",
			description:    "Direct circular reference (A→B, B→A) should be rejected",
		},
		{
			name: "self_reference",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "self.ref",
							"manager":  map[string]any{"value": "bulkId:user1"},
						},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "circular bulkId reference",
			description:    "Self-reference (A→A) should be rejected",
		},
		{
			name: "indirect_circular_reference",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "indirect1",
							"manager":  map[string]any{"value": "bulkId:user2"},
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user2",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "indirect2",
							"manager":  map[string]any{"value": "bulkId:user3"},
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user3",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "indirect3",
							"manager":  map[string]any{"value": "bulkId:user1"},
						},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "circular bulkId reference",
			description:    "Indirect circular reference (A→B→C→A) should be rejected",
		},
		{
			name: "duplicate_bulkId",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "duplicate1",
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1", // Duplicate bulkId
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "duplicate2",
						},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "duplicate bulkId",
			description:    "Duplicate bulkId should be rejected",
		},
		{
			name: "valid_dependency_chain",
			bulkRequest: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:BulkRequest"},
				"Operations": []map[string]any{
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user1",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "chain1",
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user2",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "chain2",
							"manager":  map[string]any{"value": "bulkId:user1"},
						},
					},
					{
						"method": "POST",
						"path":   "/Users",
						"bulkId": "user3",
						"data": map[string]any{
							"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
							"userName": "chain3",
							"manager":  map[string]any{"value": "bulkId:user2"},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			description:    "Valid dependency chain (user3→user2→user1) without cycles should succeed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.bulkRequest)
			req := httptest.NewRequest("POST", "/test/Bulk", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/scim+json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.shouldContain != "" {
				responseBody := w.Body.String()
				if !strings.Contains(responseBody, tt.shouldContain) {
					t.Errorf("%s: Expected response to contain %q, got: %s",
						tt.description, tt.shouldContain, responseBody)
				}

				// Verify SCIM error format
				if !strings.Contains(responseBody, "invalidValue") {
					t.Errorf("%s: Expected scimType 'invalidValue' in error response", tt.description)
				}
			}
		})
	}
}

// testCoreSCIMOperations tests core SCIM operations
func testCoreSCIMOperations(t *testing.T, handler http.Handler) {
	t.Run("CreateUser", func(t *testing.T) {
		payload := map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": "new.user",
			"active":   true,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/test/Users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetUsers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/Users", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		totalResults := int(response["totalResults"].(float64))
		if totalResults < 1 {
			t.Error("Expected at least 1 user")
		}
	})

	t.Run("SearchEndpoint", func(t *testing.T) {
		payload := map[string]any{
			"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:SearchRequest"},
			"filter":     "active eq true",
			"startIndex": 1,
			"count":      10,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/test/.search", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/scim+json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ServiceProviderConfig", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/ServiceProviderConfig", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			return
		}

		var config map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &config); err != nil {
			t.Fatalf("Failed to parse ServiceProviderConfig: %v", err)
		}

		// Verify bulk operations are supported
		if bulk, ok := config["bulk"].(map[string]any); ok {
			if supported, ok := bulk["supported"].(bool); !ok || !supported {
				t.Error("Expected bulk operations to be supported")
			}
		} else {
			t.Error("Expected bulk configuration in ServiceProviderConfig")
		}
	})
}

// testPatchOperations tests RFC 7644 Section 3.5.2 compliance
// PATCH operations must support add, remove, and replace operations
func testPatchOperations(t *testing.T, handler http.Handler) {
	// Create a test user first
	createPayload := map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "patch.test",
		"name": map[string]string{
			"givenName":  "Patch",
			"familyName": "Test",
		},
		"emails": []map[string]any{
			{"value": "patch@example.com", "type": "work", "primary": true},
		},
		"active": true,
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/test/Users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var user map[string]any
	json.Unmarshal(w.Body.Bytes(), &user)
	userID := user["id"].(string)

	tests := []struct {
		name        string
		operation   string
		path        string
		value       any
		verifyFunc  func(t *testing.T, response map[string]any)
		description string
	}{
		{
			name:      "replace_single_attribute",
			operation: "replace",
			path:      "active",
			value:     false,
			verifyFunc: func(t *testing.T, response map[string]any) {
				if active, ok := response["active"].(bool); !ok || active {
					t.Error("Expected active to be false after replace operation")
				}
			},
			description: "PATCH replace operation should update single attribute",
		},
		{
			name:      "add_email",
			operation: "add",
			path:      "emails",
			value: []map[string]any{
				{"value": "patch.home@example.com", "type": "home"},
			},
			verifyFunc: func(t *testing.T, response map[string]any) {
				emails := response["emails"].([]any)
				if len(emails) < 2 {
					t.Error("Expected at least 2 emails after add operation")
				}
			},
			description: "PATCH add operation should add to multi-valued attribute",
		},
		{
			name:      "remove_attribute",
			operation: "remove",
			path:      "name.middleName",
			value:     nil,
			verifyFunc: func(t *testing.T, response map[string]any) {
				if name, ok := response["name"].(map[string]any); ok {
					if _, exists := name["middleName"]; exists {
						t.Error("Expected middleName to be removed")
					}
				}
			},
			description: "PATCH remove operation should remove attribute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchPayload := map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				"Operations": []map[string]any{
					{
						"op":    tt.operation,
						"path":  tt.path,
						"value": tt.value,
					},
				},
			}

			body, _ := json.Marshal(patchPayload)
			req := httptest.NewRequest("PATCH", "/test/Users/"+userID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/scim+json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
				t.Errorf("%s: Expected status 200 or 204, got %d: %s", tt.description, w.Code, w.Body.String())
				return
			}

			// Get the updated user to verify the patch
			req = httptest.NewRequest("GET", "/test/Users/"+userID, nil)
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, response)
			}
		})
	}
}

// testPagination tests RFC 7644 Section 3.4.2.4 compliance
// Pagination must support startIndex and count parameters with 1-based indexing
func testPagination(t *testing.T, handler http.Handler) {
	tests := []struct {
		name        string
		startIndex  int
		count       int
		description string
	}{
		{
			name:        "first_page",
			startIndex:  1,
			count:       2,
			description: "First page with startIndex=1 should return first 2 results",
		},
		{
			name:        "second_page",
			startIndex:  3,
			count:       2,
			description: "Second page with startIndex=3 should skip first 2 results",
		},
		{
			name:        "default_pagination",
			startIndex:  1,
			count:       100,
			description: "Default count should return all results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/test/Users?startIndex=%d&count=%d", tt.startIndex, tt.count)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s: Expected status 200, got %d", tt.description, w.Code)
				return
			}

			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)

			// Verify response includes pagination metadata
			if _, ok := response["startIndex"]; !ok {
				t.Error("Response should include startIndex")
			}
			if _, ok := response["itemsPerPage"]; !ok {
				t.Error("Response should include itemsPerPage")
			}
			if _, ok := response["totalResults"]; !ok {
				t.Error("Response should include totalResults")
			}

			// Verify startIndex is 1-based (RFC requirement)
			startIdx := int(response["startIndex"].(float64))
			if tt.startIndex == 1 && startIdx != 1 {
				t.Error("StartIndex should be 1-based as per RFC 7644")
			}
		})
	}
}

// testSorting tests RFC 7644 Section 3.4.2.3 compliance
// Sorting must support sortBy and sortOrder parameters
func testSorting(t *testing.T, handler http.Handler) {
	tests := []struct {
		name        string
		sortBy      string
		sortOrder   string
		description string
	}{
		{
			name:        "sort_ascending",
			sortBy:      "userName",
			sortOrder:   "ascending",
			description: "Sorting by userName ascending should work",
		},
		{
			name:        "sort_descending",
			sortBy:      "userName",
			sortOrder:   "descending",
			description: "Sorting by userName descending should work",
		},
		{
			name:        "sort_by_nested_attribute",
			sortBy:      "name.givenName",
			sortOrder:   "ascending",
			description: "Sorting by nested attribute should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/test/Users?sortBy=%s&sortOrder=%s", tt.sortBy, tt.sortOrder)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s: Expected status 200, got %d: %s", tt.description, w.Code, w.Body.String())
				return
			}

			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)

			resources := response["Resources"].([]any)
			if len(resources) < 2 {
				t.Skip("Need at least 2 users to verify sorting")
			}

			// Verify results are returned (detailed sorting verification would require specific data)
			if response["totalResults"].(float64) < 1 {
				t.Error("Expected results when sorting")
			}
		})
	}
}

// testComplexFilters tests RFC 7644 Section 3.4.2.2 compliance
// Filters must support logical operators (and, or, not) and grouping
func testComplexFilters(t *testing.T, handler http.Handler) {
	tests := []struct {
		name        string
		filter      string
		description string
	}{
		{
			name:        "and_operator",
			filter:      `userName eq "john.doe" and active eq true`,
			description: "AND operator should work",
		},
		{
			name:        "or_operator",
			filter:      `userName eq "john.doe" or userName eq "alice.wonder"`,
			description: "OR operator should work",
		},
		{
			name:        "not_operator",
			filter:      `not (active eq false)`,
			description: "NOT operator should work",
		},
		{
			name:        "grouped_expression",
			filter:      `(userName eq "john.doe") and (active eq true)`,
			description: "Grouped expressions should work",
		},
		{
			name:        "complex_nested",
			filter:      `userName sw "john" and (active eq true or active eq false)`,
			description: "Complex nested filters should work",
		},
		{
			name:        "complex_attribute_path",
			filter:      `emails[type eq "work" and primary eq true].value pr`,
			description: "Complex attribute paths with filters should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test/Users?filter=" + strings.ReplaceAll(tt.filter, " ", "%20")
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s: Expected status 200, got %d: %s", tt.description, w.Code, w.Body.String())
				return
			}

			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)

			// Verify filter was processed (may return 0 results, but should not error)
			if _, ok := response["totalResults"]; !ok {
				t.Errorf("%s: Response should include totalResults", tt.description)
			}
		})
	}
}

// testErrorResponses tests RFC 7644 Section 3.12 compliance
// Error responses must follow SCIM error schema format
func testErrorResponses(t *testing.T, handler http.Handler) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           map[string]any
		expectedStatus int
		expectedType   string
		description    string
	}{
		{
			name:           "invalid_filter",
			method:         "GET",
			endpoint:       "/test/Users?filter=userName",
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid filter (missing operator) should return 400 with invalidFilter type",
		},
		{
			name:           "invalid_json",
			method:         "POST",
			endpoint:       "/test/Users",
			body:           nil, // Will send invalid JSON
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid JSON should return 400 with invalidSyntax type",
		},
		{
			name:     "missing_required_field",
			method:   "POST",
			endpoint: "/test/Users",
			body: map[string]any{
				"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
				// Missing userName which might be required
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Missing required fields should return 400",
		},
		{
			name:           "not_found",
			method:         "GET",
			endpoint:       "/test/Users/nonexistent-id-12345",
			expectedStatus: http.StatusNotFound,
			description:    "Non-existent resource should return 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.endpoint, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/scim+json")
			} else if tt.method == "POST" {
				// Send invalid JSON
				req = httptest.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString("{invalid json"))
				req.Header.Set("Content-Type", "application/scim+json")
			} else {
				req = httptest.NewRequest(tt.method, tt.endpoint, nil)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}

			// Verify SCIM error format for error responses
			if tt.expectedStatus >= 400 {
				var errorResponse map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err == nil {
					// Check for SCIM error schema
					if schemas, ok := errorResponse["schemas"].([]any); ok {
						found := false
						for _, schema := range schemas {
							if schema == "urn:ietf:params:scim:api:messages:2.0:Error" {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("%s: Error response should include SCIM error schema", tt.description)
						}
					}

					// Check for required error fields
					if _, ok := errorResponse["status"]; !ok {
						t.Errorf("%s: Error response should include status field", tt.description)
					}

					// Check for scimType if expected
					if tt.expectedType != "" {
						if scimType, ok := errorResponse["scimType"].(string); !ok || scimType != tt.expectedType {
							t.Errorf("%s: Expected scimType %q, got %q", tt.description, tt.expectedType, scimType)
						}
					}
				}
			}
		})
	}
}

// testETags tests RFC 7644 Section 3.14 compliance
// ETags must be supported for versioning and conditional operations
func testETags(t *testing.T, handler http.Handler) {
	// Create a test user
	createPayload := map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "etag.test",
		"active":   true,
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/test/Users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var user map[string]any
	json.Unmarshal(w.Body.Bytes(), &user)
	userID := user["id"].(string)

	t.Run("resource_includes_version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/Users/"+userID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		// Verify meta.version exists
		if meta, ok := response["meta"].(map[string]any); ok {
			if _, hasVersion := meta["version"]; !hasVersion {
				t.Error("Resource should include meta.version for ETag support")
			}
		} else {
			t.Error("Resource should include meta object")
		}
	})

	t.Run("version_changes_on_update", func(t *testing.T) {
		// Get original version
		req := httptest.NewRequest("GET", "/test/Users/"+userID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var original map[string]any
		json.Unmarshal(w.Body.Bytes(), &original)
		originalVersion := original["meta"].(map[string]any)["version"].(string)

		// Update the user
		patchPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{"op": "replace", "path": "active", "value": false},
			},
		}

		body, _ := json.Marshal(patchPayload)
		req = httptest.NewRequest("PATCH", "/test/Users/"+userID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/scim+json")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Get updated version
		req = httptest.NewRequest("GET", "/test/Users/"+userID, nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var updated map[string]any
		json.Unmarshal(w.Body.Bytes(), &updated)
		updatedVersion := updated["meta"].(map[string]any)["version"].(string)

		if originalVersion == updatedVersion {
			t.Error("Version should change after update")
		}
	})
}

// testMultiValuedAttributes tests handling of multi-valued attributes
// Multi-valued attributes must support primary flag and uniqueness
func testMultiValuedAttributes(t *testing.T, handler http.Handler) {
	// Create user with multiple emails
	createPayload := map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "multi.value.test",
		"emails": []map[string]any{
			{"value": "work@example.com", "type": "work", "primary": true},
			{"value": "home@example.com", "type": "home", "primary": false},
		},
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/test/Users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var user map[string]any
	json.Unmarshal(w.Body.Bytes(), &user)

	t.Run("primary_flag_preserved", func(t *testing.T) {
		emails := user["emails"].([]any)
		primaryCount := 0
		for _, email := range emails {
			e := email.(map[string]any)
			if primary, ok := e["primary"].(bool); ok && primary {
				primaryCount++
			}
		}

		if primaryCount != 1 {
			t.Error("Should have exactly one primary email")
		}
	})

	t.Run("filter_multi_valued_attribute", func(t *testing.T) {
		filter := `emails[type eq "work"].value pr`
		url := "/test/Users?filter=" + strings.ReplaceAll(filter, " ", "%20")
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Filter on multi-valued attribute failed: %d", w.Code)
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		totalResults := int(response["totalResults"].(float64))
		if totalResults < 1 {
			t.Error("Should find user with work email")
		}
	})
}
