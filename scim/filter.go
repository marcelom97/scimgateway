package scim

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// FilterParser parses SCIM filter expressions
type FilterParser struct {
	input string
	pos   int
}

// Filter represents a parsed SCIM filter
type Filter interface {
	Matches(resource any) bool
}

// AttributeExpression represents an attribute comparison
type AttributeExpression struct {
	AttributePath string
	Operator      string
	Value         any
}

// LogicalExpression represents a logical operation (AND, OR, NOT)
type LogicalExpression struct {
	Operator string
	Left     Filter
	Right    Filter
}

// GroupExpression represents a grouped filter
type GroupExpression struct {
	Filter Filter
}

// NewFilterParser creates a new filter parser
func NewFilterParser(filter string) *FilterParser {
	return &FilterParser{
		input: strings.TrimSpace(filter),
		pos:   0,
	}
}

// Parse parses the filter expression
func (p *FilterParser) Parse() (Filter, error) {
	if p.input == "" {
		return nil, nil
	}
	return p.parseLogicalOr()
}

// parseLogicalOr parses OR expressions
func (p *FilterParser) parseLogicalOr() (Filter, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if !p.matchKeyword("or") {
			break
		}
		p.pos += 2
		p.skipWhitespace()

		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}

		left = &LogicalExpression{
			Operator: "or",
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// parseLogicalAnd parses AND expressions
func (p *FilterParser) parseLogicalAnd() (Filter, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if !p.matchKeyword("and") {
			break
		}
		p.pos += 3
		p.skipWhitespace()

		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}

		left = &LogicalExpression{
			Operator: "and",
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// parseNot parses NOT expressions
func (p *FilterParser) parseNot() (Filter, error) {
	p.skipWhitespace()
	if p.matchKeyword("not") {
		p.pos += 3
		p.skipWhitespace()

		filter, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		return &LogicalExpression{
			Operator: "not",
			Left:     filter,
		}, nil
	}

	return p.parsePrimary()
}

// parsePrimary parses primary expressions (attribute expressions or grouped expressions)
func (p *FilterParser) parsePrimary() (Filter, error) {
	p.skipWhitespace()

	// Check for grouped expression
	if p.peek() == '(' {
		p.pos++
		filter, err := p.parseLogicalOr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.peek() != ')' {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		p.pos++
		return &GroupExpression{Filter: filter}, nil
	}

	// Parse attribute expression
	return p.parseAttributeExpression()
}

// parseAttributeExpression parses an attribute comparison
func (p *FilterParser) parseAttributeExpression() (Filter, error) {
	p.skipWhitespace()

	// Parse attribute path
	attrPath := p.parseAttributePath()
	if attrPath == "" {
		return nil, fmt.Errorf("expected attribute path at position %d", p.pos)
	}

	p.skipWhitespace()

	// Parse operator
	op := p.parseOperator()
	if op == "" {
		return nil, fmt.Errorf("expected operator at position %d", p.pos)
	}

	p.skipWhitespace()

	var value any
	// pr (present) operator doesn't need a value
	if op != "pr" {
		// Parse value
		var err error
		value, err = p.parseValue()
		if err != nil {
			return nil, err
		}
	}

	return &AttributeExpression{
		AttributePath: attrPath,
		Operator:      op,
		Value:         value,
	}, nil
}

// parseAttributePath parses an attribute path
func (p *FilterParser) parseAttributePath() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if !isAlphaNumeric(ch) && ch != '.' && ch != '[' && ch != ']' && ch != '"' && ch != ' ' {
			break
		}
		// Handle complex paths like emails[type eq "work"].value
		if ch == ' ' && p.pos > start {
			// Check if we're in a bracket expression
			inBracket := false
			for i := start; i < p.pos; i++ {
				switch p.input[i] {
				case '[':
					inBracket = true
				case ']':
					inBracket = false
				}
			}
			if !inBracket {
				break
			}
		}
		p.pos++
	}
	return strings.TrimSpace(p.input[start:p.pos])
}

// parseOperator parses a comparison operator
func (p *FilterParser) parseOperator() string {
	p.skipWhitespace()
	operators := []string{"eq", "ne", "co", "sw", "ew", "pr", "gt", "ge", "lt", "le"}

	for _, op := range operators {
		if p.matchKeyword(op) {
			p.pos += len(op)
			return op
		}
	}

	return ""
}

// parseValue parses a value (string, number, boolean, null)
func (p *FilterParser) parseValue() (any, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("expected value at position %d", p.pos)
	}

	// String value
	if p.peek() == '"' {
		p.pos++
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] != '"' {
			p.pos++
		}
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated string at position %d", start)
		}
		value := p.input[start:p.pos]
		p.pos++ // Skip closing quote
		return value, nil
	}

	// Boolean or null
	if p.matchKeyword("true") {
		p.pos += 4
		return true, nil
	}
	if p.matchKeyword("false") {
		p.pos += 5
		return false, nil
	}
	if p.matchKeyword("null") {
		p.pos += 4
		return nil, nil
	}

	// Number
	start := p.pos
	for p.pos < len(p.input) && (isDigit(p.input[p.pos]) || p.input[p.pos] == '.' || p.input[p.pos] == '-') {
		p.pos++
	}
	if p.pos > start {
		numStr := p.input[start:p.pos]
		if strings.Contains(numStr, ".") {
			return strconv.ParseFloat(numStr, 64)
		}
		return strconv.ParseInt(numStr, 10, 64)
	}

	return nil, fmt.Errorf("invalid value at position %d", p.pos)
}

