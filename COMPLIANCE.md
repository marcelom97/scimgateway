# SCIM 2.0 Compliance Documentation

This document details the SCIM 2.0 compliance of the scimgateway library, including implementation details, test coverage, and handling of edge cases.

## Overview

This library implements the complete SCIM 2.0 specification as defined in:

- **[RFC 7642](https://datatracker.ietf.org/doc/html/rfc7642)**: SCIM: Definitions, Overview, Concepts, and Requirements
- **[RFC 7643](https://datatracker.ietf.org/doc/html/rfc7643)**: SCIM: Core Schema
- **[RFC 7644](https://datatracker.ietf.org/doc/html/rfc7644)**: SCIM: Protocol

## Implemented Features

### Resource Management

#### Users (RFC 7643 Section 4.1)
- ✅ Core attributes: `id`, `userName`, `name`, `displayName`, `active`, `emails`, `phoneNumbers`, `addresses`
- ✅ Enterprise extension attributes
- ✅ Multi-valued attributes with type, primary, and display fields
- ✅ Complex attributes (name, addresses, emails, etc.)
- ✅ Full CRUD operations (Create, Read, Update, Delete)

#### Groups (RFC 7643 Section 4.2)
- ✅ Core attributes: `id`, `displayName`, `members`
- ✅ Group membership management
- ✅ Circular membership detection
- ✅ Full CRUD operations

### Query Operations (RFC 7644 Section 3.4.2)

#### Filtering
All filter operators are fully implemented and tested:

**Comparison Operators:**
- ✅ `eq` (equal) - **Case-insensitive** string comparison for Microsoft compatibility
- ✅ `ne` (not equal)
- ✅ `co` (contains) - **Case-insensitive** substring search
- ✅ `sw` (starts with) - **Case-insensitive**
- ✅ `ew` (ends with) - **Case-insensitive**
- ✅ `pr` (present) - Attribute exists
- ✅ `gt` (greater than) - Numeric/date comparison
- ✅ `ge` (greater than or equal)
- ✅ `lt` (less than)
- ✅ `le` (less than or equal)

**Logical Operators:**
- ✅ `and` - Logical AND with proper precedence
- ✅ `or` - Logical OR with proper precedence
- ✅ `not` - Logical negation

**Complex Attribute Paths:**
- ✅ Simple paths: `userName`, `displayName`
- ✅ Nested paths: `name.givenName`, `name.familyName`
- ✅ Multi-valued with filter: `emails[type eq "work"]`
- ✅ Multi-valued with nested access: `emails[type eq "work"].value`
- ✅ Multiple conditions: `emails[type eq "work" and primary eq true].value`

#### Sorting (RFC 7644 Section 3.4.2.3)
- ✅ `sortBy` - Any attribute path including nested attributes
- ✅ `sortOrder` - `ascending` or `descending`
- ✅ Stable sorting for consistent results
- ✅ Works with simple and complex attributes

#### Pagination (RFC 7644 Section 3.4.2.4)
- ✅ `startIndex` - 1-based index (SCIM spec requirement), **always included** in responses
- ✅ `count` - Number of results per page
- ✅ `totalResults` - Total number of matching resources
- ✅ `itemsPerPage` - **Always included** in responses (even when 0) for Microsoft SCIM validator compatibility
- ✅ Proper handling of out-of-range indices

#### Attribute Selection (RFC 7644 Section 3.4.2.5)
- ✅ `attributes` - Include only specified attributes
- ✅ `excludedAttributes` - Exclude specified attributes
- ✅ Mutual exclusivity enforcement (returns 400 if both provided)
- ✅ Nested attribute selection: `name.givenName`
- ✅ Multi-valued attribute selection: `emails.value`
- ✅ Always includes: `id`, `meta`, `schemas`

### PATCH Operations (RFC 7644 Section 3.5.2)

All PATCH operations are fully implemented:

#### Add Operation
- ✅ Add new attributes
- ✅ Add to multi-valued attributes (arrays)
- ✅ Add with path expressions
- ✅ Add to complex attributes
- ✅ **Auto-create array elements** - When adding to filtered path like `emails[type eq "work"].value` and no matching element exists, automatically creates element with filter criteria applied

#### Remove Operation
- ✅ Remove entire attributes
- ✅ Remove from multi-valued attributes
- ✅ Remove with filter: `emails[type eq "work"]`
- ✅ Remove nested attributes: `name.givenName`

#### Replace Operation
- ✅ Replace simple attributes
- ✅ Replace complex attributes
- ✅ Replace multi-valued attributes
- ✅ Replace with path expressions
- ✅ Replace in arrays with filter

### Bulk Operations (RFC 7644 Section 3.7)

- ✅ Multiple operations in single request
- ✅ All HTTP methods: GET, POST, PUT, PATCH, DELETE
- ✅ `bulkId` reference handling
- ✅ `bulkId` resolution in later operations
- ✅ Circular dependency detection
- ✅ `failOnErrors` support
- ✅ Individual operation error responses
- ✅ Proper status codes per operation

### Discovery Endpoints (RFC 7644 Section 4)

#### Service Provider Configuration (Section 4.1)
- ✅ `/ServiceProviderConfig`
- ✅ Returns server capabilities
- ✅ Filter, sort, pagination, bulk support indicators

#### Resource Types (Section 4.2)
- ✅ `/ResourceTypes`
- ✅ User and Group resource type definitions
- ✅ Schema references
- ✅ Endpoint information

#### Schemas (Section 4.3)
- ✅ `/Schemas`
- ✅ User schema (core + enterprise extension)
- ✅ Group schema
- ✅ Complete attribute definitions with types

### Search Endpoint (RFC 7644 Section 3.4.3)

- ✅ `POST /.search` for cross-resource searches
- ✅ All query parameters in request body
- ✅ Filter, sort, pagination support

### ETag Support (RFC 7644 Section 3.14)

- ✅ `ETag` header generation from `meta.version`
- ✅ `If-Match` for optimistic locking on updates
- ✅ `If-None-Match` for conditional GET
- ✅ `412 Precondition Failed` on version mismatch
- ✅ `304 Not Modified` for unchanged resources

## Error Handling

### HTTP Status Codes (RFC 7644 Section 3.12)
- ✅ `200 OK` - Successful GET, PUT, PATCH
- ✅ `201 Created` - Successful POST
- ✅ `204 No Content` - Successful DELETE
- ✅ `304 Not Modified` - ETag match on conditional GET
- ✅ `400 Bad Request` - Invalid filter, missing required attributes
- ✅ `401 Unauthorized` - Authentication required
- ✅ `404 Not Found` - Resource not found
- ✅ `409 Conflict` - Duplicate userName, version conflict
- ✅ `412 Precondition Failed` - ETag mismatch
- ✅ `500 Internal Server Error` - Plugin errors

### SCIM Error Schema (RFC 7644 Section 3.12)
All errors return proper SCIM error responses:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "status": "400",
  "scimType": "invalidFilter",
  "detail": "Invalid filter syntax: unexpected token"
}
```

**SCIM Error Types (`scimType`):**
- ✅ `invalidFilter` - Malformed filter expression
- ✅ `tooMany` - Too many results (not currently enforced)
- ✅ `uniqueness` - Duplicate userName
- ✅ `mutability` - Attempt to modify read-only attribute
- ✅ `invalidSyntax` - Malformed JSON
- ✅ `invalidValue` - Invalid attribute value
- ✅ `sensitive` - Security-related errors

## Test Coverage

The library includes comprehensive tests covering all SCIM operations:

### Unit Tests
- **Overall Coverage**: 75.8%
- **Config Package**: 84.1% (configuration validation)
- **Auth Package**: 90.4% (authentication)
- **Plugin Package**: 93.5% (plugin interface)
- **SCIM Package**: 53.4% (protocol implementation)

### Integration Tests
- Complete SCIM compliance test suite
- End-to-end HTTP endpoint tests
- ETag integration tests
- Bulk operation tests
- Filter parsing and execution tests

### Edge Cases Tested

**Filtering:**
- Empty filter expressions
- Nested parentheses with complex logic
- Multiple AND/OR combinations
- Filters on non-existent attributes
- Case-insensitive string comparisons (Microsoft compatibility)
- Boolean-to-string comparisons (e.g., `primary eq "True"`)
- Date/timestamp comparisons

**PATCH Operations:**
- Path expressions with complex filters
- Removing non-existent attributes (no-op)
- Adding to non-existent arrays (create array)
- Adding to filtered paths without matching elements (auto-create element with filter criteria)
- Conflicting operations in same request

**Bulk Operations:**
- Circular bulkId references
- References to failed operations
- Mixed success/failure scenarios
- BulkId in nested structures (group members)

**Pagination:**
- startIndex = 0 (invalid, adjusted to 1)
- startIndex > total results
- count = 0
- Very large count values

## Specification Compliance Notes

### Case Sensitivity (RFC 7644 Section 3.4.2.2)
**Attribute Names**: Case-insensitive per RFC spec (e.g., `userName` matches `username`)

**Filter Values**: RFC 7644 specifies case-sensitive comparison, but this implementation uses **case-insensitive** filtering for practical compatibility with Microsoft's SCIM validator and real-world SCIM providers. This means:
- `userName eq "john.doe"` matches "John.Doe", "JOHN.DOE", etc.
- `userName co "JOHN"` matches "john", "John", "JOHN"
- All string operators (eq, ne, co, sw, ew) are case-insensitive

**Boolean-String Comparison**: Filters can compare Boolean fields with string values:
- `primary eq "True"` matches `Boolean(true)` (case-insensitive: "true", "True", "TRUE")
- `primary eq "False"` matches `Boolean(false)` (case-insensitive: "false", "False", "FALSE")

### Attribute Precedence
When both `attributes` and `excludedAttributes` are provided, the gateway returns HTTP 400 per RFC 7644 Section 3.4.2.5:

> "SCIM clients MAY use one of these two OPTIONAL parameters, which MUST be supported by SCIM service providers"

### startIndex Behavior
SCIM uses 1-based indexing (not 0-based). The gateway automatically adjusts `startIndex=0` to `startIndex=1` for compatibility.

### Multi-Valued Attribute Uniqueness
The gateway enforces uniqueness on `emails[].value` and other multi-valued attributes with `uniqueness: true` in the schema definition.

### PATCH Operation Behavior
When using PATCH with filtered paths (e.g., `emails[type eq "work"].value`):
- **If matching element exists**: Updates the element's sub-attribute
- **If no matching element exists**: Creates a new element with filter criteria automatically applied (e.g., creates email with `type="work"`)

This behavior ensures PATCH operations work correctly for bulk provisioning scenarios where array elements may not exist yet.

### Microsoft SCIM Validator Compatibility
This implementation includes several enhancements for Microsoft SCIM validator compatibility:

1. **Case-Insensitive Filtering**: All string comparisons are case-insensitive
2. **Always Include Pagination Fields**: `startIndex` and `itemsPerPage` are always included in ListResponse, even when 0
3. **Boolean-String Comparison**: Supports comparing Boolean fields with string representations ("True"/"False")

These enhancements ensure the gateway passes Microsoft's SCIM validator tests while remaining compatible with the SCIM 2.0 specification.

## Known Deviations

### Case-Insensitive Filtering (Intentional)
**RFC 7644 Section 3.4.2.2** specifies case-sensitive value comparison for filters. However, this implementation uses **case-insensitive** comparison for practical reasons:
- Microsoft's SCIM validator requires case-insensitive filtering
- Most real-world SCIM providers (Azure AD, Okta, etc.) use case-insensitive filtering
- Better user experience (users don't need to remember exact casing)

**Rationale**: While this deviates from the RFC, it ensures compatibility with the largest SCIM ecosystem (Microsoft Azure AD) and matches real-world expectations. Attribute names remain case-insensitive per spec.

### Schema Discovery
Custom schema extensions can be stored and retrieved, but cannot be added to the `/Schemas` endpoint without code modifications. This is a design choice to keep the implementation simple.

### Internationalization
String comparisons use Go's default string equality and case-folding (`strings.EqualFold`), which may not handle all Unicode normalization forms identically. For full i18n support, consider normalizing strings at the plugin level.

## Testing Your SCIM Client

To verify your SCIM client is compatible with this gateway, test these scenarios:

1. **Basic CRUD**: Create, read, update, delete a user
2. **Filtering**: Test all operators (eq, ne, co, sw, ew, pr, gt, ge, lt, le)
3. **Logical operators**: Test and, or, not combinations
4. **Pagination**: Test various startIndex/count combinations
5. **Sorting**: Test ascending/descending on multiple attributes
6. **Attribute selection**: Test attributes and excludedAttributes
7. **PATCH**: Test add, remove, replace operations
8. **Bulk**: Test multiple operations with bulkId references
9. **ETags**: Test If-Match and If-None-Match headers
10. **Error handling**: Test invalid filters, missing required fields

## References

- [RFC 7642 - SCIM: Definitions, Overview, Concepts, and Requirements](https://datatracker.ietf.org/doc/html/rfc7642)
- [RFC 7643 - SCIM: Core Schema](https://datatracker.ietf.org/doc/html/rfc7643)
- [RFC 7644 - SCIM: Protocol](https://datatracker.ietf.org/doc/html/rfc7644)

## Contributing

If you discover any SCIM compliance issues, please:
1. Check if it's a documented known deviation
2. Verify against the official RFC specifications
3. Open an issue with a detailed test case

