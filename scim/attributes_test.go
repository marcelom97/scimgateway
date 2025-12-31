package scim

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAttributeSelector(t *testing.T) {
	user := &User{
		ID:          "123",
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Emails: []Email{
			{Value: "john@example.com", Primary: true, Type: "work"},
		},
		Meta: &Meta{
			ResourceType: "User",
		},
		Schemas: []string{SchemaUser},
	}

	tests := []struct {
		name       string
		attributes []string
		excluded   []string
		wantFields []string
	}{
		{
			name:       "select specific",
			attributes: []string{"userName", "active"},
			excluded:   nil,
			wantFields: []string{"id", "schemas", "meta", "userName", "active"},
		},
		{
			name:       "exclude fields",
			attributes: nil,
			excluded:   []string{"emails", "displayName"},
			wantFields: []string{"id", "schemas", "meta", "userName", "active"},
		},
		{
			name:       "select one",
			attributes: []string{"userName"},
			excluded:   nil,
			wantFields: []string{"id", "schemas", "meta", "userName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAttributeSelector(tt.attributes, tt.excluded)
			result, err := selector.FilterResource(user)
			if err != nil {
				t.Fatalf("FilterResource() error = %v", err)
			}

			data, _ := json.Marshal(result)
			var got map[string]any
			json.Unmarshal(data, &got)

			// Check that expected fields are present
			for _, field := range tt.wantFields {
				if _, exists := got[field]; !exists {
					t.Errorf("Expected field %s not found", field)
				}
			}

			// Check that excluded fields are not present (except core)
			if tt.excluded != nil {
				for _, field := range tt.excluded {
					if field != "id" && field != "schemas" && field != "meta" {
						if _, exists := got[field]; exists {
							t.Errorf("Excluded field %s should not be present", field)
						}
					}
				}
			}
		})
	}
}

