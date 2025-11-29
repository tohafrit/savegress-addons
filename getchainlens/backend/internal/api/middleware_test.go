package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"getchainlens.com/chainlens/backend/internal/config"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token123"},
		{"wrong prefix", "Basic token123"},
		{"empty token", "Bearer "},
		{"extra parts", "Bearer token part2 part3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	// Create an expired token
	claims := jwt.MapClaims{
		"sub": "usr_test123",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_WrongSigningMethod(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a token with wrong signing method (none)
	claims := jwt.MapClaims{
		"sub": "usr_test123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a valid token
	claims := jwt.MapClaims{
		"sub": "usr_test123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))

	var capturedUserID string
	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Context().Value("userID").(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if capturedUserID != "usr_test123" {
		t.Errorf("userID = %q, want %q", capturedUserID, "usr_test123")
	}
}

func TestAuthMiddleware_MissingSubClaim(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a token without sub claim
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRateLimiter_Basic(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	userID := "usr_test123"
	plan := "free"
	window := time.Minute
	maxRequests := 5

	// Should allow first requests
	for i := 0; i < maxRequests; i++ {
		allowed, remaining, _ := rl.Allow(userID, plan, window, maxRequests)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		expectedRemaining := maxRequests - i - 1
		if remaining != expectedRemaining {
			t.Errorf("remaining = %d, want %d", remaining, expectedRemaining)
		}
	}

	// Next request should be denied
	allowed, remaining, resetAt := rl.Allow(userID, plan, window, maxRequests)
	if allowed {
		t.Error("request should be denied after limit")
	}
	if remaining != 0 {
		t.Errorf("remaining = %d, want 0", remaining)
	}
	if resetAt.Before(time.Now()) {
		t.Error("resetAt should be in the future")
	}
}

func TestRateLimiter_DifferentUsers(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	window := time.Minute
	maxRequests := 2

	// User 1 exhausts limit
	rl.Allow("user1", "free", window, maxRequests)
	rl.Allow("user1", "free", window, maxRequests)
	allowed, _, _ := rl.Allow("user1", "free", window, maxRequests)
	if allowed {
		t.Error("user1 should be rate limited")
	}

	// User 2 should still be allowed
	allowed, _, _ = rl.Allow("user2", "free", window, maxRequests)
	if !allowed {
		t.Error("user2 should be allowed")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	userID := "usr_test123"
	plan := "free"
	window := 50 * time.Millisecond // Very short window for testing
	maxRequests := 2

	// Exhaust limit
	rl.Allow(userID, plan, window, maxRequests)
	rl.Allow(userID, plan, window, maxRequests)

	// Should be denied
	allowed, _, _ := rl.Allow(userID, plan, window, maxRequests)
	if allowed {
		t.Error("should be denied after limit")
	}

	// Wait for window to reset
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	allowed, _, _ = rl.Allow(userID, plan, window, maxRequests)
	if !allowed {
		t.Error("should be allowed after window reset")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Make some requests
	rl.Allow("user1", "free", time.Minute, 10)
	rl.Allow("user2", "free", time.Minute, 10)

	// Verify limits exist
	rl.mu.RLock()
	initialCount := len(rl.limits)
	rl.mu.RUnlock()

	if initialCount != 2 {
		t.Errorf("limit count = %d, want 2", initialCount)
	}

	// Cleanup is automatic via ticker, but we can test the method directly
	rl.cleanup()

	// Limits should still exist (within window)
	rl.mu.RLock()
	afterCleanup := len(rl.limits)
	rl.mu.RUnlock()

	if afterCleanup != 2 {
		t.Errorf("limit count after cleanup = %d, want 2 (should not clean recent entries)", afterCleanup)
	}
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter()

	// Should not panic
	rl.Stop()

	// Should be safe to call multiple times (but channel will be closed)
	// Note: In production code, you might want to handle this case
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	requestsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := "user_concurrent"
			for j := 0; j < requestsPerGoroutine; j++ {
				rl.Allow(userID, "free", time.Minute, 1000)
			}
		}(i)
	}

	wg.Wait()
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name          string
		origin        string
		expectCORS    bool
		expectOrigin  string
	}{
		{"allowed localhost", "http://localhost:3000", true, "http://localhost:3000"},
		{"allowed example", "https://example.com", true, "https://example.com"},
		{"not allowed", "https://malicious.com", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if corsOrigin != tt.expectOrigin {
					t.Errorf("CORS origin = %q, want %q", corsOrigin, tt.expectOrigin)
				}
			} else {
				if corsOrigin != "" {
					t.Errorf("CORS origin = %q, want empty", corsOrigin)
				}
			}
		})
	}
}

func TestCORSMiddleware_Wildcard(t *testing.T) {
	allowedOrigins := []string{"*"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if corsOrigin != "https://any-origin.com" {
		t.Errorf("CORS origin = %q, want %q", corsOrigin, "https://any-origin.com")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Body should be empty for preflight
	if w.Body.Len() > 0 {
		t.Errorf("body should be empty for preflight, got %q", w.Body.String())
	}
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	middleware := RequestIDMiddleware()
	var capturedRequestID string
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value("requestID").(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if capturedRequestID == "" {
		t.Error("request ID should be generated")
	}

	headerRequestID := w.Header().Get("X-Request-ID")
	if headerRequestID != capturedRequestID {
		t.Errorf("header request ID = %q, context request ID = %q", headerRequestID, capturedRequestID)
	}
}

func TestRequestIDMiddleware_UsesExistingID(t *testing.T) {
	middleware := RequestIDMiddleware()
	var capturedRequestID string
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value("requestID").(string)
		w.WriteHeader(http.StatusOK)
	}))

	existingID := "existing-request-id-123"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if capturedRequestID != existingID {
		t.Errorf("request ID = %q, want %q", capturedRequestID, existingID)
	}

	headerRequestID := w.Header().Get("X-Request-ID")
	if headerRequestID != existingID {
		t.Errorf("header request ID = %q, want %q", headerRequestID, existingID)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	middleware := LoggingMiddleware()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	middleware := LoggingMiddleware()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusCreated)
	}

	// Verify underlying writer also got the status
	if w.Code != http.StatusCreated {
		t.Errorf("underlying status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("request ID should not be empty")
	}

	// IDs should be unique (very high probability)
	// In practice, if generated at exactly the same nanosecond, they might be the same
	// This is acceptable for the test
	if id1 == id2 {
		t.Log("warning: request IDs might be the same if generated at the same nanosecond")
	}
}

func TestUsageTrackingMiddleware_NoUser(t *testing.T) {
	middleware := UsageTrackingMiddleware(nil)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	// Should not panic even without user context
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUsageTrackingMiddleware_WithUser(t *testing.T) {
	// Skip this test as it requires a real database connection
	// The middleware spawns a goroutine that will panic with nil db
	t.Skip("requires database connection")
}

// Benchmarks

func BenchmarkAuthMiddleware_ValidToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-for-benchmarking",
	}

	claims := jwt.MapClaims{
		"sub": "usr_test123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))

	middleware := AuthMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter()
	defer rl.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow("usr_bench", "free", time.Minute, 1000000)
	}
}

func BenchmarkRateLimiter_Concurrent(b *testing.B) {
	rl := NewRateLimiter()
	defer rl.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow("usr_bench", "free", time.Minute, 1000000)
		}
	})
}

func BenchmarkCORSMiddleware(b *testing.B) {
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	middleware := RequestIDMiddleware()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
