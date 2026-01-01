package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/marcelom97/scimgateway/scim"
)

// QueryBuilder helps construct optimized PostgreSQL queries from SCIM QueryParams
// Uses ? placeholders for compatibility with sqlx.Rebind()
type QueryBuilder struct {
	table       string
	dataColumn  string
	params      []any
	attrMapping map[string]string // Maps SCIM attribute to database column or JSONB path
}

// NewQueryBuilder creates a new query builder for the specified table
func NewQueryBuilder(table string, dataColumn string, attrMapping map[string]string) *QueryBuilder {
	return &QueryBuilder{
		table:       table,
		dataColumn:  dataColumn,
		params:      make([]any, 0),
		attrMapping: attrMapping,
	}
}

// nextParam returns the next parameter placeholder
// Uses ? for compatibility with sqlx.Rebind()
func (qb *QueryBuilder) nextParam(value any) string {
	qb.params = append(qb.params, value)
	return "?"
}

// Build constructs the full SELECT query with WHERE, ORDER BY, LIMIT, and OFFSET clauses
func (qb *QueryBuilder) Build(params scim.QueryParams) (string, []any) {
	var query strings.Builder

	// Base SELECT
	fmt.Fprintf(&query, "SELECT id, %s, data, created_at, updated_at FROM %s",
		qb.getNameColumn(), qb.table)

	// WHERE clause from filter
	whereClause := qb.buildWhereClause(params.Filter)
	if whereClause != "" {
		query.WriteString(" WHERE ")
		query.WriteString(whereClause)
	}

	// ORDER BY clause
	orderClause := qb.buildOrderClause(params.SortBy, params.SortOrder)
	if orderClause != "" {
		query.WriteString(" ")
		query.WriteString(orderClause)
	}

	// LIMIT and OFFSET for pagination
	paginationClause := qb.buildPaginationClause(params.StartIndex, params.Count)
	if paginationClause != "" {
		query.WriteString(" ")
		query.WriteString(paginationClause)
	}

	return query.String(), qb.params
}

// BuildCount constructs a COUNT query for total results
func (qb *QueryBuilder) BuildCount(params scim.QueryParams) (string, []any) {
	var query strings.Builder

	fmt.Fprintf(&query, "SELECT COUNT(*) FROM %s", qb.table)

	// WHERE clause from filter
	whereClause := qb.buildWhereClause(params.Filter)
	if whereClause != "" {
		query.WriteString(" WHERE ")
		query.WriteString(whereClause)
	}

	return query.String(), qb.params
}

// getNameColumn returns the appropriate name column for the table
func (qb *QueryBuilder) getNameColumn() string {
	if qb.table == "users" {
		return "username"
	}
	return "display_name"
}

// buildWhereClause converts a SCIM filter string to PostgreSQL WHERE clause
func (qb *QueryBuilder) buildWhereClause(filter string) string {
	if filter == "" {
		return ""
	}

	parser := scim.NewFilterParser(filter)
	parsedFilter, err := parser.Parse()
	if err != nil {
		// If filter parsing fails, return empty (let server-side filtering handle it)
		return ""
	}

	if parsedFilter == nil {
		return ""
	}

	return qb.filterToSQL(parsedFilter)
}

// filterToSQL converts a parsed SCIM filter to SQL WHERE clause
func (qb *QueryBuilder) filterToSQL(filter scim.Filter) string {
	switch f := filter.(type) {
	case *scim.AttributeExpression:
		return qb.attributeExpressionToSQL(f)
	case *scim.LogicalExpression:
		return qb.logicalExpressionToSQL(f)
	case *scim.GroupExpression:
		inner := qb.filterToSQL(f.Filter)
		if inner == "" {
			return ""
		}
		return "(" + inner + ")"
	}
	return ""
}

// attributeExpressionToSQL converts a single attribute expression to SQL
func (qb *QueryBuilder) attributeExpressionToSQL(expr *scim.AttributeExpression) string {
	sqlPath := qb.getSQLPath(expr.AttributePath)
	if sqlPath == "" {
		return ""
	}

	switch expr.Operator {
	case "eq":
		return qb.buildEqualityClause(sqlPath, expr.Value, true)
	case "ne":
		return qb.buildEqualityClause(sqlPath, expr.Value, false)
	case "co":
		return qb.buildContainsClause(sqlPath, expr.Value)
	case "sw":
		return qb.buildStartsWithClause(sqlPath, expr.Value)
	case "ew":
		return qb.buildEndsWithClause(sqlPath, expr.Value)
	case "pr":
		return qb.buildPresentClause(sqlPath)
	case "gt":
		return qb.buildComparisonClause(sqlPath, expr.Value, ">")
	case "ge":
		return qb.buildComparisonClause(sqlPath, expr.Value, ">=")
	case "lt":
		return qb.buildComparisonClause(sqlPath, expr.Value, "<")
	case "le":
		return qb.buildComparisonClause(sqlPath, expr.Value, "<=")
	}
	return ""
}

// logicalExpressionToSQL converts a logical expression (AND, OR, NOT) to SQL
func (qb *QueryBuilder) logicalExpressionToSQL(expr *scim.LogicalExpression) string {
	switch expr.Operator {
	case "and":
		left := qb.filterToSQL(expr.Left)
		right := qb.filterToSQL(expr.Right)
		if left == "" || right == "" {
			return ""
		}
		return fmt.Sprintf("(%s AND %s)", left, right)
	case "or":
		left := qb.filterToSQL(expr.Left)
		right := qb.filterToSQL(expr.Right)
		if left == "" || right == "" {
			return ""
		}
		return fmt.Sprintf("(%s OR %s)", left, right)
	case "not":
		inner := qb.filterToSQL(expr.Left)
		if inner == "" {
			return ""
		}
		return fmt.Sprintf("NOT (%s)", inner)
	}
	return ""
}

