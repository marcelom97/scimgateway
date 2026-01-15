package scimgateway

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/marcelom97/scimgateway/config"
	"github.com/marcelom97/scimgateway/plugin"
	"github.com/marcelom97/scimgateway/scim"
)

// discardLogger returns a no-op logger that discards all output
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Gateway represents a SCIM gateway instance
type Gateway struct {
	config        *config.Config
	pluginManager *plugin.Manager
	server        *scim.Server
	handler       http.Handler
	logger        *slog.Logger
}

// New creates a new Gateway instance
func New(cfg *config.Config) *Gateway {
	return &Gateway{
		config:        cfg,
		pluginManager: plugin.NewManager(),
		logger:        discardLogger(), // Default to no-op logger
	}
}

// NewWithDefaults creates a new Gateway with default valid configuration
func NewWithDefaults() *Gateway {
	return New(config.DefaultConfig())
}

// RegisterPlugin registers a plugin with the gateway
// The plugin config is automatically looked up from the gateway config by plugin name
func (g *Gateway) RegisterPlugin(p plugin.Plugin) {
	// Find the plugin config by name
	var pluginCfg *config.PluginConfig
	for i := range g.config.Plugins {
		if g.config.Plugins[i].Name == p.Name() {
			pluginCfg = &g.config.Plugins[i]
			break
		}
	}
	g.pluginManager.Register(p, pluginCfg)
}

// SetLogger sets the optional logger for the gateway.
// Pass nil to disable logging (default behavior).
// The logger will be used to log critical errors and warnings only.
func (g *Gateway) SetLogger(logger *slog.Logger) {
	if logger == nil {
		g.logger = discardLogger()
	} else {
		g.logger = logger
	}
}

// Initialize initializes the gateway (must be called before Start)
func (g *Gateway) Initialize() error {
	// Validate configuration first
	if err := g.config.Validate(); err != nil {
		g.logger.Error("configuration validation failed", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate that at least one plugin has been registered
	if len(g.pluginManager.List()) == 0 {
		err := fmt.Errorf("no plugins registered: at least one plugin must be registered via RegisterPlugin() before initialization")
		g.logger.Error("plugin registration validation failed", "error", err)
		return err
	}

	g.logger.Info("initializing SCIM gateway",
		"base_url", g.config.Gateway.BaseURL,
		"port", g.config.Gateway.Port,
		"tls_enabled", g.config.Gateway.TLS != nil && g.config.Gateway.TLS.Enabled,
	)

	// Create adapted manager
	adaptedManager := plugin.NewAdaptedManager(g.pluginManager)

	// Create SCIM server with logger
	g.server = scim.NewServerWithLogger(g.config.Gateway.BaseURL, adaptedManager, g.logger)

	// Setup handler with middleware chain
	var handler http.Handler = g.server

	// Add request logging middleware
	handler = LoggingMiddleware(g.logger)(handler)

	// Add per-plugin authentication middleware
	handler = plugin.PerPluginAuthMiddleware(g.pluginManager)(handler)

	g.handler = handler

	pluginNames := g.pluginManager.List()
	g.logger.Info("gateway initialized successfully",
		"plugins", pluginNames,
		"plugin_count", len(pluginNames),
	)

	return nil
}

// Handler returns the HTTP handler for the gateway.
// Returns an error if the gateway has not been initialized.
func (g *Gateway) Handler() (http.Handler, error) {
	if g.handler == nil {
		return nil, fmt.Errorf("gateway not initialized - call Initialize() first")
	}
	return g.handler, nil
}

// Start starts the gateway HTTP server (blocking)
func (g *Gateway) Start() error {
	if g.handler == nil {
		if err := g.Initialize(); err != nil {
			g.logger.Error("failed to initialize gateway", "error", err)
			return err
		}
	}

	if g.config.Gateway.Port == 0 {
		return fmt.Errorf("port is required for standalone mode - use Handler() for embedded mode")
	}

	addr := fmt.Sprintf(":%d", g.config.Gateway.Port)

	if g.config.Gateway.TLS != nil && g.config.Gateway.TLS.Enabled {
		g.logger.Info("starting SCIM gateway with TLS",
			"addr", addr,
			"cert_file", g.config.Gateway.TLS.CertFile,
		)
		err := http.ListenAndServeTLS(
			addr,
			g.config.Gateway.TLS.CertFile,
			g.config.Gateway.TLS.KeyFile,
			g.handler,
		)
		if err != nil {
			g.logger.Error("gateway server stopped", "error", err)
		}
		return err
	}

	g.logger.Info("starting SCIM gateway", "addr", addr)
	err := http.ListenAndServe(addr, g.handler)
	if err != nil {
		g.logger.Error("gateway server stopped", "error", err)
	}
	return err
}

// Config returns the gateway configuration
func (g *Gateway) Config() *config.Config {
	return g.config
}

// PluginManager returns the plugin manager
func (g *Gateway) PluginManager() *plugin.Manager {
	return g.pluginManager
}
