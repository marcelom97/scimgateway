package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// discardLogger returns a no-op logger that discards all output
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// PluginGetter defines the interface for getting plugin operations
type PluginGetter interface {
	GetUsers(ctx context.Context, baseEntity string, params QueryParams) (*ListResponse[*User], error)
	CreateUser(ctx context.Context, baseEntity string, user *User) (*User, error)
	// TODO: Replace attributes with QueryParams for consistency
	GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*User, error)
	ModifyUser(ctx context.Context, baseEntity string, id string, patch *PatchOp) error
	DeleteUser(ctx context.Context, baseEntity string, id string) error
	GetGroups(ctx context.Context, baseEntity string, params QueryParams) (*ListResponse[*Group], error)
	CreateGroup(ctx context.Context, baseEntity string, group *Group) (*Group, error)
	// TODO: Replace attributes with QueryParams for consistency
	GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*Group, error)
	ModifyGroup(ctx context.Context, baseEntity string, id string, patch *PatchOp) error
	DeleteGroup(ctx context.Context, baseEntity string, id string) error
}

// PluginManager defines the interface for managing plugins
type PluginManager interface {
	Get(name string) (PluginGetter, bool)
	List() []string
}

// Server represents a SCIM server instance
type Server struct {
	baseURL       string
	handler       *Handler
	pluginManager PluginManager
	mux           *http.ServeMux
	etagGen       *ETagGenerator
	logger        *slog.Logger
}

// NewServer creates a new SCIM server without logging
func NewServer(baseURL string, pluginManager PluginManager) *Server {
	return NewServerWithLogger(baseURL, pluginManager, nil)
}

// NewServerWithLogger creates a new SCIM server with an optional logger.
// Pass nil for logger to disable logging.
func NewServerWithLogger(baseURL string, pluginManager PluginManager, logger *slog.Logger) *Server {
	if logger == nil {
		logger = discardLogger()
	}

	s := &Server{
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		handler:       NewHandler(baseURL),
		pluginManager: pluginManager,
		mux:           http.NewServeMux(),
		etagGen:       NewETagGenerator(),
		logger:        logger,
	}

	s.setupRoutes()
	return s
}

// handlePluginError writes the appropriate error response based on error type
// If the error is a *SCIMError, it uses the status and scimType from the error
// Otherwise, it uses the provided fallback status and scimType
func (s *Server) handlePluginError(w http.ResponseWriter, err error, fallbackStatus int, fallbackScimType string) {
	if scimErr, ok := err.(*SCIMError); ok {
		s.handler.WriteSCIMError(w, scimErr)
	} else {
		s.handler.WriteError(w, fallbackStatus, err.Error(), fallbackScimType)
	}
}

