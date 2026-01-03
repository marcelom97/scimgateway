# Security Policy

This document describes security considerations for scimgateway and how to report vulnerabilities.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| 0.x     | :x: (use latest)   |

We recommend always using the latest version.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/marcelom97/scimgateway/security)
2. Click "Report a vulnerability"
3. Provide a detailed description

### What to Include

- Type of vulnerability
- Steps to reproduce
- Affected versions
- Potential impact
- Any suggested fixes (optional)

### Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 7 days
- **Fix timeline**: Depends on severity (critical: ASAP, high: 30 days, medium: 90 days)

## Security Features

### Authentication

scimgateway provides built-in authentication with security best practices:

**Constant-Time Comparison**

All credential comparisons use `crypto/subtle.ConstantTimeCompare` to prevent timing attacks:

```go
// Both basic auth and bearer token use constant-time comparison
subtle.ConstantTimeCompare([]byte(provided), []byte(expected))
```

**Generic Error Messages**

Authentication failures return generic messages to prevent user enumeration:
- Basic auth: "invalid credentials"
- Bearer token: "invalid token"

**Custom Authentication**

For advanced authentication (JWT, OAuth2, SAML), implement the `auth.Authenticator` interface. See `examples/jwt-auth/` for a secure JWT implementation.

### Operation Limits

Built-in limits prevent resource exhaustion:

| Limit | Default | Purpose |
|-------|---------|---------|
| Bulk MaxOperations | 1,000 | Prevents oversized bulk requests |
| Bulk MaxPayloadSize | 1 MB | Prevents memory exhaustion |
| Pagination Count | 1,000 max | Limits response size |

### Input Validation

scimgateway validates all inputs according to SCIM 2.0 specifications:

- **Schema validation**: Resources must include required schemas
- **Attribute validation**: Required attributes are enforced
- **Filter parsing**: Strict parsing with clear error messages
- **PATCH validation**: Operations validated before application

### Error Handling

Error responses follow SCIM 2.0 format without exposing internal details:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "status": "400",
  "scimType": "invalidFilter",
  "detail": "Invalid filter syntax at position 15"
}
```

Internal errors (stack traces, file paths) are never exposed to clients.

## SCIM-Specific Security Considerations

### Filter Complexity

Complex filter expressions can consume CPU resources. The filter parser handles this efficiently, but be aware:

- Deeply nested expressions (`(a and (b and (c and ...)))`) increase parsing time
- Complex filters on large datasets increase processing time
- Consider implementing filter support in your plugin for database-level optimization

**Mitigation**: The adapter handles filtering efficiently (see PERFORMANCE.md), but for very large datasets, implement `QueryParams` filtering in your plugin.

### Bulk Operations

Bulk operations can process up to 1,000 operations in a single request:

- Each operation is processed sequentially
- Circular `bulkId` references are detected and rejected
- Failed operations don't roll back successful ones (per SCIM spec)

**Mitigation**: The 1,000 operation limit is enforced. For stricter limits, validate in your plugin or middleware.

### Resource Size

Large resources (users with many group memberships, groups with many members) can impact:

- JSON parsing time
- Memory usage
- Response size

**Mitigation**: Implement pagination in your plugin for member lists. Use `attributes` parameter to request only needed fields.

## Plugin Developer Guidelines

When implementing a plugin, follow these security practices:

### Prevent SQL Injection

Always use parameterized queries:

```go
// CORRECT: Parameterized query
row := db.QueryRow("SELECT * FROM users WHERE id = $1", userID)

// WRONG: String concatenation (SQL injection risk!)
row := db.QueryRow("SELECT * FROM users WHERE id = '" + userID + "'")
```

### Validate Input

Validate all input before database operations:

```go
func (p *Plugin) GetUser(ctx context.Context, id string, attrs []string) (*scim.User, error) {
    // Validate ID format
    if !isValidUUID(id) {
        return nil, scim.ErrNotFound("User", id)
    }
    // ... proceed with database query
}
```

### Sanitize Errors

Don't expose internal details in error messages:

```go
// CORRECT: Generic error
if err != nil {
    logger.Error("database error", "error", err, "user_id", id)
    return nil, scim.ErrInternal("failed to retrieve user")
}

// WRONG: Exposes internal details
if err != nil {
    return nil, fmt.Errorf("PostgreSQL error: %v", err)
}
```

### Secure Logging

Never log sensitive data:

```go
// CORRECT: Log operation without sensitive data
logger.Info("user authenticated", "user_id", user.ID)

// WRONG: Logs credentials
logger.Info("login attempt", "password", password)
```

### Handle Authentication Context

Access authentication data securely from context:

```go
func (p *Plugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
    // Get authenticated user info (set by auth middleware)
    authInfo := ctx.Value("auth_info")
    
    // Use for audit logging, not for trust decisions
    logger.Info("user created", "created_by", authInfo, "user_id", user.ID)
    
    return user, nil
}
```

## Dependencies

scimgateway maintains minimal dependencies to reduce attack surface:

| Dependency | Purpose | Security Notes |
|------------|---------|----------------|
| `github.com/google/uuid` | UUID generation | Well-maintained, widely used |

No other external dependencies. All functionality is implemented using Go standard library.

## Security Checklist

Before deploying to production:

- [ ] Use HTTPS (TLS 1.2+)
- [ ] Configure strong authentication (bearer token or custom auth)
- [ ] Use strong, random tokens (32+ bytes, cryptographically random)
- [ ] Secure database credentials (environment variables, secret manager)
- [ ] Implement rate limiting (external load balancer or middleware)
- [ ] Enable audit logging
- [ ] Review plugin code for SQL injection
- [ ] Test error responses don't leak internal details
- [ ] Configure appropriate timeouts
- [ ] Keep scimgateway updated

## Changelog

Security-related changes are documented in [CHANGELOG.md](CHANGELOG.md) with the `security` label.
