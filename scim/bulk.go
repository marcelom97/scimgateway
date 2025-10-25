package scim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

const (
	SchemaBulkRequest  = "urn:ietf:params:scim:api:messages:2.0:BulkRequest"
	SchemaBulkResponse = "urn:ietf:params:scim:api:messages:2.0:BulkResponse"
)

// BulkRequest represents a SCIM bulk request
type BulkRequest struct {
	Schemas      []string        `json:"schemas"`
	FailOnErrors int             `json:"failOnErrors,omitempty"`
	Operations   []BulkOperation `json:"Operations"`
}

// BulkResponse represents a SCIM bulk response
type BulkResponse struct {
	Schemas    []string                `json:"schemas"`
	Operations []BulkOperationResponse `json:"Operations"`
}

// BulkOperation represents a single bulk operation
type BulkOperation struct {
	Method  string         `json:"method"`
	BulkID  string         `json:"bulkId,omitempty"`
	Version string         `json:"version,omitempty"`
	Path    string         `json:"path"`
	Data    map[string]any `json:"data,omitempty"`
}

// BulkOperationResponse represents a bulk operation response
type BulkOperationResponse struct {
	Method   string `json:"method,omitempty"`
	BulkID   string `json:"bulkId,omitempty"`
	Version  string `json:"version,omitempty"`
	Location string `json:"location,omitempty"`
	Response any    `json:"response,omitempty"`
	Status   string `json:"status"`
}

// handleBulk handles bulk operations
func (s *Server) handleBulk(w http.ResponseWriter, r *http.Request, pluginName string) {
	if r.Method != http.MethodPost {
		s.handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalidMethod")
		return
	}

	var bulkReq BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&bulkReq); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid JSON", "invalidSyntax")
		return
	}

	// Validate schemas
	validSchema := slices.Contains(bulkReq.Schemas, SchemaBulkRequest)
	if !validSchema {
		s.handler.WriteError(w, http.StatusBadRequest, "Invalid schema", "invalidValue")
		return
	}

	// Validate for circular references (RFC 7644 Section 3.7.3)
	if err := validateBulkOperations(bulkReq.Operations); err != nil {
		s.handler.WriteError(w, http.StatusBadRequest, err.Error(), "invalidValue")
		return
	}

	// Get plugin
	plugin, ok := s.pluginManager.Get(pluginName)
	if !ok {
		s.handler.WriteError(w, http.StatusNotFound, fmt.Sprintf("Plugin '%s' not found", pluginName), "invalidPath")
		return
	}

	// Process operations
	bulkResp := BulkResponse{
		Schemas:    []string{SchemaBulkResponse},
		Operations: make([]BulkOperationResponse, 0, len(bulkReq.Operations)),
	}

	errorCount := 0
	bulkIDMap := make(map[string]string) // Maps bulkId to actual resource ID

	for _, op := range bulkReq.Operations {
		// Replace bulkId references in path
		path := op.Path
		for bulkID, resourceID := range bulkIDMap {
			path = strings.ReplaceAll(path, "bulkId:"+bulkID, resourceID)
		}

		// Replace bulkId references in operation data
		if op.Data != nil {
			if replaced := replaceBulkIdReferences(op.Data, bulkIDMap); replaced != nil {
				op.Data = replaced.(map[string]any)
			}
		}

		// Process operation
		opResp := s.processBulkOperation(r.Context(), plugin, pluginName, op, path, bulkIDMap)
		bulkResp.Operations = append(bulkResp.Operations, opResp)

		// Check error count
		if opResp.Status != "200" && opResp.Status != "201" && opResp.Status != "204" {
			errorCount++
			if bulkReq.FailOnErrors > 0 && errorCount >= bulkReq.FailOnErrors {
				break
			}
		}
	}

	s.handler.WriteJSON(w, http.StatusOK, bulkResp)
}

