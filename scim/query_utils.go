package scim

import (
	"encoding/json"
)

// ApplyResourceFilter applies a SCIM filter expression to a slice of resources
// Returns filtered resources or an error if the filter is invalid
func ApplyResourceFilter[T any](resources []T, filter string) ([]T, error) {
	if filter == "" {
		return resources, nil
	}

	parser := NewFilterParser(filter)
	expr, err := parser.Parse()
	if err != nil {
		return nil, ErrInvalidFilter(err.Error())
	}

	if expr == nil {
		return resources, nil
	}

	result := make([]T, 0)
	for _, resource := range resources {
		if expr.Matches(resource) {
			result = append(result, resource)
		}
	}

	return result, nil
}

// ApplyResourcePagination applies SCIM pagination to a slice of resources
// Returns paginated slice, actual startIndex, and items per page
func ApplyResourcePagination[T any](resources []T, startIndex, count int) ([]T, int, int) {
	totalResults := len(resources)

	// Ensure startIndex is at least 1
	if startIndex < 1 {
		startIndex = 1
	}

	// If count is 0 or negative, return all results
	if count <= 0 {
		count = totalResults
	}

	// Calculate slice boundaries
	start := min(startIndex-1, totalResults)
	end := min(start+count, totalResults)

	paged := resources[start:end]

	return paged, startIndex, len(paged)
}

// ApplyAttributeSelection applies SCIM attribute selection to resources
// Returns resources with only requested attributes (or excluding specified attributes)
func ApplyAttributeSelection[T any](resources []T, attributes, excludedAttr []string) ([]T, error) {
	// If no attribute selection specified, return as-is
	if len(attributes) == 0 && len(excludedAttr) == 0 {
		return resources, nil
	}

	selector := NewAttributeSelector(attributes, excludedAttr)
	result := make([]T, len(resources))

	for i, resource := range resources {
		filtered, err := selector.FilterResource(resource)
		if err != nil {
			return nil, err
		}

		// Convert back to original type via JSON marshal/unmarshal
		data, err := json.Marshal(filtered)
		if err != nil {
			return nil, err
		}

		var filteredResource T
		if err := json.Unmarshal(data, &filteredResource); err != nil {
			return nil, err
		}

		result[i] = filteredResource
	}

	return result, nil
}

// ProcessListQuery is a convenience function that applies all SCIM query operations
// (filtering, pagination, attribute selection) to a list of resources
func ProcessListQuery[T any](allResources []T, params QueryParams) (*ListResponse[T], error) {
	// Apply filter if provided
	filtered, err := ApplyResourceFilter(allResources, params.Filter)
	if err != nil {
		return nil, err
	}

	totalResults := len(filtered)

	sorted := SortResources(filtered, params.SortBy, params.SortOrder)

	// Apply pagination
	paged, startIndex, itemsPerPage := ApplyResourcePagination(sorted, params.StartIndex, params.Count)

	// Apply attribute selection
	resources, err := ApplyAttributeSelection(paged, params.Attributes, params.ExcludedAttr)
	if err != nil {
		return nil, err
	}

	return &ListResponse[T]{
		Schemas:      []string{SchemaListResponse},
		TotalResults: totalResults,
		StartIndex:   startIndex,
		ItemsPerPage: itemsPerPage,
		Resources:    resources,
	}, nil
}
