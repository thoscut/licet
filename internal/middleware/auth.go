package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled            bool       `mapstructure:"enabled"`
	AllowAnonymousRead bool       `mapstructure:"allow_anonymous_read"`
	APIKeys            []APIKey   `mapstructure:"api_keys"`
	BasicAuth          BasicAuth  `mapstructure:"basic_auth"`
	SessionTimeout     int        `mapstructure:"session_timeout"` // Minutes
	ExemptPaths        []string   `mapstructure:"exempt_paths"`
}

// APIKey represents an API key with associated permissions
type APIKey struct {
	Name        string   `mapstructure:"name"`
	Key         string   `mapstructure:"key"`
	Role        string   `mapstructure:"role"` // "admin", "readonly", "write"
	Description string   `mapstructure:"description"`
	Enabled     bool     `mapstructure:"enabled"`
}

// BasicAuth holds basic authentication configuration
type BasicAuth struct {
	Enabled  bool         `mapstructure:"enabled"`
	Users    []BasicUser  `mapstructure:"users"`
}

// BasicUser represents a user for basic authentication
type BasicUser struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"` // Should be hashed in production
	Role     string `mapstructure:"role"`
	Enabled  bool   `mapstructure:"enabled"`
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:        false,
		SessionTimeout: 60,
		ExemptPaths: []string{
			"/api/v1/health",
			"/static/",
		},
		APIKeys:   []APIKey{},
		BasicAuth: BasicAuth{Enabled: false, Users: []BasicUser{}},
	}
}

// Role constants
const (
	RoleAdmin    = "admin"
	RoleWrite    = "write"
	RoleReadonly = "readonly"
)

// Permission constants
const (
	PermissionRead   = "read"
	PermissionWrite  = "write"
	PermissionAdmin  = "admin"
)

// AuthContext key for storing auth info in request context
type authContextKey string

const (
	AuthUserKey   authContextKey = "auth_user"
	AuthRoleKey   authContextKey = "auth_role"
	AuthMethodKey authContextKey = "auth_method"
)

// AuthInfo contains authentication information for a request
type AuthInfo struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	Method        string `json:"method"` // "api_key", "basic", "none"
}

// Authenticator handles authentication for the application
type Authenticator struct {
	config       AuthConfig
	apiKeyIndex  map[string]*APIKey
	userIndex    map[string]*BasicUser
	sessions     map[string]*session
	sessionMu    sync.RWMutex
}

type session struct {
	username  string
	role      string
	expiresAt time.Time
}

// NewAuthenticator creates a new authenticator instance
func NewAuthenticator(config AuthConfig) *Authenticator {
	auth := &Authenticator{
		config:      config,
		apiKeyIndex: make(map[string]*APIKey),
		userIndex:   make(map[string]*BasicUser),
		sessions:    make(map[string]*session),
	}

	// Build API key index
	for i := range config.APIKeys {
		key := &config.APIKeys[i]
		if key.Enabled {
			// Hash the key for secure comparison
			auth.apiKeyIndex[hashKey(key.Key)] = key
		}
	}

	// Build user index
	for i := range config.BasicAuth.Users {
		user := &config.BasicAuth.Users[i]
		if user.Enabled {
			auth.userIndex[user.Username] = user
		}
	}

	// Start session cleanup goroutine
	go auth.cleanupSessions()

	return auth
}

// cleanupSessions periodically removes expired sessions
func (a *Authenticator) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.sessionMu.Lock()
		now := time.Now()
		for token, sess := range a.sessions {
			if now.After(sess.expiresAt) {
				delete(a.sessions, token)
			}
		}
		a.sessionMu.Unlock()
	}
}

// hashKey creates a secure hash of an API key for storage/comparison
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// isExemptPath checks if a path is exempt from authentication
func (a *Authenticator) isExemptPath(path string) bool {
	for _, exempt := range a.config.ExemptPaths {
		if strings.HasPrefix(path, exempt) {
			return true
		}
	}
	return false
}

// authenticateAPIKey attempts to authenticate using an API key
func (a *Authenticator) authenticateAPIKey(r *http.Request) (*AuthInfo, bool) {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		hashedToken := hashKey(token)

		if apiKey, exists := a.apiKeyIndex[hashedToken]; exists {
			return &AuthInfo{
				Authenticated: true,
				Username:      apiKey.Name,
				Role:          apiKey.Role,
				Method:        "api_key",
			}, true
		}
	}

	// Check X-API-Key header
	apiKeyHeader := r.Header.Get("X-API-Key")
	if apiKeyHeader != "" {
		hashedKey := hashKey(apiKeyHeader)
		if apiKey, exists := a.apiKeyIndex[hashedKey]; exists {
			return &AuthInfo{
				Authenticated: true,
				Username:      apiKey.Name,
				Role:          apiKey.Role,
				Method:        "api_key",
			}, true
		}
	}

	// Check query parameter (less secure, but sometimes needed)
	apiKeyParam := r.URL.Query().Get("api_key")
	if apiKeyParam != "" {
		hashedKey := hashKey(apiKeyParam)
		if apiKey, exists := a.apiKeyIndex[hashedKey]; exists {
			return &AuthInfo{
				Authenticated: true,
				Username:      apiKey.Name,
				Role:          apiKey.Role,
				Method:        "api_key",
			}, true
		}
	}

	return nil, false
}