func TestAttributeSelectorSubAttributes(t *testing.T) {
	user := &User{
		ID:          "123",
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Emails: []Email{
			{Value: "john@example.com", Type: "work", Primary: true},
			{Value: "john.personal@example.com", Type: "personal", Primary: false},
		},
		Meta: &Meta{
			ResourceType: "User",
		},
		Schemas: []string{SchemaUser},
	}

	tests := []struct {
		name             string
		attributes       []string
		wantFields       []string
		checkEmailsFunc  func(t *testing.T, emails any)
		checkDisplayName bool
	}{
		{
			name:             "select emails.type sub-attribute only",
			attributes:       []string{"emails.type"},
			wantFields:       []string{"id", "schemas", "meta", "emails"},
			checkDisplayName: false,
			checkEmailsFunc: func(t *testing.T, emails any) {
				emailsSlice, ok := emails.([]any)
				if !ok {
					t.Fatalf("emails is not a slice, got %T", emails)
				}
				if len(emailsSlice) != 2 {
					t.Errorf("Expected 2 emails, got %d", len(emailsSlice))
				}
				for i, email := range emailsSlice {
					emailMap, ok := email.(map[string]any)
					if !ok {
						t.Fatalf("email[%d] is not a map, got %T", i, email)
					}
					// Should only have "type" field
					if len(emailMap) != 1 {
						t.Errorf("Expected email[%d] to have 1 field, got %d: %v", i, len(emailMap), emailMap)
					}
					if _, hasType := emailMap["type"]; !hasType {
						t.Errorf("Expected email[%d] to have 'type' field", i)
					}
					if _, hasValue := emailMap["value"]; hasValue {
						t.Errorf("email[%d] should not have 'value' field", i)
					}
					if _, hasPrimary := emailMap["primary"]; hasPrimary {
						t.Errorf("email[%d] should not have 'primary' field", i)
					}
				}
			},
		},
		{
			name:             "select emails.value and emails.primary",
			attributes:       []string{"emails.value", "emails.primary"},
			wantFields:       []string{"id", "schemas", "meta", "emails"},
			checkDisplayName: false,
			checkEmailsFunc: func(t *testing.T, emails any) {
				emailsSlice, ok := emails.([]any)
				if !ok {
					t.Fatalf("emails is not a slice, got %T", emails)
				}
				for i, email := range emailsSlice {
					emailMap, ok := email.(map[string]any)
					if !ok {
						t.Fatalf("email[%d] is not a map, got %T", i, email)
					}
					// Should have "value" field, and "primary" if it's not false (omitempty)
					if _, hasValue := emailMap["value"]; !hasValue {
						t.Errorf("Expected email[%d] to have 'value' field", i)
					}
					// Note: primary may not be present if false (omitempty tag)
					if _, hasType := emailMap["type"]; hasType {
						t.Errorf("email[%d] should not have 'type' field", i)
					}
				}
			},
		},
		{
			name:             "select full emails and userName",
			attributes:       []string{"emails", "userName"},
			wantFields:       []string{"id", "schemas", "meta", "emails", "userName"},
			checkDisplayName: false,
			checkEmailsFunc: func(t *testing.T, emails any) {
				emailsSlice, ok := emails.([]any)
				if !ok {
					t.Fatalf("emails is not a slice, got %T", emails)
				}
				if len(emailsSlice) != 2 {
					t.Errorf("Expected 2 emails, got %d", len(emailsSlice))
				}
				// First email should have primary=true
				email0Map, ok := emailsSlice[0].(map[string]any)
				if !ok {
					t.Fatalf("email[0] is not a map, got %T", emailsSlice[0])
				}
				if _, hasValue := email0Map["value"]; !hasValue {
					t.Error("Expected email[0] to have 'value' field")
				}
				if _, hasType := email0Map["type"]; !hasType {
					t.Error("Expected email[0] to have 'type' field")
				}
				if primary, hasPrimary := email0Map["primary"]; !hasPrimary || primary != true {
					t.Error("Expected email[0] to have 'primary' field set to true")
				}
			},
		},
		{
			name:             "mix sub-attribute and regular attribute",
			attributes:       []string{"emails.type", "userName"},
			wantFields:       []string{"id", "schemas", "meta", "emails", "userName"},
			checkDisplayName: false,
			checkEmailsFunc: func(t *testing.T, emails any) {
				emailsSlice, ok := emails.([]any)
				if !ok {
					t.Fatalf("emails is not a slice, got %T", emails)
				}
				for i, email := range emailsSlice {
					emailMap, ok := email.(map[string]any)
					if !ok {
						t.Fatalf("email[%d] is not a map, got %T", i, email)
					}
					// Should only have "type" field
					if len(emailMap) != 1 {
						t.Errorf("Expected email[%d] to have 1 field, got %d: %v", i, len(emailMap), emailMap)
					}
					if _, hasType := emailMap["type"]; !hasType {
						t.Errorf("Expected email[%d] to have 'type' field", i)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAttributeSelector(tt.attributes, nil)
			result, err := selector.FilterResource(user)
			if err != nil {
				t.Fatalf("FilterResource() error = %v", err)
			}

			data, _ := json.Marshal(result)
			var got map[string]any
			json.Unmarshal(data, &got)

			// Check that expected fields are present
			for _, field := range tt.wantFields {
				if _, exists := got[field]; !exists {
					t.Errorf("Expected field %s not found", field)
				}
			}

			// Check displayName is not included (unless specified)
			if !tt.checkDisplayName {
				if _, exists := got["displayName"]; exists {
					t.Errorf("Field displayName should not be present")
				}
			}

			// Check emails structure
			if tt.checkEmailsFunc != nil {
				if emails, exists := got["emails"]; exists {
					tt.checkEmailsFunc(t, emails)
				} else {
					t.Errorf("Expected 'emails' field to be present")
				}
			}
		})
	}
}

func TestAttributeSelectorDeepNesting(t *testing.T) {
	// Test with a resource that has deeply nested attributes
	resource := map[string]any{
		"id":      "123",
		"schemas": []string{SchemaUser},
		"meta": map[string]any{
			"resourceType": "User",
		},
		"name": map[string]any{
			"formatted":  "Mr. John Doe",
			"familyName": "Doe",
			"givenName":  "John",
			"prefix":     "Mr.",
		},
		"addresses": []any{
			map[string]any{
				"type":          "work",
				"streetAddress": "100 Universal City Plaza",
				"locality":      "Hollywood",
				"region":        "CA",
				"postalCode":    "91608",
				"country":       "USA",
				"formatted":     "100 Universal City Plaza\nHollywood, CA 91608 USA",
				"primary":       true,
			},
			map[string]any{
				"type":          "home",
				"streetAddress": "456 Home St",
				"locality":      "Los Angeles",
				"region":        "CA",
				"postalCode":    "90001",
				"country":       "USA",
				"primary":       false,
			},
		},
	}

	tests := []struct {
		name       string
		attributes []string
		checkFunc  func(t *testing.T, result map[string]any)
	}{
		{
			name:       "single nested attribute - name.formatted",
			attributes: []string{"name.formatted"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// Should have core attributes + name
				if _, exists := result["id"]; !exists {
					t.Error("Expected 'id' field")
				}
				if _, exists := result["schemas"]; !exists {
					t.Error("Expected 'schemas' field")
				}
				if _, exists := result["meta"]; !exists {
					t.Error("Expected 'meta' field")
				}

				name, exists := result["name"]
				if !exists {
					t.Fatal("Expected 'name' field")
				}

				nameMap, ok := name.(map[string]any)
				if !ok {
					t.Fatalf("name is not a map, got %T", name)
				}

				// Should only have "formatted"
				if len(nameMap) != 1 {
					t.Errorf("Expected name to have 1 field, got %d: %v", len(nameMap), nameMap)
				}
				if _, exists := nameMap["formatted"]; !exists {
					t.Error("Expected name.formatted field")
				}
				if _, exists := nameMap["familyName"]; exists {
					t.Error("name.familyName should not be present")
				}
			},
		},
		{
			name:       "multiple nested attributes from same parent",
			attributes: []string{"name.formatted", "name.familyName"},
			checkFunc: func(t *testing.T, result map[string]any) {
				name := result["name"].(map[string]any)

				// Should have "formatted" and "familyName"
				if len(name) != 2 {
					t.Errorf("Expected name to have 2 fields, got %d: %v", len(name), name)
				}
				if _, exists := name["formatted"]; !exists {
					t.Error("Expected name.formatted field")
				}
				if _, exists := name["familyName"]; !exists {
					t.Error("Expected name.familyName field")
				}
				if _, exists := name["givenName"]; exists {
					t.Error("name.givenName should not be present")
				}
			},
		},
		{
			name:       "nested attribute in multi-valued attribute",
			attributes: []string{"addresses.type", "addresses.postalCode"},
			checkFunc: func(t *testing.T, result map[string]any) {
				addresses, exists := result["addresses"]
				if !exists {
					t.Fatal("Expected 'addresses' field")
				}

				addressesSlice, ok := addresses.([]any)
				if !ok {
					t.Fatalf("addresses is not a slice, got %T", addresses)
				}

				if len(addressesSlice) != 2 {
					t.Errorf("Expected 2 addresses, got %d", len(addressesSlice))
				}

				for i, addr := range addressesSlice {
					addrMap, ok := addr.(map[string]any)
					if !ok {
						t.Fatalf("address[%d] is not a map, got %T", i, addr)
					}

					// Should only have "type" and "postalCode"
					if len(addrMap) != 2 {
						t.Errorf("Expected address[%d] to have 2 fields, got %d: %v", i, len(addrMap), addrMap)
					}
					if _, exists := addrMap["type"]; !exists {
						t.Errorf("Expected address[%d].type field", i)
					}
					if _, exists := addrMap["postalCode"]; !exists {
						t.Errorf("Expected address[%d].postalCode field", i)
					}
					if _, exists := addrMap["streetAddress"]; exists {
						t.Errorf("address[%d].streetAddress should not be present", i)
					}
				}
			},
		},
		{
			name:       "mix of nested and top-level attributes",
			attributes: []string{"name.formatted", "addresses.type"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// Check name
				name := result["name"].(map[string]any)
				if len(name) != 1 {
					t.Errorf("Expected name to have 1 field, got %d", len(name))
				}
				if _, exists := name["formatted"]; !exists {
					t.Error("Expected name.formatted field")
				}

				// Check addresses
				addressesSlice := result["addresses"].([]any)
				for i, addr := range addressesSlice {
					addrMap := addr.(map[string]any)
					if len(addrMap) != 1 {
						t.Errorf("Expected address[%d] to have 1 field, got %d", i, len(addrMap))
					}
					if _, exists := addrMap["type"]; !exists {
						t.Errorf("Expected address[%d].type field", i)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAttributeSelector(tt.attributes, nil)
			result, err := selector.FilterResource(resource)
			if err != nil {
				t.Fatalf("FilterResource() error = %v", err)
			}

			data, _ := json.Marshal(result)
			var got map[string]any
			json.Unmarshal(data, &got)

			tt.checkFunc(t, got)
		})
	}
}

func TestSortResources(t *testing.T) {
	users := []any{
		&User{UserName: "charlie", DisplayName: "Charlie"},
		&User{UserName: "alice", DisplayName: "Alice"},
		&User{UserName: "bob", DisplayName: "Bob"},
	}

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantFirst string
	}{
		{"ascending", "userName", "ascending", "alice"},
		{"descending", "userName", "descending", "charlie"},
		{"no sort", "", "", "charlie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortResources(users, tt.sortBy, tt.sortOrder)
			if len(sorted) == 0 {
				t.Fatal("No results")
			}

			first := sorted[0].(*User)
			if first.UserName != tt.wantFirst {
				t.Errorf("First user = %v, want %v", first.UserName, tt.wantFirst)
			}
		})
	}
}

func TestSortResourcesNestedFields(t *testing.T) {
	// Create users with different creation timestamps
	time1, _ := time.Parse(time.RFC3339, "2024-01-15T10:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2024-01-10T10:00:00Z")
	time3, _ := time.Parse(time.RFC3339, "2024-01-20T10:00:00Z")

	user1 := &User{
		ID:       "1",
		UserName: "user1",
		Meta: &Meta{
			Created:      &time1,
			LastModified: &time1,
		},
	}
	user2 := &User{
		ID:       "2",
		UserName: "user2",
		Meta: &Meta{
			Created:      &time2,
			LastModified: &time2,
		},
	}
	user3 := &User{
		ID:       "3",
		UserName: "user3",
		Meta: &Meta{
			Created:      &time3,
			LastModified: &time3,
		},
	}

	users := []any{user1, user2, user3}

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantFirst string // Expected first user ID
		wantLast  string // Expected last user ID
	}{
		{
			name:      "sort by meta.created ascending",
			sortBy:    "meta.created",
			sortOrder: "ascending",
			wantFirst: "2", // user2 has earliest date
			wantLast:  "3", // user3 has latest date
		},
		{
			name:      "sort by meta.created descending",
			sortBy:    "meta.created",
			sortOrder: "descending",
			wantFirst: "3", // user3 has latest date
			wantLast:  "2", // user2 has earliest date
		},
		{
			name:      "sort by meta.lastModified ascending",
			sortBy:    "meta.lastModified",
			sortOrder: "ascending",
			wantFirst: "2",
			wantLast:  "3",
		},
		{
			name:      "sort by meta.lastModified descending",
			sortBy:    "meta.lastModified",
			sortOrder: "descending",
			wantFirst: "3",
			wantLast:  "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortResources(users, tt.sortBy, tt.sortOrder)
			if len(sorted) == 0 {
				t.Fatal("No results")
			}

			first := sorted[0].(*User)
			last := sorted[len(sorted)-1].(*User)

			if first.ID != tt.wantFirst {
				t.Errorf("First user ID = %v, want %v", first.ID, tt.wantFirst)
			}
			if last.ID != tt.wantLast {
				t.Errorf("Last user ID = %v, want %v", last.ID, tt.wantLast)
			}
		})
	}
}

