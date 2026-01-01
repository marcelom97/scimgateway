package main

import (
	"log"
	"log/slog"
	"net/http"
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
				Name: "postgres",
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

	// Get PostgreSQL connection string from environment variable or use default
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Default connection string for local development
		// Format: postgres://user:password@host:port/dbname?sslmode=disable
		connStr = "postgres://postgres:postgres@localhost:5432/scimgateway?sslmode=disable"
	}

	// Create PostgreSQL plugin
	postgresPlugin, err := NewPostgresPlugin("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL plugin: %v", err)
	}
	defer postgresPlugin.Close() // nolint:errcheck
	log.Printf("PostgreSQL database connected")

	gw.RegisterPlugin(postgresPlugin)
	log.Printf("Registered plugin: postgres")

	// Initialize gateway
	if err := gw.Initialize(); err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}
	log.Printf("Gateway initialized successfully")

	hander, err := gw.Handler()
	if err != nil {
		log.Fatalf("Failed to get gateway handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", hander)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := postgresPlugin.HealthCheck(r.Context()); err != nil {
			http.Error(w, "Unhealthy", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Healthy"))
	})

	log.Printf("Starting SCIM Gateway on port %d...", cfg.Gateway.Port)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
