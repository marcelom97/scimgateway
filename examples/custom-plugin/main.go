package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/scim"
)

// CustomPlugin is a minimal plugin implementation
type CustomPlugin struct {
	name string
}

func (p *CustomPlugin) Name() string {
	return p.name
}

func (p *CustomPlugin) GetUsers(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.User, error) {
	// Return hardcoded users for demonstration
	// The adapter layer will handle filtering, pagination, and attribute selection
	users := []*scim.User{
		{
			ID:       "1",
			UserName: "john.doe",
			Active:   scim.Bool(true),
			Schemas:  []string{scim.SchemaUser},
		},
		{
			ID:       "2",
			UserName: "jane.smith",
			Active:   scim.Bool(true),
			Schemas:  []string{scim.SchemaUser},
		},
	}

	return users, nil
}

func (p *CustomPlugin) CreateUser(ctx context.Context, baseEntity string, user *scim.User) (*scim.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *CustomPlugin) GetUser(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *CustomPlugin) ModifyUser(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return fmt.Errorf("not implemented")
}

func (p *CustomPlugin) DeleteUser(ctx context.Context, baseEntity string, id string) error {
	return fmt.Errorf("not implemented")
}

func (p *CustomPlugin) GetGroups(ctx context.Context, baseEntity string, params scim.QueryParams) ([]*scim.Group, error) {
	// Return empty slice for demonstration
	// The adapter layer will handle filtering, pagination, and attribute selection
	return []*scim.Group{}, nil
}

func (p *CustomPlugin) CreateGroup(ctx context.Context, baseEntity string, group *scim.Group) (*scim.Group, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *CustomPlugin) GetGroup(ctx context.Context, baseEntity string, id string, attributes []string) (*scim.Group, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *CustomPlugin) ModifyGroup(ctx context.Context, baseEntity string, id string, patch *scim.PatchOp) error {
	return fmt.Errorf("not implemented")
}

func (p *CustomPlugin) DeleteGroup(ctx context.Context, baseEntity string, id string) error {
	return fmt.Errorf("not implemented")
}

func main() {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "custom",
				// No auth for this example
			},
		},
	}

	gw := scimgateway.New(cfg)

	// Optional: Setup structured logging for visibility into errors
	// Use nil to disable logging, or provide your own *slog.Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Only log warnings and errors
	}))
	gw.SetLogger(logger)

	// Register your custom plugin with its configuration
	customPlugin := &CustomPlugin{name: "custom"}
	gw.RegisterPlugin(customPlugin)

	// Initialize gateway
	// Note: Initialize() automatically validates the configuration
	// If validation fails, you'll get detailed error messages
	if err := gw.Initialize(); err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}
	log.Printf("Gateway initialized and validated successfully")

	log.Println("Starting SCIM Gateway with custom plugin on :8080")
	log.Println("Try: curl http://localhost:8080/custom/Users")

	if err := gw.Start(); err != nil {
		log.Fatal(err)
	}
}