// setupRoutes sets up HTTP routes using Go 1.22+ enhanced routing patterns
func (s *Server) setupRoutes() {
	// Per-plugin discovery endpoints (public, no auth required - handled by middleware)
	s.mux.HandleFunc("GET /{plugin}/ServiceProviderConfig", s.handleServiceProviderConfig)
	s.mux.HandleFunc("GET /{plugin}/ResourceTypes", s.handleResourceTypes)
	s.mux.HandleFunc("GET /{plugin}/Schemas", s.handleSchemas)

	// Search endpoints
	s.mux.HandleFunc("POST /{plugin}/.search", s.handleSearchEndpoint)
	s.mux.HandleFunc("POST /{plugin}/Users/.search", s.handleSearchEndpoint)
	s.mux.HandleFunc("POST /{plugin}/Groups/.search", s.handleSearchEndpoint)

	// Bulk endpoint
	s.mux.HandleFunc("POST /{plugin}/Bulk", s.handleBulkEndpoint)

	// User endpoints
	s.mux.HandleFunc("GET /{plugin}/Users", s.handleGetUsers)
	s.mux.HandleFunc("POST /{plugin}/Users", s.handleCreateUser)
	s.mux.HandleFunc("GET /{plugin}/Users/{id}", s.handleGetUser)
	s.mux.HandleFunc("PUT /{plugin}/Users/{id}", s.handleReplaceUser)
	s.mux.HandleFunc("PATCH /{plugin}/Users/{id}", s.handlePatchUser)
	s.mux.HandleFunc("DELETE /{plugin}/Users/{id}", s.handleDeleteUser)

	// Group endpoints
	s.mux.HandleFunc("GET /{plugin}/Groups", s.handleGetGroups)
	s.mux.HandleFunc("POST /{plugin}/Groups", s.handleCreateGroup)
	s.mux.HandleFunc("GET /{plugin}/Groups/{id}", s.handleGetGroup)
	s.mux.HandleFunc("PUT /{plugin}/Groups/{id}", s.handleReplaceGroup)
	s.mux.HandleFunc("PATCH /{plugin}/Groups/{id}", s.handlePatchGroup)
	s.mux.HandleFunc("DELETE /{plugin}/Groups/{id}", s.handleDeleteGroup)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// getPlugin retrieves a plugin by name and logs if not found
func (s *Server) getPlugin(pluginName, endpoint string, r *http.Request) (PluginGetter, bool) {
	plugin, ok := s.pluginManager.Get(pluginName)
	if !ok {
		s.logger.Warn("plugin not found",
			"plugin", pluginName,
			"endpoint", endpoint,
			"remote_addr", r.RemoteAddr,
		)
	}
	return plugin, ok
}

// handleServiceProviderConfig handles GET /{plugin}/ServiceProviderConfig
func (s *Server) handleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	// Verify plugin exists
	_, ok := s.getPlugin(pluginName, "ServiceProviderConfig", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	// Return default service provider config
	config := GetServiceProviderConfig(nil)
	s.handler.WriteJSON(w, http.StatusOK, config)
}

// handleResourceTypes handles GET /{plugin}/ResourceTypes
func (s *Server) handleResourceTypes(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	// Verify plugin exists
	_, ok := s.getPlugin(pluginName, "ResourceTypes", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	// Return default resource types
	resourceTypes := GetResourceTypes()
	s.handler.WriteJSON(w, http.StatusOK, map[string]any{"Resources": resourceTypes})
}

// handleSchemas handles GET /{plugin}/Schemas
func (s *Server) handleSchemas(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	// Verify plugin exists
	_, ok := s.getPlugin(pluginName, "Schemas", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	// Return all schemas
	schemas := []any{
		GetUserSchema(),
		GetGroupSchema(),
	}
	s.handler.WriteJSON(w, http.StatusOK, schemas)
}

// handleSearchEndpoint handles POST /{plugin}/.search
func (s *Server) handleSearchEndpoint(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	plugin, ok := s.pluginManager.Get(pluginName)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.handleSearch(w, r, plugin, pluginName)
}

// handleBulkEndpoint handles POST /{plugin}/Bulk
func (s *Server) handleBulkEndpoint(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	s.handleBulk(w, r, pluginName)
}

// handleGetUsers handles GET /{plugin}/Users
func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	plugin, ok := s.getPlugin(pluginName, "GET /Users", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.getUsers(w, r, plugin, pluginName)
}

// handleCreateUser handles POST /{plugin}/Users
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	plugin, ok := s.getPlugin(pluginName, "POST /Users", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.createUser(w, r, plugin, pluginName)
}

// handleGetUser handles GET /{plugin}/Users/{id}
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "GET /Users/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.getUser(w, r, plugin, pluginName, id)
}

// handleReplaceUser handles PUT /{plugin}/Users/{id}
func (s *Server) handleReplaceUser(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "PUT /Users/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.replaceUser(w, r, plugin, pluginName, id)
}

// handlePatchUser handles PATCH /{plugin}/Users/{id}
func (s *Server) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "PATCH /Users/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.modifyUser(w, r, plugin, pluginName, id)
}

// handleDeleteUser handles DELETE /{plugin}/Users/{id}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "DELETE /Users/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.deleteUser(w, r, plugin, pluginName, id)
}

