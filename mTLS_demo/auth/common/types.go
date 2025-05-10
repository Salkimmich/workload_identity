package common

import (
	"context"
	"fmt"
	"net/http"
)

// AuthMethod represents the authentication method used
type AuthMethod string

const (
	// AuthMethodMTLS represents mTLS authentication
	AuthMethodMTLS AuthMethod = "mtls"
	// AuthMethodJWT represents JWT authentication
	AuthMethodJWT AuthMethod = "jwt"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// AuthMethodContextKey is the key for storing the authentication method in context
	AuthMethodContextKey ContextKey = "auth_method"
	// ServiceIDContextKey is the key for storing the service ID in context
	ServiceIDContextKey ContextKey = "service_id"
	// RolesContextKey is the key for storing the roles in context
	RolesContextKey ContextKey = "roles"
)

// AuthMiddleware defines the interface for authentication middlewares
type AuthMiddleware interface {
	// Middleware returns a middleware function that validates authentication
	Middleware(next http.Handler) http.Handler

	// RequireRole creates a middleware that checks for required roles
	RequireRole(roles ...string) func(http.Handler) http.Handler
}

// AuthMetricsCollector defines the interface for authentication metrics
type AuthMetricsCollector interface {
	// RecordAuthRequest records a successful authentication request
	RecordAuthRequest(serviceName, authMethod, result string, duration float64)

	// RecordAuthError records an authentication error
	RecordAuthError(serviceName, authMethod, errorType string)
}

// Common helper functions for context values
func GetAuthMethodFromContext(ctx context.Context) (AuthMethod, error) {
	method, ok := ctx.Value(AuthMethodContextKey).(AuthMethod)
	if !ok {
		return "", fmt.Errorf("auth method not found in context")
	}
	return method, nil
}

func GetServiceIDFromContext(ctx context.Context) (string, error) {
	serviceID, ok := ctx.Value(ServiceIDContextKey).(string)
	if !ok {
		return "", fmt.Errorf("service ID not found in context")
	}
	return serviceID, nil
}

func GetRolesFromContext(ctx context.Context) ([]string, error) {
	roles, ok := ctx.Value(RolesContextKey).([]string)
	if !ok {
		return nil, fmt.Errorf("roles not found in context")
	}
	return roles, nil
}

// WithAuthMethod adds the authentication method to the context
func WithAuthMethod(ctx context.Context, method AuthMethod) context.Context {
	return context.WithValue(ctx, AuthMethodContextKey, method)
}

// WithServiceID adds the service ID to the context
func WithServiceID(ctx context.Context, serviceID string) context.Context {
	return context.WithValue(ctx, ServiceIDContextKey, serviceID)
}

// WithRoles adds the roles to the context
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, RolesContextKey, roles)
} 