// processBulkOperation processes a single bulk operation
func (s *Server) processBulkOperation(ctx context.Context, plugin PluginGetter, pluginName string, op BulkOperation, path string, bulkIDMap map[string]string) BulkOperationResponse {
	resp := BulkOperationResponse{
		Method: op.Method,
		BulkID: op.BulkID,
	}

	// Parse path to determine resource type and ID
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 {
		resp.Status = "400"
		resp.Response = map[string]any{
			"detail": "Invalid path",
		}
		return resp
	}

	resourceType := parts[0]
	var resourceID string
	if len(parts) > 1 {
		resourceID = parts[1]
	}

	baseEntity := pluginName

	switch strings.ToUpper(op.Method) {
	case "POST":
		switch resourceType {
		case "Users":
			resp = s.bulkCreateUser(ctx, plugin, baseEntity, op, pluginName, bulkIDMap)
		case "Groups":
			resp = s.bulkCreateGroup(ctx, plugin, baseEntity, op, pluginName, bulkIDMap)
		}

	case "PUT":
		switch resourceType {
		case "Users":
			resp = s.bulkUpdateUser(ctx, plugin, baseEntity, resourceID, op)
		case "Groups":
			resp = s.bulkUpdateGroup(ctx, plugin, baseEntity, resourceID, op)
		}

	case "PATCH":
		switch resourceType {
		case "Users":
			resp = s.bulkPatchUser(ctx, plugin, baseEntity, resourceID, op)
		case "Groups":
			resp = s.bulkPatchGroup(ctx, plugin, baseEntity, resourceID, op)
		}

	case "DELETE":
		switch resourceType {
		case "Users":
			resp = s.bulkDeleteUser(ctx, plugin, baseEntity, resourceID, op)
		case "Groups":
			resp = s.bulkDeleteGroup(ctx, plugin, baseEntity, resourceID, op)
		}

	default:
		resp.Status = "400"
		resp.Response = map[string]any{
			"detail": "Invalid method",
		}
	}

	return resp
}

// Bulk operation helpers

func (s *Server) bulkCreateUser(ctx context.Context, plugin PluginGetter, baseEntity string, op BulkOperation, pluginName string, bulkIDMap map[string]string) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid user data"}
		return resp
	}

	created, err := plugin.CreateUser(ctx, baseEntity, &user)
	if err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	// Store bulkId mapping
	if op.BulkID != "" {
		bulkIDMap[op.BulkID] = created.ID
	}

	resp.Status = "201"
	resp.Location = s.handler.GetResourceLocation(pluginName, "Users", created.ID)
	resp.Response = created
	return resp
}

func (s *Server) bulkCreateGroup(ctx context.Context, plugin PluginGetter, baseEntity string, op BulkOperation, pluginName string, bulkIDMap map[string]string) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var group Group
	if err := json.Unmarshal(data, &group); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid group data"}
		return resp
	}

	created, err := plugin.CreateGroup(ctx, baseEntity, &group)
	if err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	if op.BulkID != "" {
		bulkIDMap[op.BulkID] = created.ID
	}

	resp.Status = "201"
	resp.Location = s.handler.GetResourceLocation(pluginName, "Groups", created.ID)
	resp.Response = created
	return resp
}

func (s *Server) bulkUpdateUser(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid user data"}
		return resp
	}

	patch := &PatchOp{
		Schemas:    []string{SchemaPatchOp},
		Operations: []PatchOperation{{Op: "replace", Value: user}},
	}

	if err := plugin.ModifyUser(ctx, baseEntity, id, patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "200"
	return resp
}

func (s *Server) bulkUpdateGroup(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var group Group
	if err := json.Unmarshal(data, &group); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid group data"}
		return resp
	}

	patch := &PatchOp{
		Schemas:    []string{SchemaPatchOp},
		Operations: []PatchOperation{{Op: "replace", Value: group}},
	}

	if err := plugin.ModifyGroup(ctx, baseEntity, id, patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "200"
	return resp
}

func (s *Server) bulkPatchUser(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var patch PatchOp
	if err := json.Unmarshal(data, &patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid patch data"}
		return resp
	}

	if err := plugin.ModifyUser(ctx, baseEntity, id, &patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "204"
	return resp
}

func (s *Server) bulkPatchGroup(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	data, _ := json.Marshal(op.Data)
	var patch PatchOp
	if err := json.Unmarshal(data, &patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": "Invalid patch data"}
		return resp
	}

	if err := plugin.ModifyGroup(ctx, baseEntity, id, &patch); err != nil {
		resp.Status = "400"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "204"
	return resp
}

func (s *Server) bulkDeleteUser(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	if err := plugin.DeleteUser(ctx, baseEntity, id); err != nil {
		resp.Status = "404"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "204"
	return resp
}

func (s *Server) bulkDeleteGroup(ctx context.Context, plugin PluginGetter, baseEntity, id string, op BulkOperation) BulkOperationResponse {
	resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}

	if err := plugin.DeleteGroup(ctx, baseEntity, id); err != nil {
		resp.Status = "404"
		resp.Response = map[string]any{"detail": err.Error()}
		return resp
	}

	resp.Status = "204"
	return resp
}

// extractBulkIdReferences recursively searches for bulkId references in operation data
// Returns a list of bulkIds that this operation depends on
func extractBulkIdReferences(data any) []string {
	var references []string

	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			// Check if this is a bulkId reference
			if key == "value" {
				if strVal, ok := val.(string); ok {
					if after, ok0 := strings.CutPrefix(strVal, "bulkId:"); ok0 {
						bulkId := after
						references = append(references, bulkId)
					}
				}
			}
			// Recursively check nested structures
			references = append(references, extractBulkIdReferences(val)...)
		}
	case []any:
		for _, item := range v {
			references = append(references, extractBulkIdReferences(item)...)
		}
	}

	return references
}