// authenticateBasicAuth attempts to authenticate using Basic Auth
func (a *Authenticator) authenticateBasicAuth(r *http.Request) (*AuthInfo, bool) {
	if !a.config.BasicAuth.Enabled {
		return nil, false
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return nil, false
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, false
	}

	username, password := parts[0], parts[1]

	user, exists := a.userIndex[username]
	if !exists {
		return nil, false
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(password), []byte(user.Password)) != 1 {
		return nil, false
	}

	return &AuthInfo{
		Authenticated: true,
		Username:      username,
		Role:          user.Role,
		Method:        "basic",
	}, true
}

// Authenticate attempts to authenticate a request
func (a *Authenticator) Authenticate(r *http.Request) *AuthInfo {
	// Try API key first
	if info, ok := a.authenticateAPIKey(r); ok {
		return info
	}

	// Try Basic Auth
	if info, ok := a.authenticateBasicAuth(r); ok {
		return info
	}

	// No authentication
	return &AuthInfo{
		Authenticated: false,
		Method:        "none",
	}
}

// HasPermission checks if a role has a specific permission
func HasPermission(role, permission string) bool {
	switch role {
	case RoleAdmin:
		return true // Admin has all permissions
	case RoleWrite:
		return permission == PermissionRead || permission == PermissionWrite
	case RoleReadonly:
		return permission == PermissionRead
	default:
		return false
	}
}

// RequiredPermission returns the required permission for an HTTP method
func RequiredPermission(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return PermissionRead
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return PermissionWrite
	default:
		return PermissionAdmin
	}
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(auth *Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if disabled
			if !auth.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip exempt paths
			if auth.isExemptPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Authenticate the request
			authInfo := auth.Authenticate(r)

			// Allow anonymous read-only access if configured
			if !authInfo.Authenticated && auth.config.AllowAnonymousRead {
				requiredPerm := RequiredPermission(r.Method)
				if requiredPerm == PermissionRead {
					// Allow anonymous read access
					log.WithFields(log.Fields{
						"path":   r.URL.Path,
						"method": r.Method,
						"ip":     getClientIP(r),
					}).Debug("Anonymous read access allowed")

					// Set anonymous auth info in context
					ctx := context.WithValue(r.Context(), AuthUserKey, "anonymous")
					ctx = context.WithValue(ctx, AuthRoleKey, RoleReadonly)
					ctx = context.WithValue(ctx, AuthMethodKey, "anonymous")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			if !authInfo.Authenticated {
				log.WithFields(log.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
					"ip":     getClientIP(r),
				}).Warn("Authentication failed")

				// Send WWW-Authenticate header for Basic Auth
				if auth.config.BasicAuth.Enabled {
					w.Header().Set("WWW-Authenticate", `Basic realm="Licet"`)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "unauthorized",
					"message": "Authentication required",
				})
				return
			}

			// Check permission for the request method
			requiredPerm := RequiredPermission(r.Method)
			if !HasPermission(authInfo.Role, requiredPerm) {
				log.WithFields(log.Fields{
					"path":       r.URL.Path,
					"method":     r.Method,
					"user":       authInfo.Username,
					"role":       authInfo.Role,
					"required":   requiredPerm,
				}).Warn("Authorization failed - insufficient permissions")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "forbidden",
					"message": "Insufficient permissions",
					"required_permission": requiredPerm,
					"your_role": authInfo.Role,
				})
				return
			}

			// Add auth info to request context
			ctx := context.WithValue(r.Context(), AuthUserKey, authInfo.Username)
			ctx = context.WithValue(ctx, AuthRoleKey, authInfo.Role)
			ctx = context.WithValue(ctx, AuthMethodKey, authInfo.Method)

			log.WithFields(log.Fields{
				"path":   r.URL.Path,
				"method": r.Method,
				"user":   authInfo.Username,
				"role":   authInfo.Role,
			}).Debug("Request authenticated")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthInfo extracts authentication info from request context
func GetAuthInfo(r *http.Request) *AuthInfo {
	username, _ := r.Context().Value(AuthUserKey).(string)
	role, _ := r.Context().Value(AuthRoleKey).(string)
	method, _ := r.Context().Value(AuthMethodKey).(string)

	return &AuthInfo{
		Authenticated: username != "",
		Username:      username,
		Role:          role,
		Method:        method,
	}
}

// RequireRole middleware ensures the user has at least the specified role
func RequireRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(AuthRoleKey).(string)
			if !ok || role == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "unauthorized",
					"message": "Authentication required",
				})
				return
			}

			// Check role hierarchy
			roleAllowed := false
			switch minRole {
			case RoleReadonly:
				roleAllowed = role == RoleReadonly || role == RoleWrite || role == RoleAdmin
			case RoleWrite:
				roleAllowed = role == RoleWrite || role == RoleAdmin
			case RoleAdmin:
				roleAllowed = role == RoleAdmin
			}

			if !roleAllowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "forbidden",
					"message":       "Insufficient role",
					"required_role": minRole,
					"your_role":     role,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
