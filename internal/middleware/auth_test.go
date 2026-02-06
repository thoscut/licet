package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"licet/internal/config"
)

func newTestAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		Enabled:            true,
		AllowAnonymousRead: false,
		SessionTimeout:     60,
		ExemptPaths:        []string{"/api/v1/health"},
		APIKeys: []config.APIKeyConfig{
			{Name: "test-key", Key: "secret123", Role: RoleAdmin, Enabled: true},
			{Name: "readonly-key", Key: "readonly456", Role: RoleReadonly, Enabled: true},
			{Name: "disabled-key", Key: "disabled789", Role: RoleAdmin, Enabled: false},
		},
		BasicAuth: config.BasicAuthConfig{
			Enabled: true,
			Users: []config.BasicUserConfig{
				{Username: "admin", Password: "admin-pass", Role: RoleAdmin, Enabled: true},
				{Username: "reader", Password: "reader-pass", Role: RoleReadonly, Enabled: true},
				{Username: "disabled", Password: "disabled-pass", Role: RoleAdmin, Enabled: false},
			},
		},
	}
}

func TestAuthenticateAPIKey_BearerToken(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer secret123")

	info := auth.Authenticate(req)
	if !info.Authenticated {
		t.Fatal("expected authenticated")
	}
	if info.Username != "test-key" {
		t.Errorf("expected username 'test-key', got %q", info.Username)
	}
	if info.Role != RoleAdmin {
		t.Errorf("expected role 'admin', got %q", info.Role)
	}
	if info.Method != "api_key" {
		t.Errorf("expected method 'api_key', got %q", info.Method)
	}
}

func TestAuthenticateAPIKey_XAPIKeyHeader(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("X-API-Key", "readonly456")

	info := auth.Authenticate(req)
	if !info.Authenticated {
		t.Fatal("expected authenticated")
	}
	if info.Role != RoleReadonly {
		t.Errorf("expected role 'readonly', got %q", info.Role)
	}
}

func TestAuthenticateAPIKey_InvalidKey(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")

	info := auth.Authenticate(req)
	if info.Authenticated {
		t.Fatal("expected not authenticated with wrong key")
	}
}

func TestAuthenticateAPIKey_DisabledKey(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer disabled789")

	info := auth.Authenticate(req)
	if info.Authenticated {
		t.Fatal("expected not authenticated with disabled key")
	}
}

func TestAuthenticateBasicAuth_ValidCredentials(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	cred := base64.StdEncoding.EncodeToString([]byte("admin:admin-pass"))
	req.Header.Set("Authorization", "Basic "+cred)

	info := auth.Authenticate(req)
	if !info.Authenticated {
		t.Fatal("expected authenticated")
	}
	if info.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", info.Username)
	}
	if info.Method != "basic" {
		t.Errorf("expected method 'basic', got %q", info.Method)
	}
}

func TestAuthenticateBasicAuth_WrongPassword(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	cred := base64.StdEncoding.EncodeToString([]byte("admin:wrong-pass"))
	req.Header.Set("Authorization", "Basic "+cred)

	info := auth.Authenticate(req)
	if info.Authenticated {
		t.Fatal("expected not authenticated with wrong password")
	}
}

func TestAuthenticateBasicAuth_DisabledUser(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	cred := base64.StdEncoding.EncodeToString([]byte("disabled:disabled-pass"))
	req.Header.Set("Authorization", "Basic "+cred)

	info := auth.Authenticate(req)
	if info.Authenticated {
		t.Fatal("expected not authenticated with disabled user")
	}
}

func TestAuthenticateNoCredentials(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	info := auth.Authenticate(req)
	if info.Authenticated {
		t.Fatal("expected not authenticated without credentials")
	}
	if info.Method != "none" {
		t.Errorf("expected method 'none', got %q", info.Method)
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role       string
		permission string
		want       bool
	}{
		{RoleAdmin, PermissionRead, true},
		{RoleAdmin, PermissionWrite, true},
		{RoleAdmin, PermissionAdmin, true},
		{RoleWrite, PermissionRead, true},
		{RoleWrite, PermissionWrite, true},
		{RoleWrite, PermissionAdmin, false},
		{RoleReadonly, PermissionRead, true},
		{RoleReadonly, PermissionWrite, false},
		{RoleReadonly, PermissionAdmin, false},
		{"unknown", PermissionRead, false},
		{"", PermissionRead, false},
	}

	for _, tt := range tests {
		got := HasPermission(tt.role, tt.permission)
		if got != tt.want {
			t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
		}
	}
}

func TestRequiredPermission(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{http.MethodGet, PermissionRead},
		{http.MethodHead, PermissionRead},
		{http.MethodOptions, PermissionRead},
		{http.MethodPost, PermissionWrite},
		{http.MethodPut, PermissionWrite},
		{http.MethodPatch, PermissionWrite},
		{http.MethodDelete, PermissionWrite},
		{"CUSTOM", PermissionAdmin},
	}

	for _, tt := range tests {
		got := RequiredPermission(tt.method)
		if got != tt.want {
			t.Errorf("RequiredPermission(%q) = %q, want %q", tt.method, got, tt.want)
		}
	}
}

func TestAuthMiddleware_ExemptPath(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	handler := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("exempt path returned status %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_Unauthorized(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	handler := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request returned status %d, want %d", rr.Code, http.StatusUnauthorized)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %v", resp["error"])
	}
}

func TestAuthMiddleware_Forbidden(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	handler := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Readonly user trying to POST
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", nil)
	req.Header.Set("X-API-Key", "readonly456")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("insufficient permissions returned status %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAuthMiddleware_AuthenticatedRequest(t *testing.T) {
	auth := NewAuthenticator(newTestAuthConfig())
	defer auth.Stop()

	var capturedInfo *AuthInfo
	handler := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedInfo = GetAuthInfo(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("authenticated request returned status %d, want %d", rr.Code, http.StatusOK)
	}
	if capturedInfo == nil {
		t.Fatal("expected auth info in context")
	}
	if !capturedInfo.Authenticated {
		t.Error("expected authenticated=true in context")
	}
	if capturedInfo.Username != "test-key" {
		t.Errorf("expected username 'test-key' in context, got %q", capturedInfo.Username)
	}
}

func TestAuthMiddleware_AnonymousReadAccess(t *testing.T) {
	cfg := newTestAuthConfig()
	cfg.AllowAnonymousRead = true
	auth := NewAuthenticator(cfg)
	defer auth.Stop()

	handler := AuthMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// GET should be allowed for anonymous
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("anonymous GET returned status %d, want %d", rr.Code, http.StatusOK)
	}

	// POST should still be blocked
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("anonymous POST returned status %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestGetAuthInfo_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	info := GetAuthInfo(req)
	if info.Authenticated {
		t.Error("expected not authenticated when no context set")
	}
	if info.Method != "none" {
		t.Errorf("expected method 'none', got %q", info.Method)
	}
}
