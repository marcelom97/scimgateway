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
