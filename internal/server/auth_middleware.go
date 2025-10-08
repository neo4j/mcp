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

// CustomClaims contains custom claims for JWT validation
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but can be used to validate claims
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// jwtAuthMiddleware validates JWT tokens for every HTTP request
// It verifies:
// - Token signature (via JWKS from Auth0)
// - Token expiration (exp claim)
// - Audience (aud claim)
// - Issuer (iss claim)
func (s *Neo4jMCPServer) jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If Auth0 is not configured, skip JWT validation and proceed
		if s.config.Auth0Domain == "" || s.config.Auth0Audience == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("Missing Authorization header from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("Invalid Authorization header format from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
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
			log.Printf("JWT validation failed from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
			return
		}

		// Token is valid, add claims to context if needed
		ctx := context.WithValue(r.Context(), "jwt_claims", token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getJWTValidator creates and returns a JWT validator configured for Auth0
func (s *Neo4jMCPServer) getJWTValidator() (*validator.Validator, error) {
	issuerURL, err := url.Parse("https://" + s.config.Auth0Domain + "/")
	if err != nil {
		return nil, err
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{s.config.Auth0Audience},
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

	return jwtValidator, nil
}
