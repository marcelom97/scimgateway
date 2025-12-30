package memory

import (
	"context"
	"testing"

	"github.com/marcelom97/scimgateway/plugin"
	"github.com/marcelom97/scimgateway/scim"
)

func TestAttributeSelection(t *testing.T) {
	// Create plugin and adapter
	memPlugin := New("test")
	adapter := plugin.NewAdapter(memPlugin)

	// Create a test user
	user := &scim.User{
		UserName: "john.doe",
		Active:   scim.Bool(true),
		Schemas:  []string{scim.SchemaUser},
	}
	user.Name = &scim.Name{
		GivenName:  "John",
		FamilyName: "Doe",
	}

	// Create the user
	ctx := context.Background()
	_, err := memPlugin.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test 1: Get user with attributes=userName
	params := scim.QueryParams{
		Attributes: []string{"userName"},
	}

	response, err := adapter.GetUsers(ctx, params)
	if err != nil {
		t.Fatalf("Failed to get users: %v", err)
	}

	if len(response.Resources) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(response.Resources))
	}

	result := response.Resources[0]

	// Should have userName
	if result.UserName != "john.doe" {
		t.Errorf("Expected userName to be 'john.doe', got '%s'", result.UserName)
	}

	// Should have id (core attribute)
	if result.ID == "" {
		t.Error("Expected id to be present (core attribute)")
	}

	// Should have schemas (core attribute)
	if len(result.Schemas) == 0 {
		t.Error("Expected schemas to be present (core attribute)")
	}

	// Should NOT have active (not requested)
	// Note: In Go, bool default is false, so we need to check if the field was actually filtered
	// by checking if name is nil (which should be filtered out)
	if result.Name != nil {
		t.Error("Expected name to be filtered out (not requested)")
	}

	t.Logf("Result with attributes=userName: ID=%s, UserName=%s, Active=%v, Name=%v",
		result.ID, result.UserName, result.Active, result.Name)

	// Test 2: Get user with attributes=userName,name
	params2 := scim.QueryParams{
		Attributes: []string{"userName", "name"},
	}

	response2, err := adapter.GetUsers(ctx, params2)
	if err != nil {
		t.Fatalf("Failed to get users: %v", err)
	}

	result2 := response2.Resources[0]

	// Should have userName
	if result2.UserName != "john.doe" {
		t.Errorf("Expected userName to be 'john.doe', got '%s'", result2.UserName)
	}

	// Should have name
	if result2.Name == nil {
		t.Error("Expected name to be present (requested)")
	} else {
		if result2.Name.GivenName != "John" {
			t.Errorf("Expected givenName to be 'John', got '%s'", result2.Name.GivenName)
		}
	}

	t.Logf("Result with attributes=userName,name: ID=%s, UserName=%s, Active=%v, Name=%+v",
		result2.ID, result2.UserName, result2.Active, result2.Name)

	// Test 3: Get user without attributes (should return all)
	params3 := scim.QueryParams{}

	response3, err := adapter.GetUsers(ctx, params3)
	if err != nil {
		t.Fatalf("Failed to get users: %v", err)
	}

	result3 := response3.Resources[0]

	if result3.UserName != "john.doe" {
		t.Errorf("Expected userName to be 'john.doe', got '%s'", result3.UserName)
	}

	if result3.Active == nil || !*result3.Active {
		t.Errorf("Expected active to be true, got %v", result3.Active)
	}

	if result3.Name == nil {
		t.Error("Expected name to be present (all attributes)")
	}

	t.Logf("Result with no attributes filter: ID=%s, UserName=%s, Active=%v, Name=%+v",
		result3.ID, result3.UserName, result3.Active, result3.Name)
}

func TestGetUserWithAttributes(t *testing.T) {
	// Create plugin and adapter
	memPlugin := New("test")
	adapter := plugin.NewAdapter(memPlugin)

	// Create a test user
	user := &scim.User{
		UserName: "jane.doe",
		Active:   scim.Bool(true),
		Schemas:  []string{scim.SchemaUser},
	}

	ctx := context.Background()
	created, err := memPlugin.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get user with specific attributes
	result, err := adapter.GetUser(ctx, created.ID, []string{"userName"})
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	// Should have userName
	if result.UserName != "jane.doe" {
		t.Errorf("Expected userName to be 'jane.doe', got '%s'", result.UserName)
	}

	// Should have id (core attribute)
	if result.ID == "" {
		t.Error("Expected id to be present (core attribute)")
	}

	t.Logf("GetUser with attributes=userName: ID=%s, UserName=%s, Active=%v",
		result.ID, result.UserName, result.Active)
}
