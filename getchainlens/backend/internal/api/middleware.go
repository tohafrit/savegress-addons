package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"getchainlens.com/chainlens/backend/internal/config"
	"getchainlens.com/chainlens/backend/internal/database"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				respondError(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok {
				respondError(w, http.StatusUnauthorized, "Invalid user ID in token")
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), "userID", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimiter implements a sliding window rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	limits   map[string]*userLimit
	cleanupT *time.Ticker
	stopCh   chan struct{}
}

type userLimit struct {
	requests []time.Time
	plan     string
}

// NewRateLimiter creates a new rate limiter with automatic cleanup
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limits:   make(map[string]*userLimit),
		cleanupT: time.NewTicker(5 * time.Minute),
		stopCh:   make(chan struct{}),
	}

	// Background cleanup of expired entries
	go func() {
		for {
			select {
			case <-rl.cleanupT.C:
				rl.cleanup()
			case <-rl.stopCh:
				rl.cleanupT.Stop()
				return
			}
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for userID, limit := range rl.limits {
		// Remove old requests
		var valid []time.Time
		for _, t := range limit.requests {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.limits, userID)
		} else {
			limit.requests = valid
		}
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// Allow checks if a request is allowed and records it
func (rl *RateLimiter) Allow(userID, plan string, window time.Duration, maxRequests int) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	limit, exists := rl.limits[userID]
	if !exists {
		limit = &userLimit{plan: plan}
		rl.limits[userID] = limit
	}

	// Filter to requests within the window
	var valid []time.Time
	for _, t := range limit.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	remaining := maxRequests - len(valid)
	resetAt := now.Add(window)

	if len(valid) > 0 {
		resetAt = valid[0].Add(window)
	}

	if remaining <= 0 {
		limit.requests = valid
		return false, 0, resetAt
	}

	// Allow and record request
	valid = append(valid, now)
	limit.requests = valid
	limit.plan = plan

	return true, remaining - 1, resetAt
}

// RateLimitMiddleware limits API calls per user based on their plan
func RateLimitMiddleware(cfg *config.Config, db *database.DB) func(http.Handler) http.Handler {
	limiter := NewRateLimiter()

	// Rate limits per plan (requests per minute)
	planLimits := map[string]int{
		"free":       60,   // 60 requests per minute
		"pro":        300,  // 300 requests per minute
		"enterprise": 1000, // 1000 requests per minute
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("userID").(string)
			if !ok {
				// No user context, allow but with IP-based limiting
				next.ServeHTTP(w, r)
				return
			}

			// Get user's plan
			plan := "free"
			if user, err := db.GetUserByID(r.Context(), userID); err == nil {
				plan = user.Plan
			}

			limit := planLimits[plan]
			if limit == 0 {
				limit = planLimits["free"]
			}

			allowed, remaining, resetAt := limiter.Allow(userID, plan, time.Minute, limit)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt).Seconds())+1))
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UsageTrackingMiddleware tracks API usage for billing
func UsageTrackingMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Track usage asynchronously
			go func() {
				userID, ok := r.Context().Value("userID").(string)
				if !ok {
					return
				}

				duration := time.Since(start)

				// Save API usage record
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_ = db.TrackAPIUsage(ctx, userID, r.URL.Path, r.Method, rw.statusCode, duration.Milliseconds())

				// Increment user's API call count
				_, _ = db.IncrementAPIUsage(ctx, userID)
			}()
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// APILimitMiddleware checks if user has exceeded their monthly API limit
func APILimitMiddleware(cfg *config.Config, db *database.DB) func(http.Handler) http.Handler {
	planLimits := map[string]int{
		"free":       cfg.FreeAPICallsPerMonth,
		"pro":        cfg.ProAPICallsPerMonth,
		"enterprise": 1000000, // Effectively unlimited
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("userID").(string)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			user, err := db.GetUserByID(r.Context(), userID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			limit := planLimits[user.Plan]
			if limit == 0 {
				limit = planLimits["free"]
			}

			// Check if user has exceeded monthly limit
			if user.APICallsUsed >= limit {
				// Check if reset is due
				if time.Now().After(user.APICallsResetAt) {
					// Reset counter
					_ = db.ResetAPIUsage(r.Context(), userID)
				} else {
					w.Header().Set("X-API-Limit-Exceeded", "true")
					w.Header().Set("X-API-Limit-Reset", user.APICallsResetAt.Format(time.RFC3339))
					respondError(w, http.StatusPaymentRequired, "Monthly API limit exceeded. Please upgrade your plan.")
					return
				}
			}

			// Set usage headers
			w.Header().Set("X-API-Calls-Used", fmt.Sprintf("%d", user.APICallsUsed))
			w.Header().Set("X-API-Calls-Limit", fmt.Sprintf("%d", limit))

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), "requestID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			fmt.Printf("[%s] %s %s %d %s\n",
				time.Now().Format("2006-01-02 15:04:05"),
				r.Method,
				r.URL.Path,
				rw.statusCode,
				duration,
			)
		})
	}
}

// APIKeyInfo represents API key information
type APIKeyInfo struct {
	ID          string
	UserID      string
	Name        string
	Permissions []string
	RateLimit   int // requests per minute
	Active      bool
}

// APIKeyStore interface for API key validation
type APIKeyStore interface {
	ValidateKey(ctx context.Context, key string) (*APIKeyInfo, error)
}

// InMemoryAPIKeyStore provides in-memory API key storage
type InMemoryAPIKeyStore struct {
	keys map[string]*APIKeyInfo
	mu   sync.RWMutex
}

// NewInMemoryAPIKeyStore creates a new in-memory key store
func NewInMemoryAPIKeyStore() *InMemoryAPIKeyStore {
	return &InMemoryAPIKeyStore{
		keys: make(map[string]*APIKeyInfo),
	}
}

// AddKey adds an API key
func (s *InMemoryAPIKeyStore) AddKey(key string, info *APIKeyInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key] = info
}

// ValidateKey validates an API key
func (s *InMemoryAPIKeyStore) ValidateKey(ctx context.Context, key string) (*APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.keys[key]
	if !ok || !info.Active {
		return nil, nil
	}
	return info, nil
}

// APIKeyAuthMiddleware validates API keys
func APIKeyAuthMiddleware(store APIKeyStore, required bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractAPIKey(r)

			if key == "" {
				if required {
					respondError(w, http.StatusUnauthorized, "API key required")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			info, err := store.ValidateKey(r.Context(), key)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Internal server error")
				return
			}

			if info == nil {
				respondError(w, http.StatusUnauthorized, "Invalid API key")
				return
			}

			// Add key info to context
			ctx := context.WithValue(r.Context(), "apiKeyInfo", info)
			ctx = context.WithValue(ctx, "userID", info.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAPIKey extracts API key from request
func extractAPIKey(r *http.Request) string {
	// Check X-API-Key header
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	// Check Authorization header with Bearer token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check query parameter
	if key := r.URL.Query().Get("api_key"); key != "" {
		return key
	}

	return ""
}

// SecureHeadersMiddleware adds security headers
func SecureHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission middleware checks for specific permission
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := r.Context().Value("apiKeyInfo").(*APIKeyInfo)
			if !ok {
				respondError(w, http.StatusUnauthorized, "API key required")
				return
			}

			hasPermission := false
			for _, p := range info.Permissions {
				if p == permission || p == "*" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts client IP from request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
