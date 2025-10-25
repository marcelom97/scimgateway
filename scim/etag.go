package scim

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ETagGenerator generates ETags for resources
type ETagGenerator struct{}

// NewETagGenerator creates a new ETag generator
func NewETagGenerator() *ETagGenerator {
	return &ETagGenerator{}
}

// Generate generates an ETag for a resource
func (e *ETagGenerator) Generate(resource any) (string, error) {
	// We need to exclude meta.version from the hash to avoid circular dependency
	// (version contains the ETag value itself)

	// Marshal the resource
	data, err := json.Marshal(resource)
	if err != nil {
		return "", err
	}

	// Parse as map to remove meta.version
	var resourceMap map[string]any
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return "", err
	}

	// Remove version from meta if it exists
	if meta, ok := resourceMap["meta"].(map[string]any); ok {
		delete(meta, "version")
	}

	// Re-marshal without version
	data, err = json.Marshal(resourceMap)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	etag := fmt.Sprintf(`W/"%x"`, hash[:8]) // Use weak ETag with first 8 bytes
	return etag, nil
}

// CheckPreconditions checks If-Match and If-None-Match headers
func (e *ETagGenerator) CheckPreconditions(r *http.Request, currentETag string) (int, error) {
	// Check If-Match header
	ifMatch := r.Header.Get("If-Match")
	if ifMatch != "" {
		if !e.matchesETag(ifMatch, currentETag) {
			return http.StatusPreconditionFailed, fmt.Errorf("precondition failed: ETag mismatch")
		}
	}

	// Check If-None-Match header
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" {
		if e.matchesETag(ifNoneMatch, currentETag) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				return http.StatusNotModified, fmt.Errorf("not modified")
			}
			return http.StatusPreconditionFailed, fmt.Errorf("precondition failed: resource exists")
		}
	}

	return http.StatusOK, nil
}

// matchesETag checks if an ETag matches
func (e *ETagGenerator) matchesETag(headerValue, currentETag string) bool {
	// Handle * (any)
	if strings.TrimSpace(headerValue) == "*" {
		return currentETag != ""
	}

	// Parse comma-separated ETags
	tags := strings.SplitSeq(headerValue, ",")
	for tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == currentETag {
			return true
		}
	}

	return false
}

// SetETag sets the ETag header on the response
func (e *ETagGenerator) SetETag(w http.ResponseWriter, etag string) {
	if etag != "" {
		w.Header().Set("ETag", etag)
	}
}

// UpdateResourceVersion updates the version in Meta
func UpdateResourceVersion(meta *Meta, etag string) {
	if meta != nil {
		// Store ETag as version (extract from W/"hash" format)
		// ETag format is: W/"<hash>"
		version := etag
		// Remove W/ prefix
		version = strings.TrimPrefix(version, "W/")
		// Remove surrounding quotes
		version = strings.Trim(version, `"`)
		meta.Version = version
	}
}
