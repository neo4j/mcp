// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package mcpcontext

import (
	"context"
	"time"
)

type contextKey string

const (
	basicAuthUserKey  contextKey = "basicAuthUser"
	basicAuthPassKey  contextKey = "basicAuthPass"
	bearerTokenKey    contextKey = "bearerToken"
	databaseNameKey   contextKey = "databaseName"
	driverKey         contextKey = "neo4jDriver"
	readOnlyKey       contextKey = "readOnly"
	toolsKey          contextKey = "tools"
	requestTimeoutKey contextKey = "requestTimeout"
	basicAuthUserKey  contextKey = "basicAuthUser"
	basicAuthPassKey  contextKey = "basicAuthPass"
	bearerTokenKey    contextKey = "bearerToken"
	databaseNameKey   contextKey = "databaseName"
	driverKey         contextKey = "neo4jDriver"
	readOnlyKey       contextKey = "readOnly"
	toolsKey          contextKey = "tools"
	requestIDKey      contextKey = "requestID"
	neo4jTargetKey    contextKey = "neo4jTarget"
	dbIDKey           contextKey = "dbID"
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

// WithDriver stores a Neo4j driver in the context
// The driver is stored as any to avoid importing the neo4j driver package
func WithDriver(ctx context.Context, driver any) context.Context {
	return context.WithValue(ctx, driverKey, driver)
}

// GetDriver retrieves the Neo4j driver from the context
func GetDriver(ctx context.Context) (any, bool) {
	driver := ctx.Value(driverKey)
	return driver, driver != nil
}

// WithReadOnly marks the context as read-only
func WithReadOnly(ctx context.Context, readOnly bool) context.Context {
	return context.WithValue(ctx, readOnlyKey, readOnly)
}

// GetReadOnly retrieves the read-only flag from the context.
// Returns nil if the flag is not set.
func GetReadOnly(ctx context.Context) *bool {
	readOnly, ok := ctx.Value(readOnlyKey).(bool)
	if !ok {
		// The assumption here is that ctx entry is set by the setter defined in mcpcontext,
		// therefore is nil when ok is false
		return nil
	}
	return &readOnly
}

// WithTools store per-request tools in the context
func WithTools(ctx context.Context, tools []string) context.Context {
	return context.WithValue(ctx, toolsKey, tools)
}

// GetTools retrieves the per-request tools from the context.
// Returns nil if the tools are not explicitly requested.
func GetTools(ctx context.Context) *[]string {
	tools, ok := ctx.Value(toolsKey).([]string)
	if !ok {
		// The assumption here is that ctx entry is set by the setter defined in mcpcontext,
		// therefore is nil when ok is false
		return nil
	}
	return &tools
}

// WithRequestTimeout stores the effective request timeout in the context.
func WithRequestTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, requestTimeoutKey, timeout)
}

// GetRequestTimeout retrieves the effective request timeout from the context.
// Returns 0 if the timeout is not set.
func GetRequestTimeout(ctx context.Context) time.Duration {
	timeout, ok := ctx.Value(requestTimeoutKey).(time.Duration)
	if !ok {
		return 0
	}
	return timeout
}

// WithRequestID stores a server-generated request correlation ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request correlation ID from the context.
func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

// WithNeo4jTarget stores the safe Neo4j target (scheme://host) in the context.
func WithNeo4jTarget(ctx context.Context, target string) context.Context {
	return context.WithValue(ctx, neo4jTargetKey, target)
}

// GetNeo4jTarget retrieves the Neo4j target from the context.
func GetNeo4jTarget(ctx context.Context) (string, bool) {
	target, ok := ctx.Value(neo4jTargetKey).(string)
	return target, ok
}

// WithDBID stores the Aura database instance id in the context.
func WithDBID(ctx context.Context, dbID string) context.Context {
	return context.WithValue(ctx, dbIDKey, dbID)
}

// GetDBID retrieves the Aura database instance id from the context.
func GetDBID(ctx context.Context) (string, bool) {
	dbID, ok := ctx.Value(dbIDKey).(string)
	return dbID, ok
}
