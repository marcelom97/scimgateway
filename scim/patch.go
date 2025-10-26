package scim

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// PatchProcessor processes SCIM PATCH operations
type PatchProcessor struct{}

// NewPatchProcessor creates a new patch processor
func NewPatchProcessor() *PatchProcessor {
	return &PatchProcessor{}
}

// ApplyPatch applies a PATCH operation to a resource
func (pp *PatchProcessor) ApplyPatch(resource any, patch *PatchOp) error {
	for _, op := range patch.Operations {
		if err := pp.applyOperation(resource, op); err != nil {
			return err
		}
	}
	return nil
}

// applyOperation applies a single patch operation
func (pp *PatchProcessor) applyOperation(resource any, op PatchOperation) error {
	switch strings.ToLower(op.Op) {
	case "add":
		return pp.applyAdd(resource, op)
	case "remove":
		return pp.applyRemove(resource, op)
	case "replace":
		return pp.applyReplace(resource, op)
	default:
		return ErrInvalidValue(fmt.Sprintf("invalid operation: %s", op.Op))
	}
}

// applyAdd applies an ADD operation
func (pp *PatchProcessor) applyAdd(resource any, op PatchOperation) error {
	if op.Path == "" {
		// Add attributes to resource root
		return pp.addToRoot(resource, op.Value)
	}

	// Parse path
	path := parsePath(op.Path)
	return pp.addToPath(resource, path, op.Value)
}

// applyRemove applies a REMOVE operation
func (pp *PatchProcessor) applyRemove(resource any, op PatchOperation) error {
	if op.Path == "" {
		return ErrNoTarget("path is required for remove operation")
	}

	path := parsePath(op.Path)
	return pp.removeFromPath(resource, path)
}

// applyReplace applies a REPLACE operation
func (pp *PatchProcessor) applyReplace(resource any, op PatchOperation) error {
	if op.Path == "" {
		// Replace entire resource
		return pp.replaceRoot(resource, op.Value)
	}

	path := parsePath(op.Path)
	return pp.replaceAtPath(resource, path, op.Value)
}

// addToRoot adds attributes to the resource root
func (pp *PatchProcessor) addToRoot(resource any, value any) error {
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("resource must be a struct")
	}

	// Convert value to map
	valueData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var valueMap map[string]any
	if err := json.Unmarshal(valueData, &valueMap); err != nil {
		return err
	}

	// Set each attribute
	for key, val := range valueMap {
		field := findField(v, key)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		if err := pp.setValue(field, val); err != nil {
			return err
		}
	}

	return nil
}

// replaceRoot replaces the entire resource
func (pp *PatchProcessor) replaceRoot(resource any, value any) error {
	return pp.addToRoot(resource, value)
}