// Test sorting by nested name fields (name.familyName, name.givenName)
func TestSortResourcesByNameFields(t *testing.T) {
	users := []*User{
		{
			ID:       "1",
			UserName: "john.smith",
			Name: &Name{
				GivenName:  "John",
				FamilyName: "Smith",
			},
		},
		{
			ID:       "2",
			UserName: "alice.jones",
			Name: &Name{
				GivenName:  "Alice",
				FamilyName: "Jones",
			},
		},
		{
			ID:       "3",
			UserName: "bob.adams",
			Name: &Name{
				GivenName:  "Bob",
				FamilyName: "Adams",
			},
		},
	}

	tests := []struct {
		name          string
		sortBy        string
		sortOrder     string
		expectedOrder []string // Expected order of values from the sorted field
		getValueFunc  func(*User) string
	}{
		{
			name:          "ascending by name.familyName",
			sortBy:        "name.familyName",
			sortOrder:     "ascending",
			expectedOrder: []string{"Adams", "Jones", "Smith"},
			getValueFunc:  func(u *User) string { return u.Name.FamilyName },
		},
		{
			name:          "descending by name.familyName",
			sortBy:        "name.familyName",
			sortOrder:     "descending",
			expectedOrder: []string{"Smith", "Jones", "Adams"},
			getValueFunc:  func(u *User) string { return u.Name.FamilyName },
		},
		{
			name:          "ascending by name.givenName",
			sortBy:        "name.givenName",
			sortOrder:     "ascending",
			expectedOrder: []string{"Alice", "Bob", "John"},
			getValueFunc:  func(u *User) string { return u.Name.GivenName },
		},
		{
			name:          "descending by name.givenName",
			sortBy:        "name.givenName",
			sortOrder:     "descending",
			expectedOrder: []string{"John", "Bob", "Alice"},
			getValueFunc:  func(u *User) string { return u.Name.GivenName },
		},
		{
			name:          "ascending by userName",
			sortBy:        "userName",
			sortOrder:     "ascending",
			expectedOrder: []string{"alice.jones", "bob.adams", "john.smith"},
			getValueFunc:  func(u *User) string { return u.UserName },
		},
		{
			name:          "descending by userName",
			sortBy:        "userName",
			sortOrder:     "descending",
			expectedOrder: []string{"john.smith", "bob.adams", "alice.jones"},
			getValueFunc:  func(u *User) string { return u.UserName },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortResources(users, tt.sortBy, tt.sortOrder)

			if len(sorted) != len(tt.expectedOrder) {
				t.Fatalf("Expected %d results, got %d", len(tt.expectedOrder), len(sorted))
			}

			for i, expected := range tt.expectedOrder {
				actual := tt.getValueFunc(sorted[i])
				if actual != expected {
					t.Errorf("Position %d: expected %s, got %s", i, expected, actual)
				}
			}
		})
	}
}

