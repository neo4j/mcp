// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package mcpcontext

import "context"

type contextKey string

const (
	basicAuthUserKey contextKey = "basicAuthUser"
	basicAuthPassKey contextKey = "basicAuthPass"
	bearerTokenKey   contextKey = "bearerToken"
	databaseNameKey  contextKey = "databaseName"
)

// WithDatabaseName adds the target database name to the context
func WithDatabaseName(ctx context.Context, databaseName string) context.Context {
	return context.WithValue(ctx, databaseNameKey, databaseName)
}

// GetDatabaseName retrieves the database name from the context
func GetDatabaseName(ctx context.Context) (string, bool) {
	dbName, ok := ctx.Value(databaseNameKey).(string)
	return dbName, ok
}

// WithBasicAuth adds basic auth credentials to the context
func WithBasicAuth(ctx context.Context, user, pass string) context.Context {
	ctx = context.WithValue(ctx, basicAuthUserKey, user)
	ctx = context.WithValue(ctx, basicAuthPassKey, pass)
	return ctx
}

// GetBasicAuthCredentials retrieves basic auth credentials from the context
func GetBasicAuthCredentials(ctx context.Context) (string, string, bool) {
	user, okUser := ctx.Value(basicAuthUserKey).(string)
	pass, okPass := ctx.Value(basicAuthPassKey).(string)
	return user, pass, okUser && okPass
}

// WithBearerToken adds bearer token to the context
func WithBearerToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, bearerTokenKey, token)
}

// GetBearerToken retrieves bearer token from the context
func GetBearerToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(bearerTokenKey).(string)
	return token, ok
}

// HasAuth checks if either basic auth or bearer token is present in the context
func HasAuth(ctx context.Context) bool {
	_, _, okBasic := GetBasicAuthCredentials(ctx)
	_, okBearer := GetBearerToken(ctx)
	return okBasic || okBearer
}
