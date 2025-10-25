package scim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	SchemaListResponse = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SchemaError        = "urn:ietf:params:scim:api:messages:2.0:Error"
	SchemaUser         = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaGroup        = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SchemaPatchOp      = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// Handler handles HTTP requests and routing for SCIM endpoints
type Handler struct {
	baseURL string
}

// NewHandler creates a new SCIM handler
func NewHandler(baseURL string) *Handler {
	return &Handler{
		baseURL: baseURL,
	}
}

// WriteError writes a SCIM error response
func (h *Handler) WriteError(w http.ResponseWriter, status int, detail string, scimType string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)

	err := Error{
		Schemas:  []string{SchemaError},
		Status:   strconv.Itoa(status),
		Detail:   detail,
		ScimType: scimType,
	}

	json.NewEncoder(w).Encode(err)
}

// WriteJSON writes a successful JSON response
func (h *Handler) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ParseQueryParams extracts SCIM query parameters from the request
// Returns an error if both attributes and excludedAttributes are specified (RFC 7644 Section 3.9)
func (h *Handler) ParseQueryParams(r *http.Request) (QueryParams, error) {
	params := QueryParams{
		StartIndex: 1,
		Count:      100,
		SortOrder:  "ascending",
	}

	if filter := r.URL.Query().Get("filter"); filter != "" {
		params.Filter = filter
	}

	// Parse attributes parameter
	hasAttributes := false
	if attrs := r.URL.Query().Get("attributes"); attrs != "" {
		params.Attributes = strings.Split(attrs, ",")
		for i := range params.Attributes {
			params.Attributes[i] = strings.TrimSpace(params.Attributes[i])
		}
		hasAttributes = true
	}

	// Parse excludedAttributes parameter
	hasExcluded := false
	if excludedAttr := r.URL.Query().Get("excludedAttributes"); excludedAttr != "" {
		params.ExcludedAttr = strings.Split(excludedAttr, ",")
		for i := range params.ExcludedAttr {
			params.ExcludedAttr[i] = strings.TrimSpace(params.ExcludedAttr[i])
		}
		hasExcluded = true
	}

	// RFC 7644 Section 3.9: attributes and excludedAttributes are mutually exclusive
	if hasAttributes && hasExcluded {
		return params, fmt.Errorf("attributes and excludedAttributes are mutually exclusive")
	}

	if startIndex := r.URL.Query().Get("startIndex"); startIndex != "" {
		if idx, err := strconv.Atoi(startIndex); err == nil && idx > 0 {
			params.StartIndex = idx
		}
	}

	if count := r.URL.Query().Get("count"); count != "" {
		if c, err := strconv.Atoi(count); err == nil && c > 0 {
			params.Count = c
		}
	}

	if sortBy := r.URL.Query().Get("sortBy"); sortBy != "" {
		params.SortBy = sortBy
	}

	if sortOrder := r.URL.Query().Get("sortOrder"); sortOrder != "" {
		params.SortOrder = strings.ToLower(sortOrder)
	}

	return params, nil
}

// GetResourceLocation returns the location URL for a resource
func (h *Handler) GetResourceLocation(pluginName, resourceType, id string) string {
	return fmt.Sprintf("%s/%s/%s/%s", h.baseURL, pluginName, resourceType, id)
}

// ExtractPluginAndID extracts plugin name and resource ID from the request path
// Expected format: /{plugin}/Users/{id} or /{plugin}/Groups/{id}
func (h *Handler) ExtractPluginAndID(path string) (plugin, resourceType, id string, err error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid path format")
	}

	plugin = parts[0]
	resourceType = parts[1]

	if len(parts) >= 3 {
		id = parts[2]
	}

	return plugin, resourceType, id, nil
}