// Test excludedAttributes with nested paths
func TestAttributeSelectorExcludedNestedPaths(t *testing.T) {
	user := &User{
		ID:          "123",
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
			Formatted:  "John Doe",
		},
		Emails: []Email{
			{Value: "john@example.com", Type: "work", Primary: true},
		},
		Meta: &Meta{
			ResourceType: "User",
		},
		Schemas: []string{SchemaUser},
	}

	tests := []struct {
		name      string
		excluded  []string
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name:     "exclude single nested attribute",
			excluded: []string{"name.familyName"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// Name should still exist
				name, exists := result["name"]
				if !exists {
					t.Fatal("Expected 'name' field to exist")
				}

				nameMap, ok := name.(map[string]any)
				if !ok {
					t.Fatalf("name is not a map, got %T", name)
				}

				// FamilyName should be excluded
				if _, hasFamilyName := nameMap["familyName"]; hasFamilyName {
					t.Error("familyName should be excluded")
				}

				// GivenName should still be present
				if _, hasGivenName := nameMap["givenName"]; !hasGivenName {
					t.Error("givenName should be present")
				}

				// Formatted should still be present
				if _, hasFormatted := nameMap["formatted"]; !hasFormatted {
					t.Error("formatted should be present")
				}
			},
		},
		{
			name:     "exclude multiple nested attributes from same parent",
			excluded: []string{"name.familyName", "name.formatted"},
			checkFunc: func(t *testing.T, result map[string]any) {
				nameMap, ok := result["name"].(map[string]any)
				if !ok {
					t.Fatal("name should be a map")
				}

				// FamilyName should be excluded
				if _, hasFamilyName := nameMap["familyName"]; hasFamilyName {
					t.Error("familyName should be excluded")
				}

				// Formatted should be excluded
				if _, hasFormatted := nameMap["formatted"]; hasFormatted {
					t.Error("formatted should be excluded")
				}

				// GivenName should still be present
				if _, hasGivenName := nameMap["givenName"]; !hasGivenName {
					t.Error("givenName should be present")
				}
			},
		},
		{
			name:     "exclude nested attribute from multi-valued field",
			excluded: []string{"emails.type"},
			checkFunc: func(t *testing.T, result map[string]any) {
				emails, exists := result["emails"]
				if !exists {
					t.Fatal("Expected 'emails' field to exist")
				}

				emailsSlice, ok := emails.([]any)
				if !ok {
					t.Fatalf("emails is not a slice, got %T", emails)
				}

				if len(emailsSlice) == 0 {
					t.Fatal("Expected at least one email")
				}

				emailMap, ok := emailsSlice[0].(map[string]any)
				if !ok {
					t.Fatal("email should be a map")
				}

				// Type should be excluded
				if _, hasType := emailMap["type"]; hasType {
					t.Error("type should be excluded")
				}

				// Value should still be present
				if _, hasValue := emailMap["value"]; !hasValue {
					t.Error("value should be present")
				}
			},
		},
		{
			name:     "exclude top-level attribute",
			excluded: []string{"displayName"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// displayName should be excluded
				if _, hasDisplayName := result["displayName"]; hasDisplayName {
					t.Error("displayName should be excluded")
				}

				// userName should still be present
				if _, hasUserName := result["userName"]; !hasUserName {
					t.Error("userName should be present")
				}

				// name should still be present
				if _, hasName := result["name"]; !hasName {
					t.Error("name should be present")
				}
			},
		},
		{
			name:     "exclude multiple top-level and nested attributes",
			excluded: []string{"displayName", "name.formatted", "emails.type"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// displayName should be excluded
				if _, hasDisplayName := result["displayName"]; hasDisplayName {
					t.Error("displayName should be excluded")
				}

				// Check name exists and formatted is excluded
				nameMap, ok := result["name"].(map[string]any)
				if !ok {
					t.Fatal("name should be a map")
				}
				if _, hasFormatted := nameMap["formatted"]; hasFormatted {
					t.Error("name.formatted should be excluded")
				}
				if _, hasGivenName := nameMap["givenName"]; !hasGivenName {
					t.Error("name.givenName should be present")
				}

				// Check emails exists and type is excluded
				emailsSlice, ok := result["emails"].([]any)
				if !ok {
					t.Fatal("emails should be a slice")
				}
				if len(emailsSlice) > 0 {
					emailMap := emailsSlice[0].(map[string]any)
					if _, hasType := emailMap["type"]; hasType {
						t.Error("emails.type should be excluded")
					}
				}
			},
		},
		{
			name:     "exclude emails.value but keep emails.type",
			excluded: []string{"emails.value"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// Check emails exists and value is excluded
				emailsSlice, ok := result["emails"].([]any)
				if !ok {
					t.Fatal("emails should be a slice")
				}
				if len(emailsSlice) > 0 {
					emailMap := emailsSlice[0].(map[string]any)
					if _, hasValue := emailMap["value"]; hasValue {
						t.Error("emails.value should be excluded")
					}
					if _, hasType := emailMap["type"]; !hasType {
						t.Error("emails.type should be present")
					}
					if _, hasPrimary := emailMap["primary"]; !hasPrimary {
						t.Error("emails.primary should be present")
					}
				}
			},
		},
		{
			name:     "exclude multiple email sub-attributes",
			excluded: []string{"emails.value", "emails.primary"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// Check emails exists with only type
				emailsSlice, ok := result["emails"].([]any)
				if !ok {
					t.Fatal("emails should be a slice")
				}
				if len(emailsSlice) > 0 {
					emailMap := emailsSlice[0].(map[string]any)
					if _, hasValue := emailMap["value"]; hasValue {
						t.Error("emails.value should be excluded")
					}
					if _, hasPrimary := emailMap["primary"]; hasPrimary {
						t.Error("emails.primary should be excluded")
					}
					if _, hasType := emailMap["type"]; !hasType {
						t.Error("emails.type should be present")
					}
				}
			},
		},
		{
			name:     "exclude entire emails array",
			excluded: []string{"emails"},
			checkFunc: func(t *testing.T, result map[string]any) {
				// emails should be completely excluded
				if _, hasEmails := result["emails"]; hasEmails {
					t.Error("emails should be excluded")
				}

				// Other fields should be present
				if _, hasUserName := result["userName"]; !hasUserName {
					t.Error("userName should be present")
				}
				if _, hasName := result["name"]; !hasName {
					t.Error("name should be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAttributeSelector(nil, tt.excluded)
			result, err := selector.FilterResource(user)
			if err != nil {
				t.Fatalf("FilterResource() error = %v", err)
			}

			data, _ := json.Marshal(result)
			var got map[string]any
			json.Unmarshal(data, &got)

			tt.checkFunc(t, got)
		})
	}
}

