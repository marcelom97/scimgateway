package scim

import (
	"testing"
)

func TestFilterParser(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{"simple eq", `userName eq "john"`, false},
		{"simple ne", `userName ne "john"`, false},
		{"contains", `userName co "john"`, false},
		{"starts with", `userName sw "j"`, false},
		{"ends with", `userName ew "n"`, false},
		{"present", `emails pr`, false},
		{"greater than", `age gt 18`, false},
		{"greater or equal", `age ge 18`, false},
		{"less than", `age lt 65`, false},
		{"less or equal", `age le 65`, false},
		{"and operator", `userName eq "john" and active eq true`, false},
		{"or operator", `userName eq "john" or userName eq "jane"`, false},
		{"not operator", `not (active eq false)`, false},
		{"grouped", `(userName eq "john") and (active eq true)`, false},
		{"complex", `userName sw "j" and (active eq true or emails pr)`, false},
		{"complex path", `emails[type eq "work"].value co "example"`, false},
		{"invalid", `userName`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewFilterParser(tt.filter)
			_, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterMatching(t *testing.T) {
	user := &User{
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Emails: []Email{
			{Value: "john@example.com", Type: "work", Primary: true},
			{Value: "john@personal.com", Type: "home"},
		},
	}

	tests := []struct {
		name    string
		filter  string
		want    bool
		wantErr bool
	}{
		{"eq match", `userName eq "john.doe"`, true, false},
		{"eq no match", `userName eq "jane"`, false, false},
		{"ne match", `userName ne "jane"`, true, false},
		{"co match", `userName co "john"`, true, false},
		{"co no match", `userName co "jane"`, false, false},
		{"sw match", `userName sw "john"`, true, false},
		{"ew match", `userName ew "doe"`, true, false},
		{"pr match", `emails pr`, true, false},
		{"pr no match", `phoneNumbers pr`, false, false},
		{"boolean eq", `active eq true`, true, false},
		{"and true", `userName eq "john.doe" and active eq true`, true, false},
		{"and false", `userName eq "john.doe" and active eq false`, false, false},
		{"or true", `userName eq "jane" or active eq true`, true, false},
		{"or false", `userName eq "jane" or active eq false`, false, false},
		{"not true", `not (active eq false)`, true, false},
		{"complex true", `userName sw "john" and (active eq true or emails pr)`, true, false},
		{"nested email", `emails[primary eq true].value co "example"`, true, false},
		// Test Boolean custom type comparison with bool
		{"Boolean type - primary eq true match", `emails[primary eq true].value pr`, true, false},
		{"Boolean type - primary eq false match", `emails[primary eq false].value pr`, true, false},
		{"Boolean type - primary ne false match", `emails[primary ne false].value pr`, true, false},
		{"Boolean type - type eq work and primary eq true", `emails[type eq "work" and primary eq true].value pr`, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewFilterParser(tt.filter)
			filter, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			got := filter.Matches(user)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterWithComplexPaths(t *testing.T) {
	user := &User{
		UserName: "john.doe",
		Emails: []Email{
			{Value: "john@work.com", Type: "work", Primary: true},
			{Value: "john@home.com", Type: "home"},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{"filter array element", `emails[type eq "work"].value co "work"`, true},
		{"filter array no match", `emails[type eq "mobile"].value pr`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewFilterParser(tt.filter)
			filter, err := parser.Parse()
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			got := filter.Matches(user)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareEqual_BooleanType(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		// Test Boolean type with bool
		{"Boolean(true) == bool(true)", Boolean(true), true, true},
		{"Boolean(false) == bool(false)", Boolean(false), false, true},
		{"Boolean(true) != bool(false)", Boolean(true), false, false},
		{"Boolean(false) != bool(true)", Boolean(false), true, false},

		// Test reverse order (bool with Boolean)
		{"bool(true) == Boolean(true)", true, Boolean(true), true},
		{"bool(false) == Boolean(false)", false, Boolean(false), true},
		{"bool(true) != Boolean(false)", true, Boolean(false), false},
		{"bool(false) != Boolean(true)", false, Boolean(true), false},

		// Test Boolean with Boolean
		{"Boolean(true) == Boolean(true)", Boolean(true), Boolean(true), true},
		{"Boolean(false) == Boolean(false)", Boolean(false), Boolean(false), true},
		{"Boolean(true) != Boolean(false)", Boolean(true), Boolean(false), false},

		// Test regular bool with bool (should still work)
		{"bool(true) == bool(true)", true, true, true},
		{"bool(false) == bool(false)", false, false, true},
		{"bool(true) != bool(false)", true, false, false},

		// Test with other types (should not match)
		{"Boolean(true) != string", Boolean(true), "true", false},
		{"Boolean(true) != int", Boolean(true), 1, false},
		{"bool(true) != string", true, "true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
