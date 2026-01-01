package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/internal/testutil"
	"github.com/marcelom97/scimgateway/scim"
)

// TestCase represents a single HTTP test case
type TestCase struct {
	Name           string
	Method         string
	Path           string
	Body           string
	Headers        map[string]string
	Setup          func(t *testing.T, server *httptest.Server) map[string]string // Returns context (e.g., created IDs)
	ExpectedStatus int
	Validate       func(t *testing.T, resp *http.Response, context map[string]string)
}

func TestHTTPEndpoints_TableDriven(t *testing.T) {
	tests := []TestCase{
		// ============================================
		// DISCOVERY ENDPOINTS (Per-plugin)
		// ============================================
		{
			Name:           "GET /test/ServiceProviderConfig",
			Method:         "GET",
			Path:           "/test/ServiceProviderConfig",
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var result map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if schemas, ok := result["schemas"].([]any); !ok || len(schemas) == 0 {
					t.Error("Expected schemas array")
				}
			},
		},
		{
			Name:           "GET /test/ResourceTypes",
			Method:         "GET",
			Path:           "/test/ResourceTypes",
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var result map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				resources, ok := result["Resources"].([]any)
				if !ok {
					t.Error("Expected Resources array")
					return
				}
				if len(resources) == 0 {
					t.Error("Expected resource types array")
				}
			},
		},
		{
			Name:           "GET /test/Schemas",
			Method:         "GET",
			Path:           "/test/Schemas",
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var result []any
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if len(result) == 0 {
					t.Error("Expected schemas array")
				}
			},
		},

		// ============================================
		// USER CRUD OPERATIONS
		// ============================================
		{
			Name:   "POST /Users - Create user with all fields",
			Method: "POST",
			Path:   "/test/Users",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
				"userName": "john.doe",
				"name": {
					"givenName": "John",
					"familyName": "Doe"
				},
				"active": true,
				"emails": [
					{
						"value": "john@example.com",
						"type": "work",
						"primary": true
					}
				]
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusCreated,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if user.UserName != "john.doe" {
					t.Errorf("Expected userName 'john.doe', got '%s'", user.UserName)
				}
				if user.ID == "" {
					t.Error("Expected ID to be generated")
				}
				if user.Active == nil || !*user.Active {
					t.Error("Expected active to be true")
				}
				location := resp.Header.Get("Location")
				if location == "" {
					t.Error("Expected Location header")
				}
			},
		},
		{
			Name:   "POST /Users - Create user without active (should default to true)",
			Method: "POST",
			Path:   "/test/Users",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
				"userName": "default.active"
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusCreated,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				if user.Active == nil || !*user.Active {
					t.Error("Expected active to default to true")
				}
			},
		},
		{
			Name:   "POST /Users - Create user with active=false",
			Method: "POST",
			Path:   "/test/Users",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
				"userName": "inactive.user",
				"active": false
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusCreated,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				if user.Active != nil && *user.Active {
					t.Error("Expected active to be false")
				}
			},
		},
		{
			Name:           "POST /Users - Invalid JSON",
			Method:         "POST",
			Path:           "/test/Users",
			Body:           `{invalid json}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "POST /Users - Missing userName",
			Method: "POST",
			Path:   "/test/Users",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
				"active": true
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "GET /Users - List all users",
			Method: "GET",
			Path:   "/test/Users",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				// Create 3 users
				for i := 1; i <= 3; i++ {
					body := fmt.Sprintf(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "user%d"}`, i)
					http.Post(server.URL+"/test/Users", "application/scim+json", bytes.NewBufferString(body))
				}
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.User]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if listResp.TotalResults != 3 {
					t.Errorf("Expected 3 users, got %d", listResp.TotalResults)
				}
			},
		},
		{
			Name:   "GET /Users?filter=active eq true - Filter users",
			Method: "GET",
			Path:   "/test/Users?filter=active%20eq%20true",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "active1", "active": true}`))
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "inactive1", "active": false}`))
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.User]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if listResp.TotalResults != 1 {
					t.Errorf("Expected 1 active user, got %d", listResp.TotalResults)
				}
			},
		},
		{
			Name:   "GET /Users?attributes=userName - Attribute selection",
			Method: "GET",
			Path:   "/test/Users?attributes=userName",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "attr.test", "active": true, "name": {"givenName": "Test"}}`))
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.User]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if len(listResp.Resources) == 0 {
					t.Fatal("Expected at least 1 user")
				}
				user := listResp.Resources[0]
				if user.UserName != "attr.test" {
					t.Errorf("Expected userName 'attr.test', got '%s'", user.UserName)
				}
				if user.Name != nil {
					t.Error("Expected name to be filtered out")
				}
			},
		},
		{
			Name:   "GET /Users?excludedAttributes=name - Exclude attributes",
			Method: "GET",
			Path:   "/test/Users?excludedAttributes=name",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "exclude.test", "name": {"givenName": "Test"}}`))
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.User]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if len(listResp.Resources) > 0 && listResp.Resources[0].Name != nil {
					t.Error("Expected name to be excluded")
				}
			},
		},
		{
			Name:   "GET /Users?startIndex=1&count=2 - Pagination",
			Method: "GET",
			Path:   "/test/Users?startIndex=1&count=2",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				for i := 1; i <= 5; i++ {
					body := fmt.Sprintf(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "page%d"}`, i)
					http.Post(server.URL+"/test/Users", "application/scim+json", bytes.NewBufferString(body))
				}
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.User]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if listResp.ItemsPerPage != 2 {
					t.Errorf("Expected 2 items per page, got %d", listResp.ItemsPerPage)
				}
				if listResp.TotalResults != 5 {
					t.Errorf("Expected total 5, got %d", listResp.TotalResults)
				}
			},
		},
		{
			Name:   "GET /Users/{id} - Get single user",
			Method: "GET",
			Path:   "/test/Users/{userID}",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "get.user"}`))
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				resp.Body.Close()
				return map[string]string{"userID": user.ID}
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				if user.UserName != "get.user" {
					t.Errorf("Expected userName 'get.user', got '%s'", user.UserName)
				}
			},
		},
		{
			Name:           "GET /Users/{id} - Non-existent user",
			Method:         "GET",
			Path:           "/test/Users/non-existent-id",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:   "PUT /Users/{id} - Replace user",
			Method: "PUT",
			Path:   "/test/Users/{userID}",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
				"userName": "updated.user",
				"active": false
			}`,
			Headers: map[string]string{"Content-Type": "application/scim+json"},
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "original.user"}`))
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				resp.Body.Close()
				return map[string]string{"userID": user.ID}
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				if user.UserName != "updated.user" {
					t.Errorf("Expected userName 'updated.user', got '%s'", user.UserName)
				}
				if user.Active != nil && *user.Active {
					t.Error("Expected active to be false")
				}
			},
		},
		{
			Name:   "PATCH /Users/{id} - Modify user",
			Method: "PATCH",
			Path:   "/test/Users/{userID}",
			Body: `{
				"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
				"Operations": [
					{
						"op": "replace",
						"path": "active",
						"value": false
					}
				]
			}`,
			Headers: map[string]string{"Content-Type": "application/scim+json"},
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "patch.user", "active": true}`))
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				resp.Body.Close()
				return map[string]string{"userID": user.ID}
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				if user.Active != nil && *user.Active {
					t.Error("Expected active to be false after patch")
				}
			},
		},
		{
			Name:   "DELETE /Users/{id} - Delete user",
			Method: "DELETE",
			Path:   "/test/Users/{userID}",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "delete.user"}`))
				var user scim.User
				json.NewDecoder(resp.Body).Decode(&user)
				resp.Body.Close()
				return map[string]string{"userID": user.ID}
			},
			ExpectedStatus: http.StatusNoContent,
		},
		{
			Name:           "DELETE /Users/{id} - Non-existent user",
			Method:         "DELETE",
			Path:           "/test/Users/non-existent-id",
			ExpectedStatus: http.StatusNotFound,
		},

		// ============================================
		// GROUP CRUD OPERATIONS
		// ============================================
		{
			Name:   "POST /Groups - Create group",
			Method: "POST",
			Path:   "/test/Groups",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
				"displayName": "Admins"
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusCreated,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var group scim.Group
				json.NewDecoder(resp.Body).Decode(&group)
				if group.DisplayName != "Admins" {
					t.Errorf("Expected displayName 'Admins', got '%s'", group.DisplayName)
				}
				if group.ID == "" {
					t.Error("Expected ID to be generated")
				}
			},
		},
		{
			Name:   "POST /Groups - Missing displayName",
			Method: "POST",
			Path:   "/test/Groups",
			Body: `{
				"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"]
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "GET /Groups - List all groups",
			Method: "GET",
			Path:   "/test/Groups",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				http.Post(server.URL+"/test/Groups", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"], "displayName": "Group1"}`))
				http.Post(server.URL+"/test/Groups", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"], "displayName": "Group2"}`))
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[*scim.Group]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if listResp.TotalResults != 2 {
					t.Errorf("Expected 2 groups, got %d", listResp.TotalResults)
				}
			},
		},
		{
			Name:   "GET /Groups/{id} - Get single group",
			Method: "GET",
			Path:   "/test/Groups/{groupID}",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Groups", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"], "displayName": "TestGroup"}`))
				var group scim.Group
				json.NewDecoder(resp.Body).Decode(&group)
				resp.Body.Close()
				return map[string]string{"groupID": group.ID}
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var group scim.Group
				json.NewDecoder(resp.Body).Decode(&group)
				if group.DisplayName != "TestGroup" {
					t.Errorf("Expected displayName 'TestGroup', got '%s'", group.DisplayName)
				}
			},
		},
		{
			Name:   "DELETE /Groups/{id} - Delete group",
			Method: "DELETE",
			Path:   "/test/Groups/{groupID}",
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				resp, _ := http.Post(server.URL+"/test/Groups", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"], "displayName": "DeleteMe"}`))
				var group scim.Group
				json.NewDecoder(resp.Body).Decode(&group)
				resp.Body.Close()
				return map[string]string{"groupID": group.ID}
			},
			ExpectedStatus: http.StatusNoContent,
		},

		// ============================================
		// SEARCH OPERATIONS
		// ============================================
		{
			Name:   "POST /.search - Global search",
			Method: "POST",
			Path:   "/test/.search",
			Body: `{
				"schemas": ["urn:ietf:params:scim:api:messages:2.0:SearchRequest"],
				"filter": "active eq true",
				"startIndex": 1,
				"count": 10
			}`,
			Headers: map[string]string{"Content-Type": "application/scim+json"},
			Setup: func(t *testing.T, server *httptest.Server) map[string]string {
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "search1", "active": true}`))
				http.Post(server.URL+"/test/Users", "application/scim+json",
					bytes.NewBufferString(`{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "search2", "active": false}`))
				return nil
			},
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, resp *http.Response, context map[string]string) {
				var listResp scim.ListResponse[map[string]any]
				json.NewDecoder(resp.Body).Decode(&listResp)
				if listResp.TotalResults != 1 {
					t.Errorf("Expected 1 result, got %d", listResp.TotalResults)
				}
			},
		},
		{
			Name:   "POST /.search - Invalid schema",
			Method: "POST",
			Path:   "/test/.search",
			Body: `{
				"schemas": ["invalid:schema"],
				"filter": "active eq true"
			}`,
			Headers:        map[string]string{"Content-Type": "application/scim+json"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "GET /.search - Method not allowed",
			Method:         "GET",
			Path:           "/test/.search",
			ExpectedStatus: http.StatusMethodNotAllowed,
		},

		// ============================================
		// EDGE CASES & ERROR SCENARIOS
		// ============================================
		{
			Name:           "GET /UnknownPlugin/Users - Invalid plugin",
			Method:         "GET",
			Path:           "/unknown/Users",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "GET /test/InvalidResource - Invalid resource type",
			Method:         "GET",
			Path:           "/test/InvalidResource",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "POST /Users - Missing Content-Type header",
			Method:         "POST",
			Path:           "/test/Users",
			Body:           `{"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"], "userName": "test"}`,
			ExpectedStatus: http.StatusCreated, // Should still work
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create fresh server for each test to ensure isolation
			cfg := &config.Config{
				Gateway: config.GatewayConfig{
					BaseURL: "http://localhost:8080",
					Port:    8080,
				},
				Plugins: []config.PluginConfig{{Name: "test"}},
			}
			gw := scimgateway.New(cfg)
			gw.RegisterPlugin(testutil.NewMemoryPlugin("test"))
			if err := gw.Initialize(); err != nil {
				t.Fatalf("Failed to initialize gateway: %v", err)
			}
			handler, err := gw.Handler()
			if err != nil {
				t.Fatalf("Handler() error = %v", err)
			}
			server := httptest.NewServer(handler)
			defer server.Close()

			// Run setup if provided
			var context map[string]string
			if tt.Setup != nil {
				context = tt.Setup(t, server)
			}

			// Replace placeholders in path with actual values from context
			path := tt.Path
			for key, value := range context {
				path = strings.ReplaceAll(path, "{"+key+"}", value)
			}

			// Create request
			var req *http.Request
			var reqErr error
			if tt.Body != "" {
				req, reqErr = http.NewRequest(tt.Method, server.URL+path, bytes.NewBufferString(tt.Body))
			} else {
				req, reqErr = http.NewRequest(tt.Method, server.URL+path, nil)
			}
			if reqErr != nil {
				t.Fatalf("Failed to create request: %v", reqErr)
			}

			// Set headers
			if tt.Headers != nil {
				for key, value := range tt.Headers {
					req.Header.Set(key, value)
				}
			}

			// Execute request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.ExpectedStatus {
				body := new(bytes.Buffer)
				body.ReadFrom(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.ExpectedStatus, resp.StatusCode, body.String())
				return
			}

			// Run custom validation if provided
			if tt.Validate != nil {
				// Need to recreate response body reader for validation
				bodyBytes := new(bytes.Buffer)
				bodyBytes.ReadFrom(resp.Body)
				resp.Body.Close()
				resp.Body = http.NoBody

				// Create new response with body
				newResp := &http.Response{
					Status:        resp.Status,
					StatusCode:    resp.StatusCode,
					Proto:         resp.Proto,
					ProtoMajor:    resp.ProtoMajor,
					ProtoMinor:    resp.ProtoMinor,
					Header:        resp.Header,
					Body:          http.NoBody,
					ContentLength: resp.ContentLength,
					Close:         resp.Close,
					Uncompressed:  resp.Uncompressed,
					Trailer:       resp.Trailer,
					Request:       resp.Request,
					TLS:           resp.TLS,
				}
				newResp.Body = io.NopCloser(bytes.NewReader(bodyBytes.Bytes()))

				tt.Validate(t, newResp, context)
			}
		})
	}
}