func TestApplyPagination(t *testing.T) {
	resources := []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tests := []struct {
		name       string
		startIndex int
		count      int
		wantLen    int
		wantStart  int
	}{
		{"first page", 1, 5, 5, 1},
		{"second page", 6, 5, 5, 6},
		{"partial page", 8, 5, 3, 8},
		{"beyond range", 15, 5, 0, 15},
		{"zero index", 0, 5, 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paged, startIdx, itemsPerPage := ApplyPagination(resources, tt.startIndex, tt.count)

			if len(paged) != tt.wantLen {
				t.Errorf("len(paged) = %d, want %d", len(paged), tt.wantLen)
			}

			if startIdx != tt.wantStart {
				t.Errorf("startIndex = %d, want %d", startIdx, tt.wantStart)
			}

			if itemsPerPage != tt.wantLen {
				t.Errorf("itemsPerPage = %d, want %d", itemsPerPage, tt.wantLen)
			}
		})
	}
}

func TestFilterByFilter(t *testing.T) {
	resources := []any{
		&User{UserName: "john", Active: Bool(true)},
		&User{UserName: "jane", Active: Bool(false)},
		&User{UserName: "bob", Active: Bool(true)},
	}

	tests := []struct {
		name    string
		filter  string
		wantLen int
		wantErr bool
	}{
		{"active users", `active eq true`, 2, false},
		{"specific user", `userName eq "john"`, 1, false},
		{"no match", `userName eq "alice"`, 0, false},
		{"empty filter", "", 3, false},
		{"invalid filter", "userName", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := FilterByFilter(resources, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterByFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(filtered) != tt.wantLen {
				t.Errorf("len(filtered) = %d, want %d", len(filtered), tt.wantLen)
			}
		})
	}
}

