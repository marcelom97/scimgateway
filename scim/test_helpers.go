package scim

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// mockPlugin is a simple in-memory plugin for testing
type mockPlugin struct {
	users  map[string]*User
	groups map[string]*Group
	mu     sync.RWMutex
}

func newMockPlugin() *mockPlugin {
	return &mockPlugin{
		users:  make(map[string]*User),
		groups: make(map[string]*Group),
	}
}

func (m *mockPlugin) GetUsers(ctx context.Context, params QueryParams) (*ListResponse[*User], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}

	// Apply filter if provided
	if params.Filter != "" {
		filtered, err := FilterByFilter(users, params.Filter)
		if err != nil {
			return nil, err
		}
		users = filtered
	}

	return &ListResponse[*User]{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(users),
		Resources:    users,
	}, nil
}

func (m *mockPlugin) CreateUser(ctx context.Context, user *User) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if len(user.Schemas) == 0 {
		user.Schemas = []string{SchemaUser}
	}

	m.users[user.ID] = user
	return user, nil
}

func (m *mockPlugin) GetUser(ctx context.Context, id string, attributes []string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *mockPlugin) ModifyUser(ctx context.Context, id string, patch *PatchOp) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}

	patcher := NewPatchProcessor()
	return patcher.ApplyPatch(user, patch)
}

func (m *mockPlugin) DeleteUser(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(m.users, id)
	return nil
}

func (m *mockPlugin) GetGroups(ctx context.Context, params QueryParams) (*ListResponse[*Group], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make([]*Group, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}

	// Apply filter if provided
	if params.Filter != "" {
		filtered, err := FilterByFilter(groups, params.Filter)
		if err != nil {
			return nil, err
		}
		groups = filtered
	}

	return &ListResponse[*Group]{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(groups),
		Resources:    groups,
	}, nil
}

func (m *mockPlugin) CreateGroup(ctx context.Context, group *Group) (*Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	if len(group.Schemas) == 0 {
		group.Schemas = []string{SchemaGroup}
	}

	m.groups[group.ID] = group
	return group, nil
}

func (m *mockPlugin) GetGroup(ctx context.Context, id string, attributes []string) (*Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	return group, nil
}

func (m *mockPlugin) ModifyGroup(ctx context.Context, id string, patch *PatchOp) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[id]
	if !ok {
		return fmt.Errorf("group not found")
	}

	patcher := NewPatchProcessor()
	return patcher.ApplyPatch(group, patch)
}

func (m *mockPlugin) DeleteGroup(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[id]; !ok {
		return fmt.Errorf("group not found")
	}
	delete(m.groups, id)
	return nil
}

// mockPluginManager is a mock plugin manager for testing
type mockPluginManager struct {
	plugin PluginGetter
}

func (m *mockPluginManager) Get(name string) (PluginGetter, bool) {
	if m.plugin != nil {
		return m.plugin, true
	}
	return nil, false
}

func (m *mockPluginManager) List() []string {
	return []string{"test"}
}
