package scim

import (
	"fmt"
	"net/http"
)

// SCIM error types as defined in RFC 7644
const (
	ScimTypeInvalidFilter = "invalidFilter"
	ScimTypeInvalidPath   = "invalidPath"
	ScimTypeInvalidSyntax = "invalidSyntax"
	ScimTypeInvalidValue  = "invalidValue"
	ScimTypeInvalidVers   = "invalidVers"
	ScimTypeMutability    = "mutability"
	ScimTypeNoTarget      = "noTarget"
	ScimTypeSensitive     = "sensitive"
	ScimTypeTooMany       = "tooMany"
	ScimTypeUniqueness    = "uniqueness"
)

// SCIMError represents a SCIM error
type SCIMError struct {
	Status   int
	Detail   string
	ScimType string
}

// Error implements the error interface
func (e *SCIMError) Error() string {
	return e.Detail
}

// NewSCIMError creates a new SCIM error
func NewSCIMError(status int, detail, scimType string) *SCIMError {
	return &SCIMError{
		Status:   status,
		Detail:   detail,
		ScimType: scimType,
	}
}

// Common SCIM errors
var (
	ErrInvalidFilter = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeInvalidFilter)
	}

	ErrInvalidPath = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeInvalidPath)
	}

	ErrInvalidSyntax = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeInvalidSyntax)
	}

	ErrInvalidValue = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeInvalidValue)
	}

	ErrInvalidVersion = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeInvalidVers)
	}

	ErrMutability = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeMutability)
	}

	ErrNoTarget = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeNoTarget)
	}

	ErrSensitive = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeSensitive)
	}

	ErrTooMany = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusBadRequest, detail, ScimTypeTooMany)
	}

	ErrUniqueness = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusConflict, detail, ScimTypeUniqueness)
	}

	ErrNotFound = func(resourceType, id string) *SCIMError {
		return NewSCIMError(http.StatusNotFound, fmt.Sprintf("%s %s not found", resourceType, id), "")
	}

	ErrUnauthorized = func() *SCIMError {
		return NewSCIMError(http.StatusUnauthorized, "Unauthorized", "")
	}

	ErrForbidden = func() *SCIMError {
		return NewSCIMError(http.StatusForbidden, "Forbidden", "")
	}

	ErrMethodNotAllowed = func(method string) *SCIMError {
		return NewSCIMError(http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed", method), "")
	}

	ErrConflict = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusConflict, detail, "")
	}

	ErrInternalServer = func(detail string) *SCIMError {
		return NewSCIMError(http.StatusInternalServerError, detail, "")
	}

	ErrNotImplemented = func(feature string) *SCIMError {
		return NewSCIMError(http.StatusNotImplemented, fmt.Sprintf("%s not implemented", feature), "")
	}
)

// WriteSCIMError writes a SCIM error response
func (h *Handler) WriteSCIMError(w http.ResponseWriter, err *SCIMError) {
	h.WriteError(w, err.Status, err.Detail, err.ScimType)
}