// Matches checks if an attribute expression matches a resource
func (ae *AttributeExpression) Matches(resource any) bool {
	value := getAttributeValue(resource, ae.AttributePath)

	switch ae.Operator {
	case "eq":
		return compareEqual(value, ae.Value)
	case "ne":
		return !compareEqual(value, ae.Value)
	case "co":
		return contains(value, ae.Value)
	case "sw":
		return startsWith(value, ae.Value)
	case "ew":
		return endsWith(value, ae.Value)
	case "pr":
		return value != nil && !isZeroValue(value)
	case "gt":
		return compareGreater(value, ae.Value)
	case "ge":
		return compareGreaterOrEqual(value, ae.Value)
	case "lt":
		return compareLess(value, ae.Value)
	case "le":
		return compareLessOrEqual(value, ae.Value)
	}

	return false
}

// Matches checks if a logical expression matches a resource
func (le *LogicalExpression) Matches(resource any) bool {
	switch le.Operator {
	case "and":
		return le.Left.Matches(resource) && le.Right.Matches(resource)
	case "or":
		return le.Left.Matches(resource) || le.Right.Matches(resource)
	case "not":
		return !le.Left.Matches(resource)
	}
	return false
}

// Matches checks if a group expression matches a resource
func (ge *GroupExpression) Matches(resource any) bool {
	return ge.Filter.Matches(resource)
}

// Helper functions

func (p *FilterParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *FilterParser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n') {
		p.pos++
	}
}

func (p *FilterParser) matchKeyword(keyword string) bool {
	if p.pos+len(keyword) > len(p.input) {
		return false
	}
	match := strings.EqualFold(p.input[p.pos:p.pos+len(keyword)], keyword)
	if !match {
		return false
	}
	// Check that keyword is not part of a larger word
	if p.pos+len(keyword) < len(p.input) {
		nextChar := p.input[p.pos+len(keyword)]
		if isAlphaNumeric(nextChar) {
			return false
		}
	}
	return true
}

func isAlphaNumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// getAttributeValue extracts a value from a resource by attribute path
func getAttributeValue(resource any, path string) any {
	if resource == nil {
		return nil
	}

	// Handle complex paths like emails[type eq "work"].value
	if strings.Contains(path, "[") {
		return getComplexAttributeValue(resource, path)
	}

	// For nested paths (e.g., "meta.created"), use JSON-based navigation
	// This is more reliable than reflection, especially for nested fields
	if strings.Contains(path, ".") {
		return getNestedAttributeValueJSON(resource, path)
	}

	// For simple paths, try reflection first (faster for direct struct access)
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := findField(v, path)
		if field.IsValid() {
			return field.Interface()
		}
	}

	// Fallback to JSON-based navigation for non-struct types or when field not found
	return getNestedAttributeValueJSON(resource, path)
}

// getNestedAttributeValueJSON uses JSON marshaling to navigate nested paths
// This works reliably for any JSON-serializable resource and nested paths like "meta.created"
func getNestedAttributeValueJSON(resource any, path string) any {
	// Marshal resource to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		return nil
	}

	// Unmarshal to map for navigation
	var resourceMap map[string]any
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return nil
	}

	// Navigate through the path
	parts := strings.Split(path, ".")
	var current any = resourceMap

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// Current must be a map to navigate further
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		// Try to find the key (case-insensitive)
		var found bool
		for key, value := range currentMap {
			if strings.EqualFold(key, part) {
				current = value
				found = true
				break
			}
		}

		if !found {
			return nil
		}
	}

	return current
}

