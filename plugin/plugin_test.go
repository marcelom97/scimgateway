package plugin

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/scim"
)

// mockPlugin for testing
type mockPlugin struct {
	name string
}

func (p *mockPlugin) Name() string { return p.name }

func (p *mockPlugin) GetUsers(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.User, error) {
	return []*scim.User{}, nil
}

func (p *mockPlugin) CreateUser(ctx context.Context, baseEntity string, user *scim.User) (*scim.User, error) {
	return user, nil
}

func (p *mockPlugin) GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.User, error) {
	return &scim.User{ID: id, UserName: "test"}, nil
}

func (p *mockPlugin) ModifyUser(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return nil
}

func (p *mockPlugin) DeleteUser(ctx context.Context, baseEntity string, id string) error {
	return nil
}

func (p *mockPlugin) GetGroups(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.Group, error) {
	return []*scim.Group{}, nil
}

func (p *mockPlugin) CreateGroup(ctx context.Context, baseEntity string, group *scim.Group) (*scim.Group, error) {
	return group, nil
}

func (p *mockPlugin) GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.Group, error) {
	return &scim.Group{ID: id, DisplayName: "test"}, nil
}

func (p *mockPlugin) ModifyGroup(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return nil
}

func (p *mockPlugin) DeleteGroup(ctx context.Context, baseEntity string, id string) error {
	return nil
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if m.plugins == nil {
		t.Error("plugins map not initialized")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager()
	p := &mockPlugin{name: "test"}

	m.Register(p, nil)

	if len(m.plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(m.plugins))
	}

	if _, exists := m.plugins["test"]; !exists {
		t.Error("Plugin not registered with correct name")
	}
}

func TestManagerRegisterMultiple(t *testing.T) {
	m := NewManager()
	p1 := &mockPlugin{name: "plugin1"}
	p2 := &mockPlugin{name: "plugin2"}
	p3 := &mockPlugin{name: "plugin3"}

	m.Register(p1, nil)
	m.Register(p2, nil)
	m.Register(p3, nil)

	if len(m.plugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(m.plugins))
	}
}

func TestManagerRegisterOverwrite(t *testing.T) {
	m := NewManager()
	p1 := &mockPlugin{name: "test"}
	p2 := &mockPlugin{name: "test"}

	m.Register(p1, nil)
	m.Register(p2, nil)

	if len(m.plugins) != 1 {
		t.Errorf("Expected 1 plugin after overwrite, got %d", len(m.plugins))
	}

	// Should have the second plugin
	if m.plugins["test"] != p2 {
		t.Error("Plugin not overwritten correctly")
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager()
	p := &mockPlugin{name: "test"}
	m.Register(p, nil)

	retrieved, ok := m.Get("test")
	if !ok {
		t.Fatal("Get() returned false for existing plugin")
	}

	if retrieved != p {
		t.Error("Get() returned wrong plugin")
	}
}

func TestManagerGetNonexistent(t *testing.T) {
	m := NewManager()

	_, ok := m.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent plugin")
	}
}

func TestManagerList(t *testing.T) {
	m := NewManager()

	// Empty list
	list := m.List()
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}

	// Add plugins
	p1 := &mockPlugin{name: "alpha"}
	p2 := &mockPlugin{name: "beta"}
	p3 := &mockPlugin{name: "gamma"}

	m.Register(p1, nil)
	m.Register(p2, nil)
	m.Register(p3, nil)

	list = m.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(list))
	}

	// Check that all names are present
	names := make(map[string]bool)
	for _, name := range list {
		names[name] = true
	}

	if !names["alpha"] || !names["beta"] || !names["gamma"] {
		t.Error("List() missing expected plugin names")
	}
}

