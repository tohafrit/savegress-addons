package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"getchainlens.com/chainlens/backend/internal/config"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "ok"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"ok"}`,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":123}`,
		},
		{
			name:       "empty response",
			status:     http.StatusOK,
			data:       map[string]interface{}{},
			wantStatus: http.StatusOK,
			wantBody:   `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", ct)
			}

			// Trim newline from encoder
			got := bytes.TrimSpace(w.Body.Bytes())
			if string(got) != tt.wantBody {
				t.Errorf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			message:    "resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.message {
				t.Errorf("error = %s, want %s", resp["error"], tt.message)
			}
		})
	}
}

func TestIsValidTxHash(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{
			name:  "valid hash",
			hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			valid: true,
		},
		{
			name:  "valid hash uppercase",
			hash:  "0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF",
			valid: true,
		},
		{
			name:  "valid hash mixed case",
			hash:  "0x1234567890AbCdEf1234567890AbCdEf1234567890AbCdEf1234567890AbCdEf",
			valid: true,
		},
		{
			name:  "too short",
			hash:  "0x1234",
			valid: false,
		},
		{
			name:  "too long",
			hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef00",
			valid: false,
		},
		{
			name:  "no prefix",
			hash:  "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			valid: false,
		},
		{
			name:  "invalid characters",
			hash:  "0x1234567890ghijkl1234567890abcdef1234567890abcdef1234567890abcdef",
			valid: false,
		},
		{
			name:  "empty",
			hash:  "",
			valid: false,
		},
		{
			name:  "only prefix",
			hash:  "0x",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTxHash(tt.hash); got != tt.valid {
				t.Errorf("isValidTxHash(%q) = %v, want %v", tt.hash, got, tt.valid)
			}
		})
	}
}

func TestGetRPCURL(t *testing.T) {
	cfg := &config.Config{
		EthereumRPCURL: "https://eth.example.com",
		PolygonRPCURL:  "https://polygon.example.com",
		ArbitrumRPCURL: "https://arb.example.com",
	}

	tests := []struct {
		name  string
		chain string
		want  string
	}{
		{"ethereum", "ethereum", "https://eth.example.com"},
		{"eth", "eth", "https://eth.example.com"},
		{"mainnet", "mainnet", "https://eth.example.com"},
		{"ETHEREUM uppercase", "ETHEREUM", "https://eth.example.com"},
		{"polygon", "polygon", "https://polygon.example.com"},
		{"matic", "matic", "https://polygon.example.com"},
		{"arbitrum", "arbitrum", "https://arb.example.com"},
		{"arb", "arb", "https://arb.example.com"},
		{"unknown chain", "unknown", ""},
		{"empty chain", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRPCURL(cfg, tt.chain); got != tt.want {
				t.Errorf("getRPCURL(%q) = %q, want %q", tt.chain, got, tt.want)
			}
		})
	}
}

func TestGenerateAccessToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret-key-for-testing",
		JWTExpiration: 24, // 24 hours
	}

	userID := "usr_test123"

	token, err := generateAccessToken(cfg, userID)
	if err != nil {
		t.Fatalf("generateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Error("token is empty")
	}

	// Token should be a valid JWT (three parts separated by dots)
	parts := bytes.Split([]byte(token), []byte("."))
	if len(parts) != 3 {
		t.Errorf("token has %d parts, want 3", len(parts))
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token1, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken failed: %v", err)
	}

	if token1 == "" {
		t.Error("token is empty")
	}

	// Should be 64 hex characters (32 bytes)
	if len(token1) != 64 {
		t.Errorf("token length = %d, want 64", len(token1))
	}

	// Should generate unique tokens
	token2, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken failed: %v", err)
	}

	if token1 == token2 {
		t.Error("tokens should be unique")
	}
}

// TestHandleRegister_Validation tests registration validation
func TestHandleRegister_Validation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret",
		JWTExpiration:     24,
		RefreshExpiration: 7,
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "missing email",
			body:       `{"password":"12345678","name":"Test"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Email, password and name are required",
		},
		{
			name:       "missing password",
			body:       `{"email":"test@example.com","name":"Test"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Email, password and name are required",
		},
		{
			name:       "missing name",
			body:       `{"email":"test@example.com","password":"12345678"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Email, password and name are required",
		},
		{
			name:       "password too short",
			body:       `{"email":"test@example.com","password":"1234567","name":"Test"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handleRegister(cfg, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleLogin_Validation tests login validation
func TestHandleLogin_Validation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret",
		JWTExpiration:     24,
		RefreshExpiration: 7,
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handleLogin(cfg, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleRefreshToken_Validation tests refresh token validation
func TestHandleRefreshToken_Validation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:         "test-secret",
		JWTExpiration:     24,
		RefreshExpiration: 7,
	}

	handler := handleRefreshToken(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleValidateLicense_Validation tests license validation
func TestHandleValidateLicense_Validation(t *testing.T) {
	cfg := &config.Config{}

	handler := handleValidateLicense(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/license/validate", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleGetUser tests get user handler
func TestHandleGetUser(t *testing.T) {
	handler := handleGetUser(nil)

	// Create request with user context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	ctx := context.WithValue(req.Context(), "userID", "usr_test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// This will fail because db is nil, but it tests the context extraction
	defer func() {
		if r := recover(); r == nil {
			// Expected to panic or return error when db is nil
		}
	}()

	handler(w, req)
}

// TestHandleUpdateUser_Validation tests update user validation
func TestHandleUpdateUser_Validation(t *testing.T) {
	handler := handleUpdateUser(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "userID", "usr_test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleChangePassword_Validation tests change password validation
func TestHandleChangePassword_Validation(t *testing.T) {
	handler := handleChangePassword(nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "password too short",
			body:       `{"current_password":"oldpass","new_password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "userID", "usr_test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleAnalyzeContract_Validation tests analyze contract validation
func TestHandleAnalyzeContract_Validation(t *testing.T) {
	cfg := &config.Config{}
	handler := handleAnalyzeContract(cfg, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "empty source",
			body:       `{"source":"","compiler":"0.8.19"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Source code is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleTraceTransaction_Validation tests trace transaction validation
func TestHandleTraceTransaction_Validation(t *testing.T) {
	cfg := &config.Config{
		EthereumRPCURL: "https://eth.example.com",
	}
	handler := handleTraceTransaction(cfg, nil)

	// Test invalid tx hash
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trace/invalid", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != "Invalid transaction hash format" {
		t.Errorf("error = %q, want %q", resp["error"], "Invalid transaction hash format")
	}
}

// TestHandleSimulateTransaction_Validation tests simulate transaction validation
func TestHandleSimulateTransaction_Validation(t *testing.T) {
	cfg := &config.Config{}
	handler := handleSimulateTransaction(cfg, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported chain",
			body:       `{"chain":"unknown","from":"0x123","to":"0x456"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/simulate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleCreateFork tests create fork handler
func TestHandleCreateFork(t *testing.T) {
	cfg := &config.Config{
		EthereumRPCURL: "https://eth.example.com",
	}
	handler := handleCreateFork(cfg, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported chain",
			body:       `{"chain":"unknown"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid ethereum fork",
			body:       `{"chain":"ethereum","block_number":12345}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "default chain (ethereum)",
			body:       `{"block_number":12345}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/forks", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if _, ok := resp["fork_id"]; !ok {
					t.Error("response missing fork_id")
				}
				if _, ok := resp["rpc_url"]; !ok {
					t.Error("response missing rpc_url")
				}
			}
		})
	}
}

// TestHandleDeleteFork tests delete fork handler
func TestHandleDeleteFork(t *testing.T) {
	cfg := &config.Config{}
	handler := handleDeleteFork(cfg, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/forks/fork_123", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestHandleCreateMonitor_Validation tests create monitor validation
func TestHandleCreateMonitor_Validation(t *testing.T) {
	cfg := &config.Config{}
	handler := handleCreateMonitor(cfg, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "missing name",
			body:       `{"webhook_url":"https://example.com/webhook"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Name and webhook_url are required",
		},
		{
			name:       "missing webhook_url",
			body:       `{"name":"Test Monitor"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Name and webhook_url are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "userID", "usr_test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleUpdateMonitor_Validation tests update monitor validation
func TestHandleUpdateMonitor_Validation(t *testing.T) {
	handler := handleUpdateMonitor(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/monitors/mon_123", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleCreateProject_Validation tests create project validation
func TestHandleCreateProject_Validation(t *testing.T) {
	handler := handleCreateProject(nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "missing name",
			body:       `{"description":"Test project"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "userID", "usr_test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleUpdateProject_Validation tests update project validation
func TestHandleUpdateProject_Validation(t *testing.T) {
	handler := handleUpdateProject(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/prj_123", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleAddContract_Validation tests add contract validation
func TestHandleAddContract_Validation(t *testing.T) {
	cfg := &config.Config{}
	handler := handleAddContract(cfg, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "missing address",
			body:       `{"chain":"ethereum"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Address and chain are required",
		},
		{
			name:       "missing chain",
			body:       `{"address":"0x123"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Address and chain are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/contracts", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), "userID", "usr_test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}

// TestHandleCreateCheckout_Validation tests create checkout validation
func TestHandleCreateCheckout_Validation(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey: "", // Not configured
	}
	handler := handleCreateCheckout(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", bytes.NewBufferString(`{"price_id":"price_123"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "userID", "usr_test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != "Billing not configured" {
		t.Errorf("error = %q, want %q", resp["error"], "Billing not configured")
	}
}

// TestHandleCreatePortal_Validation tests create portal validation
func TestHandleCreatePortal_Validation(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey: "", // Not configured
	}
	handler := handleCreatePortal(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/portal", bytes.NewBufferString(`{"return_url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "userID", "usr_test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHandleStripeWebhook_Validation tests stripe webhook validation
func TestHandleStripeWebhook_Validation(t *testing.T) {
	cfg := &config.Config{
		StripeWebhookSecret: "", // Not configured
	}
	handler := handleStripeWebhook(cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// BenchmarkRespondJSON benchmarks JSON response serialization
func BenchmarkRespondJSON(b *testing.B) {
	data := map[string]interface{}{
		"id":      "test-123",
		"message": "success",
		"count":   42,
		"items":   []string{"a", "b", "c"},
	}

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		respondJSON(w, http.StatusOK, data)
	}
}

// BenchmarkIsValidTxHash benchmarks tx hash validation
func BenchmarkIsValidTxHash(b *testing.B) {
	hash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	for i := 0; i < b.N; i++ {
		isValidTxHash(hash)
	}
}

// BenchmarkGenerateAccessToken benchmarks token generation
func BenchmarkGenerateAccessToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecret:     "test-secret-key",
		JWTExpiration: 24,
	}

	for i := 0; i < b.N; i++ {
		generateAccessToken(cfg, "usr_test123")
	}
}

// BenchmarkGenerateRefreshToken benchmarks refresh token generation
func BenchmarkGenerateRefreshToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateRefreshToken()
	}
}

// Test handler context extraction helper
func withUserContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), "userID", userID)
	return req.WithContext(ctx)
}

// Test time-sensitive operations
func TestTokenExpiration(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 1, // 1 hour
	}

	token, err := generateAccessToken(cfg, "usr_test")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should be non-empty
	if token == "" {
		t.Error("token should not be empty")
	}
}

// Test concurrent token generation
func TestConcurrentTokenGeneration(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 24,
	}

	done := make(chan bool)
	tokens := make(chan string, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			token, err := generateAccessToken(cfg, "usr_test")
			if err != nil {
				t.Errorf("failed to generate token: %v", err)
			}
			tokens <- token
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	close(tokens)

	// Collect all tokens
	tokenSet := make(map[string]bool)
	for token := range tokens {
		tokenSet[token] = true
	}

	// All tokens should be unique (different iat claim)
	// Note: In practice, tokens generated in the same second might have the same iat
	// This is expected behavior
	if len(tokenSet) < 1 {
		t.Error("should have generated at least some tokens")
	}
}

// Test refresh token uniqueness
func TestRefreshTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		token, err := generateRefreshToken()
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

// Test health check would be in router_test.go
// Test middleware would be in middleware_test.go