// getComplexAttributeValue handles complex attribute paths with filters
func getComplexAttributeValue(resource any, path string) any {
	// Simple regex to parse paths like: emails[type eq "work"].value
	re := regexp.MustCompile(`^(\w+)\[(.+?)\]\.?(.*)$`)
	matches := re.FindStringSubmatch(path)

	if len(matches) == 0 {
		return getAttributeValue(resource, path)
	}

	arrayAttr := matches[1]
	filterExpr := matches[2]
	remainingPath := matches[3]

	// Get the array attribute
	arrayValue := getAttributeValue(resource, arrayAttr)
	if arrayValue == nil {
		return nil
	}

	v := reflect.ValueOf(arrayValue)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	// Parse the filter
	parser := NewFilterParser(filterExpr)
	filter, err := parser.Parse()
	if err != nil {
		return nil
	}

	// Find matching element
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		if filter.Matches(elem) {
			if remainingPath != "" {
				return getAttributeValue(elem, remainingPath)
			}
			return elem
		}
	}

	return nil
}

// findField finds a struct field by name (case-insensitive)
func findField(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// Check field name
		if strings.EqualFold(field.Name, name) {
			return v.Field(i)
		}
		// Check json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if strings.EqualFold(jsonName, name) {
				return v.Field(i)
			}
		}
	}
	return reflect.Value{}
}

// Comparison functions

func compareEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Dereference pointers for comparison
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	if aVal.Kind() == reflect.Ptr && !aVal.IsNil() {
		a = aVal.Elem().Interface()
	}
	if bVal.Kind() == reflect.Ptr && !bVal.IsNil() {
		b = bVal.Elem().Interface()
	}

	// String comparison (case-sensitive per RFC 7644 Section 3.4.2.2)
	// Note: Attribute names are case-insensitive, but values are case-sensitive
	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)
	if aIsStr && bIsStr {
		return aStr == bStr
	}

	// Handle comparison between bool and custom bool types (like Boolean)
	// Convert both to bool if either has underlying bool kind
	aVal = reflect.ValueOf(a)
	bVal = reflect.ValueOf(b)
	if (aVal.Kind() == reflect.Bool || bVal.Kind() == reflect.Bool) &&
		aVal.Type().ConvertibleTo(reflect.TypeOf(true)) &&
		bVal.Type().ConvertibleTo(reflect.TypeOf(true)) {
		return aVal.Convert(reflect.TypeOf(true)).Bool() == bVal.Convert(reflect.TypeOf(true)).Bool()
	}

	// Handle comparison between boolean and string representation
	// This handles cases like: primary eq "True" where primary is Boolean type
	if aVal.Kind() == reflect.Bool && bIsStr {
		boolVal := aVal.Bool()
		return (boolVal && strings.ToLower(bStr) == "true") || (!boolVal && strings.ToLower(bStr) == "false")
	}
	if bVal.Kind() == reflect.Bool && aIsStr {
		boolVal := bVal.Bool()
		return (boolVal && strings.ToLower(aStr) == "true") || (!boolVal && strings.ToLower(aStr) == "false")
	}

	return reflect.DeepEqual(a, b)
}

// contains checks if string a contains string b (case-sensitive per RFC 7644 Section 3.4.2.2)
func contains(a, b any) bool {
	aStr, ok := a.(string)
	if !ok {
		return false
	}
	bStr, ok := b.(string)
	if !ok {
		return false
	}
	return strings.Contains(aStr, bStr)
}

// startsWith checks if string a starts with string b (case-sensitive per RFC 7644 Section 3.4.2.2)
func startsWith(a, b any) bool {
	aStr, ok := a.(string)
	if !ok {
		return false
	}
	bStr, ok := b.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(aStr, bStr)
}

// endsWith checks if string a ends with string b (case-sensitive per RFC 7644 Section 3.4.2.2)
func endsWith(a, b any) bool {
	aStr, ok := a.(string)
	if !ok {
		return false
	}
	bStr, ok := b.(string)
	if !ok {
		return false
	}
	return strings.HasSuffix(aStr, bStr)
}

func compareGreater(a, b any) bool {
	return compareNumeric(a, b, func(x, y float64) bool { return x > y })
}

func compareGreaterOrEqual(a, b any) bool {
	return compareNumeric(a, b, func(x, y float64) bool { return x >= y })
}

func compareLess(a, b any) bool {
	return compareNumeric(a, b, func(x, y float64) bool { return x < y })
}

func compareLessOrEqual(a, b any) bool {
	return compareNumeric(a, b, func(x, y float64) bool { return x <= y })
}

func compareNumeric(a, b any, op func(float64, float64) bool) bool {
	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum == nil || bNum == nil {
		return false
	}
	return op(*aNum, *bNum)
}

func toFloat64(v any) *float64 {
	var result float64
	switch val := v.(type) {
	case float64:
		result = val
	case float32:
		result = float64(val)
	case int:
		result = float64(val)
	case int32:
		result = float64(val)
	case int64:
		result = float64(val)
	default:
		return nil
	}
	return &result
}

func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero()
}
