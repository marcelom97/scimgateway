# Plugin Development Guide

This guide walks you through creating custom SCIM plugins for the scimgateway library.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Plugin Interface](#plugin-interface)
- [Implementation Approaches](#implementation-approaches)
- [Best Practices](#best-practices)
- [Complete Examples](#complete-examples)
- [Testing Your Plugin](#testing-your-plugin)
- [Common Patterns](#common-patterns)

## Overview

A SCIM plugin connects the gateway to your backend identity store (database, LDAP, API, etc.). The plugin architecture is designed to be simple yet flexible:

- **Simple approach**: Return all data, let the adapter handle filtering/pagination
- **Optimized approach**: Process filters natively in your backend for better performance

The adapter layer automatically handles:
- SCIM filter processing
- Pagination (startIndex, count)
- Sorting (sortBy, sortOrder)
- Attribute selection (attributes, excludedAttributes)
- PATCH operations
- Error responses

## Quick Start

### 1. Create Plugin Structure

```go
package myplugin

import (
    "context"
    "github.com/marcelom97/scimgateway/scim"
)

type MyPlugin struct {
    name string
    // Add your backend connection (db, client, etc.)
}

func New(name string) *MyPlugin {
    return &MyPlugin{
        name: name,
    }
}

func (p *MyPlugin) Name() string {
    return p.name
}
```

### 2. Implement Required Methods

```go
// User operations
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error)
func (p *MyPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error)
func (p *MyPlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error)
func (p *MyPlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error
func (p *MyPlugin) DeleteUser(ctx context.Context, id string) error

// Group operations
func (p *MyPlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error)
func (p *MyPlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error)
func (p *MyPlugin) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error)
func (p *MyPlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error
func (p *MyPlugin) DeleteGroup(ctx context.Context, id string) error
```

### 3. Register and Use

```go
import (
    gateway "github.com/marcelom97/scimgateway"
    "github.com/marcelom97/scimgateway/config"
    "yourproject/myplugin"
)

cfg := &config.Config{
    Gateway: config.GatewayConfig{
        BaseURL: "http://localhost:8080",
        Port:    8080,
    },
    Plugins: []config.PluginConfig{
        {Name: "myplugin"},
    },
}

gw := gateway.New(cfg)
gw.RegisterPlugin(myplugin.New("myplugin"))

if err := gw.Initialize(); err != nil {
    log.Fatal(err)
}

gw.Start()
```

## Plugin Interface

The complete `plugin.Plugin` interface from `plugin/plugin.go`:

```go
type Plugin interface {
    // Name returns the plugin name
    Name() string

    // GetUsers retrieves all users. The adapter layer will apply filtering,
    // pagination, and attribute selection based on params.
    GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error)

    // CreateUser creates a new user
    CreateUser(ctx context.Context, user *scim.User) (*scim.User, error)

    // GetUser retrieves a specific user by ID
    GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error)

    // ModifyUser updates a user's attributes using PATCH operations
    ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error

    // DeleteUser deletes a user
    DeleteUser(ctx context.Context, id string) error

    // GetGroups retrieves all groups. The adapter layer will apply filtering,
    // pagination, and attribute selection based on params.
    GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error)

    // CreateGroup creates a new group
    CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error)

    // GetGroup retrieves a specific group by ID
    GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error)

    // ModifyGroup updates a group's attributes using PATCH operations
    ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error

    // DeleteGroup deletes a group
    DeleteGroup(ctx context.Context, id string) error
}
```

### Method Parameters Explained

- **`ctx context.Context`**: Request context for cancellation, timeouts, and tracing. **Always respect context cancellation**.
- **`params scim.QueryParams`**: Contains filter, sorting, pagination info. You can ignore this and return all data, or use it to optimize queries.
- **`attributes []string`**: Requested attributes for optimization. Adapter handles selection, but you can use this for database query optimization.
- **`patch *scim.PatchOp`**: PATCH operations to apply. Use `scim.NewPatchProcessor().ApplyPatch()` to apply patches.

## Implementation Approaches

### Approach 1: Simple (Return All Data)

**Best for**: Small datasets, simple backends, rapid development

```go
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // Just return all users - adapter handles everything
    return p.fetchAllUsers(ctx)
}
```

**Pros**:
- Simple implementation
- No need to parse SCIM filters
- Adapter handles all SCIM operations

**Cons**:
- May be slow with large datasets
- Transfers all data even if only a few records match

### Approach 2: Optimized (Process Filters Natively)

**Best for**: Large datasets, SQL/NoSQL backends, performance-critical applications

```go
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // Convert SCIM filter to native query
    if params.Filter != "" {
        sqlWhere := convertFilterToSQL(params.Filter)
        return p.db.QueryUsers(ctx, sqlWhere)
    }
    
    // Apply pagination at database level
    if params.StartIndex > 0 || params.Count > 0 {
        return p.db.QueryUsersWithPagination(ctx, params.StartIndex, params.Count)
    }
    
    return p.fetchAllUsers(ctx)
}
```

**Pros**:
- Much better performance with large datasets
- Reduced memory usage
- Database-level optimization (indexes, query planning)

**Cons**:
- More complex implementation
- Need to parse/convert SCIM filters
- Must ensure converted queries match SCIM semantics exactly

## Design Philosophy & API Decisions

### Why `attributes` is passed but `excludedAttributes` is not

The plugin interface passes only `attributes []string` to `GetUser`/`GetGroup` methods,
not the full `QueryParams` with both `attributes` and `excludedAttributes`. This is
an intentional design decision based on practical considerations.

**Rationale:**

1. **Positive vs Negative Selection:**
   - `attributes` (inclusion): Easy to optimize - "give me ONLY id, userName, emails"
   - `excludedAttributes` (exclusion): Hard to optimize - requires complete schema knowledge

2. **Query Optimization:**
   - With `attributes`: `SELECT id, userName, emails FROM users WHERE id = ?` ✅
   - With `excludedAttributes`: Must know ALL possible columns to exclude ❌

3. **Schema Coupling:**
   - `attributes` optimization works regardless of schema changes
   - `excludedAttributes` optimization breaks when new attributes are added

4. **Real-World Performance:**
   - Server-layer filtering of `excludedAttributes` is fast enough for typical SCIM payloads
   - Optimization only matters for truly large binary fields (photos, certificates)
   - Most SCIM resources are small (< 10KB) where filtering overhead is negligible

**Both query parameters are fully supported:**

```bash
# Works perfectly - inclusion optimization at plugin level
GET /Users/123?attributes=id,userName,emails

# Works perfectly - exclusion filtering at server level  
GET /Users/123?excludedAttributes=photos,x509Certificates

# Rejected per RFC 7644 - mutually exclusive
GET /Users/123?attributes=id&excludedAttributes=name
```

**When to optimize `attributes`:**

- ✅ SQL databases with normalized schema (column projection reduces I/O)
- ✅ Large datasets where partial queries significantly reduce transfer size
- ❌ JSON/JSONB storage (whole document is retrieved anyway)
- ❌ In-memory storage (filtering is instant)
- ❌ REST API backends (typically return full objects)

**Bottom line:** The current API provides the valuable optimization (inclusion)
while avoiding the complexity and tight coupling of exclusion-based optimization.
This design makes the library "good enough for the vast majority" of use cases.

## Best Practices

### 1. ID Generation

Always generate IDs if not provided:

```go
func (p *MyPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    if user.ID == "" {
        user.ID = uuid.New().String()
    }
    // ... rest of implementation
}
```

### 2. Schema Assignment

Set default schemas if not provided:

```go
if len(user.Schemas) == 0 {
    user.Schemas = []string{scim.SchemaUser}
}
```

### 3. Metadata Management

Always set meta fields correctly:

```go
now := time.Now()
user.Meta = &scim.Meta{
    ResourceType: "User",
    Created:      &now,
    LastModified: &now,
    Version:      fmt.Sprintf("W/\"%s\"", user.ID), // ETag format
}
```

For updates, update `LastModified` and `Version`:

```go
now := time.Now()
user.Meta.LastModified = &now
user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())
```

### 4. Error Handling

Use SCIM error types for proper HTTP responses:

```go
import "github.com/marcelom97/scimgateway/scim"

// Not found (404)
if !found {
    return nil, scim.ErrNotFound("User", id)
}

// Uniqueness violation (409)
if duplicate {
    return nil, scim.ErrUniqueness("userName 'john' already exists")
}

// Internal error (500)
if err != nil {
    return nil, scim.ErrInternalServer(fmt.Sprintf("database error: %v", err))
}

// Invalid filter (400)
if invalidFilter {
    return nil, scim.ErrInvalidFilter("invalid filter syntax")
}
```

### 5. Context Handling

Always check context cancellation in long operations:

```go
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // Check cancellation before expensive operations
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context in database queries
    rows, err := p.db.QueryContext(ctx, "SELECT * FROM users")
    // ...
}
```

### 6. Thread Safety

Protect shared state with mutexes:

```go
type MyPlugin struct {
    name  string
    data  map[string]*scim.User
    mu    sync.RWMutex  // Use RWMutex for read-heavy workloads
}

func (p *MyPlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    user, ok := p.data[id]
    if !ok {
        return nil, scim.ErrNotFound("User", id)
    }
    return user, nil
}

func (p *MyPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.data[user.ID] = user
    return user, nil
}
```

### 7. PATCH Operations

Use the built-in patch processor:

```go
func (p *MyPlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
    user, err := p.GetUser(ctx, id, nil)
    if err != nil {
        return err
    }
    
    // Apply patch using built-in processor
    patcher := scim.NewPatchProcessor()
    if err := patcher.ApplyPatch(user, patch); err != nil {
        return scim.ErrInvalidSyntax(fmt.Sprintf("patch failed: %v", err))
    }
    
    // Update metadata
    now := time.Now()
    user.Meta.LastModified = &now
    user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())
    
    // Save to backend
    return p.saveUser(ctx, user)
}
```

## Complete Examples

### Example 1: In-Memory Plugin

See `memory/memory.go` for a complete, production-ready in-memory plugin with:
- Thread-safe operations using `sync.RWMutex`
- Proper ID generation and metadata management
- Full SCIM error handling
- PATCH operation support

### Example 2: SQLite Plugin

See `examples/sqlite/plugin.go` for a database-backed plugin showing:
- Database schema initialization
- Context-aware database queries
- JSON serialization of SCIM resources
- Uniqueness constraint handling
- Proper error mapping (sql.ErrNoRows → scim.ErrNotFound)

### Example 3: Minimal Custom Plugin

```go
package simple

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/marcelom97/scimgateway/scim"
)

type SimplePlugin struct {
    name   string
    users  map[string]*scim.User
    groups map[string]*scim.Group
    mu     sync.RWMutex
}

func New(name string) *SimplePlugin {
    return &SimplePlugin{
        name:   name,
        users:  make(map[string]*scim.User),
        groups: make(map[string]*scim.Group),
    }
}

func (p *SimplePlugin) Name() string {
    return p.name
}

// GetUsers - Simple approach: return all users
func (p *SimplePlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()

    users := make([]*scim.User, 0, len(p.users))
    for _, user := range p.users {
        users = append(users, user)
    }
    return users, nil
}

// CreateUser with proper metadata
func (p *SimplePlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Generate ID if needed
    if user.ID == "" {
        user.ID = uuid.New().String()
    }

    // Set schemas
    if len(user.Schemas) == 0 {
        user.Schemas = []string{scim.SchemaUser}
    }

    // Set metadata
    now := time.Now()
    user.Meta = &scim.Meta{
        ResourceType: "User",
        Created:      &now,
        LastModified: &now,
        Version:      fmt.Sprintf("W/\"%s\"", user.ID),
    }

    p.users[user.ID] = user
    return user, nil
}

// GetUser with proper error handling
func (p *SimplePlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()

    user, ok := p.users[id]
    if !ok {
        return nil, scim.ErrNotFound("User", id)
    }
    return user, nil
}

// ModifyUser with PATCH support
func (p *SimplePlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    user, ok := p.users[id]
    if !ok {
        return scim.ErrNotFound("User", id)
    }

    // Apply patch
    patcher := scim.NewPatchProcessor()
    if err := patcher.ApplyPatch(user, patch); err != nil {
        return err
    }

    // Update metadata
    now := time.Now()
    user.Meta.LastModified = &now
    user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

    return nil
}

// DeleteUser with proper error handling
func (p *SimplePlugin) DeleteUser(ctx context.Context, id string) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if _, ok := p.users[id]; !ok {
        return scim.ErrNotFound("User", id)
    }

    delete(p.users, id)
    return nil
}

// Implement Group methods similarly...
func (p *SimplePlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error) {
    // Similar to GetUsers
    return nil, scim.ErrNotImplemented("GetGroups")
}

func (p *SimplePlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
    return nil, scim.ErrNotImplemented("CreateGroup")
}

func (p *SimplePlugin) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error) {
    return nil, scim.ErrNotImplemented("GetGroup")
}

func (p *SimplePlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
    return scim.ErrNotImplemented("ModifyGroup")
}

func (p *SimplePlugin) DeleteGroup(ctx context.Context, id string) error {
    return scim.ErrNotImplemented("DeleteGroup")
}
```

## Testing Your Plugin

### Unit Tests

```go
package myplugin

import (
    "context"
    "testing"

    "github.com/marcelom97/scimgateway/scim"
)

func TestCreateUser(t *testing.T) {
    plugin := New("test")
    ctx := context.Background()

    user := &scim.User{
        UserName: "john.doe",
        Name: &scim.Name{
            GivenName:  "John",
            FamilyName: "Doe",
        },
    }

    created, err := plugin.CreateUser(ctx, "", user)
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }

    if created.ID == "" {
        t.Error("Expected ID to be generated")
    }

    if created.Meta == nil {
        t.Error("Expected Meta to be set")
    }

    if created.Meta.Created == nil {
        t.Error("Expected Created timestamp")
    }
}

func TestGetUser_NotFound(t *testing.T) {
    plugin := New("test")
    ctx := context.Background()

    _, err := plugin.GetUser(ctx, "", "nonexistent", nil)
    if err == nil {
        t.Error("Expected error for nonexistent user")
    }

    // Check it's a SCIM error with correct status
    scimErr, ok := err.(*scim.SCIMError)
    if !ok {
        t.Errorf("Expected SCIMError, got %T", err)
    }
    if scimErr.Status != 404 {
        t.Errorf("Expected status 404, got %d", scimErr.Status)
    }
}
```

### Integration Tests

```go
func TestPluginWithGateway(t *testing.T) {
    cfg := &config.Config{
        Gateway: config.GatewayConfig{
            BaseURL: "http://localhost:8080",
            Port:    8080,
        },
        Plugins: []config.PluginConfig{
            {Name: "test"},
        },
    }

    gw := gateway.New(cfg)
    gw.RegisterPlugin(New("test"))

    if err := gw.Initialize(); err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }

    handler, err := gw.Handler()
    if err != nil {
        t.Fatalf("Handler failed: %v", err)
    }

    // Test with httptest
    req := httptest.NewRequest("GET", "/test/Users", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != 200 {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

## Common Patterns

### Pattern 1: Database Connection Management

```go
type DBPlugin struct {
    name string
    db   *sql.DB
}

func New(name string, connString string) (*DBPlugin, error) {
    db, err := sql.Open("postgres", connString)
    if err != nil {
        return nil, err
    }

    return &DBPlugin{name: name, db: db}, nil
}

func (p *DBPlugin) Close() error {
    return p.db.Close()
}
```

### Pattern 2: Connection Pooling

```go
func New(name string, connString string) (*DBPlugin, error) {
    db, err := sql.Open("postgres", connString)
    if err != nil {
        return nil, err
    }

    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    return &DBPlugin{name: name, db: db}, nil
}
```

### Pattern 3: Retry Logic

```go
func (p *DBPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    var err error
    for i := 0; i < 3; i++ {
        err = p.tryCreateUser(ctx, user)
        if err == nil {
            return user, nil
        }
        if !isRetryable(err) {
            break
        }
        time.Sleep(time.Duration(i*100) * time.Millisecond)
    }
    return nil, err
}
```

## Custom Authentication Patterns

Implement the `auth.Authenticator` interface:

```go
type Authenticator interface {
    Authenticate(r *http.Request) error
}
```

### Building Blocks

**1. Implement authenticator:**

```go
type APIKeyAuthenticator struct {
    validKeys map[string]string  // key -> user_id
}

func (a *APIKeyAuthenticator) Authenticate(r *http.Request) error {
    apiKey := r.Header.Get("X-API-Key")
    if apiKey == "" {
        return fmt.Errorf("missing API key")
    }
    
    userID, ok := a.validKeys[apiKey]
    if !ok {
        return fmt.Errorf("invalid API key")
    }
    
    // Add data to context for plugins to access
    ctx := context.WithValue(r.Context(), "user_id", userID)
    *r = *r.WithContext(ctx)
    
    return nil
}
```

**2. Pass via config:**

```go
myAuth := &APIKeyAuthenticator{validKeys: map[string]string{"key123": "user456"}}

cfg := &config.Config{
    Plugins: []config.PluginConfig{{
        Name: "myplugin",
        Auth: &config.AuthConfig{
            Type: "custom",
            Custom: &config.CustomAuth{Authenticator: myAuth},
        },
    }},
}
```

**3. Access auth data in plugins:**

```go
func (p *MyPlugin) GetUser(ctx context.Context, id string, attrs []string) (*scim.User, error) {
    userID, _ := ctx.Value("user_id").(string)
    // Use for authorization, logging, etc.
}
```

**Multiple auth methods:**

```go
multi := auth.NewMultiAuthenticator(
    &APIKeyAuthenticator{...},
    &JWTAuthenticator{...},
)
```

**See:** `examples/jwt-auth/` for complete JWT implementation (~100 lines)

## Additional Resources

- **Plugin Interface**: See `plugin/plugin.go` for the complete interface definition
- **SCIM Types**: See `scim/types.go` for User, Group, and other SCIM types
- **Error Types**: See `scim/errors.go` for SCIM error constructors
- **Patch Operations**: See `scim/patch.go` for PATCH operation details
- **Auth Interface**: See `auth/auth.go` for the Authenticator interface
- **Examples**: See `examples/` directory for complete working examples
  - `examples/jwt-auth/` - Custom JWT authentication implementation

## Support

If you have questions or need help:
- Check existing examples in `examples/` directory
- Review the in-memory plugin (`memory/memory.go`) as reference
- Open an issue on GitHub with your question

## Contributing

Contributions of new plugin examples are welcome! Please ensure:
- Full interface implementation
- Proper error handling with SCIM error types
- Thread safety where applicable
- Tests demonstrating functionality
- Documentation of any external dependencies
