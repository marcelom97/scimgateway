# SCIM Gateway

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/marcelom97/scimgateway)](https://goreportcard.com/report/github.com/marcelom97/scimgateway)

A production-ready SCIM 2.0 (System for Cross-domain Identity Management) gateway library for Go. This library provides a complete implementation of the SCIM protocol with a flexible plugin architecture that allows you to connect to any identity backend.

> **Inspired by**: This Go implementation is inspired by the popular Node.js [scimgateway](https://github.com/jelhub/scimgateway) project by Jarle Elshaug. We've reimagined the architecture for Go, bringing type safety, excellent performance, and the simplicity of Go's concurrency model to SCIM gateway implementations.

## Features

- **Full SCIM 2.0 Protocol Support** (RFC 7643 & 7644)
  - User and Group resource management
  - Advanced filtering with all operators (eq, ne, co, sw, ew, pr, gt, ge, lt, le)
  - Logical operators (and, or, not) with proper precedence
  - PATCH operations (add, remove, replace) with path expressions
  - Bulk operations with bulkId reference handling and circular dependency detection
  - Pagination with startIndex and count parameters
  - Sorting by any attribute
  - Attribute selection and exclusion
  - ETag support for optimistic concurrency control
  - Schema discovery endpoints

- **Flexible Plugin Architecture**
  - Simple plugin interface for connecting any backend
  - Included in-memory plugin for testing
  - Plugins can be simple (return all data) or optimized (process filters natively)
  - Multiple plugins can be registered simultaneously

- **Per-Plugin Authentication**
  - Each plugin can have its own authentication configuration
  - Basic authentication support
  - Bearer token authentication support
  - Custom authenticators via simple interface
  - No authentication (public access) option
  - Constant-time credential comparison for security

- **Observability & Validation**
  - Optional structured logging with `log/slog` integration
  - HTTP request logging middleware (method, path, status, duration, client IP)
  - Comprehensive configuration validation with detailed error messages
  - Validates port ranges, URLs, TLS config, authentication, and plugin setup
  - Automatic validation on gateway initialization

- **Production Ready**
  - Thread-safe operations
  - Comprehensive error handling with no panics
  - Excellent test coverage (75%+)
  - TLS support
  - Can run as standalone server or embedded HTTP handler
  - Fail-fast validation with clear error messages

## Why Choose This Library?

- **🚀 Production Ready**: Comprehensive error handling, no panics, excellent test coverage (75%+)
- **🔌 Plugin Architecture**: Connect to any backend (LDAP, SQL, NoSQL, APIs) with a simple interface
- **✅ Full SCIM 2.0 Compliance**: Implements RFC 7643 & 7644 specifications completely
- **📊 Observable**: Built-in structured logging with `log/slog`, HTTP request tracking
- **🛡️ Secure by Default**: Constant-time auth comparison, automatic config validation, TLS support
- **⚡ High Performance**: Efficient filtering, thread-safe operations, optimized for Go
- **🎯 Type Safe**: Leverage Go's type system for compile-time safety
- **📖 Well Documented**: Clear examples, comprehensive tests, detailed API documentation

## Requirements

- Go 1.22 or higher (uses enhanced routing patterns)

## Installation

```bash
go get github.com/marcelom97/scimgateway
```

## Quick Start

Here's a minimal example to get started:

```go
package main

import (
    "log"

    gateway "github.com/marcelom97/scimgateway"
    "github.com/marcelom97/scimgateway/config"
    "github.com/marcelom97/scimgateway/memory"
)

func main() {
    // Create configuration with per-plugin authentication
    cfg := &config.Config{
        Gateway: config.GatewayConfig{
            BaseURL: "http://localhost:8080",
            Port:    8080,
        },
        Plugins: []config.PluginConfig{
            {
                Name: "memory",
                Auth: &config.AuthConfig{
                    Type: "basic",
                    Basic: &config.BasicAuth{
                        Username: "admin",
                        Password: "secret",
                    },
                },
            },
        },
    }

    // Create and configure gateway
    gw := gateway.New(cfg)

    // Register plugin (auth is automatically configured from cfg.Plugins)
    gw.RegisterPlugin(memory.New("memory"))

    // Optional: Enable structured logging
    // gw.SetLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

    // Initialize and start (config is validated automatically)
    if err := gw.Initialize(); err != nil {
        log.Fatal(err)
    }

    log.Println("Starting SCIM Gateway on :8080")
    if err := gw.Start(); err != nil {
        log.Fatal(err)
    }
}
```

Test it:
```bash
# List users
curl -u admin:secret http://localhost:8080/memory/Users

# Create a user
curl -u admin:secret -X POST http://localhost:8080/memory/Users \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "john.doe",
    "name": {
      "givenName": "John",
      "familyName": "Doe"
    },
    "emails": [{"value": "john.doe@example.com", "primary": true}]
  }'

# Search with filter
curl -u admin:secret "http://localhost:8080/memory/Users?filter=userName%20eq%20%22john.doe%22"
```

## Usage Examples

### Embedded Mode (Use as HTTP Handler)

```go
gw := gateway.New(cfg)
gw.RegisterPlugin(memory.New("memory"))

if err := gw.Initialize(); err != nil {
    log.Fatal(err)
}

// Get the handler
handler, err := gw.Handler()
if err != nil {
    log.Fatal(err)
}

// Use as part of your existing HTTP server
mux := http.NewServeMux()
mux.Handle("/scim/", http.StripPrefix("/scim", handler))
http.ListenAndServe(":8080", mux)
```

### Multiple Plugins with Different Authentication

Each plugin can have its own independent authentication configuration:

```go
cfg := &config.Config{
    Gateway: config.GatewayConfig{
        BaseURL: "http://localhost:8080",
        Port:    8080,
    },
    Plugins: []config.PluginConfig{
        {
            Name: "memory",
            Auth: &config.AuthConfig{
                Type: "basic",
                Basic: &config.BasicAuth{Username: "admin", Password: "secret"},
            },
        },
        {
            Name: "ldap",
            Auth: &config.AuthConfig{
                Type: "bearer",
                Bearer: &config.BearerAuth{Token: "ldap-token-123"},
            },
        },
        {
            Name: "public",
            // No Auth field means no authentication required
        },
    },
}

gw := gateway.New(cfg)

// Register multiple backends (auth is configured per plugin)
gw.RegisterPlugin(memory.New("memory"))
gw.RegisterPlugin(ldapPlugin.New("ldap"))
gw.RegisterPlugin(sqlPlugin.New("public"))

if err := gw.Initialize(); err != nil {
    log.Fatal(err)
}

if err := gw.Start(); err != nil {
    log.Fatal(err)
}

// Access different backends with different auth:
// http://localhost:8080/memory/Users    (requires basic auth: admin/secret)
// http://localhost:8080/ldap/Users      (requires bearer token: ldap-token-123)
// http://localhost:8080/public/Users    (no auth required)
```

### TLS Configuration

```go
cfg := &config.Config{
    Gateway: config.GatewayConfig{
        BaseURL: "https://localhost:8443",
        Port:    8443,
        TLS: &config.TLS{
            Enabled:  true,
            CertFile: "/path/to/cert.pem",
            KeyFile:  "/path/to/key.pem",
        },
    },
    Plugins: []config.PluginConfig{
        {
            Name: "memory",
            Auth: &config.AuthConfig{
                Type: "basic",
                Basic: &config.BasicAuth{Username: "admin", Password: "secret"},
            },
        },
    },
}
```

### Bearer Token Authentication (Per-Plugin)

Configure bearer token authentication for a specific plugin:

```go
cfg := &config.Config{
    Gateway: config.GatewayConfig{
        BaseURL: "http://localhost:8080",
        Port:    8080,
    },
    Plugins: []config.PluginConfig{
        {
            Name: "memory",
            Auth: &config.AuthConfig{
                Type: "bearer",
                Bearer: &config.BearerAuth{Token: "my-secret-token"},
            },
        },
    },
}

gw := gateway.New(cfg)
gw.RegisterPlugin(memory.New("memory"))
```

Test with:
```bash
curl -H "Authorization: Bearer my-secret-token" http://localhost:8080/memory/Users
```

### Custom Authentication

Implement the `auth.Authenticator` interface:

```go
type Authenticator interface {
    Authenticate(r *http.Request) error
}
```

Pass it via config:

```go
jwtAuth := &MyJWTAuthenticator{publicKey, audience, issuer}

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

**Example:** `examples/jwt-auth/` - JWT with RSA signatures (~100 lines)

Only Basic and Bearer auth are built-in to keep the core minimal.

## Creating Custom Plugins

Implement the `plugin.Plugin` interface:

```go
package myplugin

import (
    "context"
    "github.com/marcelom97/scimgateway/plugin"
    "github.com/marcelom97/scimgateway/scim"
)

type MyPlugin struct {
    name string
    // your backend connection
}

func New(name string) *MyPlugin {
    return &MyPlugin{name: name}
}

func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // Simple approach: return all users, adapter handles filtering/pagination
    users := p.fetchAllUsers()
    return users, nil

    // OR optimized: process filters in your backend
    // if params.Filter != "" {
    //     return p.queryUsersWithFilter(params.Filter)
    // }
}

func (p *MyPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    // Create user in your backend
    return user, nil
}

// Implement other methods: GetUser, ModifyUser, DeleteUser, GetGroups, CreateGroup, etc.
```

See `examples/custom-plugin/` for a complete example.

## API Endpoints

The gateway provides standard SCIM 2.0 endpoints:

### Users
- `GET /{plugin}/Users` - List all users (supports filtering, pagination, sorting, attributes)
- `POST /{plugin}/Users` - Create a user
- `GET /{plugin}/Users/{id}` - Get a specific user
- `PUT /{plugin}/Users/{id}` - Replace a user
- `PATCH /{plugin}/Users/{id}` - Modify a user
- `DELETE /{plugin}/Users/{id}` - Delete a user

### Groups
- `GET /{plugin}/Groups` - List all groups
- `POST /{plugin}/Groups` - Create a group
- `GET /{plugin}/Groups/{id}` - Get a specific group
- `PUT /{plugin}/Groups/{id}` - Replace a group
- `PATCH /{plugin}/Groups/{id}` - Modify a group
- `DELETE /{plugin}/Groups/{id}` - Delete a group

### Search
- `POST /{plugin}/.search` - Search across resource types

### Bulk Operations
- `POST /{plugin}/Bulk` - Perform multiple operations in a single request

### Discovery
- `GET /ServiceProviderConfig` - Server capabilities
- `GET /Schemas` - Supported schemas
- `GET /ResourceTypes` - Resource type definitions

## Query Parameters

### Filtering
Use the `filter` parameter with SCIM filter expressions:

```bash
# Simple equality
filter=userName eq "john.doe"

# Contains
filter=emails co "example.com"

# Logical operators
filter=userName eq "john" and active eq true

# Complex attribute paths
filter=emails[type eq "work"].value co "example.com"
```

Supported operators: `eq`, `ne`, `co`, `sw`, `ew`, `pr`, `gt`, `ge`, `lt`, `le`, `and`, `or`, `not`

### Pagination
```bash
# Get items 11-20
?startIndex=11&count=10
```

### Sorting
```bash
# Sort by username ascending
?sortBy=userName&sortOrder=ascending
```

### Attribute Selection
```bash
# Return only specific attributes
?attributes=userName,emails,name

# Exclude attributes
?excludedAttributes=groups,roles
```

## PATCH Operations

PATCH requests support three operations:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "add",
      "path": "emails",
      "value": [{"value": "new@example.com", "type": "work"}]
    },
    {
      "op": "replace",
      "path": "active",
      "value": false
    },
    {
      "op": "remove",
      "path": "emails[type eq \"work\"]"
    }
  ]
}
```

## Bulk Operations

Perform multiple operations in a single request:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "user1",
      "data": {
        "userName": "john.doe",
        "name": {"givenName": "John", "familyName": "Doe"}
      }
    },
    {
      "method": "POST",
      "path": "/Groups",
      "data": {
        "displayName": "Admins",
        "members": [{"value": "bulkId:user1"}]
      }
    }
  ]
}
```

The gateway automatically detects circular bulkId references and returns proper error responses.

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./scim -v
go test ./plugin -v
go test ./auth -v

# Using Makefile
make test
make build
make all  # tidy, fmt, test, build
```

