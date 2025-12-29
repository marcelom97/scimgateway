package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
)

func main() {
	// Create configuration programmatically
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "sqlite",
				Auth: &config.AuthConfig{
					Type: "bearer",
					Bearer: &config.BearerAuth{
						Token: "my-secret-token",
					},
				},
			},
		},
	}

	// Create gateway
	gw := scimgateway.New(cfg)

	// Optional: Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	gw.SetLogger(logger)
	log.Printf("Structured logging enabled")

	// Create SQLite plugin
	// Database file will be created in current directory
	dbPath := "./scim.db"
	sqlitePlugin, err := NewSQLitePlugin("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to create SQLite plugin: %v", err)
	}
	defer sqlitePlugin.Close() // nolint:errcheck
	log.Printf("SQLite database initialized at: %s", dbPath)

	gw.RegisterPlugin(sqlitePlugin)
	log.Printf("Registered plugin: sqlite")

	// Initialize gateway
	if err := gw.Initialize(); err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}
	log.Printf("Gateway initialized successfully")

	log.Printf("\nExample commands:")
	log.Printf("  Create user: curl -H 'Authorization: Bearer my-secret-token' -X POST http://localhost:8080/sqlite/Users -H 'Content-Type: application/json' -d '{\"userName\":\"john.doe\",\"active\":true}'")
	log.Printf("  List users:  curl -H 'Authorization: Bearer my-secret-token' http://localhost:8080/sqlite/Users")
	log.Printf("  Get user:    curl -H 'Authorization: Bearer my-secret-token' http://localhost:8080/sqlite/Users/{id}")
	log.Printf("  List groups: curl -H 'Authorization: Bearer my-secret-token' http://localhost:8080/sqlite/Groups\n")

	if err := gw.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