// TestManager_ConcurrentAccess verifies thread-safe access to Manager
func TestManager_ConcurrentAccess(t *testing.T) {
	manager := NewManager()
	plugin1 := &mockPlugin{name: "plugin1"}
	plugin2 := &mockPlugin{name: "plugin2"}

	pluginCfg1 := &config.PluginConfig{
		Name: "plugin1",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token1"},
		},
	}

	pluginCfg2 := &config.PluginConfig{
		Name: "plugin2",
		Auth: &config.AuthConfig{
			Type:  "basic",
			Basic: &config.BasicAuth{Username: "user", Password: "pass"},
		},
	}

	// Register initial plugins
	manager.Register(plugin1, pluginCfg1)
	manager.Register(plugin2, pluginCfg2)

	// Run concurrent operations
	var wg sync.WaitGroup
	errChan := make(chan error, 200)

	// Concurrent reads via Get()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pluginName := "plugin1"
			if n%2 == 0 {
				pluginName = "plugin2"
			}
			if _, ok := manager.Get(pluginName); !ok {
				errChan <- fmt.Errorf("failed to get %s", pluginName)
			}
		}(i)
	}

	// Concurrent reads via GetAuthenticator()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pluginName := "plugin1"
			if n%2 == 0 {
				pluginName = "plugin2"
			}
			if _, ok := manager.GetAuthenticator(pluginName); !ok {
				errChan <- fmt.Errorf("failed to get authenticator for %s", pluginName)
			}
		}(i)
	}

	// Concurrent List() operations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			list := manager.List()
			if len(list) != 2 {
				errChan <- fmt.Errorf("expected 2 plugins, got %d", len(list))
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}
}

// ============================================================================
// Manager Authentication Tests
// ============================================================================

func TestManager_RegisterWithAuth(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "test"}
	pluginCfg := &config.PluginConfig{
		Name: "test",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token"},
		},
	}

	manager.Register(plugin, pluginCfg)

	// Verify plugin is registered
	p, ok := manager.Get("test")
	if !ok {
		t.Error("Expected plugin to be registered")
	}
	if p.Name() != "test" {
		t.Errorf("Expected plugin name 'test', got '%s'", p.Name())
	}

	// Verify authenticator is registered
	a, ok := manager.GetAuthenticator("test")
	if !ok {
		t.Error("Expected authenticator to be registered")
	}
	if a == nil {
		t.Error("Expected non-nil authenticator")
	}
}

func TestManager_RegisterWithAuth_NilAuth(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "test"}

	manager.Register(plugin, nil)

	// Verify plugin is registered
	_, ok := manager.Get("test")
	if !ok {
		t.Error("Expected plugin to be registered")
	}

	// Verify authenticator is NOT registered (nil was passed)
	_, ok = manager.GetAuthenticator("test")
	if ok {
		t.Error("Expected no authenticator for nil auth")
	}
}

func TestManager_Register_NoAuth(t *testing.T) {
	manager := NewManager()
	plugin := &mockPlugin{name: "test"}

	manager.Register(plugin, nil)

	// Verify plugin is registered
	_, ok := manager.Get("test")
	if !ok {
		t.Error("Expected plugin to be registered")
	}

	// Verify no authenticator is registered
	_, ok = manager.GetAuthenticator("test")
	if ok {
		t.Error("Expected no authenticator for Register() without auth")
	}
}

func TestManager_GetAuthenticator_NotFound(t *testing.T) {
	manager := NewManager()

	_, ok := manager.GetAuthenticator("nonexistent")
	if ok {
		t.Error("Expected GetAuthenticator to return false for nonexistent plugin")
	}
}

