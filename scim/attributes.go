package scim

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// AttributeSelector handles attribute selection and exclusion
type AttributeSelector struct {
	attributes            map[string]bool
	excluded              map[string]bool
	subAttributes         map[string][]string // parent -> list of sub-attributes to include
	excludedSubAttributes map[string][]string // parent -> list of sub-attributes to exclude
	includeAll            bool
	excludeAny            bool
}

// NewAttributeSelector creates a new attribute selector
func NewAttributeSelector(attributes, excluded []string) *AttributeSelector {
	as := &AttributeSelector{
		attributes:            make(map[string]bool),
		excluded:              make(map[string]bool),
		subAttributes:         make(map[string][]string),
		excludedSubAttributes: make(map[string][]string),
		includeAll:            len(attributes) == 0,
		excludeAny:            len(excluded) > 0,
	}

	for _, attr := range attributes {
		lowerAttr := strings.ToLower(attr)
		as.attributes[lowerAttr] = true

		// Parse sub-attributes (e.g., "emails.type" -> parent: "emails", sub: "type")
		// Supports arbitrary nesting levels (e.g., "name.formatted", "addresses.street.postalCode")
		if strings.Contains(lowerAttr, ".") {
			parts := strings.SplitN(lowerAttr, ".", 2)
			parent := parts[0]
			sub := parts[1]
			as.subAttributes[parent] = append(as.subAttributes[parent], sub)
		}
	}

	for _, attr := range excluded {
		lowerAttr := strings.ToLower(attr)
		as.excluded[lowerAttr] = true

		// Parse excluded sub-attributes (e.g., "name.familyName" -> parent: "name", sub: "familyName")
		// Supports arbitrary nesting levels
		if strings.Contains(lowerAttr, ".") {
			parts := strings.SplitN(lowerAttr, ".", 2)
			parent := parts[0]
			sub := parts[1]
			as.excludedSubAttributes[parent] = append(as.excludedSubAttributes[parent], sub)
		}
	}

	return as
}

// FilterResource filters a resource based on attribute selection
func (as *AttributeSelector) FilterResource(resource any) (any, error) {
	// If no filtering needed, return as-is
	if as.includeAll && !as.excludeAny {
		return resource, nil
	}

	// Convert to JSON and back to map for manipulation
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Always include these core attributes
	coreAttributes := map[string]bool{
		"id":      true,
		"schemas": true,
		"meta":    true,
	}

	// Filter attributes
	filtered := make(map[string]any)

	for key, value := range result {
		lowerKey := strings.ToLower(key)

		// Always include core attributes
		if coreAttributes[lowerKey] {
			filtered[key] = value
			continue
		}

		// Check if excluded
		if as.excluded[lowerKey] {
			continue
		}

		// If specific attributes requested, only include those
		if !as.includeAll {
			// Check if this attribute or its sub-attributes are requested
			if as.attributes[lowerKey] {
				filtered[key] = value
			} else if subs, hasSubAttrs := as.subAttributes[lowerKey]; hasSubAttrs {
				// Filter sub-attributes for complex/multi-valued attributes
				filteredValue := as.filterSubAttributes(value, subs)
				if filteredValue != nil {
					filtered[key] = filteredValue
				}
			}
		} else {
			// Include all except excluded
			// Check if this attribute has excluded sub-attributes
			if excludedSubs, hasExcludedSubs := as.excludedSubAttributes[lowerKey]; hasExcludedSubs {
				// Filter out excluded sub-attributes
				filteredValue := as.excludeSubAttributes(value, excludedSubs)
				if filteredValue != nil {
					filtered[key] = filteredValue
				}
			} else {
				filtered[key] = value
			}
		}
	}

	return filtered, nil
}

// FilterResources filters a list of resources
func (as *AttributeSelector) FilterResources(resources []any) ([]any, error) {
	if as.includeAll && !as.excludeAny {
		return resources, nil
	}

	filtered := make([]any, 0, len(resources))
	for _, resource := range resources {
		f, err := as.FilterResource(resource)
		if err != nil {
			return nil, err
		}
		filtered = append(filtered, f)
	}

	return filtered, nil
}

