package scim

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
)

const (
	SchemaSearchRequest = "urn:ietf:params:scim:api:messages:2.0:SearchRequest"
)

// SearchRequest represents a SCIM search request
type SearchRequest struct {
	Schemas            []string `json:"schemas"`
	Attributes         []string `json:"attributes,omitempty"`
	ExcludedAttributes []string `json:"excludedAttributes,omitempty"`
	Filter             string   `json:"filter,omitempty"`
	SortBy             string   `json:"sortBy,omitempty"`
	SortOrder          string   `json:"sortOrder,omitempty"`
	StartIndex         int      `json:"startIndex,omitempty"`
	Count              int      `json:"count,omitempty"`
}

// handleSearch handles POST /.search
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, plugin PluginGetter) {
	if r.Method != http.MethodPost {
		s.handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalidMethod")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Failed to read request body", "invalidSyntax")
		return
	}
	defer r.Body.Close()

	var searchReq SearchRequest
	if err := json.Unmarshal(body, &searchReq); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate schemas
	validSchema := slices.Contains(searchReq.Schemas, SchemaSearchRequest)
	if !validSchema {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid schema", "invalidValue")
		return
	}

	// Set defaults
	if searchReq.StartIndex == 0 {
		searchReq.StartIndex = 1
	}
	if searchReq.Count == 0 {
		searchReq.Count = 100
	}

	// Convert to QueryParams
	params := QueryParams{
		Filter:       searchReq.Filter,
		Attributes:   searchReq.Attributes,
		ExcludedAttr: searchReq.ExcludedAttributes,
		StartIndex:   searchReq.StartIndex,
		Count:        searchReq.Count,
		SortBy:       searchReq.SortBy,
		SortOrder:    searchReq.SortOrder,
	}

	// Search across both Users and Groups
	var allResources []any

	// Get users
	usersResp, err := plugin.GetUsers(r.Context(), params)
	if err == nil {
		for _, user := range usersResp.Resources {
			allResources = append(allResources, user)
		}
	}

	// Get groups
	groupsResp, err := plugin.GetGroups(r.Context(), params)
	if err == nil {
		for _, group := range groupsResp.Resources {
			allResources = append(allResources, group)
		}
	}

	// Apply filtering, sorting, and pagination to combined results
	filtered, err := FilterByFilter(allResources, params.Filter)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidFilter")
		return
	}

	sorted := SortResources(filtered, params.SortBy, params.SortOrder)
	paged, startIndex, itemsPerPage := ApplyPagination(sorted, params.StartIndex, params.Count)

	// Apply attribute selection
	selector := NewAttributeSelector(params.Attributes, params.ExcludedAttr)
	resources, err := selector.FilterResources(paged)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response := &ListResponse[any]{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(filtered),
		StartIndex:   startIndex,
		ItemsPerPage: itemsPerPage,
		Resources:    resources,
	}

	s.handler.WriteJSON(w, http.StatusOK, response)
}
