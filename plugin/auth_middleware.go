package plugin

import (
	"net/http"
	"strings"

	"github.com/marcelom97/scimgateway/auth"
)

// PerPluginAuthMiddleware creates middleware that applies authentication per plugin
// based on the plugin name extracted from the request path (/{plugin}/...)
func PerPluginAuthMiddleware(manager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract plugin name from path: /{plugin}/...
			path := strings.TrimPrefix(r.URL.Path, "/")
			parts := strings.SplitN(path, "/", 2)

			if len(parts) == 0 {
				// No plugin in path, continue without auth
				next.ServeHTTP(w, r)
				return
			}

			pluginName := parts[0]

			// Get authenticator for this plugin
			authenticator, hasAuth := manager.GetAuthenticator(pluginName)

			if !hasAuth {
				// No auth configured for this plugin, allow request
				next.ServeHTTP(w, r)
				return
			}

			// Apply authentication for this plugin
			authHandler := auth.Middleware(authenticator)(next)
			authHandler.ServeHTTP(w, r)
		})
	}
}