func TestManager_MultiplePlugins_DifferentAuth(t *testing.T) {
	manager := NewManager()

	// Plugin 1 with bearer auth
	plugin1 := &mockPlugin{name: "plugin1"}
	pluginCfg1 := &config.PluginConfig{
		Name: "plugin1",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token1"},
		},
	}
	manager.Register(plugin1, pluginCfg1)

	// Plugin 2 with basic auth
	plugin2 := &mockPlugin{name: "plugin2"}
	pluginCfg2 := &config.PluginConfig{
		Name: "plugin2",
		Auth: &config.AuthConfig{
			Type:  "basic",
			Basic: &config.BasicAuth{Username: "user", Password: "pass"},
		},
	}
	manager.Register(plugin2, pluginCfg2)

	// Plugin 3 without auth
	plugin3 := &mockPlugin{name: "plugin3"}
	manager.Register(plugin3, nil)

	// Verify plugin 1
	p1, ok := manager.Get("plugin1")
	if !ok {
		t.Error("Expected plugin1 to be registered")
	}
	if p1.Name() != "plugin1" {
		t.Errorf("Expected plugin1 name, got %s", p1.Name())
	}
	a1, ok := manager.GetAuthenticator("plugin1")
	if !ok {
		t.Error("Expected plugin1 to have authenticator")
	}
	if a1 == nil {
		t.Error("Expected plugin1 authenticator to be non-nil")
	}

	// Verify plugin 2
	p2, ok := manager.Get("plugin2")
	if !ok {
		t.Error("Expected plugin2 to be registered")
	}
	if p2.Name() != "plugin2" {
		t.Errorf("Expected plugin2 name, got %s", p2.Name())
	}
	a2, ok := manager.GetAuthenticator("plugin2")
	if !ok {
		t.Error("Expected plugin2 to have authenticator")
	}
	if a2 == nil {
		t.Error("Expected plugin2 authenticator to be non-nil")
	}

	// Verify plugin 3
	p3, ok := manager.Get("plugin3")
	if !ok {
		t.Error("Expected plugin3 to be registered")
	}
	if p3.Name() != "plugin3" {
		t.Errorf("Expected plugin3 name, got %s", p3.Name())
	}
	_, ok = manager.GetAuthenticator("plugin3")
	if ok {
		t.Error("Expected plugin3 to NOT have authenticator")
	}

	// Verify authenticators are different instances
	if a1 == a2 {
		t.Error("Expected plugin1 and plugin2 to have different authenticator instances")
	}
}

func TestManager_List_WithAuth(t *testing.T) {
	manager := NewManager()

	plugin1 := &mockPlugin{name: "plugin1"}
	pluginCfg1 := &config.PluginConfig{
		Name: "plugin1",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token"},
		},
	}
	manager.Register(plugin1, pluginCfg1)

	plugin2 := &mockPlugin{name: "plugin2"}
	manager.Register(plugin2, nil)

	names := manager.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(names))
	}

	// Check both plugin names are in the list
	found1, found2 := false, false
	for _, name := range names {
		if name == "plugin1" {
			found1 = true
		}
		if name == "plugin2" {
			found2 = true
		}
	}

	if !found1 {
		t.Error("Expected plugin1 in list")
	}
	if !found2 {
		t.Error("Expected plugin2 in list")
	}
}

func TestManager_OverwritePlugin(t *testing.T) {
	manager := NewManager()

	// Register plugin with auth
	plugin1 := &mockPlugin{name: "test"}
	pluginCfg1 := &config.PluginConfig{
		Name: "test",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token1"},
		},
	}
	manager.Register(plugin1, pluginCfg1)

	// Re-register with different auth
	plugin2 := &mockPlugin{name: "test"}
	pluginCfg2 := &config.PluginConfig{
		Name: "test",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token2"},
		},
	}
	manager.Register(plugin2, pluginCfg2)

	// Should have the second plugin and auth
	p, ok := manager.Get("test")
	if !ok {
		t.Error("Expected plugin to be registered")
	}
	if p != plugin2 {
		t.Error("Expected second plugin instance")
	}

	_, ok = manager.GetAuthenticator("test")
	if !ok {
		t.Error("Expected authenticator to be registered")
	}
}

func TestManager_RegisterWithAuth_ThenRegisterWithoutAuth(t *testing.T) {
	manager := NewManager()

	// First register with auth
	plugin := &mockPlugin{name: "test"}
	pluginCfg := &config.PluginConfig{
		Name: "test",
		Auth: &config.AuthConfig{
			Type:   "bearer",
			Bearer: &config.BearerAuth{Token: "token"},
		},
	}
	manager.Register(plugin, pluginCfg)

	// Then register same plugin without auth
	plugin2 := &mockPlugin{name: "test"}
	manager.Register(plugin2, nil)

	// Plugin should be updated
	p, ok := manager.Get("test")
	if !ok {
		t.Error("Expected plugin to be registered")
	}
	if p != plugin2 {
		t.Error("Expected second plugin instance")
	}

	// Authenticator should NOT exist (Register with nil config doesn't set auth)
	_, ok = manager.GetAuthenticator("test")
	if ok {
		t.Error("Expected no authenticator after registering with nil config")
	}
}