// addToPath adds value to a specific path
func (pp *PatchProcessor) addToPath(resource any, path *Path, value any) error {
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Navigate to the target
	target := v
	var parentField reflect.Value // Track the field for map operations

	for i, segment := range path.Segments {
		if i == len(path.Segments)-1 {
			// Last segment - perform the add

			// Check if parent is a map (for extension attributes)
			if parentField.IsValid() && parentField.Kind() == reflect.Map {
				// Initialize map if nil
				if parentField.IsNil() {
					parentField.Set(reflect.MakeMap(parentField.Type()))
				}
				// Set value in map
				valueData, err := json.Marshal(value)
				if err != nil {
					return err
				}
				var val any
				if err := json.Unmarshal(valueData, &val); err != nil {
					return err
				}
				parentField.SetMapIndex(reflect.ValueOf(segment.Attribute), reflect.ValueOf(val))
				return nil
			}

			// Normal struct field access
			field := findField(target, segment.Attribute)
			if !field.IsValid() {
				return ErrNoTarget(fmt.Sprintf("attribute %s not found", segment.Attribute))
			}

			if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				// Add to array
				return pp.addToArray(field, value, segment.Filter)
			}

			if !field.CanSet() {
				return ErrMutability(fmt.Sprintf("attribute %s is not mutable", segment.Attribute))
			}

			return pp.setValue(field, value)
		}

		// Navigate deeper
		field := findField(target, segment.Attribute)
		if !field.IsValid() {
			return ErrNoTarget(fmt.Sprintf("attribute %s not found", segment.Attribute))
		}

		// Handle intermediate segments with filters (e.g., emails[type eq "work"].value)
		if segment.Filter != nil && (field.Kind() == reflect.Slice || field.Kind() == reflect.Array) {
			// Find the first matching element in the array
			matchFound := false
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if segment.Filter.Matches(elem.Interface()) {
					// Navigate into the matching element
					if elem.Kind() == reflect.Ptr {
						target = elem.Elem()
					} else {
						target = elem
					}
					matchFound = true
					break
				}
			}
			if !matchFound {
				return ErrNoTarget(fmt.Sprintf("no matching element found for filter in attribute %s", segment.Attribute))
			}
			parentField = reflect.Value{} // Clear parent field
		} else if field.Kind() == reflect.Map {
			// Next segment will access this map
			parentField = field
			// Don't update target for maps - we'll use parentField
		} else if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Create new instance
				field.Set(reflect.New(field.Type().Elem()))
			}
			target = field.Elem()
			parentField = reflect.Value{} // Clear parent field
		} else {
			target = field
			parentField = reflect.Value{} // Clear parent field
		}
	}

	return nil
}

// removeFromPath removes value from a specific path
func (pp *PatchProcessor) removeFromPath(resource any, path *Path) error {
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	target := v
	var parentField reflect.Value // Track the field for map operations

	for i, segment := range path.Segments {
		if i == len(path.Segments)-1 {
			// Last segment - perform the remove

			// Check if parent is a map (for extension attributes)
			if parentField.IsValid() && parentField.Kind() == reflect.Map {
				// Delete key from map
				parentField.SetMapIndex(reflect.ValueOf(segment.Attribute), reflect.Value{})
				return nil
			}

			// Normal struct field access
			field := findField(target, segment.Attribute)
			if !field.IsValid() {
				return nil // Attribute doesn't exist, nothing to remove
			}

			if segment.Filter != nil {
				// Remove from filtered array
				if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
					return pp.removeFromArray(field, segment.Filter)
				}
			}

			if !field.CanSet() {
				return ErrMutability(fmt.Sprintf("attribute %s is not mutable", segment.Attribute))
			}

			// Set to zero value
			field.Set(reflect.Zero(field.Type()))
			return nil
		}

		// Navigate deeper
		field := findField(target, segment.Attribute)
		if !field.IsValid() {
			return nil // Attribute doesn't exist, nothing to remove
		}

		// Handle intermediate segments with filters (e.g., emails[type eq "work"].value)
		if segment.Filter != nil && (field.Kind() == reflect.Slice || field.Kind() == reflect.Array) {
			// Find the first matching element in the array
			matchFound := false
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if segment.Filter.Matches(elem.Interface()) {
					// Navigate into the matching element
					if elem.Kind() == reflect.Ptr {
						target = elem.Elem()
					} else {
						target = elem
					}
					matchFound = true
					break
				}
			}
			if !matchFound {
				return nil // No matching element, nothing to remove
			}
			parentField = reflect.Value{} // Clear parent field
		} else if field.Kind() == reflect.Map {
			// Next segment will access this map
			parentField = field
			// Don't update target for maps - we'll use parentField
		} else if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				return nil
			}
			target = field.Elem()
			parentField = reflect.Value{} // Clear parent field
		} else {
			target = field
			parentField = reflect.Value{} // Clear parent field
		}
	}

	return nil
}

// replaceAtPath replaces value at a specific path
func (pp *PatchProcessor) replaceAtPath(resource any, path *Path, value any) error {
	return pp.addToPath(resource, path, value)
}

