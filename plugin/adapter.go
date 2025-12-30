package plugin

import (
	"context"

	"github.com/marcelom97/scimgateway/scim"
)

// Adapter adapts the plugin interface to the scim.PluginGetter interface
type Adapter struct {
	plugin Plugin
}

// NewAdapter creates a new plugin adapter
func NewAdapter(plugin Plugin) *Adapter {
	return &Adapter{plugin: plugin}
}

// GetUsers implements scim.PluginGetter
// The adapter applies SCIM protocol operations (filtering, pagination, attribute selection)
func (a *Adapter) GetUsers(ctx context.Context, params scim.QueryParams) (*scim.ListResponse[*scim.User], error) {
	// Get raw data from plugin
	users, err := a.plugin.GetUsers(ctx, params)
	if err != nil {
		return nil, err
	}

	// Apply SCIM query operations internally
	return scim.ProcessListQuery(users, params)
}

// CreateUser implements scim.PluginGetter
func (a *Adapter) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
	return a.plugin.CreateUser(ctx, user)
}

// GetUser implements scim.PluginGetter
func (a *Adapter) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
	return a.plugin.GetUser(ctx, id, attributes)
}

// ModifyUser implements scim.PluginGetter
func (a *Adapter) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
	return a.plugin.ModifyUser(ctx, id, patch)
}

// DeleteUser implements scim.PluginGetter
func (a *Adapter) DeleteUser(ctx context.Context, id string) error {
	return a.plugin.DeleteUser(ctx, id)
}

// GetGroups implements scim.PluginGetter
// The adapter applies SCIM protocol operations (filtering, pagination, attribute selection)
func (a *Adapter) GetGroups(ctx context.Context, params scim.QueryParams) (*scim.ListResponse[*scim.Group], error) {
	// Get raw data from plugin
	groups, err := a.plugin.GetGroups(ctx, params)
	if err != nil {
		return nil, err
	}

	// Apply SCIM query operations internally
	return scim.ProcessListQuery(groups, params)
}

// CreateGroup implements scim.PluginGetter
func (a *Adapter) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
	return a.plugin.CreateGroup(ctx, group)
}

// GetGroup implements scim.PluginGetter
func (a *Adapter) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error) {
	return a.plugin.GetGroup(ctx, id, attributes)
}

// ModifyGroup implements scim.PluginGetter
func (a *Adapter) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
	return a.plugin.ModifyGroup(ctx, id, patch)
}

// DeleteGroup implements scim.PluginGetter
func (a *Adapter) DeleteGroup(ctx context.Context, id string) error {
	return a.plugin.DeleteGroup(ctx, id)
}

// AdaptedManager wraps Manager to provide adapted plugins
type AdaptedManager struct {
	manager *Manager
}

// NewAdaptedManager creates a new adapted manager
func NewAdaptedManager(manager *Manager) *AdaptedManager {
	return &AdaptedManager{manager: manager}
}

// Get retrieves an adapted plugin by name
func (am *AdaptedManager) Get(name string) (scim.PluginGetter, bool) {
	plugin, ok := am.manager.Get(name)
	if !ok {
		return nil, false
	}
	return NewAdapter(plugin), true
}

// List returns all registered plugin names
func (am *AdaptedManager) List() []string {
	return am.manager.List()
}
