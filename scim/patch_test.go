package scim

import (
	"testing"
)

func TestPatchProcessor_Replace(t *testing.T) {
	user := &User{
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
	}

	tests := []struct {
		name      string
		patch     *PatchOp
		checkFunc func(*User) bool
		wantErr   bool
	}{
		{
			name: "replace active",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "active", Value: false},
				},
			},
			checkFunc: func(u *User) bool { return u.Active != nil && !*u.Active },
			wantErr:   false,
		},
		{
			name: "replace displayName",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "displayName", Value: "Jane Doe"},
				},
			},
			checkFunc: func(u *User) bool { return u.DisplayName == "Jane Doe" },
			wantErr:   false,
		},
		{
			name: "replace root",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Value: map[string]any{"active": false, "displayName": "Test"}},
				},
			},
			checkFunc: func(u *User) bool { return u.Active != nil && !*u.Active && u.DisplayName == "Test" },
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewPatchProcessor()
			err := processor.ApplyPatch(user, tt.patch)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.checkFunc(user) {
				t.Errorf("Patch did not apply correctly")
			}
		})
	}
}

func TestPatchProcessor_Add(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Emails:   []Email{},
	}

	patch := &PatchOp{
		Schemas: []string{SchemaPatchOp},
		Operations: []PatchOperation{
			{
				Op:   "add",
				Path: "emails",
				Value: []any{
					map[string]any{
						"value":   "john@example.com",
						"type":    "work",
						"primary": true,
					},
				},
			},
		},
	}

	processor := NewPatchProcessor()
	err := processor.ApplyPatch(user, patch)

	if err != nil {
		t.Fatalf("ApplyPatch() error = %v", err)
	}

	if len(user.Emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(user.Emails))
	}

	if user.Emails[0].Value != "john@example.com" {
		t.Errorf("Email value = %v, want john@example.com", user.Emails[0].Value)
	}
}

func TestPatchProcessor_Remove(t *testing.T) {
	user := &User{
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
	}

	patch := &PatchOp{
		Schemas: []string{SchemaPatchOp},
		Operations: []PatchOperation{
			{Op: "remove", Path: "displayName"},
		},
	}

	processor := NewPatchProcessor()
	err := processor.ApplyPatch(user, patch)

	if err != nil {
		t.Fatalf("ApplyPatch() error = %v", err)
	}

	if user.DisplayName != "" {
		t.Errorf("DisplayName should be empty, got %v", user.DisplayName)
	}
}

func TestPatchProcessor_ComplexPath(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Emails: []Email{
			{Value: "john@work.com", Type: "work", Primary: true},
			{Value: "john@home.com", Type: "home"},
		},
	}

	patch := &PatchOp{
		Schemas: []string{SchemaPatchOp},
		Operations: []PatchOperation{
			{Op: "remove", Path: "emails[type eq \"work\"]"},
		},
	}

	processor := NewPatchProcessor()
	err := processor.ApplyPatch(user, patch)

	if err != nil {
		t.Fatalf("ApplyPatch() error = %v", err)
	}

	if len(user.Emails) != 1 {
		t.Errorf("Expected 1 email after removal, got %d", len(user.Emails))
	}

	if len(user.Emails) > 0 && user.Emails[0].Type == "work" {
		t.Errorf("Work email should be removed")
	}
}

