// Package main demonstrates custom JWT authentication with scimgateway.
//
// This example shows how to use a custom JWT authenticator by passing it
// directly in the gateway configuration using the "custom" auth type.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/scim"
)

func main() {
	// Get configuration from environment
	publicKeyPath := getEnv("JWT_PUBLIC_KEY", "public-key.pem")
	audience := getEnv("JWT_AUDIENCE", "scim-gateway")
	issuer := getEnv("JWT_ISSUER", "auth.example.com")
	port := getEnv("PORT", "8080")

	// Create custom JWT authenticator
	jwtAuth, err := NewJWTAuthenticator(publicKeyPath, audience, issuer)
	if err != nil {
		log.Println("Failed to create JWT authenticator:", err)
		log.Println()
		log.Println("Generate keys first:")
		log.Println("  openssl genrsa -out private-key.pem 2048")
		log.Println("  openssl rsa -in private-key.pem -pubout -out public-key.pem")
		log.Fatal(err)
	}

	// Create gateway config with custom authenticator
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost:8080",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "memory",
				Auth: &config.AuthConfig{
					Type: "custom",
					Custom: &config.CustomAuth{
						Authenticator: jwtAuth,
					},
				},
			},
		},
	}

	gw := scimgateway.New(cfg)
	gw.RegisterPlugin(NewMemoryPlugin())

	if err := gw.Initialize(); err != nil {
		log.Fatal(err)
	}

	log.Printf("╔══════════════════════════════════════════════════════════════╗")
	log.Printf("║  SCIM Gateway with Custom JWT Authentication                ║")
	log.Printf("╚══════════════════════════════════════════════════════════════╝")
	log.Println()
	log.Printf("  Address:     http://localhost:8080")
	log.Printf("  Public Key:  %s", publicKeyPath)
	log.Printf("  Audience:    %s", audience)
	log.Printf("  Issuer:      %s", issuer)
	log.Println()
	log.Println("Generate a JWT token with your preferred tool/library, then:")
	log.Printf("  curl -H \"Authorization: Bearer <token>\" http://localhost:8080/memory/Users\n")
	log.Println()

	if err := gw.Start(); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewMemoryPlugin() *MemoryPlugin {
	return &MemoryPlugin{
		name:   "memory",
		users:  make(map[string]*scim.User),
		groups: make(map[string]*scim.Group),
	}
}

// MemoryPlugin is a minimal in-memory plugin for demonstration
type MemoryPlugin struct {
	name   string
	users  map[string]*scim.User
	groups map[string]*scim.Group
	mu     sync.RWMutex
}

func (p *MemoryPlugin) Name() string { return p.name }

func (p *MemoryPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	users := make([]*scim.User, 0, len(p.users))
	for _, u := range p.users {
		users = append(users, u)
	}
	return users, nil
}

func (p *MemoryPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	user.ID = uuid.New().String()
	user.Schemas = []string{scim.SchemaUser}
	now := time.Now()
	user.Meta = &scim.Meta{ResourceType: "User", Created: &now, LastModified: &now}
	p.users[user.ID] = user
	return user, nil
}

func (p *MemoryPlugin) GetUser(ctx context.Context, id string, attrs []string) (*scim.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if u, ok := p.users[id]; ok {
		return u, nil
	}
	return nil, scim.ErrNotFound("User", id)
}

func (p *MemoryPlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	user, ok := p.users[id]
	if !ok {
		return scim.ErrNotFound("User", id)
	}
	return scim.NewPatchProcessor().ApplyPatch(user, patch)
}

func (p *MemoryPlugin) DeleteUser(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.users[id]; !ok {
		return scim.ErrNotFound("User", id)
	}
	delete(p.users, id)
	return nil
}

func (p *MemoryPlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	groups := make([]*scim.Group, 0, len(p.groups))
	for _, g := range p.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

func (p *MemoryPlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	group.ID = uuid.New().String()
	group.Schemas = []string{scim.SchemaGroup}
	now := time.Now()
	group.Meta = &scim.Meta{ResourceType: "Group", Created: &now, LastModified: &now}
	p.groups[group.ID] = group
	return group, nil
}

func (p *MemoryPlugin) GetGroup(ctx context.Context, id string, attrs []string) (*scim.Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if g, ok := p.groups[id]; ok {
		return g, nil
	}
	return nil, scim.ErrNotFound("Group", id)
}

func (p *MemoryPlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	group, ok := p.groups[id]
	if !ok {
		return scim.ErrNotFound("Group", id)
	}
	return scim.NewPatchProcessor().ApplyPatch(group, patch)
}

func (p *MemoryPlugin) DeleteGroup(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.groups[id]; !ok {
		return scim.ErrNotFound("Group", id)
	}
	delete(p.groups, id)
	return nil
}
