# In-Memory SCIM Example

This is a **complete runnable example** of a SCIM gateway using in-memory storage. It demonstrates how to implement the `plugin.Plugin` interface and run a SCIM server.

## ⚠️ Not for Production Use

This example stores all data in memory and is **not suitable for production use** because:

- **Data is lost on restart** - no persistence
- **Not scalable** - limited by available RAM
- **No concurrent instance support** - each instance has its own isolated data
- **No audit trail** - no logging of changes
- **No backup/recovery** - data cannot be recovered if lost

## Purpose

This example serves as:

1. **Learning resource** - Shows how to implement the plugin interface
2. **Testing tool** - Useful for unit tests and local development
3. **Prototype base** - Quick starting point for experimenting with SCIM
4. **Complete working example** - Runnable SCIM server you can test immediately

## Quick Start

### Run the Example

```bash
cd examples/memory
go run .
```

The server will start on `http://localhost:8080`.

### Test It

```bash
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
    "emails": [{
      "value": "john.doe@example.com",
      "primary": true
    }]
  }'

# List users
curl -u admin:secret http://localhost:8080/memory/Users

# Get a specific user
curl -u admin:secret http://localhost:8080/memory/Users/{id}
```

## Files

- **`main.go`** - Complete runnable example showing gateway setup
- **`plugin.go`** - In-memory plugin implementation
- **`plugin_test.go`** - Tests demonstrating plugin functionality

## Implementation Details

### Thread Safety

The plugin uses `sync.RWMutex` to protect concurrent access to in-memory maps:
- Read operations (Get) use `RLock()` for concurrent reads
- Write operations (Create, Update, Delete) use `Lock()` for exclusive writes

### SCIM Compliance

The plugin follows the **adapter pattern**:
- `GetUsers()`/`GetGroups()` return **all data** - filtering/pagination happens in the adapter layer
- `GetUser()`/`GetGroup()` receive an `attributes` parameter for optimization hints (unused in this implementation)
- Patch operations are delegated to `scim.PatchProcessor`

This design keeps the plugin simple while the framework handles SCIM protocol complexity.

### Resource Metadata

The plugin automatically manages SCIM metadata:
- **ID**: Generated using `github.com/google/uuid` if not provided
- **Created**: Set to current timestamp on creation
- **LastModified**: Updated on every modification
- **Version**: ETag formatted as `W/"<id>"` (creation) or `W/"<id>-<timestamp>"` (modifications)

## Building a Production Plugin

For production use, see:
- **[examples/postgres](../postgres/)** - PostgreSQL plugin with connection pooling, health checks, and graceful shutdown
- **[examples/sqlite](../sqlite/)** - SQLite plugin for single-instance deployments
- **[PLUGIN_DEVELOPMENT.md](../../PLUGIN_DEVELOPMENT.md)** - Complete plugin development guide

### Key Differences for Production

A production plugin should:
1. ✅ Use a persistent database (PostgreSQL, MySQL, MongoDB, etc.)
2. ✅ Implement proper error handling with structured logging
3. ✅ Support connection pooling and health checks
4. ✅ Optimize queries using the `attributes` and `QueryParams` hints
5. ✅ Handle database migrations and schema versioning
6. ✅ Implement audit logging for compliance
7. ✅ Support graceful shutdown and connection cleanup
8. ✅ Use prepared statements to prevent SQL injection
9. ✅ Implement retry logic for transient failures
10. ✅ Monitor performance with metrics

## Testing

Run tests:
```bash
cd examples/memory
go test -v
```

The test suite demonstrates:
- Attribute selection (`attributes` parameter)
- Attribute exclusion (`excludedAttributes` parameter)
- Integration with the adapter layer

## License

Same as parent project - see [LICENSE](../../LICENSE)
