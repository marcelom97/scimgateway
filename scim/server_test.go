package scim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServer("http://localhost:8080", pm)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	if srv.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %s, want http://localhost:8080", srv.baseURL)
	}

	if srv.logger == nil {
		t.Error("logger should not be nil")
	}
}

// TestNewServerWithLogger tests server creation with logger
func TestNewServerWithLogger(t *testing.T) {
	pm := &mockPluginManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServerWithLogger("http://localhost:8080", pm, logger)

	if srv == nil {
		t.Fatal("NewServerWithLogger returned nil")
	}

	if srv.logger != logger {
		t.Error("logger not set correctly")
	}
}

// TestNewServerWithNilLogger tests server handles nil logger
func TestNewServerWithNilLogger(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServerWithLogger("http://localhost:8080", pm, nil)

	if srv == nil {
		t.Fatal("NewServerWithLogger returned nil")
	}

	if srv.logger == nil {
		t.Error("logger should not be nil when nil is passed")
	}
}

// TestBaseURLTrailingSlash tests that trailing slash is removed
func TestBaseURLTrailingSlash(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServer("http://localhost:8080/", pm)

	if srv.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %s, want http://localhost:8080 (trailing slash should be removed)", srv.baseURL)
	}
}

// TestGetPluginNotFound tests plugin not found scenario
func TestGetPluginNotFound(t *testing.T) {
	pm := &mockPluginManager{plugin: nil}
	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/nonexistent/Users", nil)
	_, ok := srv.getPlugin("nonexistent", "/Users", req)

	if ok {
		t.Error("getPlugin should return false when plugin is nil")
	}
}

// TestGetPluginFound tests plugin found scenario
func TestGetPluginFound(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Users", nil)
	p, ok := srv.getPlugin("test", "/Users", req)

	if !ok {
		t.Error("getPlugin should return true for existing plugin")
	}

	if p == nil {
		t.Error("getPlugin should return plugin, got nil")
	}
}

// TestHandlePluginErrorWithSCIMError tests handlePluginError with SCIMError
func TestHandlePluginErrorWithSCIMError(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServer("http://localhost:8080", pm)

	w := httptest.NewRecorder()
	err := &SCIMError{
		Status:   http.StatusBadRequest,
		Detail:   "test error",
		ScimType: "invalidValue",
	}

	srv.handlePluginError(w, err, http.StatusInternalServerError, "serverError")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["detail"] != "test error" {
		t.Errorf("detail = %s, want 'test error'", resp["detail"])
	}
}

// TestHandlePluginErrorWithRegularError tests handlePluginError with regular error
func TestHandlePluginErrorWithRegularError(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServer("http://localhost:8080", pm)

	w := httptest.NewRecorder()
	err := fmt.Errorf("regular error")

	srv.handlePluginError(w, err, http.StatusInternalServerError, "serverError")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["detail"] != "regular error" {
		t.Errorf("detail = %s, want 'regular error'", resp["detail"])
	}
}

// TestHandleServiceProviderConfig tests ServiceProviderConfig endpoint
func TestHandleServiceProviderConfig(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/ServiceProviderConfig", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check basic structure
	if _, ok := resp["schemas"]; !ok {
		t.Error("response missing 'schemas' field")
	}
}

// TestHandleServiceProviderConfigPluginNotFound tests ServiceProviderConfig with non-existent plugin
func TestHandleServiceProviderConfigPluginNotFound(t *testing.T) {
	pm := &mockPluginManager{plugin: nil}
	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/nonexistent/ServiceProviderConfig", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestHandleResourceTypes tests ResourceTypes endpoint
func TestHandleResourceTypes(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/ResourceTypes", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// ResourceTypes returns {"Resources": [...]}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := resp["Resources"]; !ok {
		t.Error("response missing 'Resources' field")
	}
}

// TestHandleSchemas tests Schemas endpoint
func TestHandleSchemas(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Schemas", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Schemas returns an array of schema definitions
	var resp []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) == 0 {
		t.Error("response should contain schemas")
	}
}

// TestHandleGetUsers tests GET /Users endpoint
func TestHandleGetUsers(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Add a test user
	user := &User{
		ID:       "user1",
		UserName: "testuser",
		Active:   Bool(true),
	}
	plugin.CreateUser(context.Background(), user)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Users", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ListResponse[*User]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalResults != 1 {
		t.Errorf("totalResults = %d, want 1", resp.TotalResults)
	}
}

