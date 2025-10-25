package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeBearer AuthType = "bearer"
)

// Authenticator defines the interface for authentication
type Authenticator interface {
	Authenticate(r *http.Request) error
}

// BasicAuthenticator implements HTTP Basic authentication
type BasicAuthenticator struct {
	Username string
	Password string
}

// NewBasicAuthenticator creates a new basic authenticator
func NewBasicAuthenticator(username, password string) *BasicAuthenticator {
	return &BasicAuthenticator{
		Username: username,
		Password: password,
	}
}

// Authenticate validates basic authentication credentials
func (ba *BasicAuthenticator) Authenticate(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return fmt.Errorf("invalid authorization type")
	}

	payload, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return fmt.Errorf("invalid base64 encoding")
	}

	parts := strings.SplitN(string(payload), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid authorization format")
	}

	username, password := parts[0], parts[1]

	// Use constant-time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(ba.Username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(ba.Password)) == 1

	if !usernameMatch || !passwordMatch {
		return fmt.Errorf("invalid credentials")
	}

	return nil
}

// BearerAuthenticator implements Bearer token authentication
type BearerAuthenticator struct {
	Token string
}

// NewBearerAuthenticator creates a new bearer token authenticator
func NewBearerAuthenticator(token string) *BearerAuthenticator {
	return &BearerAuthenticator{
		Token: token,
	}
}

// Authenticate validates bearer token
func (ba *BearerAuthenticator) Authenticate(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return fmt.Errorf("invalid authorization type")
	}

	token := auth[7:]

	// Use constant-time comparison
	if subtle.ConstantTimeCompare([]byte(token), []byte(ba.Token)) != 1 {
		return fmt.Errorf("invalid token")
	}

	return nil
}

// MultiAuthenticator supports multiple authentication methods
type MultiAuthenticator struct {
	Authenticators []Authenticator
}

// NewMultiAuthenticator creates a new multi-authenticator
func NewMultiAuthenticator(authenticators ...Authenticator) *MultiAuthenticator {
	return &MultiAuthenticator{
		Authenticators: authenticators,
	}
}

// Authenticate tries each authenticator until one succeeds
func (ma *MultiAuthenticator) Authenticate(r *http.Request) error {
	if len(ma.Authenticators) == 0 {
		return nil // No authentication required
	}

	var lastErr error
	for _, auth := range ma.Authenticators {
		if err := auth.Authenticate(r); err == nil {
			return nil // Authentication successful
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("authentication failed")
}

// Middleware creates an authentication middleware
func Middleware(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator == nil {
				next.ServeHTTP(w, r)
				return
			}

			if err := authenticator.Authenticate(r); err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="SCIM Gateway"`)
				w.Header().Set("Content-Type", "application/scim+json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"schemas":["urn:ietf:params:scim:api:messages:2.0:Error"],"status":"401","detail":"Unauthorized"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NoAuth returns a no-op authenticator
type NoAuth struct{}

// Authenticate always succeeds
func (n *NoAuth) Authenticate(r *http.Request) error {
	return nil
}