// buildDependencyGraph creates a directed graph of bulkId dependencies
// Returns: adjacency list (graph), bulkId to index mapping, error if duplicate bulkIds found
func buildDependencyGraph(operations []BulkOperation) (map[int][]int, map[string]int, error) {
	// Map bulkId to operation index
	bulkIdToIndex := make(map[string]int)
	for i, op := range operations {
		if op.BulkID != "" {
			if _, exists := bulkIdToIndex[op.BulkID]; exists {
				return nil, nil, fmt.Errorf("duplicate bulkId: %s", op.BulkID)
			}
			bulkIdToIndex[op.BulkID] = i
		}
	}

	// Build adjacency list: graph[i] = list of operations that operation i depends on
	graph := make(map[int][]int)
	for i, op := range operations {
		references := extractBulkIdReferences(op.Data)
		for _, bulkId := range references {
			if depIndex, exists := bulkIdToIndex[bulkId]; exists {
				graph[i] = append(graph[i], depIndex)
			}
			// Note: References to non-existent bulkIds will be caught during execution
		}
	}

	return graph, bulkIdToIndex, nil
}

// detectCycle uses DFS with coloring to detect cycles in dependency graph
// Returns: (hasCycle bool, cycle []int with operation indices in cycle)
func detectCycle(graph map[int][]int, numNodes int) (bool, []int) {
	const (
		white = 0 // not visited
		gray  = 1 // visiting (in current DFS path)
		black = 2 // visited (completed)
	)

	color := make([]int, numNodes)
	parent := make([]int, numNodes)
	for i := range parent {
		parent[i] = -1
	}

	var dfs func(node int) (bool, []int)
	dfs = func(node int) (bool, []int) {
		color[node] = gray

		for _, neighbor := range graph[node] {
			if color[neighbor] == gray {
				// Found cycle - build path
				cycle := []int{neighbor}
				current := node
				for current != neighbor && current != -1 {
					cycle = append(cycle, current)
					current = parent[current]
				}
				cycle = append(cycle, neighbor)
				return true, cycle
			}

			if color[neighbor] == white {
				parent[neighbor] = node
				if found, cycle := dfs(neighbor); found {
					return true, cycle
				}
			}
		}

		color[node] = black
		return false, nil
	}

	// Try DFS from each unvisited node
	for i := range numNodes {
		if color[i] == white {
			if found, cycle := dfs(i); found {
				return true, cycle
			}
		}
	}

	return false, nil
}

// validateBulkOperations checks for circular dependencies per RFC 7644 Section 3.7.3
// Returns error if circular bulkId references are detected
func validateBulkOperations(operations []BulkOperation) error {
	if len(operations) == 0 {
		return nil
	}

	// Build dependency graph
	graph, bulkIdToIndex, err := buildDependencyGraph(operations)
	if err != nil {
		return err
	}

	// Detect cycles
	hasCycle, cycle := detectCycle(graph, len(operations))
	if hasCycle {
		// Build error message with bulkIds in cycle
		cycleIds := make([]string, 0, len(cycle))
		for _, idx := range cycle {
			// Find the bulkId for this index
			for bulkId, bulkIdx := range bulkIdToIndex {
				if bulkIdx == idx {
					cycleIds = append(cycleIds, bulkId)
					break
				}
			}
		}
		return fmt.Errorf("circular bulkId reference detected: %s", strings.Join(cycleIds, " → "))
	}

	return nil
}

// replaceBulkIdReferences recursively replaces bulkId references in operation data
// with actual resource IDs from the bulkIDMap
func replaceBulkIdReferences(data any, bulkIDMap map[string]string) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, val := range v {
			// Check if this is a bulkId reference value
			if key == "value" {
				if strVal, ok := val.(string); ok {
					if after, found := strings.CutPrefix(strVal, "bulkId:"); found {
						// Replace with actual resource ID if available
						if resourceID, exists := bulkIDMap[after]; exists {
							result[key] = resourceID
							continue
						}
					}
				}
			}
			// Recursively process nested structures
			result[key] = replaceBulkIdReferences(val, bulkIDMap)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = replaceBulkIdReferences(item, bulkIDMap)
		}
		return result
	default:
		return v
	}
}