// TestHandleGetUsersPluginNotFound tests GET /Users with non-existent plugin
func TestHandleGetUsersPluginNotFound(t *testing.T) {
	pm := &mockPluginManager{plugin: nil}
	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/nonexistent/Users", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestHandleCreateUser tests POST /Users endpoint
func TestHandleCreateUser(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	user := &User{
		UserName: "newuser",
		Active:   Bool(true),
	}

	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/test/Users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.UserName != "newuser" {
		t.Errorf("userName = %s, want 'newuser'", resp.UserName)
	}

	if resp.ID == "" {
		t.Error("ID should be generated")
	}
}

// TestHandleCreateUserInvalidJSON tests POST /Users with invalid JSON
func TestHandleCreateUserInvalidJSON(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("POST", "/test/Users", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleGetUser tests GET /Users/{id} endpoint
func TestHandleGetUser(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test user
	user := &User{
		ID:       "user1",
		UserName: "testuser",
		Active:   Bool(true),
	}
	plugin.CreateUser(context.Background(), user)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Users/user1", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "user1" {
		t.Errorf("id = %s, want 'user1'", resp.ID)
	}
}

// TestHandleGetUserNotFound tests GET /Users/{id} with non-existent user
func TestHandleGetUserNotFound(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Users/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestHandlePatchUser tests PATCH /Users/{id} endpoint
func TestHandlePatchUser(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test user
	user := &User{
		ID:       "user1",
		UserName: "testuser",
		Active:   Bool(true),
	}
	plugin.CreateUser(context.Background(), user)

	srv := NewServer("http://localhost:8080", pm)

	patch := &PatchOp{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		Operations: []PatchOperation{
			{
				Op:    "replace",
				Path:  "active",
				Value: false,
			},
		},
	}

	body, _ := json.Marshal(patch)
	req := httptest.NewRequest("PATCH", "/test/Users/user1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestHandleDeleteUser tests DELETE /Users/{id} endpoint
func TestHandleDeleteUser(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test user
	user := &User{
		ID:       "user1",
		UserName: "testuser",
		Active:   Bool(true),
	}
	plugin.CreateUser(context.Background(), user)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("DELETE", "/test/Users/user1", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TestHandleGetGroups tests GET /Groups endpoint
func TestHandleGetGroups(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Add a test group
	group := &Group{
		ID:          "group1",
		DisplayName: "testgroup",
	}
	plugin.CreateGroup(context.Background(), group)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Groups", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ListResponse[*Group]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalResults != 1 {
		t.Errorf("totalResults = %d, want 1", resp.TotalResults)
	}
}

// TestHandleCreateGroup tests POST /Groups endpoint
func TestHandleCreateGroup(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	srv := NewServer("http://localhost:8080", pm)

	group := &Group{
		DisplayName: "newgroup",
	}

	body, _ := json.Marshal(group)
	req := httptest.NewRequest("POST", "/test/Groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp Group
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DisplayName != "newgroup" {
		t.Errorf("displayName = %s, want 'newgroup'", resp.DisplayName)
	}
}

// TestHandleGetGroup tests GET /Groups/{id} endpoint
func TestHandleGetGroup(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test group
	group := &Group{
		ID:          "group1",
		DisplayName: "testgroup",
	}
	plugin.CreateGroup(context.Background(), group)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/test/Groups/group1", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp Group
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "group1" {
		t.Errorf("id = %s, want 'group1'", resp.ID)
	}
}

// TestHandleDeleteGroup tests DELETE /Groups/{id} endpoint
func TestHandleDeleteGroup(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test group
	group := &Group{
		ID:          "group1",
		DisplayName: "testgroup",
	}
	plugin.CreateGroup(context.Background(), group)

	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("DELETE", "/test/Groups/group1", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TestServeHTTP tests that ServeHTTP delegates to mux
func TestServeHTTP(t *testing.T) {
	pm := &mockPluginManager{}
	srv := NewServer("http://localhost:8080", pm)

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Should get 404 for unknown route
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestHandleReplaceUser tests PUT /Users/{id} endpoint
func TestHandleReplaceUser(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test user
	user := &User{
		ID:       "user1",
		UserName: "testuser",
		Active:   Bool(true),
	}
	plugin.CreateUser(context.Background(), user)

	srv := NewServer("http://localhost:8080", pm)

	// Replace user
	newUser := &User{
		ID:       "user1",
		UserName: "updated",
		Active:   Bool(false),
	}

	body, _ := json.Marshal(newUser)
	req := httptest.NewRequest("PUT", "/test/Users/user1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp User
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.UserName != "updated" {
		t.Errorf("userName = %s, want 'updated'", resp.UserName)
	}
}

// TestHandleReplaceGroup tests PUT /Groups/{id} endpoint
func TestHandleReplaceGroup(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test group
	group := &Group{
		ID:          "group1",
		DisplayName: "testgroup",
	}
	plugin.CreateGroup(context.Background(), group)

	srv := NewServer("http://localhost:8080", pm)

	// Replace group
	newGroup := &Group{
		ID:          "group1",
		DisplayName: "updated",
	}

	body, _ := json.Marshal(newGroup)
	req := httptest.NewRequest("PUT", "/test/Groups/group1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp Group
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DisplayName != "updated" {
		t.Errorf("displayName = %s, want 'updated'", resp.DisplayName)
	}
}

// TestHandlePatchGroup tests PATCH /Groups/{id} endpoint
func TestHandlePatchGroup(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}

	// Create a test group
	group := &Group{
		ID:          "group1",
		DisplayName: "testgroup",
	}
	plugin.CreateGroup(context.Background(), group)

	srv := NewServer("http://localhost:8080", pm)

	patch := &PatchOp{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		Operations: []PatchOperation{
			{
				Op:    "replace",
				Path:  "displayName",
				Value: "newname",
			},
		},
	}

	body, _ := json.Marshal(patch)
	req := httptest.NewRequest("PATCH", "/test/Groups/group1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