## Project Structure

```
.
├── auth/           # Authentication middleware and providers
├── config/         # Configuration types and defaults
├── examples/       # Example implementations
│   ├── in-memory/     # Simple in-memory server
│   ├── sqlite/        # Simple sqlite-backed server
│   └── custom-plugin/ # Custom plugin example
├── memory/         # In-memory plugin implementation
├── plugin/         # Plugin interface and manager
├── scim/           # SCIM protocol implementation
│   ├── attributes.go  # Attribute selection
│   ├── bulk.go        # Bulk operations
│   ├── discovery.go   # Schema endpoints
│   ├── etag.go        # ETag generation
│   ├── filter.go      # Filter parser
│   ├── handler.go     # HTTP handlers
│   ├── patch.go       # PATCH operations
│   ├── query_utils.go # Query processing
│   ├── search.go      # Search endpoint
│   ├── server.go      # HTTP routing
│   ├── types.go       # SCIM resource types
│   └── validation.go  # Input validation
└── gateway.go      # Main gateway implementation
```

## SCIM 2.0 Compliance

This library is fully compliant with SCIM 2.0 specifications:

- **[RFC 7642](https://datatracker.ietf.org/doc/html/rfc7642)**: SCIM Definitions, Overview, Concepts, and Requirements
- **[RFC 7643](https://datatracker.ietf.org/doc/html/rfc7643)**: SCIM Core Schema - Defines User, Group, and other resource schemas
- **[RFC 7644](https://datatracker.ietf.org/doc/html/rfc7644)**: SCIM Protocol - Defines HTTP endpoints, filtering, pagination, bulk operations

### Compliance Features

**Filter Processing:**
- ✅ All comparison operators: `eq`, `ne`, `co`, `sw`, `ew`, `pr`, `gt`, `ge`, `lt`, `le`
- ✅ Logical operators: `and`, `or`, `not` with proper precedence
- ✅ Complex attribute paths: `emails[type eq "work"].value`
- ✅ Case-sensitive value comparisons (as per spec)

**Query Parameters:**
- ✅ `attributes` and `excludedAttributes` with mutual exclusivity enforcement
- ✅ Sorting by any attribute (`sortBy`, `sortOrder`)
- ✅ Pagination with `startIndex` and `count`
- ✅ Nested attribute support (e.g., `name.familyName`, `emails.value`)

**Operations:**
- ✅ PATCH operations (`add`, `remove`, `replace`) with path expressions
- ✅ Bulk operations with `bulkId` reference resolution
- ✅ Circular dependency detection in bulk operations
- ✅ ETag support for optimistic concurrency control

**Error Handling:**
- ✅ Proper HTTP status codes (400, 404, 409, 500, etc.)
- ✅ SCIM error schema responses with `scimType`
- ✅ Detailed error messages

**Metadata:**
- ✅ `meta.created`, `meta.lastModified`, `meta.version`, `meta.location`
- ✅ `meta.resourceType` for resource identification

For detailed compliance test results and edge case handling, see [COMPLIANCE.md](COMPLIANCE.md).

## Logging and Observability

The gateway includes optional structured logging using Go's standard `log/slog` package:

```go
import "log/slog"

// JSON logging to stdout
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
gw.SetLogger(logger)

// Text logging to a file
file, _ := os.OpenFile("gateway.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
logger := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{
    Level: slog.LevelWarn, // Only log warnings and errors
}))
gw.SetLogger(logger)

// No logging (default)
gw.SetLogger(nil) // Or simply don't call SetLogger()
```

**What gets logged:**
- Gateway initialization and startup events
- Plugin registration and lookup failures
- Configuration validation errors
- HTTP requests (method, path, status code, duration, client IP, user agent)
- Request failures and errors

**Log Levels:**
- `INFO`: Successful requests (status 2xx-3xx)
- `WARN`: Client errors (status 4xx)
- `ERROR`: Server errors (status 5xx), initialization failures

## Configuration Validation

The gateway automatically validates your configuration on initialization:

```go
cfg := &config.Config{
    Gateway: config.GatewayConfig{
        BaseURL: "invalid-url",  // ❌ Invalid
        Port:    99999,          // ❌ Out of range
    },
    Plugins: []config.PluginConfig{}, // ❌ No plugins
}

gw := gateway.New(cfg)
err := gw.Initialize()
// Returns detailed error:
// "invalid configuration: config validation failed with 3 errors:
//   1. config validation error [gateway.baseURL]: invalid URL format
//   2. config validation error [gateway.port]: port 99999 is out of range: must be between 1 and 65535
//   3. config validation error [plugins]: at least one plugin must be configured"
```

**Validation checks:**
- BaseURL format (must be valid http/https URL with host)
- Port range (1-65535)
- TLS configuration (cert and key files required when enabled)
- Per-plugin authentication configuration (credentials required for basic/bearer auth types)
- Plugin configuration (at least one plugin, no duplicates, valid names)
- Plugin registration (at least one plugin must be registered)

## Known Limitations

- **Case Sensitivity**: SCIM attribute names are case-insensitive per spec, but this implementation treats them as case-sensitive. Use the exact attribute names as defined in the schema (e.g., `userName`, not `username`).

- **Attribute Path Complexity**: Complex filter paths like `emails[type eq "work" and primary eq true].value` are supported, but deeply nested paths (3+ levels) may have performance implications with large datasets.

- **Bulk Operation Size**: No built-in limit on bulk operation size. Consider implementing size limits in your plugins or reverse proxy to prevent resource exhaustion.

- **ETag and Versioning**: The gateway uses `meta.version` for ETag generation (optimistic concurrency control). If your plugin doesn't maintain stable version numbers across reads, the gateway will regenerate ETags, which means:
  - `If-Match` headers may not work reliably for detecting concurrent modifications
  - `If-None-Match` for conditional GET will still work correctly
  - Recommendation: Have your plugin maintain a version counter or timestamp for each resource

- **Schema Extensions**: SCIM schema extensions are supported for data storage and retrieval, but custom schema definitions cannot be added to the `/Schemas` endpoint without code changes.

- **Internationalization**: String comparisons in filters use Go's default string comparison, which may not handle all Unicode normalization cases as expected.

## Performance Considerations

### Plugin Optimization
Plugins can return all data and let the adapter handle filtering (simple approach), or they can optimize by processing filters natively:

```go
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // Optimized: convert SCIM filter to SQL WHERE clause
    if params.Filter != "" {
        sqlWhere := convertSCIMFilterToSQL(params.Filter)
        return p.db.Query("SELECT * FROM users WHERE " + sqlWhere)
    }
    return p.db.Query("SELECT * FROM users")
}
```

### Thread Safety
- All gateway operations are thread-safe
- The included memory plugin uses `sync.RWMutex` for concurrent access
- Custom plugins should implement their own concurrency controls

## Examples

See the `examples/` directory for complete working examples:

- `examples/memory/` - In-memory storage reference implementation
- `examples/postgres/` - PostgreSQL backend with query builder
- `examples/sqlite/` - SQLite backend implementation
- `examples/jwt-auth/` - Custom JWT authentication example
- `examples/custom-plugin/` - Template for implementing custom plugins

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code is formatted: `go fmt ./...`
3. Changes comply with SCIM 2.0 specifications (RFC 7642 & 7643 & 7644)
4. New features include tests

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Documentation

- [Security](SECURITY.md) - Security considerations and vulnerability reporting
- [Plugin Development](PLUGIN_DEVELOPMENT.md) - Guide to building custom plugins
- [Compliance](COMPLIANCE.md) - SCIM 2.0 RFC compliance status

## Links

### SCIM 2.0 Specifications
- [RFC 7642 - SCIM: Definitions, Overview, Concepts, and Requirements](https://datatracker.ietf.org/doc/html/rfc7642)
- [RFC 7643 - SCIM: Core Schema](https://datatracker.ietf.org/doc/html/rfc7643)
- [RFC 7644 - SCIM: Protocol](https://datatracker.ietf.org/doc/html/rfc7644)

## Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/marcelom97/scimgateway/issues)
- **Questions**: Use [GitHub Discussions](https://github.com/marcelom97/scimgateway/discussions) for questions and community support

## Acknowledgments

This project was inspired by the excellent [Node.js SCIM Gateway](https://github.com/jelhub/scimgateway) by Jarle Elshaug. While this is a complete reimplementation in Go with different architectural decisions, we appreciate the pioneering work done in the Node.js ecosystem.
