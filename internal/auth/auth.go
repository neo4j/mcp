package auth

import "context"

type contextKey string

const (
	basicAuthUserKey contextKey = "basicAuthUser"
	basicAuthPassKey contextKey = "basicAuthPass"
)

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