func generateBenchmarkUsers(n int) []*User {
	users := make([]*User, n)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range n {
		createdTime := baseTime.Add(time.Duration(i) * time.Hour)
		users[i] = &User{
			ID:          string(rune(i)),
			UserName:    string(rune(n - i)),
			DisplayName: string(rune(i)),
			Active:      Bool(i%2 == 0),
			Meta: &Meta{
				ResourceType: "User",
				Created:      &createdTime,
				LastModified: &createdTime,
			},
			Schemas: []string{SchemaUser},
		}
	}
	return users
}

func BenchmarkSortResources_SmallDataset(b *testing.B) {
	users := generateBenchmarkUsers(10)
	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = u
	}

	for b.Loop() {
		_ = SortResources(resources, "userName", "ascending")
	}
}

func BenchmarkSortResources_MediumDataset(b *testing.B) {
	users := generateBenchmarkUsers(100)
	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = u
	}

	for b.Loop() {
		_ = SortResources(resources, "userName", "ascending")
	}
}

func BenchmarkSortResources_LargeDataset(b *testing.B) {
	users := generateBenchmarkUsers(1000)
	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = u
	}

	for b.Loop() {
		_ = SortResources(resources, "userName", "ascending")
	}
}

func BenchmarkSortResources_NestedPath(b *testing.B) {
	users := generateBenchmarkUsers(1000)
	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = u
	}

	for b.Loop() {
		_ = SortResources(resources, "meta.created", "ascending")
	}
}

