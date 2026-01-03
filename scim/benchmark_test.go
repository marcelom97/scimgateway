package scim

import (
	"fmt"
	"testing"
)

// Benchmark helper: create test users
func createBenchUsers(n int) []*User {
	users := make([]*User, n)
	for i := 0; i < n; i++ {
		users[i] = &User{
			ID:       fmt.Sprintf("user%d", i),
			UserName: fmt.Sprintf("user%d@example.com", i),
			Active:   Bool(i%2 == 0), // Alternate active/inactive
			Name: &Name{
				GivenName:  fmt.Sprintf("User%d", i),
				FamilyName: "Testson",
			},
		}
	}
	return users
}

// createBenchGroups creates test groups for benchmarks
func createBenchGroups(n int) []*Group {
	groups := make([]*Group, n)
	for i := 0; i < n; i++ {
		groups[i] = &Group{
			ID:          fmt.Sprintf("group%d", i),
			DisplayName: fmt.Sprintf("Group %d", i),
		}
	}
	return groups
}

// ============================================================================
// Filter Parsing Benchmarks
// ============================================================================

func BenchmarkFilterParsing_Simple(b *testing.B) {
	filter := `userName eq "john@example.com"`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewFilterParser(filter)
		_, err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFilterParsing_Complex(b *testing.B) {
	filter := `userName eq "john" and (emails.type eq "work" or emails.type eq "home")`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewFilterParser(filter)
		_, err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFilterParsing_Nested(b *testing.B) {
	filter := `(userName eq "john" or userName eq "jane") and (active eq true and (emails.type eq "work" or emails.type eq "home"))`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewFilterParser(filter)
		_, err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFilterParsing_AttributePath(b *testing.B) {
	filter := `emails[type eq "work"].value eq "john@work.com"`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewFilterParser(filter)
		_, err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Filtering Benchmarks (by dataset size)
// ============================================================================

func BenchmarkFiltering_100Users_Simple(b *testing.B) {
	users := createBenchUsers(100)
	filter := `active eq true`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FilterByFilter(users, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFiltering_1000Users_Simple(b *testing.B) {
	users := createBenchUsers(1000)
	filter := `active eq true`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FilterByFilter(users, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFiltering_10000Users_Simple(b *testing.B) {
	users := createBenchUsers(10000)
	filter := `active eq true`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FilterByFilter(users, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFiltering_1000Users_Complex(b *testing.B) {
	users := createBenchUsers(1000)
	filter := `userName sw "user" and active eq true`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FilterByFilter(users, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Pagination Benchmarks
// ============================================================================

func BenchmarkPagination_SmallDataset(b *testing.B) {
	users := createBenchUsers(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyResourcePagination(users, 1, 10)
	}
}

func BenchmarkPagination_LargeDataset(b *testing.B) {
	users := createBenchUsers(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyResourcePagination(users, 1, 100)
	}
}

func BenchmarkPagination_MiddleOfLargeDataset(b *testing.B) {
	users := createBenchUsers(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyResourcePagination(users, 5000, 100)
	}
}

// ============================================================================
// Sorting Benchmarks
// ============================================================================

func BenchmarkSorting_100Users_SingleField(b *testing.B) {
	users := createBenchUsers(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortResources(users, "userName", "ascending")
	}
}

func BenchmarkSorting_1000Users_SingleField(b *testing.B) {
	users := createBenchUsers(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortResources(users, "userName", "ascending")
	}
}

func BenchmarkSorting_10000Users_SingleField(b *testing.B) {
	users := createBenchUsers(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortResources(users, "userName", "ascending")
	}
}

func BenchmarkSorting_1000Users_NestedField(b *testing.B) {
	users := createBenchUsers(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortResources(users, "name.givenName", "ascending")
	}
}

// ============================================================================
// PATCH Operation Benchmarks
// ============================================================================

func BenchmarkPatchOperation_SimpleReplace(b *testing.B) {
	pp := NewPatchProcessor()
	user := &User{
		ID:       "user1",
		UserName: "john",
		Active:   Bool(true),
	}

	patch := &PatchOp{
		Operations: []PatchOperation{
			{Op: "replace", Path: "active", Value: false},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userCopy := *user
		err := pp.ApplyPatch(&userCopy, patch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPatchOperation_MultipleReplace(b *testing.B) {
	pp := NewPatchProcessor()
	user := &User{
		ID:       "user1",
		UserName: "john",
		Active:   Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	patch := &PatchOp{
		Operations: []PatchOperation{
			{Op: "replace", Path: "active", Value: false},
			{Op: "replace", Path: "userName", Value: "jane"},
			{Op: "replace", Path: "name.givenName", Value: "Jane"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userCopy := *user
		err := pp.ApplyPatch(&userCopy, patch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPatchOperation_Add(b *testing.B) {
	pp := NewPatchProcessor()
	user := &User{
		ID:       "user1",
		UserName: "john",
	}

	patch := &PatchOp{
		Operations: []PatchOperation{
			{Op: "add", Value: map[string]any{"active": true, "displayName": "John Doe"}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userCopy := *user
		err := pp.ApplyPatch(&userCopy, patch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPatchOperation_Remove(b *testing.B) {
	pp := NewPatchProcessor()
	user := &User{
		ID:          "user1",
		UserName:    "john",
		DisplayName: "John Doe",
		Active:      Bool(true),
	}

	patch := &PatchOp{
		Operations: []PatchOperation{
			{Op: "remove", Path: "displayName"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userCopy := *user
		err := pp.ApplyPatch(&userCopy, patch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Attribute Selection Benchmarks
// ============================================================================

func BenchmarkAttributeSelection_Include(b *testing.B) {
	user := &User{
		ID:          "user1",
		UserName:    "john@example.com",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	attributes := []string{"id", "userName", "active"}
	selector := NewAttributeSelector(attributes, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.FilterResource(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAttributeSelection_Exclude(b *testing.B) {
	user := &User{
		ID:          "user1",
		UserName:    "john@example.com",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	excludedAttributes := []string{"name", "displayName"}
	selector := NewAttributeSelector(nil, excludedAttributes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.FilterResource(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAttributeSelection_NestedInclude(b *testing.B) {
	user := &User{
		ID:          "user1",
		UserName:    "john@example.com",
		DisplayName: "John Doe",
		Active:      Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	attributes := []string{"id", "userName", "name.givenName"}
	selector := NewAttributeSelector(attributes, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.FilterResource(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Full Query Processing Benchmarks (realistic scenarios)
// ============================================================================

func BenchmarkFullQuery_FilterSortPaginate_1000Users(b *testing.B) {
	users := createBenchUsers(1000)
	params := QueryParams{
		Filter:     `active eq true`,
		SortBy:     "userName",
		SortOrder:  "ascending",
		StartIndex: 1,
		Count:      50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessListQuery(users, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullQuery_ComplexFilter_1000Users(b *testing.B) {
	users := createBenchUsers(1000)
	params := QueryParams{
		Filter: `userName sw "user" and active eq true`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessListQuery(users, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullQuery_NoFilter_10000Users(b *testing.B) {
	users := createBenchUsers(10000)
	params := QueryParams{
		SortBy:     "userName",
		SortOrder:  "ascending",
		StartIndex: 1,
		Count:      100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessListQuery(users, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// ETag Generation Benchmarks
// ============================================================================

func BenchmarkETagGeneration(b *testing.B) {
	gen := NewETagGenerator()
	user := &User{
		ID:       "user1",
		UserName: "john@example.com",
		Active:   Bool(true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.Generate(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Validation Benchmarks
// ============================================================================

func BenchmarkValidation_User(b *testing.B) {
	validator := NewValidator()
	user := &User{
		UserName: "john@example.com",
		Active:   Bool(true),
		Name: &Name{
			GivenName:  "John",
			FamilyName: "Doe",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateUser(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidation_Group(b *testing.B) {
	validator := NewValidator()
	group := &Group{
		DisplayName: "Administrators",
		Members: []MemberRef{
			{Value: "user1"},
			{Value: "user2"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateGroup(group)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidation_PatchOp(b *testing.B) {
	validator := NewValidator()
	patch := &PatchOp{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		Operations: []PatchOperation{
			{Op: "replace", Path: "active", Value: false},
			{Op: "add", Value: map[string]any{"displayName": "John Doe"}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidatePatchOp(patch)
		if err != nil {
			b.Fatal(err)
		}
	}
}
