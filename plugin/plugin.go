package plugin

import (
	"context"

	"github.com/marcelom97/scimgateway/auth"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/scim"
)

// Plugin defines the interface that all SCIM plugins must implement
// Note: Plugins return raw data. The adapter layer handles SCIM protocol operations
// (filtering, pagination, attribute selection) via ProcessListQuery.
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// GetUsers retrieves all users. The adapter layer will apply filtering, pagination,
	// and attribute selection based on params. Plugins can optionally use params to
	// optimize queries (e.g., convert filters to SQL WHERE clauses).
	GetUsers(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.User, error)

	// CreateUser creates a new user
	CreateUser(ctx context.Context, baseEntity string, user *scim.User) (*scim.User, error)

	// GetUser retrieves a specific user by ID
	GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.User, error)

	// ModifyUser updates a user's attributes
	ModifyUser(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error

	// DeleteUser deletes a user
	DeleteUser(ctx context.Context, baseEntity string, id string) error

	// GetGroups retrieves all groups. The adapter layer will apply filtering, pagination,
	// and attribute selection based on params. Plugins can optionally use params to
	// optimize queries (e.g., convert filters to SQL WHERE clauses).
	GetGroups(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.Group, error)

	// CreateGroup creates a new group
	CreateGroup(ctx context.Context, baseEntity string, group *scim.Group) (*scim.Group, error)

	// GetGroup retrieves a specific group by ID
	GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.Group, error)

	// ModifyGroup updates a group's attributes
	ModifyGroup(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error

	// DeleteGroup deletes a group
	DeleteGroup(ctx context.Context, baseEntity string, id string) error
}

// Manager manages multiple plugins and their authentication
type Manager struct {
	plugins        map[string]Plugin
	authenticators map[string]auth.Authenticator
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
	authenticator, ok := m.authenticators[name]
	return authenticator, ok
}

// Get retrieves a plugin by name
func (m *Manager) Get(name string) (Plugin, bool) {
	plugin, ok := m.plugins[name]
	return plugin, ok
}

// List returns all registered plugin names
func (m *Manager) List() []string {
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}