func BenchmarkSortResources_SimplePath(b *testing.B) {
	users := generateBenchmarkUsers(1000)
	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = u
	}

	for b.Loop() {
		_ = SortResources(resources, "userName", "ascending")
	}
}

// TestCompareForSort_TimeValues is a regression test ensuring time.Time comparison works correctly.
func TestCompareForSort_TimeValues(t *testing.T) {
	time1 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		a        any
		b        any
		expected int
	}{
		{
			name:     "time.Time: earlier < later",
			a:        time1,
			b:        time3,
			expected: -1,
		},
		{
			name:     "time.Time: later > earlier",
			a:        time3,
			b:        time1,
			expected: 1,
		},
		{
			name:     "time.Time: equal times",
			a:        time2,
			b:        time2,
			expected: 0,
		},
		{
			name:     "*time.Time: earlier < later",
			a:        &time1,
			b:        &time3,
			expected: -1,
		},
		{
			name:     "*time.Time: later > earlier",
			a:        &time3,
			b:        &time1,
			expected: 1,
		},
		{
			name:     "*time.Time: equal times",
			a:        &time2,
			b:        &time2,
			expected: 0,
		},
		{
			name:     "mixed: time.Time vs *time.Time",
			a:        time1,
			b:        &time3,
			expected: -1,
		},
		{
			name:     "mixed: *time.Time vs time.Time",
			a:        &time3,
			b:        time1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareForSort(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareForSort(%v, %v) = %d, expected %d",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestSortResources_TemporalFieldsRegression ensures temporal sorting works correctly.
func TestSortResources_TemporalFieldsRegression(t *testing.T) {
	time1 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)

	user1 := &User{ID: "user1", Meta: &Meta{Created: &time1}}
	user2 := &User{ID: "user2", Meta: &Meta{Created: &time2}}
	user3 := &User{ID: "user3", Meta: &Meta{Created: &time3}}

	t.Run("ascending", func(t *testing.T) {
		users := []any{user2, user3, user1}
		sorted := SortResources(users, "meta.created", "ascending")

		if sorted[0].(*User).ID != "user1" {
			t.Errorf("First user should be user1 (earliest), got %s", sorted[0].(*User).ID)
		}
		if sorted[1].(*User).ID != "user2" {
			t.Errorf("Second user should be user2 (middle), got %s", sorted[1].(*User).ID)
		}
		if sorted[2].(*User).ID != "user3" {
			t.Errorf("Third user should be user3 (latest), got %s", sorted[2].(*User).ID)
		}
	})

	t.Run("descending", func(t *testing.T) {
		users := []any{user2, user1, user3}
		sorted := SortResources(users, "meta.created", "descending")

		if sorted[0].(*User).ID != "user3" {
			t.Errorf("First user should be user3 (latest), got %s", sorted[0].(*User).ID)
		}
		if sorted[1].(*User).ID != "user2" {
			t.Errorf("Second user should be user2 (middle), got %s", sorted[1].(*User).ID)
		}
		if sorted[2].(*User).ID != "user1" {
			t.Errorf("Third user should be user1 (earliest), got %s", sorted[2].(*User).ID)
		}
	})
}
