package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/marcelom97/scimgateway/scim"
)

// Plugin implements an in-memory SCIM plugin
type Plugin struct {
	name   string
	users  map[string]*scim.User
	groups map[string]*scim.Group
	mu     sync.RWMutex
}

// New creates a new in-memory plugin
func New(name string) *Plugin {
	return &Plugin{
		name:   name,
		users:  make(map[string]*scim.User),
		groups: make(map[string]*scim.Group),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return p.name
}

// GetUsers retrieves all users
// Note: The adapter layer handles filtering, pagination, and attribute selection
func (p *Plugin) GetUsers(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return all users - adapter will apply SCIM operations
	allUsers := make([]*scim.User, 0, len(p.users))
	for _, user := range p.users {
		allUsers = append(allUsers, user)
	}

	return allUsers, nil
}

// CreateUser creates a new user
func (p *Plugin) CreateUser(ctx context.Context, baseEntity string, user *scim.User) (*scim.User, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Generate ID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Set schemas if not provided
	if len(user.Schemas) == 0 {
		user.Schemas = []string{scim.SchemaUser}
	}

	// Set meta
	now := time.Now()
	user.Meta = &scim.Meta{
		ResourceType: "User",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", user.ID),
	}

	// Store user
	p.users[user.ID] = user

	return user, nil
}

// GetUser retrieves a specific user by ID
func (p *Plugin) GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	user, ok := p.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}

	// Note: Attribute selection is handled by the server layer
	// The attributes parameter is provided for plugins that need to optimize queries
	// For in-memory plugin, we just return the full user
	return user, nil
}

// ModifyUser updates a user's attributes
func (p *Plugin) ModifyUser(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	user, ok := p.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}

	// Apply patch operations
	patcher := scim.NewPatchProcessor()
	err := patcher.ApplyPatch(user, patch)
	if err != nil {
		return err
	}

	// Update user metadata
	now := time.Now()
	user.Meta.LastModified = &now
	user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	return nil
}

// DeleteUser deletes a user
func (p *Plugin) DeleteUser(ctx context.Context, baseEntity string, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.users[id]; !ok {
		return fmt.Errorf("user not found")
	}

	delete(p.users, id)
	return nil
}

// GetGroups retrieves all groups
// Note: The adapter layer handles filtering, pagination, and attribute selection
func (p *Plugin) GetGroups(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return all groups - adapter will apply SCIM operations
	allGroups := make([]*scim.Group, 0, len(p.groups))
	for _, group := range p.groups {
		allGroups = append(allGroups, group)
	}

	return allGroups, nil
}

// CreateGroup creates a new group
func (p *Plugin) CreateGroup(ctx context.Context, baseEntity string, group *scim.Group) (*scim.Group, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Generate ID if not provided
	if group.ID == "" {
		group.ID = uuid.New().String()
	}

	// Set schemas if not provided
	if len(group.Schemas) == 0 {
		group.Schemas = []string{scim.SchemaGroup}
	}

	// Set meta
	now := time.Now()
	group.Meta = &scim.Meta{
		ResourceType: "Group",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", group.ID),
	}

	// Store group
	p.groups[group.ID] = group

	return group, nil
}

// GetGroup retrieves a specific group by ID
func (p *Plugin) GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	group, ok := p.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}

	// Note: Attribute selection is handled by the server layer
	// The attributes parameter is provided for plugins that need to optimize queries
	// For in-memory plugin, we just return the full group
	return group, nil
}

// ModifyGroup updates a group's attributes
func (p *Plugin) ModifyGroup(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	group, ok := p.groups[id]
	if !ok {
		return fmt.Errorf("group not found")
	}

	// Apply patch operations
	patcher := scim.NewPatchProcessor()
	err := patcher.ApplyPatch(group, patch)
	if err != nil {
		return err
	}

	// Update group metadata
	now := time.Now()
	group.Meta.LastModified = &now
	group.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	return nil
}

// DeleteGroup deletes a group
func (p *Plugin) DeleteGroup(ctx context.Context, baseEntity string, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.groups[id]; !ok {
		return fmt.Errorf("group not found")
	}

	delete(p.groups, id)
	return nil
}
