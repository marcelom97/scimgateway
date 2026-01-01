// Package testutil provides test utilities for the scimgateway project.
// This package is internal and not part of the public API.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/marcelom97/scimgateway/scim"
)

// MemoryPlugin is a full in-memory plugin implementation for testing.
// This is used in tests that require actual data storage (auth tests, integration tests, etc.).
//
// Note: This is a test utility and NOT intended for production use.
// For production, use a real database-backed plugin.
type MemoryPlugin struct {
	name   string
	users  map[string]*scim.User
	groups map[string]*scim.Group
	mu     sync.RWMutex
}

// NewMemoryPlugin creates a new in-memory plugin for testing.
func NewMemoryPlugin(name string) *MemoryPlugin {
	return &MemoryPlugin{
		name:   name,
		users:  make(map[string]*scim.User),
		groups: make(map[string]*scim.Group),
	}
}

// Name returns the plugin name.
func (p *MemoryPlugin) Name() string {
	return p.name
}

// GetUsers retrieves all users.
// Note: The adapter layer handles filtering, pagination, and attribute selection.
func (p *MemoryPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allUsers := make([]*scim.User, 0, len(p.users))
	for _, user := range p.users {
		allUsers = append(allUsers, user)
	}
	return allUsers, nil
}

// CreateUser creates a new user.
func (p *MemoryPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if len(user.Schemas) == 0 {
		user.Schemas = []string{scim.SchemaUser}
	}

	now := time.Now()
	user.Meta = &scim.Meta{
		ResourceType: "User",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", user.ID),
	}

	p.users[user.ID] = user
	return user, nil
}

// GetUser retrieves a specific user by ID.
func (p *MemoryPlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	user, ok := p.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ModifyUser updates a user's attributes using PATCH operations.
func (p *MemoryPlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	user, ok := p.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}

	patcher := scim.NewPatchProcessor()
	err := patcher.ApplyPatch(user, patch)
	if err != nil {
		return err
	}

	now := time.Now()
	user.Meta.LastModified = &now
	user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	return nil
}

// DeleteUser deletes a user.
func (p *MemoryPlugin) DeleteUser(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(p.users, id)
	return nil
}

// GetGroups retrieves all groups.
// Note: The adapter layer handles filtering, pagination, and attribute selection.
func (p *MemoryPlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allGroups := make([]*scim.Group, 0, len(p.groups))
	for _, group := range p.groups {
		allGroups = append(allGroups, group)
	}
	return allGroups, nil
}

// CreateGroup creates a new group.
func (p *MemoryPlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	if len(group.Schemas) == 0 {
		group.Schemas = []string{scim.SchemaGroup}
	}

	now := time.Now()
	group.Meta = &scim.Meta{
		ResourceType: "Group",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", group.ID),
	}

	p.groups[group.ID] = group
	return group, nil
}

// GetGroup retrieves a specific group by ID.
func (p *MemoryPlugin) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	group, ok := p.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	return group, nil
}

// ModifyGroup updates a group's attributes using PATCH operations.
func (p *MemoryPlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	group, ok := p.groups[id]
	if !ok {
		return fmt.Errorf("group not found")
	}

	patcher := scim.NewPatchProcessor()
	err := patcher.ApplyPatch(group, patch)
	if err != nil {
		return err
	}

	now := time.Now()
	group.Meta.LastModified = &now
	group.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	return nil
}

// DeleteGroup deletes a group.
func (p *MemoryPlugin) DeleteGroup(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.groups[id]; !ok {
		return fmt.Errorf("group not found")
	}
	delete(p.groups, id)
	return nil
}
