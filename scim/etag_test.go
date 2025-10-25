package scim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestETagGenerator_Generate(t *testing.T) {
	gen := NewETagGenerator()

	user1 := &User{UserName: "john", Active: Bool(true)}
	user2 := &User{UserName: "john", Active: Bool(true)}
	user3 := &User{UserName: "jane", Active: Bool(true)}

	etag1, err := gen.Generate(user1)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	etag2, err := gen.Generate(user2)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	etag3, err := gen.Generate(user3)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Same data should generate same ETag
	if etag1 != etag2 {
		t.Errorf("Same data should generate same ETag: %v != %v", etag1, etag2)
	}

	// Different data should generate different ETag
	if etag1 == etag3 {
		t.Errorf("Different data should generate different ETag")
	}

	// ETag should be weak
	if etag1[:2] != "W/" {
		t.Errorf("ETag should be weak (start with W/), got %v", etag1)
	}
}

func TestETagGenerator_CheckPreconditions(t *testing.T) {
	gen := NewETagGenerator()
	currentETag := `W/"abc123"`

	tests := []struct {
		name        string
		method      string
		ifMatch     string
		ifNoneMatch string
		wantStatus  int
		wantErr     bool
	}{
		{
			name:       "If-Match success",
			method:     "PUT",
			ifMatch:    `W/"abc123"`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "If-Match fail",
			method:     "PUT",
			ifMatch:    `W/"xyz789"`,
			wantStatus: http.StatusPreconditionFailed,
			wantErr:    true,
		},
		{
			name:       "If-Match wildcard",
			method:     "PUT",
			ifMatch:    "*",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:        "If-None-Match GET not modified",
			method:      "GET",
			ifNoneMatch: `W/"abc123"`,
			wantStatus:  http.StatusNotModified,
			wantErr:     true,
		},
		{
			name:        "If-None-Match GET modified",
			method:      "GET",
			ifNoneMatch: `W/"xyz789"`,
			wantStatus:  http.StatusOK,
			wantErr:     false,
		},
		{
			name:        "If-None-Match PUT fail",
			method:      "PUT",
			ifNoneMatch: `W/"abc123"`,
			wantStatus:  http.StatusPreconditionFailed,
			wantErr:     true,
		},
		{
			name:       "No preconditions",
			method:     "GET",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.ifMatch != "" {
				req.Header.Set("If-Match", tt.ifMatch)
			}
			if tt.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", tt.ifNoneMatch)
			}

			status, err := gen.CheckPreconditions(req, currentETag)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPreconditions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if status != tt.wantStatus {
				t.Errorf("CheckPreconditions() status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestETagGenerator_SetETag(t *testing.T) {
	gen := NewETagGenerator()
	w := httptest.NewRecorder()

	etag := `W/"abc123"`
	gen.SetETag(w, etag)

	if w.Header().Get("ETag") != etag {
		t.Errorf("ETag header = %v, want %v", w.Header().Get("ETag"), etag)
	}
}

func TestUpdateResourceVersion(t *testing.T) {
	meta := &Meta{}
	etag := `W/"abc123"`

	UpdateResourceVersion(meta, etag)

	if meta.Version != `abc123` {
		t.Errorf("Version = %v, want abc123", meta.Version)
	}

	// Test with nil meta
	UpdateResourceVersion(nil, etag)
	// Should not panic
}

func TestETagGenerator_MatchesETag(t *testing.T) {
	gen := NewETagGenerator()

	tests := []struct {
		name        string
		headerValue string
		currentETag string
		want        bool
	}{
		{"exact match", `W/"abc123"`, `W/"abc123"`, true},
		{"no match", `W/"abc123"`, `W/"xyz789"`, false},
		{"wildcard", "*", `W/"abc123"`, true},
		{"multiple match", `W/"abc123", W/"xyz789"`, `W/"abc123"`, true},
		{"multiple no match", `W/"aaa", W/"bbb"`, `W/"ccc"`, false},
		{"empty current", "*", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.matchesETag(tt.headerValue, tt.currentETag)
			if got != tt.want {
				t.Errorf("matchesETag() = %v, want %v", got, tt.want)
			}
		})
	}
}
