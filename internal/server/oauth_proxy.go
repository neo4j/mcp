package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// handleProtectedResourceMetadata returns protected resource metadata per RFC 9728
// This endpoint tells OAuth clients:
// 1. What resource identifier to use when requesting tokens (for RFC 8707 compliance)
// 2. Where to find the authorization server(s) that can issue tokens for this resource
//
// Per MCP specification and RFC 9728, this endpoint is discovered via the WWW-Authenticate
// header's realm parameter when the server returns 401 Unauthorized.
func (s *Neo4jMCPServer) handleProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	if s.config.ResourceIdentifier == "" {
		log.Printf("ERROR: Cannot serve protected resource metadata - MCP_RESOURCE_IDENTIFIER not configured")
		http.Error(w, "Resource identifier not configured", http.StatusInternalServerError)
		return
	}

	if s.config.Auth0Domain == "" {
		log.Printf("ERROR: Cannot serve protected resource metadata - AUTH0_DOMAIN not configured")
		http.Error(w, "Authorization server not configured", http.StatusInternalServerError)
		return
	}

	// Per RFC 9728, the protected resource metadata includes:
	// - resource: The unique identifier for this resource server (used in token requests)
	// - authorization_servers: List of authorization servers that can issue tokens
	metadata := map[string]interface{}{
		"resource": s.config.ResourceIdentifier,
		"authorization_servers": []string{
			"https://" + s.config.Auth0Domain + "/",
		},
		// "scopes_supported":         []string{"read:tools", "execute:tools"}, // todo: define scopes if needed
		"bearer_methods_supported": []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("Failed to encode protected resource metadata: %v", err)
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Served protected resource metadata (resource=%s)", s.config.ResourceIdentifier)
}

// handleAuthorizationServerMetadata redirects to Auth0's authorization server metadata endpoint
// VS Code expects this endpoint to discover OAuth endpoints (authorize, token, etc.)
func (s *Neo4jMCPServer) handleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	if s.config.Auth0Domain == "" {
		log.Printf("ERROR: Cannot serve authorization server metadata - AUTH0_DOMAIN not configured")
		http.Error(w, "Authorization server not configured", http.StatusInternalServerError)
		return
	}

	// Redirect to Auth0's OpenID Connect Discovery endpoint
	auth0MetadataURL := fmt.Sprintf("https://%s/.well-known/oauth-authorization-server", s.config.Auth0Domain)

	log.Printf("→ Redirecting to Auth0 authorization server metadata")
	log.Printf("  Auth0 URL: %s", auth0MetadataURL)

	http.Redirect(w, r, auth0MetadataURL, http.StatusFound)
}

// handleAuthorize logs the request and redirects to Auth0 authorization endpoint
func (s *Neo4jMCPServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	log.Printf("→ OAuth /authorize request received")
	log.Printf("  Method: %s", r.Method)
	log.Printf("  URL: %s", r.URL.String())
	log.Printf("  Remote: %s", r.RemoteAddr)

	// Log query parameters
	for key, values := range r.URL.Query() {
		for _, value := range values {
			log.Printf("  Query: %s = %s", key, value)
		}
	}

	// Redirect to Auth0 authorize endpoint
	auth0URL := fmt.Sprintf("https://%s/authorize?%s", s.config.Auth0Domain, r.URL.RawQuery)
	log.Printf("→ Redirecting to Auth0: %s", auth0URL)
	http.Redirect(w, r, auth0URL, http.StatusFound)
}

// handleCallback logs the request and redirects to Auth0 callback endpoint
func (s *Neo4jMCPServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("← OAuth /callback request received")
	log.Printf("  Method: %s", r.Method)
	log.Printf("  URL: %s", r.URL.String())
	log.Printf("  Remote: %s", r.RemoteAddr)

	// Log query parameters
	for key, values := range r.URL.Query() {
		for _, value := range values {
			log.Printf("  Query: %s = %s", key, value)
		}
	}

	// Redirect to Auth0 callback endpoint (typically not used, but for completeness)
	auth0URL := fmt.Sprintf("https://%s/login/callback?%s", s.config.Auth0Domain, r.URL.RawQuery)
	log.Printf("→ Redirecting to Auth0: %s", auth0URL)
	http.Redirect(w, r, auth0URL, http.StatusFound)
}

// handleToken proxies OAuth token requests to Auth0 and injects the resource parameter
// This endpoint is essential for PKCE flows where the client has the code_verifier
func (s *Neo4jMCPServer) handleToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("→ OAuth /token request received")
	log.Printf("  Method: %s", r.Method)
	log.Printf("  URL: %s", r.URL.String())
	log.Printf("  Remote: %s", r.RemoteAddr)
	log.Printf("  Content-Type: %s", r.Header.Get("Content-Type"))

	// Log form data if present
	if err := r.ParseForm(); err == nil {
		for key, values := range r.PostForm {
			for _, value := range values {
				log.Printf("  Form: %s = %s", key, value)
			}
		}
	}

	// Redirect to Auth0 token endpoint
	auth0URL := fmt.Sprintf("https://%s/oauth/token", s.config.Auth0Domain)
	log.Printf("→ Redirecting to Auth0: %s", auth0URL)
	http.Redirect(w, r, auth0URL, http.StatusFound)
}
