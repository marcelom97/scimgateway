# Performance

This document describes the performance characteristics of scimgateway operations.

## Benchmark Environment

- **Platform:** macOS (darwin/arm64)
- **CPU:** Apple M3
- **Go Version:** 1.25
- **Benchmark Tool:** `go test -bench`

Run benchmarks yourself:
```bash
go test ./scim -bench=Benchmark -benchmem
```

## Summary

| Operation | Latency | Notes |
|-----------|---------|-------|
| Filter Parsing | 75-500 ns | Depends on complexity |
| Pagination | < 1 ns | Zero allocations |
| PATCH (simple) | 1.4 μs | Single field replace |
| PATCH (complex) | 3 μs | Multiple operations |
| Attribute Selection | 2.2 μs | Per resource |
| ETag Generation | 1.7 μs | Per resource |
| Validation (User) | 1.3 μs | Full user validation |

## Filter Parsing

The filter parser converts SCIM filter expressions to an AST for evaluation.

| Filter Type | Example | Latency | Allocations |
|-------------|---------|---------|-------------|
| Simple | `userName eq "john"` | 75 ns | 2 |
| Attribute Path | `emails.value eq "x"` | 120 ns | 2 |
| Complex | `userName eq "x" and active eq true` | 291 ns | 9 |
| Nested | `(a eq "1" and b eq "2") or c eq "3"` | 508 ns | 16 |

**Guidance:**
- Filter parsing is fast and rarely a bottleneck
- Complex nested filters scale linearly with depth
- Parser is called once per request, then the AST is reused

## Filtering (Resource Matching)

Filtering applies parsed filters to resources. Performance scales with dataset size.

| Dataset Size | Simple Filter | Complex Filter | Memory |
|--------------|---------------|----------------|--------|
| 100 users | 112 μs | - | 43 KB |
| 1,000 users | 1.1 ms | 1.5 ms | 426 KB |
| 10,000 users | 11.3 ms | - | 4.3 MB |

**Guidance:**
- Linear scaling with dataset size (O(n))
- For datasets > 10K resources, consider plugin-side filtering
- Memory usage is proportional to matching results

### When to Optimize in Your Plugin

The adapter can handle filtering for you, but for large datasets, implement filtering in your plugin:

```go
func (p *MyPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
    // For small datasets (< 1000): return all, let adapter filter
    if p.estimatedUserCount() < 1000 {
        return p.getAllUsers()
    }
    
    // For large datasets: translate filter to backend query
    if params.Filter != "" {
        return p.queryWithFilter(params.Filter)
    }
    return p.getAllUsers()
}
```

## Sorting

Sorting is performed in-memory after filtering.

| Dataset Size | Simple Field | Nested Field | Memory |
|--------------|--------------|--------------|--------|
| 100 users | 36 μs | - | 15 KB |
| 1,000 users | 409 μs | 1.9 ms | 145 KB |
| 10,000 users | 4.4 ms | - | 1.4 MB |

**Guidance:**
- Simple fields (e.g., `userName`) are fastest
- Nested fields (e.g., `name.familyName`) require path traversal, ~5x slower
- For sorted queries on large datasets, consider database-level sorting

## Pagination

Pagination uses slice operations and is extremely fast.

| Operation | Latency | Allocations |
|-----------|---------|-------------|
| Small dataset (10 items) | < 1 ns | 0 |
| Large dataset (10K items) | < 1 ns | 0 |
| Middle of dataset | < 1 ns | 0 |

**Guidance:**
- Pagination is essentially free (zero allocations)
- Always use `startIndex` and `count` for large datasets
- The gateway handles 1-based indexing per SCIM spec

## PATCH Operations

PATCH applies modifications to resources.

| Operation | Example | Latency | Allocations |
|-----------|---------|---------|-------------|
| Simple Replace | `replace userName` | 1.4 μs | 22 |
| Multiple Replace | 3 operations | 3.0 μs | 48 |
| Add | `add emails` | 2.8 μs | 47 |
| Remove | `remove emails[type eq "work"]` | 555 ns | 10 |

**Guidance:**
- Remove is fastest (just deletion)
- Add/replace require value copying
- Batch multiple changes in one PATCH request for efficiency

## Attribute Selection

Attribute selection filters response fields.

| Operation | Latency | Allocations |
|-----------|---------|-------------|
| Include attributes | 2.2 μs | 38 |
| Exclude attributes | 2.2 μs | 38 |
| Nested include | 2.4 μs | 42 |

**Guidance:**
- Use `attributes` parameter to reduce response size
- Both include and exclude have similar performance
- Reduces network bandwidth for large resources

## Full Query Processing

Combined operations (filter + sort + paginate) for realistic scenarios.

| Scenario | Dataset | Latency | Memory |
|----------|---------|---------|--------|
| Filter + Sort + Paginate | 1,000 users | 1.4 ms | 499 KB |
| Complex Filter | 1,000 users | 1.5 ms | 539 KB |
| Sort + Paginate (no filter) | 10,000 users | 4.5 ms | 1.4 MB |

**Guidance:**
- Most queries complete in 1-5 ms for typical datasets
- Memory usage is dominated by result set size
- For production, monitor query latency and memory

## Validation & ETag

| Operation | Latency | Allocations |
|-----------|---------|-------------|
| User validation | 1.3 μs | 37 |
| Group validation | 3 ns | 0 |
| PatchOp validation | 20 ns | 0 |
| ETag generation | 1.7 μs | 32 |

**Guidance:**
- Validation is fast and always performed
- ETag generation uses content hashing
- Group validation is minimal (just displayName check)

## Memory Usage Patterns

### Per-Request Memory

| Request Type | Typical Memory |
|--------------|----------------|
| GET single resource | 2-5 KB |
| GET list (100 resources) | 50-100 KB |
| GET list (1000 resources) | 500 KB - 1 MB |
| PATCH operation | 2-5 KB |
| Bulk (10 operations) | 20-50 KB |

### Optimization Tips

1. **Use pagination** - Always set `count` for list operations
2. **Use attribute selection** - Request only needed fields
3. **Implement plugin-side filtering** - For datasets > 10K resources
4. **Batch changes** - Use bulk operations instead of many single requests

## Scaling Recommendations

### Small Scale (< 1,000 resources)
- Default adapter filtering is sufficient
- No special optimization needed
- Expected latency: < 5 ms per request

### Medium Scale (1,000 - 10,000 resources)
- Consider plugin-side filtering for common queries
- Use pagination consistently
- Expected latency: 5-50 ms per request

### Large Scale (> 10,000 resources)
- Implement plugin-side filtering (translate to SQL/NoSQL queries)
- Implement plugin-side sorting
- Use database indexes for filtered fields
- Consider caching for frequently accessed resources
- Expected latency: depends on backend optimization

## Running Benchmarks

```bash
# Run all benchmarks
go test ./scim -bench=Benchmark -benchmem

# Run specific benchmark
go test ./scim -bench=BenchmarkFilterParsing -benchmem

# Run with longer duration for stability
go test ./scim -bench=Benchmark -benchtime=2s -benchmem

# Save results to file
go test ./scim -bench=Benchmark -benchmem > benchmark_results.txt
```

## Comparing Performance

To compare performance across versions:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save
go test ./scim -bench=Benchmark -count=5 > old.txt

# After changes
go test ./scim -bench=Benchmark -count=5 > new.txt

# Compare
benchstat old.txt new.txt
```
