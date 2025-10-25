package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthenticator(t *testing.T) {
	auth := NewBasicAuthenticator("admin", "secret")

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{
			name:    "valid credentials",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			wantErr: false,
		},
		{
			name:    "invalid password",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrong")),
			wantErr: true,
		},
		{
			name:    "invalid username",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("user:secret")),
			wantErr: true,
		},
		{
			name:    "missing header",
			header:  "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			header:  "Basic invalid",
			wantErr: true,
		},
		{
			name:    "wrong auth type",
			header:  "Bearer token123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			err := auth.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBearerAuthenticator(t *testing.T) {
	auth := NewBearerAuthenticator("my-secret-token")

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{
			name:    "valid token",
			header:  "Bearer my-secret-token",
			wantErr: false,
		},
		{
			name:    "invalid token",
			header:  "Bearer wrong-token",
			wantErr: true,
		},
		{
			name:    "missing header",
			header:  "",
			wantErr: true,
		},
		{
			name:    "wrong auth type",
			header:  "Basic dGVzdDp0ZXN0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			err := auth.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiAuthenticator(t *testing.T) {
	basic := NewBasicAuthenticator("admin", "secret")
	bearer := NewBearerAuthenticator("my-token")
	multi := NewMultiAuthenticator(basic, bearer)

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{
			name:    "valid basic",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			wantErr: false,
		},
		{
			name:    "valid bearer",
			header:  "Bearer my-token",
			wantErr: false,
		},
		{
			name:    "invalid both",
			header:  "Bearer wrong",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tt.header)

			err := multi.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	auth := NewBasicAuthenticator("admin", "secret")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := Middleware(auth)(handler)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{
			name:       "authenticated",
			header:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			header:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrong")),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing auth",
			header:     "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusUnauthorized {
				if w.Header().Get("WWW-Authenticate") == "" {
					t.Error("Missing WWW-Authenticate header")
				}
			}
		})
	}
}

func TestNoAuth(t *testing.T) {
	noAuth := &NoAuth{}
	req := httptest.NewRequest("GET", "/", nil)

	err := noAuth.Authenticate(req)
	if err != nil {
		t.Errorf("NoAuth should always succeed, got error: %v", err)
	}
}