// getSQLPath converts a SCIM attribute path to PostgreSQL JSONB path
func (qb *QueryBuilder) getSQLPath(attrPath string) string {
	// Normalize attribute path to lowercase for mapping
	normalized := strings.ToLower(attrPath)

	// Check if there's a direct column mapping
	if col, ok := qb.attrMapping[normalized]; ok {
		return col
	}

	// Handle nested paths (e.g., "name.givenName" -> data->'name'->>'givenName')
	parts := strings.Split(attrPath, ".")
	if len(parts) == 1 {
		// Simple attribute - extract as text from JSONB
		return fmt.Sprintf("%s->>'%s'", qb.dataColumn, attrPath)
	}

	// Nested path - navigate through JSONB
	var path strings.Builder
	path.WriteString(qb.dataColumn)
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - extract as text
			fmt.Fprintf(&path, "->>'%s'", part)
		} else {
			// Intermediate part - keep as JSON
			fmt.Fprintf(&path, "->'%s'", part)
		}
	}
	return path.String()
}

// buildEqualityClause builds an equality (eq) or inequality (ne) clause
func (qb *QueryBuilder) buildEqualityClause(sqlPath string, value any, equal bool) string {
	op := "="
	if !equal {
		op = "<>"
	}

	switch v := value.(type) {
	case string:
		// Case-insensitive string comparison using LOWER()
		param := qb.nextParam(strings.ToLower(v))
		return fmt.Sprintf("LOWER(%s) %s %s", sqlPath, op, param)
	case bool:
		param := qb.nextParam(strconv.FormatBool(v))
		return fmt.Sprintf("%s %s %s", sqlPath, op, param)
	case int64, float64:
		param := qb.nextParam(fmt.Sprintf("%v", v))
		return fmt.Sprintf("(%s)::numeric %s %s", sqlPath, op, param)
	case nil:
		if equal {
			return fmt.Sprintf("%s IS NULL", sqlPath)
		}
		return fmt.Sprintf("%s IS NOT NULL", sqlPath)
	default:
		param := qb.nextParam(fmt.Sprintf("%v", v))
		return fmt.Sprintf("%s %s %s", sqlPath, op, param)
	}
}

// buildContainsClause builds a LIKE clause for "co" operator
func (qb *QueryBuilder) buildContainsClause(sqlPath string, value any) string {
	strVal, ok := value.(string)
	if !ok {
		return ""
	}
	// Escape special LIKE characters and wrap with wildcards
	escaped := escapeLikePattern(strVal)
	param := qb.nextParam("%" + strings.ToLower(escaped) + "%")
	return fmt.Sprintf("LOWER(%s) LIKE %s", sqlPath, param)
}

// buildStartsWithClause builds a LIKE clause for "sw" operator
func (qb *QueryBuilder) buildStartsWithClause(sqlPath string, value any) string {
	strVal, ok := value.(string)
	if !ok {
		return ""
	}
	escaped := escapeLikePattern(strVal)
	param := qb.nextParam(strings.ToLower(escaped) + "%")
	return fmt.Sprintf("LOWER(%s) LIKE %s", sqlPath, param)
}

// buildEndsWithClause builds a LIKE clause for "ew" operator
func (qb *QueryBuilder) buildEndsWithClause(sqlPath string, value any) string {
	strVal, ok := value.(string)
	if !ok {
		return ""
	}
	escaped := escapeLikePattern(strVal)
	param := qb.nextParam("%" + strings.ToLower(escaped))
	return fmt.Sprintf("LOWER(%s) LIKE %s", sqlPath, param)
}

// buildPresentClause builds an IS NOT NULL clause for "pr" operator
func (qb *QueryBuilder) buildPresentClause(sqlPath string) string {
	return fmt.Sprintf("(%s IS NOT NULL AND %s <> '')", sqlPath, sqlPath)
}

// buildComparisonClause builds a numeric comparison clause
func (qb *QueryBuilder) buildComparisonClause(sqlPath string, value any, op string) string {
	param := qb.nextParam(fmt.Sprintf("%v", value))
	return fmt.Sprintf("(%s)::numeric %s %s", sqlPath, op, param)
}

// buildOrderClause constructs the ORDER BY clause
func (qb *QueryBuilder) buildOrderClause(sortBy string, sortOrder string) string {
	if sortBy == "" {
		// Default ordering by created_at for consistent results
		return "ORDER BY created_at ASC"
	}

	sqlPath := qb.getSQLPath(sortBy)
	direction := "ASC"
	if strings.EqualFold(sortOrder, "descending") {
		direction = "DESC"
	}

	return fmt.Sprintf("ORDER BY %s %s NULLS LAST", sqlPath, direction)
}

// buildPaginationClause constructs the LIMIT and OFFSET clause
func (qb *QueryBuilder) buildPaginationClause(startIndex int, count int) string {
	var parts []string

	// SCIM uses 1-based indexing
	if count > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", count))
	}

	// Convert 1-based startIndex to 0-based offset
	if startIndex > 1 {
		offset := startIndex - 1
		parts = append(parts, fmt.Sprintf("OFFSET %d", offset))
	}

	return strings.Join(parts, " ")
}

// escapeLikePattern escapes special characters in LIKE patterns
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// UserAttributeMapping defines how SCIM user attributes map to database columns/paths
var UserAttributeMapping = map[string]string{
	"id":       "id",
	"username": "username",
	"userName": "username", // Handle case variations
}

// GroupAttributeMapping defines how SCIM group attributes map to database columns/paths
var GroupAttributeMapping = map[string]string{
	"id":          "id",
	"displayname": "display_name",
	"displayName": "display_name", // Handle case variations
}
