package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/marcelom97/scimgateway"
	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/memory"
)

func main() {
	// Create configuration programmatically with per-plugin authentication
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			BaseURL: "http://localhost",
			Port:    8080,
		},
		Plugins: []config.PluginConfig{
			{
				Name: "memory",
				Auth: &config.AuthConfig{
					Type: "basic",
					Basic: &config.BasicAuth{
						Username: "admin",
						Password: "secret",
					},
				},
			},
		},
	}

	// Create gateway
	gw := scimgateway.New(cfg)

	// Optional: Setup structured logging
	// Pass nil to disable logging (default), or provide your own *slog.Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	gw.SetLogger(logger)
	log.Printf("Structured logging enabled")

	gw.RegisterPlugin(memory.New("memory"))
	log.Printf("Registered plugin: memory with basic auth")

	// Initialize gateway
	// Note: Initialize() automatically validates the configuration
	// If the config is invalid, it will return a detailed error
	if err := gw.Initialize(); err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}
	log.Printf("Gateway initialized successfully")

	// Start server
	log.Printf("Starting SCIM Gateway on :%d", cfg.Gateway.Port)
	log.Printf("Base URL: %s", cfg.Gateway.BaseURL)
	log.Printf("Try: curl -u admin:secret http://localhost:8080/memory/Users")

	if err := gw.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
