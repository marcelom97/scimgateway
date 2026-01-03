package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthenticator implements auth.Authenticator for JWT tokens
type JWTAuthenticator struct {
	publicKey *rsa.PublicKey
	audience  string
	issuer    string
}

// contextKey is used for storing JWT claims in request context
type contextKey string

const ClaimsContextKey contextKey = "jwt_claims"

// NewJWTAuthenticator creates a JWT authenticator from a public key file
func NewJWTAuthenticator(publicKeyPath, audience, issuer string) (*JWTAuthenticator, error) {
	// Read public key
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse PEM
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse RSA public key
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return &JWTAuthenticator{
		publicKey: rsaKey,
		audience:  audience,
		issuer:    issuer,
	}, nil
}

// Authenticate implements the auth.Authenticator interface
func (j *JWTAuthenticator) Authenticate(r *http.Request) error {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("invalid authorization type")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("token is invalid")
	}

	// Extract and validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid claims format")
	}

	// Validate audience if configured
	if j.audience != "" {
		aud, ok := claims["aud"].(string)
		if !ok || aud != j.audience {
			return fmt.Errorf("invalid audience")
		}
	}

	// Validate issuer if configured
	if j.issuer != "" {
		iss, ok := claims["iss"].(string)
		if !ok || iss != j.issuer {
			return fmt.Errorf("invalid issuer")
		}
	}

	// Add claims to request context
	ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
	*r = *r.WithContext(ctx)

	return nil
}

// ClaimsFromContext extracts JWT claims from the request context
func ClaimsFromContext(ctx context.Context) jwt.MapClaims {
	claims, _ := ctx.Value(ClaimsContextKey).(jwt.MapClaims)
	return claims
}
