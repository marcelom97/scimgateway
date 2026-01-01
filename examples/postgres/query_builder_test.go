package main

import (
	"testing"

	"github.com/marcelom97/scimgateway/scim"
)

func TestQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name       string
		table      string
		dataColumn string
		mapping    map[string]string
		params     scim.QueryParams
		wantSQL    string
		wantArgs   []any
	}{
		{
			name:       "simple select without params",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params:     scim.QueryParams{},
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC",
			wantArgs:   []any{},
		},
		{
			name:       "simple select for groups",
			table:      "groups",
			dataColumn: "data",
			mapping:    GroupAttributeMapping,
			params:     scim.QueryParams{},
			wantSQL:    "SELECT id, display_name, data, created_at, updated_at FROM groups ORDER BY created_at ASC",
			wantArgs:   []any{},
		},
		{
			name:       "filter by userName eq",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Filter: `userName eq "john"`,
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) = ? ORDER BY created_at ASC",
			wantArgs: []any{"john"},
		},
		{
			name:       "filter by displayName eq for groups",
			table:      "groups",
			dataColumn: "data",
			mapping:    GroupAttributeMapping,
			params: scim.QueryParams{
				Filter: `displayName eq "Admins"`,
			},
			wantSQL:  "SELECT id, display_name, data, created_at, updated_at FROM groups WHERE LOWER(display_name) = ? ORDER BY created_at ASC",
			wantArgs: []any{"admins"},
		},
		{
			name:       "filter with pagination",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Filter:     `userName eq "john"`,
				StartIndex: 11,
				Count:      10,
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) = ? ORDER BY created_at ASC LIMIT 10 OFFSET 10",
			wantArgs: []any{"john"},
		},
		{
			name:       "filter with sorting ascending",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Filter:    `active eq true`,
				SortBy:    "userName",
				SortOrder: "ascending",
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE data->>'active' = ? ORDER BY username ASC NULLS LAST",
			wantArgs: []any{"true"},
		},
		{
			name:       "filter with sorting descending",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				SortBy:    "userName",
				SortOrder: "descending",
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users ORDER BY username DESC NULLS LAST",
			wantArgs: []any{},
		},
		{
			name:       "pagination only with count",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Count: 25,
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 25",
			wantArgs: []any{},
		},
		{
			name:       "pagination with startIndex 1 (no offset)",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				StartIndex: 1,
				Count:      10,
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 10",
			wantArgs: []any{},
		},
		{
			name:       "full query with all params",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Filter:     `userName sw "john"`,
				SortBy:     "userName",
				SortOrder:  "ascending",
				StartIndex: 21,
				Count:      20,
			},
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY username ASC NULLS LAST LIMIT 20 OFFSET 20",
			wantArgs: []any{"john%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(tt.table, tt.dataColumn, tt.mapping)
			gotSQL, gotArgs := qb.Build(tt.params)

			if gotSQL != tt.wantSQL {
				t.Errorf("Build() SQL =\n%v\nwant:\n%v", gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Build() args count = %d, want %d", len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_BuildCount(t *testing.T) {
	tests := []struct {
		name       string
		table      string
		dataColumn string
		mapping    map[string]string
		params     scim.QueryParams
		wantSQL    string
		wantArgs   []any
	}{
		{
			name:       "count without filter",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params:     scim.QueryParams{},
			wantSQL:    "SELECT COUNT(*) FROM users",
			wantArgs:   []any{},
		},
		{
			name:       "count with filter",
			table:      "users",
			dataColumn: "data",
			mapping:    UserAttributeMapping,
			params: scim.QueryParams{
				Filter: `active eq true`,
			},
			wantSQL:  "SELECT COUNT(*) FROM users WHERE data->>'active' = ?",
			wantArgs: []any{"true"},
		},
		{
			name:       "count ignores pagination and sorting",
			table:      "groups",
			dataColumn: "data",
			mapping:    GroupAttributeMapping,
			params: scim.QueryParams{
				Filter:     `displayName co "admin"`,
				StartIndex: 10,
				Count:      5,
				SortBy:     "displayName",
			},
			wantSQL:  "SELECT COUNT(*) FROM groups WHERE LOWER(display_name) LIKE ?",
			wantArgs: []any{"%admin%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(tt.table, tt.dataColumn, tt.mapping)
			gotSQL, gotArgs := qb.BuildCount(tt.params)

			if gotSQL != tt.wantSQL {
				t.Errorf("BuildCount() SQL =\n%v\nwant:\n%v", gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("BuildCount() args count = %d, want %d", len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("BuildCount() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_FilterOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		// Equality operators
		{
			name:     "eq with string value",
			filter:   `userName eq "john.doe"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) = ? ORDER BY created_at ASC",
			wantArgs: []any{"john.doe"},
		},
		{
			name:     "eq with boolean true",
			filter:   `active eq true`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE data->>'active' = ? ORDER BY created_at ASC",
			wantArgs: []any{"true"},
		},
		{
			name:     "eq with boolean false",
			filter:   `active eq false`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE data->>'active' = ? ORDER BY created_at ASC",
			wantArgs: []any{"false"},
		},
		{
			name:     "ne with string value",
			filter:   `userName ne "admin"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) <> ? ORDER BY created_at ASC",
			wantArgs: []any{"admin"},
		},

		// String operators
		{
			name:     "co contains",
			filter:   `userName co "john"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%john%"},
		},
		{
			name:     "sw starts with",
			filter:   `userName sw "john"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"john%"},
		},
		{
			name:     "ew ends with",
			filter:   `userName ew "doe"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%doe"},
		},

		// Presence operator
		{
			name:     "pr present",
			filter:   `userName pr`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (username IS NOT NULL AND username <> '') ORDER BY created_at ASC",
			wantArgs: []any{},
		},

		// Comparison operators
		{
			name:     "gt greater than",
			filter:   `age gt 18`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (data->>'age')::numeric > ? ORDER BY created_at ASC",
			wantArgs: []any{"18"},
		},
		{
			name:     "ge greater than or equal",
			filter:   `age ge 21`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (data->>'age')::numeric >= ? ORDER BY created_at ASC",
			wantArgs: []any{"21"},
		},
		{
			name:     "lt less than",
			filter:   `age lt 65`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (data->>'age')::numeric < ? ORDER BY created_at ASC",
			wantArgs: []any{"65"},
		},
		{
			name:     "le less than or equal",
			filter:   `age le 30`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (data->>'age')::numeric <= ? ORDER BY created_at ASC",
			wantArgs: []any{"30"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Filter operator %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Filter operator %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Filter operator %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_LogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "AND operator",
			filter:   `userName eq "john" and active eq true`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (LOWER(username) = ? AND data->>'active' = ?) ORDER BY created_at ASC",
			wantArgs: []any{"john", "true"},
		},
		{
			name:     "OR operator",
			filter:   `userName eq "john" or userName eq "jane"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (LOWER(username) = ? OR LOWER(username) = ?) ORDER BY created_at ASC",
			wantArgs: []any{"john", "jane"},
		},
		{
			name:     "NOT operator",
			filter:   `not userName eq "admin"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE NOT (LOWER(username) = ?) ORDER BY created_at ASC",
			wantArgs: []any{"admin"},
		},
		{
			name:     "complex AND OR combination",
			filter:   `userName eq "john" and (active eq true or role eq "admin")`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (LOWER(username) = ? AND ((data->>'active' = ? OR LOWER(data->>'role') = ?))) ORDER BY created_at ASC",
			wantArgs: []any{"john", "true", "admin"},
		},
		{
			name:     "multiple AND",
			filter:   `userName eq "john" and active eq true and verified eq true`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE ((LOWER(username) = ? AND data->>'active' = ?) AND data->>'verified' = ?) ORDER BY created_at ASC",
			wantArgs: []any{"john", "true", "true"},
		},
		{
			name:     "grouped expression",
			filter:   `(userName eq "john")`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (LOWER(username) = ?) ORDER BY created_at ASC",
			wantArgs: []any{"john"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Logical operator %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Logical operator %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Logical operator %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_NestedAttributes(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "single level nested attribute",
			filter:   `name.givenName eq "John"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(data->'name'->>'givenName') = ? ORDER BY created_at ASC",
			wantArgs: []any{"john"},
		},
		{
			name:     "two level nested attribute",
			filter:   `name.familyName eq "Doe"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(data->'name'->>'familyName') = ? ORDER BY created_at ASC",
			wantArgs: []any{"doe"},
		},
		{
			name:     "three level nested attribute",
			filter:   `enterprise.manager.displayName eq "Boss"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(data->'enterprise'->'manager'->>'displayName') = ? ORDER BY created_at ASC",
			wantArgs: []any{"boss"},
		},
		{
			name:     "nested attribute with sw operator",
			filter:   `name.givenName sw "Jo"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(data->'name'->>'givenName') LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"jo%"},
		},
		{
			name:     "nested attribute presence check",
			filter:   `name.givenName pr`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE (data->'name'->>'givenName' IS NOT NULL AND data->'name'->>'givenName' <> '') ORDER BY created_at ASC",
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Nested attribute %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Nested attribute %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Nested attribute %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_SortingNestedAttributes(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantSQL   string
	}{
		{
			name:      "sort by direct column",
			sortBy:    "userName",
			sortOrder: "ascending",
			wantSQL:   "SELECT id, username, data, created_at, updated_at FROM users ORDER BY username ASC NULLS LAST",
		},
		{
			name:      "sort by nested attribute ascending",
			sortBy:    "name.familyName",
			sortOrder: "ascending",
			wantSQL:   "SELECT id, username, data, created_at, updated_at FROM users ORDER BY data->'name'->>'familyName' ASC NULLS LAST",
		},
		{
			name:      "sort by nested attribute descending",
			sortBy:    "name.givenName",
			sortOrder: "descending",
			wantSQL:   "SELECT id, username, data, created_at, updated_at FROM users ORDER BY data->'name'->>'givenName' DESC NULLS LAST",
		},
		{
			name:      "sort by JSONB field",
			sortBy:    "active",
			sortOrder: "ascending",
			wantSQL:   "SELECT id, username, data, created_at, updated_at FROM users ORDER BY data->>'active' ASC NULLS LAST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, _ := qb.Build(scim.QueryParams{
				SortBy:    tt.sortBy,
				SortOrder: tt.sortOrder,
			})

			if gotSQL != tt.wantSQL {
				t.Errorf("Sorting %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}
		})
	}
}

func TestQueryBuilder_PaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		startIndex int
		count      int
		wantSQL    string
	}{
		{
			name:       "startIndex 0 treated as no offset",
			startIndex: 0,
			count:      10,
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 10",
		},
		{
			name:       "startIndex 1 no offset",
			startIndex: 1,
			count:      10,
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 10",
		},
		{
			name:       "startIndex 2 offset 1",
			startIndex: 2,
			count:      10,
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 10 OFFSET 1",
		},
		{
			name:       "large pagination",
			startIndex: 1001,
			count:      100,
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 100 OFFSET 1000",
		},
		{
			name:       "only offset no limit",
			startIndex: 50,
			count:      0,
			wantSQL:    "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC OFFSET 49",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, _ := qb.Build(scim.QueryParams{
				StartIndex: tt.startIndex,
				Count:      tt.count,
			})

			if gotSQL != tt.wantSQL {
				t.Errorf("Pagination %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}
		})
	}
}

func TestQueryBuilder_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "LIKE pattern with percent",
			filter:   `userName co "100%"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%100\\%%"},
		},
		{
			name:     "LIKE pattern with underscore",
			filter:   `userName co "user_name"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%user\\_name%"},
		},
		{
			name:     "LIKE pattern with backslash",
			filter:   `userName co "path\\file"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(username) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%path\\\\\\\\file%"}, // Double escaping: filter parser + LIKE escape
		},
		{
			name:     "string with spaces",
			filter:   `displayName eq "John Doe"`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE LOWER(data->>'displayName') = ? ORDER BY created_at ASC",
			wantArgs: []any{"john doe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Special chars %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Special chars %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Special chars %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestQueryBuilder_InvalidFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantSQL string
	}{
		{
			name:    "invalid filter syntax",
			filter:  "userName eq",
			wantSQL: "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC",
		},
		{
			name:    "unclosed quote",
			filter:  `userName eq "john`,
			wantSQL: "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC",
		},
		{
			name:    "unknown operator",
			filter:  "userName xyz value",
			wantSQL: "SELECT id, username, data, created_at, updated_at FROM users ORDER BY created_at ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, _ := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Invalid filter %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}
		})
	}
}

func TestQueryBuilder_NullHandling(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "eq null",
			filter:   `middleName eq null`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE data->>'middleName' IS NULL ORDER BY created_at ASC",
			wantArgs: []any{},
		},
		{
			name:     "ne null",
			filter:   `middleName ne null`,
			wantSQL:  "SELECT id, username, data, created_at, updated_at FROM users WHERE data->>'middleName' IS NOT NULL ORDER BY created_at ASC",
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Null handling %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Null handling %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
			}
		})
	}
}

func TestQueryBuilder_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantArgs []any
	}{
		{
			name:     "uppercase value converted to lowercase",
			filter:   `userName eq "JOHN"`,
			wantArgs: []any{"john"},
		},
		{
			name:     "mixed case value converted to lowercase",
			filter:   `userName eq "John.Doe"`,
			wantArgs: []any{"john.doe"},
		},
		{
			name:     "contains with uppercase",
			filter:   `userName co "ADMIN"`,
			wantArgs: []any{"%admin%"},
		},
		{
			name:     "starts with mixed case",
			filter:   `userName sw "Super"`,
			wantArgs: []any{"super%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("users", "data", UserAttributeMapping)
			_, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Case insensitivity %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Case insensitivity %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"100%", "100\\%"},
		{"user_name", "user\\_name"},
		{"path\\file", "path\\\\file"},
		{"a%b_c\\d", "a\\%b\\_c\\\\d"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeLikePattern(tt.input)
			if got != tt.expected {
				t.Errorf("escapeLikePattern(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetSQLPath(t *testing.T) {
	qb := NewQueryBuilder("users", "data", UserAttributeMapping)

	tests := []struct {
		attrPath string
		expected string
	}{
		// Direct column mappings
		{"id", "id"},
		{"userName", "username"},
		{"username", "username"},

		// Single-level JSONB
		{"active", "data->>'active'"},
		{"displayName", "data->>'displayName'"},

		// Nested JSONB paths
		{"name.givenName", "data->'name'->>'givenName'"},
		{"name.familyName", "data->'name'->>'familyName'"},
		{"enterprise.manager.displayName", "data->'enterprise'->'manager'->>'displayName'"},
	}

	for _, tt := range tests {
		t.Run(tt.attrPath, func(t *testing.T) {
			got := qb.getSQLPath(tt.attrPath)
			if got != tt.expected {
				t.Errorf("getSQLPath(%q) = %q, want %q", tt.attrPath, got, tt.expected)
			}
		})
	}
}

func TestQueryBuilder_GroupsTable(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "filter by displayName",
			filter:   `displayName eq "Administrators"`,
			wantSQL:  "SELECT id, display_name, data, created_at, updated_at FROM groups WHERE LOWER(display_name) = ? ORDER BY created_at ASC",
			wantArgs: []any{"administrators"},
		},
		{
			name:     "filter by displayName contains",
			filter:   `displayName co "admin"`,
			wantSQL:  "SELECT id, display_name, data, created_at, updated_at FROM groups WHERE LOWER(display_name) LIKE ? ORDER BY created_at ASC",
			wantArgs: []any{"%admin%"},
		},
		{
			name:     "filter by nested members",
			filter:   `members.value eq "user123"`,
			wantSQL:  "SELECT id, display_name, data, created_at, updated_at FROM groups WHERE LOWER(data->'members'->>'value') = ? ORDER BY created_at ASC",
			wantArgs: []any{"user123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("groups", "data", GroupAttributeMapping)
			gotSQL, gotArgs := qb.Build(scim.QueryParams{Filter: tt.filter})

			if gotSQL != tt.wantSQL {
				t.Errorf("Groups %s:\nSQL =\n%v\nwant:\n%v", tt.name, gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Groups %s: args count = %d, want %d", tt.name, len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("Groups %s: args[%d] = %v, want %v", tt.name, i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}