func TestPatchProcessor_ReplaceFilteredArraySubAttribute(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Emails: []Email{
			{Value: "john@work.com", Type: "work", Primary: true},
			{Value: "john@home.com", Type: "home"},
		},
		PhoneNumbers: []PhoneNumber{
			{Value: "555-1234", Type: "work", Primary: true},
			{Value: "555-5678", Type: "mobile"},
			{Value: "555-9999", Type: "fax"},
		},
		Addresses: []Address{
			{Formatted: "123 Main St", Type: "work", Primary: true},
			{Formatted: "456 Home St", Type: "home"},
		},
	}

	tests := []struct {
		name      string
		patch     *PatchOp
		checkFunc func(*User) bool
		wantErr   bool
	}{
		{
			name: "replace email value in filtered array",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[type eq \"work\"].value", Value: "newemail@work.com"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, email := range u.Emails {
					if email.Type == "work" {
						return email.Value == "newemail@work.com"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "replace email primary flag in filtered array",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[type eq \"work\"].primary", Value: false},
				},
			},
			checkFunc: func(u *User) bool {
				for _, email := range u.Emails {
					if email.Type == "work" {
						return email.Primary == false
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "replace phone number in filtered array",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "phoneNumbers[type eq \"mobile\"].value", Value: "999-9999"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, phone := range u.PhoneNumbers {
					if phone.Type == "mobile" {
						return phone.Value == "999-9999"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "replace address formatted in filtered array",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "addresses[type eq \"work\"].formatted", Value: "789 New St"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, addr := range u.Addresses {
					if addr.Type == "work" {
						return addr.Formatted == "789 New St"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "multiple replace operations with filtered paths",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[type eq \"work\"].value", Value: "updated@work.com"},
					{Op: "replace", Path: "phoneNumbers[type eq \"work\"].value", Value: "111-2222"},
					{Op: "replace", Path: "addresses[type eq \"home\"].formatted", Value: "999 Updated St"},
				},
			},
			checkFunc: func(u *User) bool {
				emailOk := false
				phoneOk := false
				addrOk := false
				for _, email := range u.Emails {
					if email.Type == "work" && email.Value == "updated@work.com" {
						emailOk = true
					}
				}
				for _, phone := range u.PhoneNumbers {
					if phone.Type == "work" && phone.Value == "111-2222" {
						phoneOk = true
					}
				}
				for _, addr := range u.Addresses {
					if addr.Type == "home" && addr.Formatted == "999 Updated St" {
						addrOk = true
					}
				}
				return emailOk && phoneOk && addrOk
			},
			wantErr: false,
		},
		{
			name: "replace non-existent filter match should fail",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[type eq \"business\"].value", Value: "test@business.com"},
				},
			},
			checkFunc: func(u *User) bool { return true },
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh copy for each test
			testUser := &User{
				UserName:     user.UserName,
				Emails:       make([]Email, len(user.Emails)),
				PhoneNumbers: make([]PhoneNumber, len(user.PhoneNumbers)),
				Addresses:    make([]Address, len(user.Addresses)),
			}
			copy(testUser.Emails, user.Emails)
			copy(testUser.PhoneNumbers, user.PhoneNumbers)
			copy(testUser.Addresses, user.Addresses)

			processor := NewPatchProcessor()
			err := processor.ApplyPatch(testUser, tt.patch)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.checkFunc(testUser) {
				t.Errorf("Patch did not apply correctly")
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name         string
		pathStr      string
		wantSegments int
		wantAttr     string
	}{
		{"simple", "userName", 1, "userName"},
		{"nested", "name.givenName", 2, "name"},
		{"filtered", "emails[type eq \"work\"]", 1, "emails"},
		{"complex", "emails[type eq \"work\"].value", 2, "emails"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := parsePath(tt.pathStr)

			if len(path.Segments) != tt.wantSegments {
				t.Errorf("segments = %d, want %d", len(path.Segments), tt.wantSegments)
			}

			if path.Segments[0].Attribute != tt.wantAttr {
				t.Errorf("first attribute = %v, want %v", path.Segments[0].Attribute, tt.wantAttr)
			}
		})
	}
}

func TestPatchProcessor_EnterpriseExtension(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Active:   Bool(true),
		EnterpriseUser: map[string]any{
			"employeeNumber": "EMP001",
		},
	}

	tests := []struct {
		name      string
		patch     *PatchOp
		checkFunc func(*User) bool
		wantErr   bool
	}{
		{
			name: "replace enterprise extension attribute",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber", Value: "EMP12345"},
				},
			},
			checkFunc: func(u *User) bool {
				val, ok := u.EnterpriseUser["employeeNumber"]
				return ok && val == "EMP12345"
			},
			wantErr: false,
		},
		{
			name: "add new enterprise extension attribute",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department", Value: "Engineering"},
				},
			},
			checkFunc: func(u *User) bool {
				val, ok := u.EnterpriseUser["department"]
				return ok && val == "Engineering"
			},
			wantErr: false,
		},
		{
			name: "replace multiple enterprise extension attributes",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department", Value: "Sales"},
					{Op: "replace", Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:costCenter", Value: "CC9999"},
				},
			},
			checkFunc: func(u *User) bool {
				dept, deptOk := u.EnterpriseUser["department"]
				cost, costOk := u.EnterpriseUser["costCenter"]
				return deptOk && dept == "Sales" && costOk && cost == "CC9999"
			},
			wantErr: false,
		},
		{
			name: "remove enterprise extension attribute",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "remove", Path: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber"},
				},
			},
			checkFunc: func(u *User) bool {
				_, exists := u.EnterpriseUser["employeeNumber"]
				return !exists
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh copy for each test
			testUser := &User{
				UserName:       user.UserName,
				Active:         user.Active,
				EnterpriseUser: make(map[string]any),
			}
			// Copy map contents
			for k, v := range user.EnterpriseUser {
				testUser.EnterpriseUser[k] = v
			}

			processor := NewPatchProcessor()
			err := processor.ApplyPatch(testUser, tt.patch)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.checkFunc(testUser) {
				t.Errorf("Patch did not apply correctly")
			}
		})
	}
}

