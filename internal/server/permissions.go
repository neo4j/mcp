package server

import (
	"context"
	"fmt"
	"log"
	"slices"
)

// ToolPermission defines the required permissions for each tool
// Permissions are mapped as scope strings from Auth0 JWT tokens
var ToolPermissions = map[string][]string{
	"get-schema":          {"read:schema", "admin:all"},
	"read-cypher":         {"read:data", "admin:all"},
	"write-cypher":        {"write:data", "admin:all"},
	"list-gds-procedures": {"read:gds", "admin:all"},
}

// CheckToolPermission verifies if the user has permission to execute the specified tool
// Returns an error if:
// - Authentication is enabled but no user context is found
// - User doesn't have the required permissions
// Returns nil if:
// - Authentication is disabled (no user context)
// - User has at least one of the required permissions
func CheckToolPermission(ctx context.Context, toolName string) error {
	userCtx := GetUserContext(ctx)

	// If no user context exists, auth is disabled - allow all
	if userCtx == nil {
		log.Printf("No user context found - authentication disabled, allowing tool: %s", toolName)
		return nil
	}

	// Get required permissions for this tool
	requiredPerms, exists := ToolPermissions[toolName]
	if !exists {
		// Tool not in permission map - deny by default for security
		log.Printf("⚠ Tool '%s' not found in permission map - denying access for user %s", toolName, userCtx.UserID)
		return fmt.Errorf("tool '%s' is not configured for permission checking", toolName)
	}

	// Check if user has at least one of the required permissions
	for _, userPerm := range userCtx.Permissions {
		if slices.Contains(requiredPerms, userPerm) {
			log.Printf("✓ Permission check PASSED for user %s: tool '%s' requires %v, user has '%s'",
				userCtx.UserID, toolName, requiredPerms, userPerm)
			return nil
		}
	}

	// User doesn't have any of the required permissions
	log.Printf("⚠ Permission check FAILED for user %s: tool '%s' requires one of %v, user has %v",
		userCtx.UserID, toolName, requiredPerms, userCtx.Permissions)
	return fmt.Errorf("insufficient permissions: tool '%s' requires one of %v, but user has %v",
		toolName, requiredPerms, userCtx.Permissions)
}