// setValue sets a value to a reflect.Value
func (pp *PatchProcessor) setValue(field reflect.Value, value any) error {
	// Convert value to the field's type
	valueData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	newValue := reflect.New(field.Type())
	if err := json.Unmarshal(valueData, newValue.Interface()); err != nil {
		return err
	}

	field.Set(newValue.Elem())
	return nil
}

// addToArray adds an element to an array
func (pp *PatchProcessor) addToArray(field reflect.Value, value any, filter *AttributeExpression) error {
	// Convert value to array element type
	elemType := field.Type().Elem()
	valueData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Handle both single values and arrays
	var values []any
	if err := json.Unmarshal(valueData, &values); err != nil {
		// Single value
		newElem := reflect.New(elemType)
		if err := json.Unmarshal(valueData, newElem.Interface()); err != nil {
			return err
		}
		field.Set(reflect.Append(field, newElem.Elem()))
	} else {
		// Array of values
		for _, v := range values {
			vData, _ := json.Marshal(v)
			newElem := reflect.New(elemType)
			if err := json.Unmarshal(vData, newElem.Interface()); err != nil {
				return err
			}
			field.Set(reflect.Append(field, newElem.Elem()))
		}
	}

	return nil
}

// removeFromArray removes elements from an array based on a filter
func (pp *PatchProcessor) removeFromArray(field reflect.Value, filter *AttributeExpression) error {
	newArray := reflect.MakeSlice(field.Type(), 0, field.Len())

	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		if !filter.Matches(elem.Interface()) {
			newArray = reflect.Append(newArray, elem)
		}
	}

	field.Set(newArray)
	return nil
}

// Path represents a parsed SCIM path
type Path struct {
	Segments []PathSegment
}

// PathSegment represents a segment of a path
type PathSegment struct {
	Attribute string
	Filter    *AttributeExpression
}

// parsePath parses a SCIM path expression
func parsePath(pathStr string) *Path {
	path := &Path{
		Segments: []PathSegment{},
	}

	// Parse different path types:
	// - Simple: emails[type eq "work"].value
	// - Nested: name.givenName
	// - Filtered: addresses[type eq "work"]
	// - Schema URN: urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber

	var parts []string

	// Check if this is a schema URN path
	if strings.HasPrefix(pathStr, "urn:") {
		// Schema URN paths have format: <schema-urn>:<attribute-path>
		// Example: urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber
		// The schema URN ends at ":User" or ":Group", then comes the attribute path

		var schemaURN string
		var attrPath string

		if idx := strings.Index(pathStr, ":User:"); idx != -1 {
			schemaURN = pathStr[:idx+5] // Include ":User"
			attrPath = pathStr[idx+6:]  // Skip ":User:"
		} else if idx := strings.Index(pathStr, ":Group:"); idx != -1 {
			schemaURN = pathStr[:idx+6] // Include ":Group"
			attrPath = pathStr[idx+7:]  // Skip ":Group:"
		} else {
			// Fallback: just the URN without attribute
			schemaURN = pathStr
			attrPath = ""
		}

		// First segment is the schema URN (maps to extension field)
		parts = append(parts, schemaURN)

		// If there's an attribute path, split it on dots
		if attrPath != "" {
			dotParts := strings.Split(attrPath, ".")
			parts = append(parts, dotParts...)
		}
	} else {
		// Normal path - split on dots
		parts = strings.Split(pathStr, ".")
	}

	// Process each part into path segments
	for _, part := range parts {
		segment := PathSegment{}

		// Check for filter
		if strings.Contains(part, "[") {
			openIdx := strings.Index(part, "[")
			closeIdx := strings.Index(part, "]")

			segment.Attribute = part[:openIdx]
			filterStr := part[openIdx+1 : closeIdx]

			// Parse filter
			parser := NewFilterParser(filterStr)
			filter, err := parser.Parse()
			if err == nil {
				if attrExpr, ok := filter.(*AttributeExpression); ok {
					segment.Filter = attrExpr
				}
			}
		} else {
			segment.Attribute = part
		}

		path.Segments = append(path.Segments, segment)
	}

	return path
}
