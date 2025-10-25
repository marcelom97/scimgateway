package scim

import (
	"testing"
)

func TestValidator_ValidateUser(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{
			name: "valid user",
			user: &User{
				UserName: "john.doe",
				Emails: []Email{
					{Value: "john@example.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing userName",
			user: &User{
				DisplayName: "John",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			user: &User{
				UserName: "john.doe",
				Emails: []Email{
					{Value: "invalid-email"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid userName chars",
			user: &User{
				UserName: "john$doe!",
			},
			wantErr: true,
		},
		{
			name:    "nil user",
			user:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUser(tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateGroup(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		group   *Group
		wantErr bool
	}{
		{
			name: "valid group",
			group: &Group{
				DisplayName: "Admins",
			},
			wantErr: false,
		},
		{
			name: "missing displayName",
			group: &Group{
				ID: "123",
			},
			wantErr: true,
		},
		{
			name:    "nil group",
			group:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateGroup(tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePatchOp(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		patch   *PatchOp
		wantErr bool
	}{
		{
			name: "valid replace",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "replace", Path: "active", Value: false},
				},
			},
			wantErr: false,
		},
		{
			name: "valid add",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "add", Path: "emails", Value: []any{}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid remove",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "remove", Path: "displayName"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid schema",
			patch: &PatchOp{
				Schemas: []string{"invalid"},
				Operations: []PatchOperation{
					{Op: "replace", Path: "active", Value: false},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid op",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "invalid", Path: "active", Value: false},
				},
			},
			wantErr: true,
		},
		{
			name: "remove without path",
			patch: &PatchOp{
				Schemas: []string{SchemaPatchOp},
				Operations: []PatchOperation{
					{Op: "remove"},
				},
			},
			wantErr: true,
		},
		{
			name: "no operations",
			patch: &PatchOp{
				Schemas:    []string{SchemaPatchOp},
				Operations: []PatchOperation{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePatchOp(tt.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePatchOp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		params  *QueryParams
		wantErr bool
		check   func(*QueryParams) bool
	}{
		{
			name:    "valid params",
			params:  &QueryParams{StartIndex: 1, Count: 10},
			wantErr: false,
			check:   func(p *QueryParams) bool { return p.StartIndex == 1 && p.Count == 10 },
		},
		{
			name:    "fix negative startIndex",
			params:  &QueryParams{StartIndex: -1, Count: 10},
			wantErr: false,
			check:   func(p *QueryParams) bool { return p.StartIndex == 1 },
		},
		{
			name:    "fix zero count",
			params:  &QueryParams{StartIndex: 1, Count: 0},
			wantErr: false,
			check:   func(p *QueryParams) bool { return p.Count == 100 },
		},
		{
			name:    "limit max count",
			params:  &QueryParams{StartIndex: 1, Count: 2000},
			wantErr: false,
			check:   func(p *QueryParams) bool { return p.Count == 1000 },
		},
		{
			name:    "invalid sortOrder",
			params:  &QueryParams{SortOrder: "invalid"},
			wantErr: true,
			check:   nil,
		},
		{
			name:    "normalize sortOrder",
			params:  &QueryParams{SortOrder: "ASCENDING"},
			wantErr: false,
			check:   func(p *QueryParams) bool { return p.SortOrder == "ascending" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQueryParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQueryParams() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.check != nil && !tt.check(tt.params) {
				t.Errorf("Params not validated correctly")
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "john.doe", "john.doe"},
		{"with spaces", "  john.doe  ", "john.doe"},
		{"with null bytes", "john\x00doe", "johndoe"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}
