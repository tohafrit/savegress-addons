package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"getchainlens.com/chainlens/backend/internal/config"
)

func TestNewRouter_Health(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	expectedBody := `{"status":"ok","service":"chainlens-api","version":"0.1.0"}`
	if w.Body.String() != expectedBody {
		t.Errorf("body = %q, want %q", w.Body.String(), expectedBody)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestNewRouter_PublicRoutes(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	// These routes should be accessible without auth (though they may fail due to nil db)
	publicRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/license/validate"},
	}

	for _, route := range publicRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not get 404 (route exists)
			if w.Code == http.StatusNotFound {
				t.Errorf("route %s %s not found", route.method, route.path)
			}

			// Should not get 401 (public route)
			if w.Code == http.StatusUnauthorized {
				t.Errorf("route %s %s should be public", route.method, route.path)
			}
		})
	}
}

func TestNewRouter_ProtectedRoutes_NoAuth(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	// These routes require authentication
	protectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/user"},
		{http.MethodPut, "/api/v1/user"},
		{http.MethodPut, "/api/v1/user/password"},
		{http.MethodPost, "/api/v1/analyze"},
		{http.MethodGet, "/api/v1/trace/0x123"},
		{http.MethodPost, "/api/v1/simulate"},
		{http.MethodPost, "/api/v1/forks"},
		{http.MethodDelete, "/api/v1/forks/123"},
		{http.MethodGet, "/api/v1/monitors"},
		{http.MethodPost, "/api/v1/monitors"},
		{http.MethodPut, "/api/v1/monitors/123"},
		{http.MethodDelete, "/api/v1/monitors/123"},
		{http.MethodGet, "/api/v1/projects"},
		{http.MethodPost, "/api/v1/projects"},
		{http.MethodGet, "/api/v1/projects/123"},
		{http.MethodPut, "/api/v1/projects/123"},
		{http.MethodDelete, "/api/v1/projects/123"},
		{http.MethodGet, "/api/v1/contracts"},
		{http.MethodPost, "/api/v1/contracts"},
		{http.MethodDelete, "/api/v1/contracts/123"},
		{http.MethodGet, "/api/v1/billing/subscription"},
		{http.MethodPost, "/api/v1/billing/checkout"},
		{http.MethodPost, "/api/v1/billing/portal"},
		{http.MethodGet, "/api/v1/usage"},
	}

	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should get 401 without auth
			if w.Code != http.StatusUnauthorized {
				t.Errorf("route %s %s status = %d, want %d (should require auth)",
					route.method, route.path, w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestNewRouter_WebhookRoute(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:           "test-secret",
		StripeWebhookSecret: "", // Not configured
	}

	router := NewRouter(cfg, nil)

	// Webhook route should be public but fail without proper config
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should not get 401 (webhook is public)
	if w.Code == http.StatusUnauthorized {
		t.Error("webhook route should be public")
	}

	// Should get 503 (webhook not configured)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (webhook not configured)", w.Code, http.StatusServiceUnavailable)
	}
}

func TestNewRouter_NotFound(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewRouter_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	// Try GET on POST-only route
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewRouter_CORS(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	// Test CORS preflight
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/user", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check CORS headers
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == "" {
		t.Error("missing Access-Control-Allow-Origin header")
	}
}

func TestNewRouter_ContentType(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// Benchmark router performance
func BenchmarkRouter_Health(b *testing.B) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_AuthRequired(b *testing.B) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	router := NewRouter(cfg, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
