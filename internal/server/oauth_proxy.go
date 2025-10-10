package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
)

// handleAuthorize redirects OAuth authorization requests to Auth0
// This handles VS Code's OAuth flow by proxying to the Auth0 authorization endpoint
func (s *Neo4jMCPServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if s.config.Auth0Domain == "" {
		http.Error(w, "Auth0 not configured", http.StatusInternalServerError)
		return
	}

	// Parse existing query parameters from VS Code
	params := r.URL.Query()

	// Inject the audience parameter so Auth0 issues a JWT for our API
	// Without this, Auth0 would issue an opaque token for the userinfo endpoint
	if s.config.Auth0Audience != "" {
		params.Set("audience", s.config.Auth0Audience)
	}

	// Build Auth0 authorization URL with modified parameters
	auth0URL := "https://" + s.config.Auth0Domain + "/authorize?" + params.Encode()

	log.Printf("→ Redirecting authorization to Auth0 (audience=%s)", s.config.Auth0Audience)
	http.Redirect(w, r, auth0URL, http.StatusFound)
}

// handleToken proxies OAuth token requests to Auth0
// This handles the token exchange after VS Code receives the authorization code
func (s *Neo4jMCPServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if s.config.Auth0Domain == "" {
		http.Error(w, "Auth0 not configured", http.StatusInternalServerError)
		return
	}

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read token request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the form data to log it (optional, for debugging)
	formData, _ := url.ParseQuery(string(body))
	log.Printf("→ Proxying token request to Auth0 (grant_type=%s)", formData.Get("grant_type"))

	// Create request to Auth0 token endpoint
	auth0TokenURL := "https://" + s.config.Auth0Domain + "/oauth/token"
	req, err := http.NewRequest("POST", auth0TokenURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("Failed to create Auth0 token request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy content-type header
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	// Forward the request to Auth0
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to call Auth0 token endpoint: %v", err)
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read Auth0's response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Auth0 token response: %v", err)
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	// Log success or failure
	if resp.StatusCode == http.StatusOK {
		log.Printf("← Token obtained successfully from Auth0")
	} else {
		log.Printf("← Auth0 token request failed: %d %s", resp.StatusCode, string(respBody))
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Return Auth0's response to VS Code
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(respBody); err != nil {
		log.Printf("Failed to write response to client: %v", err)
	}
}

// handleOAuthMetadata returns OAuth authorization server metadata
// This helps OAuth clients discover endpoints
func (s *Neo4jMCPServer) handleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	if s.config.Auth0Domain == "" {
		http.Error(w, "Auth0 not configured", http.StatusInternalServerError)
		return
	}

	// Determine the base URL for this server
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + r.Host

	metadata := map[string]interface{}{
		"issuer":                           "https://" + s.config.Auth0Domain + "/",
		"authorization_endpoint":           baseURL + "/authorize",
		"token_endpoint":                   baseURL + "/token",
		"jwks_uri":                         "https://" + s.config.Auth0Domain + "/.well-known/jwks.json",
		"response_types_supported":         []string{"code"},
		"grant_types_supported":            []string{"authorization_code"},
		"code_challenge_methods_supported": []string{"S256", "plain"},
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(metadata)
	if err != nil {
		log.Printf("Failed to encode OAuth metadata: %v", err)
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}

	log.Printf("← Served OAuth authorization server metadata")
}
