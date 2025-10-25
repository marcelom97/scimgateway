package scim

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Validator validates SCIM resources
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateUser validates a User resource
func (v *Validator) ValidateUser(user *User) error {
	if user == nil {
		return ErrInvalidValue("user cannot be nil")
	}

	// userName is required
	if strings.TrimSpace(user.UserName) == "" {
		return ErrInvalidValue("userName is required")
	}

	// Validate userName format (alphanumeric, dots, underscores, hyphens, @ allowed)
	if !isValidUserName(user.UserName) {
		return ErrInvalidValue("userName contains invalid characters")
	}

	// Validate emails
	for _, email := range user.Emails {
		if err := v.validateEmail(email.Value); err != nil {
			return err
		}
	}

	// Validate schemas
	if len(user.Schemas) == 0 {
		user.Schemas = []string{SchemaUser}
	}

	return nil
}

// ValidateGroup validates a Group resource
func (v *Validator) ValidateGroup(group *Group) error {
	if group == nil {
		return ErrInvalidValue("group cannot be nil")
	}

	// displayName is required
	if strings.TrimSpace(group.DisplayName) == "" {
		return ErrInvalidValue("displayName is required")
	}

	// Validate schemas
	if len(group.Schemas) == 0 {
		group.Schemas = []string{SchemaGroup}
	}

	return nil
}

// ValidatePatchOp validates a PATCH operation
func (v *Validator) ValidatePatchOp(patch *PatchOp) error {
	if patch == nil {
		return ErrInvalidSyntax("patch operation cannot be nil")
	}

	// Validate schemas
	validSchema := slices.Contains(patch.Schemas, SchemaPatchOp)
	if !validSchema {
		return ErrInvalidValue(fmt.Sprintf("invalid schema, expected %s", SchemaPatchOp))
	}

	// Validate operations
	if len(patch.Operations) == 0 {
		return ErrInvalidValue("at least one operation is required")
	}

	for i, op := range patch.Operations {
		if err := v.validatePatchOperation(op); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}
	}

	return nil
}

// validatePatchOperation validates a single patch operation
func (v *Validator) validatePatchOperation(op PatchOperation) error {
	// Validate op
	opLower := strings.ToLower(op.Op)
	if opLower != "add" && opLower != "remove" && opLower != "replace" {
		return ErrInvalidValue(fmt.Sprintf("invalid op: %s", op.Op))
	}

	// Remove requires a path
	if opLower == "remove" && op.Path == "" {
		return ErrNoTarget("path is required for remove operation")
	}

	// Add and Replace require a value (unless path targets specific attribute)
	if (opLower == "add" || opLower == "replace") && op.Value == nil && op.Path == "" {
		return ErrInvalidValue(fmt.Sprintf("value is required for %s operation", op.Op))
	}

	return nil
}

// validateEmail validates an email address
func (v *Validator) validateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidValue(fmt.Sprintf("invalid email format: %s", email))
	}

	return nil
}

// isValidUserName checks if a userName is valid
func isValidUserName(userName string) bool {
	// Allow alphanumeric, dots, underscores, hyphens, and @
	validUserNameRegex := regexp.MustCompile(`^[a-zA-Z0-9._@\-]+$`)
	return validUserNameRegex.MatchString(userName)
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// ValidateQueryParams validates query parameters
func ValidateQueryParams(params *QueryParams) error {
	// Validate startIndex
	if params.StartIndex < 1 {
		params.StartIndex = 1
	}

	// Validate count
	if params.Count < 1 {
		params.Count = 100
	}
	if params.Count > 1000 {
		params.Count = 1000 // Max limit
	}

	// Validate sortOrder
	if params.SortOrder != "" {
		sortOrder := strings.ToLower(params.SortOrder)
		if sortOrder != "ascending" && sortOrder != "descending" {
			return ErrInvalidValue(fmt.Sprintf("invalid sortOrder: %s", params.SortOrder))
		}
		params.SortOrder = sortOrder
	}

	return nil
}
