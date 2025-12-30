package plugin

import (
	"context"
	"sync"

	"github.com/marcelom97/scimgateway/auth"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/scim"
)

// Plugin defines the interface that all SCIM plugins must implement.
//
// Thread Safety:
// Plugin implementations must be thread-safe as they will be called concurrently
// by multiple HTTP request handlers.
//
// Design Philosophy:
//   - Plugins return raw/complete resources from the backend
//   - The adapter layer applies SCIM protocol operations (filtering, pagination, sorting)
//   - This separation keeps plugins simple and focused on backend integration
//
// Query Optimization:
//   - GetUsers/GetGroups: Can return ALL data (simple) or optimize using params.Filter
//   - GetUser/GetGroup: 'attributes' parameter is an optimization hint only
//     The server layer handles both 'attributes' and 'excludedAttributes' query parameters
//
// Error Handling:
//   - Use scim.ErrNotFound() for missing resources (becomes HTTP 404)
//   - Use scim.ErrUniqueness() for duplicate keys (becomes HTTP 409)
//   - Use scim.ErrInternalServer() for backend errors (becomes HTTP 500)
//   - See scim/errors.go for the complete list of error constructors
//
// See PLUGIN_DEVELOPMENT.md for detailed plugin development guide.
type Plugin interface {
	// Name returns the unique plugin identifier used in URL paths.
	// Example: If Name() returns "ldap", URLs will be /ldap/Users, /ldap/Groups, etc.
	Name() string

	// GetUsers retrieves users from the backend.
	//
	// Simple Implementation:
	//   Return all users. The adapter will apply filtering, pagination, and sorting.
	//
	// Optimized Implementation:
	//   Process params.Filter natively in your backend (e.g., convert to SQL WHERE clause).
	//   This can dramatically improve performance with large datasets.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and timeouts. Always respect ctx.Done().
	//   - params: Query parameters containing:
	//       * Filter: SCIM filter expression (e.g., "userName eq \"john\"")
	//       * StartIndex/Count: Pagination parameters (1-based index)
	//       * SortBy/SortOrder: Sorting parameters
	//       * Attributes: Requested attributes for optimization
	//
	// Returns:
	//   - Slice of users matching the query (or all users if not optimizing)
	//   - Error if backend operation fails
	GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error)

	// CreateUser creates a new user in the backend.
	//
	// Implementation Requirements:
	//   - Generate user.ID if not provided (e.g., using uuid.New())
	//   - Set user.Schemas if not provided (default: []string{scim.SchemaUser})
	//   - Set user.Meta with Created, LastModified, Version, ResourceType
	//   - Return scim.ErrUniqueness() if userName already exists
	//
	// The created user is returned with all metadata populated.
	CreateUser(ctx context.Context, user *scim.User) (*scim.User, error)

	// GetUser retrieves a specific user by ID.
	//
	// The attributes parameter is an OPTIMIZATION HINT. Plugins can use it to
	// fetch partial resources, reducing I/O:
	//
	//   Example SQL optimization:
	//     if len(attributes) > 0 {
	//         cols := mapAttributesToColumns(attributes)  // ["id", "username", "emails"]
	//         query = "SELECT " + strings.Join(cols, ",") + " FROM users WHERE id = ?"
	//     }
	//
	// Important Notes:
	//   - Return the full resource if optimization is not feasible
	//   - The server handles 'excludedAttributes' query parameter via post-processing
	//   - Only 'attributes' (positive selection) is passed to plugins because:
	//       * Easier to optimize without knowing the complete schema
	//       * Avoids tight coupling between plugin and SCIM schema evolution
	//       * Server-layer filtering is fast enough for typical payloads
	//
	// Returns:
	//   - The requested user with all or selected attributes
	//   - scim.ErrNotFound() if user doesn't exist
	GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error)

	// ModifyUser applies PATCH operations to update a user's attributes.
	//
	// Implementation:
	//   1. Retrieve the current user (return ErrNotFound if missing)
	//   2. Apply patch operations using scim.NewPatchProcessor().ApplyPatch()
	//   3. Update user.Meta.LastModified and user.Meta.Version
	//   4. Save the modified user to backend
	//
	// The patch parameter contains Operations (add, remove, replace) with paths and values.
	// The PatchProcessor handles all RFC 7644 PATCH semantics automatically.
	//
	// Returns error if user not found or patch application fails.
	ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error

	// DeleteUser deletes a user from the backend.
	//
	// Returns scim.ErrNotFound() if the user doesn't exist.
	DeleteUser(ctx context.Context, id string) error

	// GetGroups retrieves groups from the backend.
	// See GetUsers documentation for details - same pattern applies.
	GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error)

	// CreateGroup creates a new group in the backend.
	// See CreateUser documentation for details - same pattern applies.
	CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error)

	// GetGroup retrieves a specific group by ID.
	// See GetUser documentation for details - same pattern applies.
	GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error)

	// ModifyGroup applies PATCH operations to update a group's attributes.
	// See ModifyUser documentation for details - same pattern applies.
	ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error

	// DeleteGroup deletes a group from the backend.
	// See DeleteUser documentation for details - same pattern applies.
	DeleteGroup(ctx context.Context, id string) error
}

// Manager manages multiple plugins and their authentication.
//
// Thread Safety:
// Manager is thread-safe and can be accessed concurrently from multiple goroutines.
// All methods use sync.RWMutex to protect concurrent access to internal maps.
//
// Typical Usage:
//
//	manager := plugin.NewManager()
//	manager.Register(myPlugin, pluginConfig)  // Register plugins before gateway starts
//	plugin, ok := manager.Get("myPlugin")     // Lookup during request handling
//
// Note: While Manager supports concurrent access, plugins are typically registered
// during application startup before the HTTP server starts serving requests.
type Manager struct {
	plugins        map[string]Plugin
	authenticators map[string]auth.Authenticator
	mu             sync.RWMutex // Protects concurrent access to both maps
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins:        make(map[string]Plugin),
		authenticators: make(map[string]auth.Authenticator),
	}
}

// Register registers a plugin with its configuration
func (m *Manager) Register(plugin Plugin, cfg *config.PluginConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins[plugin.Name()] = plugin

	// Clear any existing authenticator first
	delete(m.authenticators, plugin.Name())

	// Setup authentication from config if provided
	if cfg != nil && cfg.Auth != nil {
		authenticator := createAuthenticator(cfg.Auth)
		if authenticator != nil {
			m.authenticators[plugin.Name()] = authenticator
		}
	}
}

// createAuthenticator creates an authenticator from config
func createAuthenticator(authCfg *config.AuthConfig) auth.Authenticator {
	switch authCfg.Type {
	case "basic":
		if authCfg.Basic != nil {
			return auth.NewBasicAuthenticator(authCfg.Basic.Username, authCfg.Basic.Password)
		}
	case "bearer":
		if authCfg.Bearer != nil {
			return auth.NewBearerAuthenticator(authCfg.Bearer.Token)
		}
	case "none", "":
		return nil
	}
	return nil
}

// GetAuthenticator retrieves the authenticator for a plugin by name
func (m *Manager) GetAuthenticator(name string) (auth.Authenticator, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	authenticator, ok := m.authenticators[name]
	return authenticator, ok
}

// Get retrieves a plugin by name
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, ok := m.plugins[name]
	return plugin, ok
}

// List returns all registered plugin names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}