// handleGetGroups handles GET /{plugin}/Groups
func (s *Server) handleGetGroups(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	plugin, ok := s.getPlugin(pluginName, "GET /Groups", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.getGroups(w, r, plugin, pluginName)
}

// handleCreateGroup handles POST /{plugin}/Groups
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")

	plugin, ok := s.getPlugin(pluginName, "POST /Groups", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.createGroup(w, r, plugin, pluginName)
}

// handleGetGroup handles GET /{plugin}/Groups/{id}
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "GET /Groups/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.getGroup(w, r, plugin, pluginName, id)
}

// handleReplaceGroup handles PUT /{plugin}/Groups/{id}
func (s *Server) handleReplaceGroup(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "PUT /Groups/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.replaceGroup(w, r, plugin, pluginName, id)
}

// handlePatchGroup handles PATCH /{plugin}/Groups/{id}
func (s *Server) handlePatchGroup(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "PATCH /Groups/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.modifyGroup(w, r, plugin, pluginName, id)
}

// handleDeleteGroup handles DELETE /{plugin}/Groups/{id}
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	pluginName := r.PathValue("plugin")
	id := r.PathValue("id")

	plugin, ok := s.getPlugin(pluginName, "DELETE /Groups/{id}", r)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	s.deleteGroup(w, r, plugin, pluginName, id)
}

// getUsers handles GET /plugin/Users
func (s *Server) getUsers(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string) {
	params, err := s.handler.ParseQueryParams(r)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidFilter")
		return
	}

	response, err := plugin.GetUsers(r.Context(), pluginName, params)
	if err != nil {
		if scimErr, ok := err.(*SCIMError); ok {
			s.handler.WriteSCIMError(w, scimErr)
		} else {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
		}
		return
	}

	// Apply attribute selection if specified
	if len(params.Attributes) > 0 || len(params.ExcludedAttr) > 0 {
		selector := NewAttributeSelector(params.Attributes, params.ExcludedAttr)
		// Convert to []any for filtering
		resources := make([]any, len(response.Resources))
		for i, user := range response.Resources {
			resources[i] = user
		}

		filteredResources, err := selector.FilterResources(resources)
		if err != nil {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
			return
		}

		// Return filtered response with []any type
		filteredResponse := &ListResponse[any]{
			Schemas:      response.Schemas,
			TotalResults: response.TotalResults,
			StartIndex:   response.StartIndex,
			ItemsPerPage: response.ItemsPerPage,
			Resources:    filteredResources,
		}
		s.handler.WriteJSON(w, http.StatusOK, filteredResponse)
		return
	}

	s.handler.WriteJSON(w, http.StatusOK, response)
}

// createUser handles POST /plugin/Users
func (s *Server) createUser(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Failed to read request body", "invalidSyntax")
		return
	}
	defer r.Body.Close()

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate user
	validator := NewValidator()
	if err := validator.ValidateUser(&user); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	// Set active to true by default if not provided
	// Note: We check against the raw JSON to see if 'active' was explicitly set
	var rawData map[string]any
	json.Unmarshal(body, &rawData)
	if _, exists := rawData["active"]; !exists {
		user.Active = Bool(true)
	}

	created, err := plugin.CreateUser(r.Context(), pluginName, &user)
	if err != nil {
		s.handlePluginError(w, err, http.StatusInternalServerError, "internalError")
		return
	}

	// Set location header
	location := s.handler.GetResourceLocation(pluginName, "Users", created.ID)
	w.Header().Set("Location", location)
	if created.Meta != nil {
		created.Meta.Location = location
	}

	// Generate ETag for the created resource
	etag, err := s.etagGen.Generate(created)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(created.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusCreated, created)
}