func TestParsePath_URNPaths(t *testing.T) {
	tests := []struct {
		name         string
		pathStr      string
		wantSegments int
		wantFirst    string
		wantSecond   string
	}{
		{
			name:         "enterprise extension simple attribute",
			pathStr:      "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber",
			wantSegments: 2,
			wantFirst:    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
			wantSecond:   "employeeNumber",
		},
		{
			name:         "enterprise extension nested attribute",
			pathStr:      "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager.value",
			wantSegments: 3,
			wantFirst:    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
			wantSecond:   "manager",
		},
		{
			name:         "normal attribute path",
			pathStr:      "name.givenName",
			wantSegments: 2,
			wantFirst:    "name",
			wantSecond:   "givenName",
		},
		{
			name:         "filtered array path",
			pathStr:      "emails[type eq \"work\"].value",
			wantSegments: 2,
			wantFirst:    "emails",
			wantSecond:   "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := parsePath(tt.pathStr)

			if len(path.Segments) != tt.wantSegments {
				t.Errorf("segments = %d, want %d", len(path.Segments), tt.wantSegments)
			}

			if path.Segments[0].Attribute != tt.wantFirst {
				t.Errorf("first attribute = %v, want %v", path.Segments[0].Attribute, tt.wantFirst)
			}

			if len(path.Segments) >= 2 && path.Segments[1].Attribute != tt.wantSecond {
				t.Errorf("second attribute = %v, want %v", path.Segments[1].Attribute, tt.wantSecond)
			}
		})
	}
}

// TestPatchProcessor_BooleanStringFilterComparison tests PATCH operations with
// filtered paths where boolean fields are compared to string representations
func TestPatchProcessor_BooleanStringFilterComparison(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Emails: []Email{
			{Value: "john@work.com", Type: "work", Primary: Boolean(true)},
			{Value: "john@home.com", Type: "home", Primary: Boolean(false)},
		},
		Roles: []Role{
			{Value: "admin", Type: "work", Primary: Boolean(true)},
			{Value: "user", Type: "app", Primary: Boolean(false)},
		},
	}

	tests := []struct {
		name      string
		patch     *PatchOp
		checkFunc func(*User) bool
		wantErr   bool
	}{
		{
			name: "filter with string \"True\" matches Boolean(true)",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[primary eq \"True\"].value", Value: "newemail@work.com"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, email := range u.Emails {
					if bool(email.Primary) {
						return email.Value == "newemail@work.com"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "filter with string \"true\" (lowercase) matches Boolean(true)",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "roles[primary eq \"true\"].value", Value: "superadmin"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, role := range u.Roles {
					if bool(role.Primary) {
						return role.Value == "superadmin"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "filter with string \"False\" matches Boolean(false)",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[primary eq \"False\"].value", Value: "newhome@example.com"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, email := range u.Emails {
					if !bool(email.Primary) {
						return email.Value == "newhome@example.com"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "filter with string \"false\" (lowercase) matches Boolean(false)",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "roles[primary eq \"false\"].value", Value: "guest"},
				},
			},
			checkFunc: func(u *User) bool {
				for _, role := range u.Roles {
					if !bool(role.Primary) {
						return role.Value == "guest"
					}
				}
				return false
			},
			wantErr: false,
		},
		{
			name: "multiple operations with boolean string filters",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "emails[primary eq \"True\"].type", Value: "business"},
					{Op: "replace", Path: "roles[primary eq \"True\"].display", Value: "Administrator"},
				},
			},
			checkFunc: func(u *User) bool {
				emailOk := false
				roleOk := false
				for _, email := range u.Emails {
					if bool(email.Primary) && email.Type == "business" {
						emailOk = true
					}
				}
				for _, role := range u.Roles {
					if bool(role.Primary) && role.Display == "Administrator" {
						roleOk = true
					}
				}
				return emailOk && roleOk
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewPatchProcessor()
			err := processor.ApplyPatch(user, tt.patch)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.checkFunc(user) {
				t.Errorf("ApplyPatch() result check failed")
			}
		})
	}
}
