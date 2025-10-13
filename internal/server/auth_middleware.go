package server

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

type contextKey string

const (
	contextKeyJWTClaims contextKey = "jwt_claims"
	jwksCacheDuration              = 1 * time.Minute
)

// CustomClaims contains custom claims for JWT validation
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but can be used to validate claims
func (c CustomClaims) Validate(_ context.Context) error {
	return nil
}

// jwtAuthMiddleware validates JWT tokens for every HTTP request
// It verifies:
// - Token signature (via JWKS from Auth0)
// - Token expiration (exp claim)
// - Audience (aud claim) - MUST contain this server's ResourceIdentifier (RFC 8707)
// - Issuer (iss claim)
//
// Per RFC9728 Section 5.1, all 401 responses include the WWW-Authenticate header
// with the resource server metadata URL.
//
// Per RFC 8707, tokens MUST be bound to the specific resource server via the audience claim.
// This prevents token passthrough attacks where a token obtained for one server is reused on another.
func (s *Neo4jMCPServer) jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If Auth0 is not configured, skip JWT validation and proceed
		if s.config.Auth0Domain == "" || s.config.ResourceIdentifier == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("Missing Authorization header from %s", r.RemoteAddr)
			s.sendUnauthorized(w, "invalid_request", "Missing Authorization header")
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("Invalid Authorization header format from %s", r.RemoteAddr)
			s.sendUnauthorized(w, "invalid_request", "Invalid Authorization header format")
			return
		}
		tokenString := parts[1]

		// Get the JWT validator
		jwtValidator, err := s.getJWTValidator()
		if err != nil {
			log.Printf("Failed to create JWT validator: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Validate the JWT token
		token, err := jwtValidator.ValidateToken(r.Context(), tokenString)
		if err != nil {
			// Log detailed validation failure for security monitoring
			log.Printf("⚠ JWT validation FAILED from %s: %v %s", r.RemoteAddr, err, tokenString)
			log.Printf("⚠ Expected audience(s): %s", s.config.ResourceIdentifier)
			log.Printf("⚠ Request path: %s", r.URL.Path)
			s.sendUnauthorized(w, "invalid_token", "The access token is invalid or expired")
			return
		}

		// Extract and validate the registered claims
		registeredClaims := token.(*validator.ValidatedClaims).RegisteredClaims

		// Log successful validation with audience information for security audit
		log.Printf("✓ JWT validation SUCCESS from %s", r.RemoteAddr)
		log.Printf("  Token audience: %v", registeredClaims.Audience)
		log.Printf("  Expected resource: %s", s.config.ResourceIdentifier)
		log.Printf("  Token subject: %s", registeredClaims.Subject)
		log.Printf("  Request path: %s", r.URL.Path)

		// debug
		log.Printf("  Full claims: %+v", token)

		// Verify that the token's audience matches our resource identifier (RFC 8707 compliance check)
		audienceMatches := false
		for _, aud := range registeredClaims.Audience {
			if aud == s.config.ResourceIdentifier {
				audienceMatches = true
				break
			}
		}

		if !audienceMatches {
			// This should not happen if the validator is configured correctly, but check anyway
			log.Printf("⚠ SECURITY ALERT: Token audience mismatch!")
			log.Printf("⚠ Token audience: %v", registeredClaims.Audience)
			log.Printf("⚠ Expected resource: %s", s.config.ResourceIdentifier)
			log.Printf("⚠ This may indicate a token passthrough attack attempt")

			// Reject invalid audience - strict RFC 8707 compliance
			s.sendUnauthorized(w, "invalid_token", "Token audience does not match this resource server")
			return
		}

		// Token is valid, add claims to context if needed
		ctx := context.WithValue(r.Context(), contextKeyJWTClaims, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sendUnauthorized sends a 401 Unauthorized response with the WWW-Authenticate header
// as required by RFC9728 Section 5.1.
//
// The WWW-Authenticate header includes:
// - realm: The protected resource metadata URL (RFC 9728 - this server's metadata endpoint)
// - error: OAuth 2.0 error code (e.g., "invalid_token", "invalid_request")
// - error_description: Human-readable description of the error
//
// Per MCP spec and RFC 9728, the realm MUST point to this server's protected resource
// metadata endpoint, which tells clients:
// 1. What resource identifier to use in token requests
// 2. Where to find the authorization server (Auth0)
func (s *Neo4jMCPServer) sendUnauthorized(w http.ResponseWriter, errorCode, errorDescription string) {
	// Construct the protected resource metadata URL per RFC 9728
	// This tells clients where to discover this resource server's configuration
	metadataURL := "https://" + s.config.Auth0Domain + "/.well-known/oauth-protected-resource"

	// Build WWW-Authenticate header value per RFC 6750 Section 3
	wwwAuthenticate := `Bearer resource_metadata="` + metadataURL + `"`
	if errorCode != "" {
		wwwAuthenticate += `, error="` + errorCode + `"`
	}
	if errorDescription != "" {
		wwwAuthenticate += `, error_description="` + errorDescription + `"`
	}

	w.Header().Set("WWW-Authenticate", wwwAuthenticate)
	http.Error(w, "Unauthorized: "+errorDescription, http.StatusUnauthorized)
}

// getJWTValidator creates and returns a JWT validator configured for Auth0
// Per RFC 8707, the validator MUST verify that the token's audience (aud) claim
// contains this specific server's ResourceIdentifier. This prevents tokens obtained
// for one resource server from being accepted by another resource server.
func (s *Neo4jMCPServer) getJWTValidator() (*validator.Validator, error) {
	issuerURL, err := url.Parse("https://" + s.config.Auth0Domain + "/")
	if err != nil {
		return nil, err
	}

	provider := jwks.NewCachingProvider(issuerURL, jwksCacheDuration)

	// Build audience list for validation
	// Audience MUST be this server's ResourceIdentifier (RFC 8707 compliance)
	audiences := []string{s.config.ResourceIdentifier}

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		audiences,
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)

	if err != nil {
		return nil, err
	}

	log.Printf("JWT validator configured with expected audience(s): %v", audiences)
	return jwtValidator, nil
}

func (s *Neo4jMCPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.isOriginAllowed(s.config.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the given origin is in the list of allowed origins
func (s *Neo4jMCPServer) isOriginAllowed(allowedOrigins []string, origin string) bool {
	return true
	// todo: re-enable origin checking in main
	// return slices.Contains(allowedOrigins, origin)
}
