# Custom JWT Authentication Example

Demonstrates implementing JWT authentication by creating a custom authenticator and passing it via config.

## Quick Start

### 1. Generate Keys

```bash
openssl genrsa -out private-key.pem 2048
openssl rsa -in private-key.pem -pubout -out public-key.pem
```

### 2. Run the Server

```bash
go run .
```

### 3. Create a Token

Use any JWT library. Example:

```go
package main

import (
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "os"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func main() {
    keyData, _ := os.ReadFile("private-key.pem")
    block, _ := pem.Decode(keyData)
    privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
        "sub": "user123",
        "aud": "scim-gateway",
        "iss": "auth.example.com",
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    })

    tokenString, _ := token.SignedString(privateKey)
    fmt.Println(tokenString)
}
```

### 4. Test

```bash
TOKEN="your-jwt-token"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/memory/Users
```

## Building Blocks

**1. Authenticator** (`jwt_authenticator.go` ~100 lines)

```go
type JWTAuthenticator struct {
    publicKey *rsa.PublicKey
    audience  string
    issuer    string
}

func (j *JWTAuthenticator) Authenticate(r *http.Request) error {
    // Extract token, validate signature/claims, add to context
}
```

**2. Config** (`main.go`)

```go
jwtAuth, _ := NewJWTAuthenticator("public-key.pem", "aud", "iss")

cfg := &config.Config{
    Plugins: []config.PluginConfig{{
        Name: "memory",
        Auth: &config.AuthConfig{
            Type: "custom",
            Custom: &config.CustomAuth{Authenticator: jwtAuth},
        },
    }},
}
```

**3. Access in plugins**

```go
claims := ClaimsFromContext(ctx)
userID, _ := claims["sub"].(string)
```

## Extending

**Multiple algorithms:**
```go
case *jwt.SigningMethodRSA:  return j.rsaKey, nil
case *jwt.SigningMethodECDSA: return j.ecdsaKey, nil
```

**JWKS support:**
```go
kid := extractKID(token)
publicKey := j.fetchFromJWKS(kid)
```

**Token caching:**
```go
if result, ok := j.cache.Get(tokenHash); ok { return result }
```

## License

MIT
