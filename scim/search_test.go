package scim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestServer_Search(t *testing.T) {
	plugin := newMockPlugin()
	userID1 := uuid.New().String()
	userID2 := uuid.New().String()
	groupID1 := uuid.New().String()
	plugin.users[userID1] = &User{ID: userID1, UserName: "alice", Active: Bool(true)}
	plugin.users[userID2] = &User{ID: userID2, UserName: "bob", Active: Bool(false)}
	plugin.groups[groupID1] = &Group{ID: groupID1, DisplayName: "Admins"}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	searchJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:SearchRequest"],
		"filter": "active eq true",
		"startIndex": 1,
		"count": 10
	}`

	req := httptest.NewRequest("POST", "/test/.search", bytes.NewBufferString(searchJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ListResponse[any]
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should find only active users (alice)
	if resp.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want 1", resp.TotalResults)
	}
}

func TestServer_SearchWithSorting(t *testing.T) {
	plugin := newMockPlugin()
	userID1 := uuid.New().String()
	userID2 := uuid.New().String()
	userID3 := uuid.New().String()
	plugin.users[userID1] = &User{ID: userID1, UserName: "charlie", Active: Bool(true)}
	plugin.users[userID2] = &User{ID: userID2, UserName: "alice", Active: Bool(true)}
	plugin.users[userID3] = &User{ID: userID3, UserName: "bob", Active: Bool(true)}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	searchJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:SearchRequest"],
		"sortBy": "userName",
		"sortOrder": "ascending",
		"startIndex": 1,
		"count": 10
	}`

	req := httptest.NewRequest("POST", "/test/.search", bytes.NewBufferString(searchJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ListResponse[any]
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Resources) < 1 {
		t.Fatal("No resources returned")
	}

	// First result should be alice (alphabetically first)
	firstResource, _ := json.Marshal(resp.Resources[0])
	var firstUser User
	json.Unmarshal(firstResource, &firstUser)

	if firstUser.UserName != "alice" {
		t.Errorf("First user = %v, want alice", firstUser.UserName)
	}
}

func TestServer_SearchWithPagination(t *testing.T) {
	plugin := newMockPlugin()
	for i := 1; i <= 10; i++ {
		id := uuid.New().String()
		plugin.users[id] = &User{
			ID:       id,
			UserName: fmt.Sprintf("user%d", i),
			Active:   Bool(true),
		}
	}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	searchJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:SearchRequest"],
		"startIndex": 1,
		"count": 5
	}`

	req := httptest.NewRequest("POST", "/test/.search", bytes.NewBufferString(searchJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ListResponse[any]
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Search returns up to count items
	if resp.ItemsPerPage > 5 {
		t.Errorf("ItemsPerPage = %d, should be <= 5", resp.ItemsPerPage)
	}

	// Should have results
	if len(resp.Resources) == 0 {
		t.Error("Expected some resources")
	}
}

func TestServer_SearchInvalidSchema(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	searchJSON := `{
		"schemas": ["invalid"],
		"filter": "active eq true"
	}`

	req := httptest.NewRequest("POST", "/test/.search", bytes.NewBufferString(searchJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestServer_SearchMethodNotAllowed(t *testing.T) {
	plugin := newMockPlugin()
	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	req := httptest.NewRequest("GET", "/test/.search", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServer_SearchWithAttributes(t *testing.T) {
	plugin := newMockPlugin()
	userID := uuid.New().String()
	plugin.users[userID] = &User{
		ID:          userID,
		UserName:    "alice",
		DisplayName: "Alice Smith",
		Active:      Bool(true),
	}

	pm := &mockPluginManager{plugin: plugin}
	server := NewServer("http://localhost:8880", pm)

	searchJSON := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:SearchRequest"],
		"attributes": ["userName", "active"],
		"startIndex": 1,
		"count": 10
	}`

	req := httptest.NewRequest("POST", "/test/.search", bytes.NewBufferString(searchJSON))
	req.Header.Set("Content-Type", "application/scim+json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ListResponse[any]
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Resources) == 0 {
		t.Fatal("No resources returned")
	}

	// Check that only requested attributes are present (plus core attributes)
	resourceJSON, _ := json.Marshal(resp.Resources[0])
	var resourceMap map[string]any
	json.Unmarshal(resourceJSON, &resourceMap)

	// Core attributes should always be present
	if _, ok := resourceMap["id"]; !ok {
		t.Error("id should be present")
	}

	// userName should be present (requested)
	if _, ok := resourceMap["userName"]; !ok {
		t.Error("userName should be present")
	}

	// displayName should NOT be present (not requested and not core)
	if _, ok := resourceMap["displayName"]; ok {
		t.Error("displayName should not be present")
	}
}