// filterSubAttributes filters a complex or multi-valued attribute to only include requested sub-attributes
// Supports arbitrary nesting levels (e.g., ["type"], ["street.postalCode"], etc.)
func (as *AttributeSelector) filterSubAttributes(value any, requestedSubs []string) any {
	if value == nil {
		return nil
	}

	// Group sub-attributes by their immediate child
	// e.g., ["type", "street.postalCode"] -> {"type": [], "street": ["postalCode"]}
	immediateChildren := make(map[string][]string)
	for _, sub := range requestedSubs {
		if strings.Contains(sub, ".") {
			// Nested sub-attribute (e.g., "street.postalCode")
			parts := strings.SplitN(sub, ".", 2)
			parent := strings.ToLower(parts[0])
			remainder := parts[1]
			immediateChildren[parent] = append(immediateChildren[parent], remainder)
		} else {
			// Direct sub-attribute (e.g., "type")
			immediateChildren[strings.ToLower(sub)] = []string{}
		}
	}

	// Handle multi-valued attributes (arrays)
	if arr, ok := value.([]any); ok {
		filtered := make([]any, 0, len(arr))
		for _, item := range arr {
			if itemMap, ok := item.(map[string]any); ok {
				filteredItem := as.filterMapBySubAttributes(itemMap, immediateChildren)
				if len(filteredItem) > 0 {
					filtered = append(filtered, filteredItem)
				}
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}

	// Handle single complex attributes (objects)
	if objMap, ok := value.(map[string]any); ok {
		filteredObj := as.filterMapBySubAttributes(objMap, immediateChildren)
		if len(filteredObj) > 0 {
			return filteredObj
		}
		return nil
	}

	// If it's neither an array nor an object, return as-is
	return value
}

// filterMapBySubAttributes filters a map based on requested sub-attributes
func (as *AttributeSelector) filterMapBySubAttributes(objMap map[string]any, requestedChildren map[string][]string) map[string]any {
	filteredObj := make(map[string]any)

	for k, v := range objMap {
		lowerK := strings.ToLower(k)
		if children, exists := requestedChildren[lowerK]; exists {
			if len(children) == 0 {
				// Direct match - include the whole attribute
				filteredObj[k] = v
			} else {
				// Nested attribute - recursively filter
				filtered := as.filterSubAttributes(v, children)
				if filtered != nil {
					filteredObj[k] = filtered
				}
			}
		}
	}

	return filteredObj
}

// excludeSubAttributes excludes specific sub-attributes from a complex or multi-valued attribute
// Supports arbitrary nesting levels (e.g., ["familyName"], ["street.postalCode"], etc.)
func (as *AttributeSelector) excludeSubAttributes(value any, excludedSubs []string) any {
	if value == nil {
		return nil
	}

	// Group excluded sub-attributes by their immediate child
	// e.g., ["familyName", "street.postalCode"] -> {"familyname": [], "street": ["postalCode"]}
	immediateExclusions := make(map[string][]string)
	for _, sub := range excludedSubs {
		if strings.Contains(sub, ".") {
			// Nested excluded sub-attribute (e.g., "street.postalCode")
			parts := strings.SplitN(sub, ".", 2)
			parent := strings.ToLower(parts[0])
			remainder := parts[1]
			immediateExclusions[parent] = append(immediateExclusions[parent], remainder)
		} else {
			// Direct excluded sub-attribute (e.g., "familyName")
			immediateExclusions[strings.ToLower(sub)] = []string{}
		}
	}

	// Handle multi-valued attributes (arrays)
	if arr, ok := value.([]any); ok {
		filtered := make([]any, 0, len(arr))
		for _, item := range arr {
			if itemMap, ok := item.(map[string]any); ok {
				filteredItem := as.excludeFromMap(itemMap, immediateExclusions)
				if len(filteredItem) > 0 {
					filtered = append(filtered, filteredItem)
				}
			} else {
				// If item is not a map, include as-is
				filtered = append(filtered, item)
			}
		}
		return filtered
	}

	// Handle single complex attributes (objects)
	if objMap, ok := value.(map[string]any); ok {
		return as.excludeFromMap(objMap, immediateExclusions)
	}

	// If it's neither an array nor an object, return as-is
	return value
}

// excludeFromMap excludes specific keys from a map based on exclusion rules
func (as *AttributeSelector) excludeFromMap(objMap map[string]any, exclusions map[string][]string) map[string]any {
	filteredObj := make(map[string]any)

	for k, v := range objMap {
		lowerK := strings.ToLower(k)
		if children, shouldExclude := exclusions[lowerK]; shouldExclude {
			if len(children) == 0 {
				// Direct exclusion - skip this attribute entirely
				continue
			} else {
				// Nested exclusion - recursively exclude sub-attributes
				filtered := as.excludeSubAttributes(v, children)
				if filtered != nil {
					filteredObj[k] = filtered
				}
			}
		} else {
			// Not in exclusion list - include as-is
			filteredObj[k] = v
		}
	}

	return filteredObj
}

// SortResources sorts resources based on sortBy and sortOrder.
// Pre-extracts attribute values once per resource for optimal performance,
// especially important for nested attributes that require JSON marshaling.
func SortResources[T any](resources []T, sortBy, sortOrder string) []T {
	if sortBy == "" || len(resources) == 0 {
		return resources
	}

	sorted := make([]T, len(resources))
	copy(sorted, resources)

	ascending := strings.ToLower(sortOrder) != "descending"

	type resourceValue struct {
		resource T
		value    any
	}
	pairs := make([]resourceValue, len(sorted))
	for i := range sorted {
		pairs[i] = resourceValue{
			resource: sorted[i],
			value:    getAttributeValue(sorted[i], sortBy),
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		cmp := compareForSort(pairs[i].value, pairs[j].value)
		if ascending {
			return cmp < 0
		}
		return cmp > 0
	})

	for i := range pairs {
		sorted[i] = pairs[i].resource
	}

	return sorted
}

// compareForSort compares two values for sorting.
//
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareForSort(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)
	if aIsStr && bIsStr {
		if aStr < bStr {
			return -1
		}
		if aStr > bStr {
			return 1
		}
		return 0
	}

	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum != nil && bNum != nil {
		if *aNum < *bNum {
			return -1
		}
		if *aNum > *bNum {
			return 1
		}
		return 0
	}

	aBool, aIsBool := a.(bool)
	bBool, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		if !aBool && bBool {
			return -1
		}
		if aBool && !bBool {
			return 1
		}
		return 0
	}

	aTime := toTime(a)
	bTime := toTime(b)
	if aTime != nil && bTime != nil {
		if aTime.Before(*bTime) {
			return -1
		}
		if aTime.After(*bTime) {
			return 1
		}
		return 0
	}

	return 0
}

// toTime converts a value to *time.Time if possible.
func toTime(v any) *time.Time {
	switch val := v.(type) {
	case time.Time:
		return &val
	case *time.Time:
		return val
	default:
		return nil
	}
}

// ApplyPagination applies pagination to resources
func ApplyPagination[T any](resources []T, startIndex, count int) ([]T, int, int) {
	total := len(resources)

	// Adjust startIndex (SCIM uses 1-based indexing)
	if startIndex < 1 {
		startIndex = 1
	}

	// Calculate array indices (0-based)
	start := startIndex - 1
	if start >= total {
		return []T{}, startIndex, 0
	}

	end := min(start+count, total)

	paged := resources[start:end]
	return paged, startIndex, len(paged)
}

// FilterByFilter applies a SCIM filter to resources
func FilterByFilter[T any](resources []T, filterStr string) ([]T, error) {
	if filterStr == "" {
		return resources, nil
	}

	parser := NewFilterParser(filterStr)
	filter, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return resources, nil
	}

	filtered := make([]T, 0)
	for _, resource := range resources {
		if filter.Matches(resource) {
			filtered = append(filtered, resource)
		}
	}

	return filtered, nil
}