// getUser handles GET /plugin/Users/{id}
func (s *Server) getUser(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	params, err := s.handler.ParseQueryParams(r)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidFilter")
		return
	}

	user, err := plugin.GetUser(r.Context(), pluginName, id, params.Attributes)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for the resource
	etag, err := s.etagGen.Generate(user)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-None-Match for conditional GET (304 Not Modified)
	status, err := s.etagGen.CheckPreconditions(r, etag)
	if err != nil && status == http.StatusNotModified {
		// Resource hasn't changed, return 304
		s.etagGen.SetETag(w, etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(user.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	// Apply attribute selection
	if len(params.Attributes) > 0 || len(params.ExcludedAttr) > 0 {
		selector := NewAttributeSelector(params.Attributes, params.ExcludedAttr)
		filtered, err := selector.FilterResource(user)
		if err != nil {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
			return
		}
		// Return the filtered map directly to preserve exact attribute selection
		s.handler.WriteJSON(w, http.StatusOK, filtered)
		return
	}

	s.handler.WriteJSON(w, http.StatusOK, user)
}

// replaceUser handles PUT /plugin/Users/{id}
func (s *Server) replaceUser(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentUser, err := plugin.GetUser(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentUser)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate user
	validator := NewValidator()
	if err := validator.ValidateUser(&user); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	// Ensure ID matches
	user.ID = id

	// Delete and recreate (simple replace strategy)
	if err := plugin.DeleteUser(r.Context(), pluginName, id); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	created, err := plugin.CreateUser(r.Context(), pluginName, &user)
	if err != nil {
		s.handlePluginError(w, err, http.StatusInternalServerError, "internalError")
		return
	}

	// Generate ETag for the updated resource
	etag, err := s.etagGen.Generate(created)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(created.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusOK, created)
}

// modifyUser handles PATCH /plugin/Users/{id}
func (s *Server) modifyUser(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentUser, err := plugin.GetUser(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentUser)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	var patch PatchOp
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate patch
	validator := NewValidator()
	if err := validator.ValidatePatchOp(&patch); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	if err := plugin.ModifyUser(r.Context(), pluginName, id, &patch); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Return updated user
	user, err := plugin.GetUser(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for the updated resource
	etag, err := s.etagGen.Generate(user)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(user.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusOK, user)
}

// deleteUser handles DELETE /plugin/Users/{id}
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentUser, err := plugin.GetUser(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentUser)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	if err := plugin.DeleteUser(r.Context(), pluginName, id); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getGroups handles GET /plugin/Groups
func (s *Server) getGroups(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string) {
	params, err := s.handler.ParseQueryParams(r)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidFilter")
		return
	}

	response, err := plugin.GetGroups(r.Context(), pluginName, params)
	if err != nil {
		if scimErr, ok := err.(*SCIMError); ok {
			s.handler.WriteSCIMError(w, scimErr)
		} else {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
		}
		return
	}

	// Apply attribute selection if specified
	if len(params.Attributes) > 0 || len(params.ExcludedAttr) > 0 {
		selector := NewAttributeSelector(params.Attributes, params.ExcludedAttr)
		// Convert to []any for filtering
		resources := make([]any, len(response.Resources))
		for i, group := range response.Resources {
			resources[i] = group
		}

		filteredResources, err := selector.FilterResources(resources)
		if err != nil {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
			return
		}

		// Return filtered response with []any type
		filteredResponse := &ListResponse[any]{
			Schemas:      response.Schemas,
			TotalResults: response.TotalResults,
			StartIndex:   response.StartIndex,
			ItemsPerPage: response.ItemsPerPage,
			Resources:    filteredResources,
		}
		s.handler.WriteJSON(w, http.StatusOK, filteredResponse)
		return
	}

	s.handler.WriteJSON(w, http.StatusOK, response)
}

// createGroup handles POST /plugin/Groups
func (s *Server) createGroup(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string) {
	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate group
	validator := NewValidator()
	if err := validator.ValidateGroup(&group); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	created, err := plugin.CreateGroup(r.Context(), pluginName, &group)
	if err != nil {
		s.handlePluginError(w, err, http.StatusInternalServerError, "internalError")
		return
	}

	// Set location header
	location := s.handler.GetResourceLocation(pluginName, "Groups", created.ID)
	w.Header().Set("Location", location)
	if created.Meta != nil {
		created.Meta.Location = location
	}

	// Generate ETag for the created resource
	etag, err := s.etagGen.Generate(created)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(created.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusCreated, created)
}

// getGroup handles GET /plugin/Groups/{id}
func (s *Server) getGroup(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	params, err := s.handler.ParseQueryParams(r)
	if err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidFilter")
		return
	}

	group, err := plugin.GetGroup(r.Context(), pluginName, id, params.Attributes)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for the resource
	etag, err := s.etagGen.Generate(group)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-None-Match for conditional GET (304 Not Modified)
	status, err := s.etagGen.CheckPreconditions(r, etag)
	if err != nil && status == http.StatusNotModified {
		// Resource hasn't changed, return 304
		s.etagGen.SetETag(w, etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(group.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	// Apply attribute selection
	if len(params.Attributes) > 0 || len(params.ExcludedAttr) > 0 {
		selector := NewAttributeSelector(params.Attributes, params.ExcludedAttr)
		filtered, err := selector.FilterResource(group)
		if err != nil {
			s.handler.WriteError(w, http.StatusInternalServerError, err.Error(), "internalError")
			return
		}
		// Return the filtered map directly to preserve exact attribute selection
		s.handler.WriteJSON(w, http.StatusOK, filtered)
		return
	}

	s.handler.WriteJSON(w, http.StatusOK, group)
}

// replaceGroup handles PUT /plugin/Groups/{id}
func (s *Server) replaceGroup(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentGroup, err := plugin.GetGroup(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentGroup)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate group
	validator := NewValidator()
	if err := validator.ValidateGroup(&group); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	// Ensure ID matches
	group.ID = id

	// Delete and recreate (simple replace strategy)
	if err := plugin.DeleteGroup(r.Context(), pluginName, id); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	created, err := plugin.CreateGroup(r.Context(), pluginName, &group)
	if err != nil {
		s.handlePluginError(w, err, http.StatusInternalServerError, "internalError")
		return
	}

	// Generate ETag for the updated resource
	etag, err := s.etagGen.Generate(created)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(created.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusOK, created)
}

// modifyGroup handles PATCH /plugin/Groups/{id}
func (s *Server) modifyGroup(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentGroup, err := plugin.GetGroup(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentGroup)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	var patch PatchOp
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate patch
	validator := NewValidator()
	if err := validator.ValidatePatchOp(&patch); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	if err := plugin.ModifyGroup(r.Context(), pluginName, id, &patch); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Return updated group
	group, err := plugin.GetGroup(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for the updated resource
	etag, err := s.etagGen.Generate(group)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Update meta.version with ETag value
	UpdateResourceVersion(group.Meta, etag)

	// Set ETag header on response
	s.etagGen.SetETag(w, etag)

	s.handler.WriteJSON(w, http.StatusOK, group)
}

// deleteGroup handles DELETE /plugin/Groups/{id}
func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request, plugin PluginGetter, pluginName string, id string) {
	// Get current resource to check ETag preconditions
	currentGroup, err := plugin.GetGroup(r.Context(), pluginName, id, nil)
	if err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	// Generate ETag for current resource
	currentETag, err := s.etagGen.Generate(currentGroup)
	if err != nil {
		s.handler.WriteError(w, http.StatusInternalServerError, "Failed to generate ETag", "internalError")
		return
	}

	// Check If-Match precondition
	status, err := s.etagGen.CheckPreconditions(r, currentETag)
	if err != nil && status == http.StatusPreconditionFailed {
		s.handler.WriteError(w, http.StatusPreconditionFailed, err.Error(), "invalidVers")
		return
	}

	if err := plugin.DeleteGroup(r.Context(), pluginName, id); err != nil {
		s.handlePluginError(w, err, http.StatusNotFound, "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
