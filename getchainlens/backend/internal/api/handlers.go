package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"getchainlens.com/chainlens/backend/internal/analyzer"
	"getchainlens.com/chainlens/backend/internal/billing"
	"getchainlens.com/chainlens/backend/internal/config"
	"getchainlens.com/chainlens/backend/internal/database"
	"getchainlens.com/chainlens/backend/internal/tracer"
)

// Response helpers
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// =============================================================================
// Auth handlers
// =============================================================================

func handleRegister(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate input
		if req.Email == "" || req.Password == "" || req.Name == "" {
			respondError(w, http.StatusBadRequest, "Email, password and name are required")
			return
		}

		if len(req.Password) < 8 {
			respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}

		// Create user
		user, err := db.CreateUser(r.Context(), req.Email, string(hashedPassword), req.Name)
		if err != nil {
			if errors.Is(err, database.ErrDuplicateEmail) {
				respondError(w, http.StatusConflict, "Email already registered")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		// Generate tokens
		accessToken, err := generateAccessToken(cfg, user.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken, err := generateRefreshToken()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
			return
		}

		// Save refresh token
		expiresAt := time.Now().UTC().AddDate(0, 0, cfg.RefreshExpiration)
		_, err = db.CreateRefreshToken(r.Context(), user.ID, refreshToken, expiresAt)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to save refresh token")
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
				"plan":  user.Plan,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_in":    cfg.JWTExpiration * 3600,
		})
	}
}

func handleLogin(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Get user by email
		user, err := db.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				respondError(w, http.StatusUnauthorized, "Invalid email or password")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to authenticate")
			return
		}

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		// Generate tokens
		accessToken, err := generateAccessToken(cfg, user.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken, err := generateRefreshToken()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
			return
		}

		// Save refresh token
		expiresAt := time.Now().UTC().AddDate(0, 0, cfg.RefreshExpiration)
		_, err = db.CreateRefreshToken(r.Context(), user.ID, refreshToken, expiresAt)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to save refresh token")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
				"plan":  user.Plan,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_in":    cfg.JWTExpiration * 3600,
		})
	}
}

func handleRefreshToken(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate refresh token
		rt, err := db.GetRefreshToken(r.Context(), req.RefreshToken)
		if err != nil {
			if errors.Is(err, database.ErrInvalidToken) {
				respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to validate token")
			return
		}

		// Delete old refresh token
		_ = db.DeleteRefreshToken(r.Context(), req.RefreshToken)

		// Generate new tokens
		accessToken, err := generateAccessToken(cfg, rt.UserID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		newRefreshToken, err := generateRefreshToken()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
			return
		}

		// Save new refresh token
		expiresAt := time.Now().UTC().AddDate(0, 0, cfg.RefreshExpiration)
		_, err = db.CreateRefreshToken(r.Context(), rt.UserID, newRefreshToken, expiresAt)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to save refresh token")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": newRefreshToken,
			"expires_in":    cfg.JWTExpiration * 3600,
		})
	}
}

func handleValidateLicense(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			LicenseKey string `json:"license_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		license, err := db.GetLicenseByKey(r.Context(), req.LicenseKey)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				respondJSON(w, http.StatusOK, map[string]interface{}{
					"valid":    false,
					"tier":     "free",
					"features": []string{},
				})
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to validate license")
			return
		}

		// Check expiration
		if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"valid":   false,
				"tier":    "free",
				"message": "License expired",
			})
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":    true,
			"tier":     license.Tier,
			"features": license.Features,
		})
	}
}

// =============================================================================
// User handlers
// =============================================================================

func handleGetUser(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				respondError(w, http.StatusNotFound, "User not found")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to get user")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"plan":           user.Plan,
			"api_calls_used": user.APICallsUsed,
			"email_verified": user.EmailVerified,
			"created_at":     user.CreatedAt,
		})
	}
}

func handleUpdateUser(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := db.UpdateUser(r.Context(), userID, req.Name); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "User updated successfully",
		})
	}
}

func handleChangePassword(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if len(req.NewPassword) < 8 {
			respondError(w, http.StatusBadRequest, "New password must be at least 8 characters")
			return
		}

		// Get user and verify current password
		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get user")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			respondError(w, http.StatusUnauthorized, "Current password is incorrect")
			return
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}

		if err := db.UpdateUserPassword(r.Context(), userID, string(hashedPassword)); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}

		// Invalidate all refresh tokens
		_ = db.DeleteUserRefreshTokens(r.Context(), userID)

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Password changed successfully",
		})
	}
}

// =============================================================================
// Analysis handlers
// =============================================================================

func handleAnalyzeContract(cfg *config.Config, db *database.DB) http.HandlerFunc {
	// Create analyzer with worker pool (4 workers for analysis tasks)
	contractAnalyzer, err := analyzer.NewAnalyzer(4)
	if err != nil {
		// Log error but don't fail - will return error on requests
		fmt.Printf("Failed to create analyzer: %v\n", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if contractAnalyzer == nil {
			respondError(w, http.StatusInternalServerError, "Analyzer not available")
			return
		}

		var req struct {
			Source   string `json:"source"`
			Compiler string `json:"compiler"`
			Options  struct {
				Security bool `json:"security"`
				Gas      bool `json:"gas"`
				Patterns bool `json:"patterns"`
			} `json:"options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Source == "" {
			respondError(w, http.StatusBadRequest, "Source code is required")
			return
		}

		// Run analysis using worker pool
		result, err := contractAnalyzer.Analyze(r.Context(), req.Source, req.Compiler)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
			return
		}

		// Save analysis result if user is authenticated
		if userID := r.Context().Value("userID"); userID != nil {
			issuesJSON, _ := json.Marshal(result.Issues)
			gasJSON, _ := json.Marshal(result.GasEstimates)
			scoreJSON, _ := json.Marshal(result.Score)
			_, _ = db.SaveAnalysisResult(r.Context(), userID.(string), result.SourceHash, result.Status,
				string(issuesJSON), string(gasJSON), string(scoreJSON), result.Duration.Milliseconds())
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// =============================================================================
// Transaction tracing
// =============================================================================

func handleTraceTransaction(cfg *config.Config, db *database.DB) http.HandlerFunc {
	txTracer := tracer.NewTracer()

	return func(w http.ResponseWriter, r *http.Request) {
		txHash := chi.URLParam(r, "txHash")
		chain := r.URL.Query().Get("chain")
		if chain == "" {
			chain = "ethereum"
		}

		// Validate tx hash format
		if !isValidTxHash(txHash) {
			respondError(w, http.StatusBadRequest, "Invalid transaction hash format")
			return
		}

		// Get RPC URL for chain
		rpcURL := getRPCURL(cfg, chain)
		if rpcURL == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported chain: %s", chain))
			return
		}

		// Trace transaction
		result, err := txTracer.TraceTransaction(r.Context(), rpcURL, txHash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to trace transaction: %v", err))
			return
		}

		result.Chain = chain
		respondJSON(w, http.StatusOK, result)
	}
}

// =============================================================================
// Simulation
// =============================================================================

func handleSimulateTransaction(cfg *config.Config, db *database.DB) http.HandlerFunc {
	txTracer := tracer.NewTracer()

	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Chain    string `json:"chain"`
			From     string `json:"from"`
			To       string `json:"to"`
			Data     string `json:"data"`
			Value    string `json:"value"`
			GasLimit uint64 `json:"gas_limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Chain == "" {
			req.Chain = "ethereum"
		}

		rpcURL := getRPCURL(cfg, req.Chain)
		if rpcURL == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported chain: %s", req.Chain))
			return
		}

		result, err := txTracer.SimulateTransaction(r.Context(), rpcURL, req.From, req.To, req.Data, req.Value, req.GasLimit)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Simulation failed: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// =============================================================================
// Fork management
// =============================================================================

func handleCreateFork(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Chain       string `json:"chain"`
			BlockNumber uint64 `json:"block_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Chain == "" {
			req.Chain = "ethereum"
		}

		rpcURL := getRPCURL(cfg, req.Chain)
		if rpcURL == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported chain: %s", req.Chain))
			return
		}

		// For now, return a mock response
		// Full implementation would start an Anvil instance
		forkID := fmt.Sprintf("fork_%d", time.Now().UnixNano())

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"fork_id":      forkID,
			"rpc_url":      fmt.Sprintf("https://fork.getchainlens.com/%s", forkID),
			"chain":        req.Chain,
			"block_number": req.BlockNumber,
			"expires_at":   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		})
	}
}

func handleDeleteFork(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forkID := chi.URLParam(r, "forkID")

		// For now, just acknowledge deletion
		respondJSON(w, http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Fork %s deleted", forkID),
		})
	}
}

// =============================================================================
// Monitor handlers
// =============================================================================

func handleGetMonitors(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		monitors, err := db.GetMonitorsByUser(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get monitors")
			return
		}

		respondJSON(w, http.StatusOK, monitors)
	}
}

func handleCreateMonitor(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			ContractID   string   `json:"contract_id"`
			Name         string   `json:"name"`
			EventFilters []string `json:"event_filters"`
			WebhookURL   string   `json:"webhook_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" || req.WebhookURL == "" {
			respondError(w, http.StatusBadRequest, "Name and webhook_url are required")
			return
		}

		monitor, err := db.CreateMonitor(r.Context(), userID, req.ContractID, req.Name, req.WebhookURL, req.EventFilters)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create monitor")
			return
		}

		respondJSON(w, http.StatusCreated, monitor)
	}
}

func handleUpdateMonitor(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "monitorID")

		var req struct {
			Name         string   `json:"name"`
			EventFilters []string `json:"event_filters"`
			WebhookURL   string   `json:"webhook_url"`
			IsActive     bool     `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := db.UpdateMonitor(r.Context(), monitorID, req.Name, req.WebhookURL, req.EventFilters, req.IsActive); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update monitor")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Monitor updated",
		})
	}
}

func handleDeleteMonitor(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "monitorID")

		if err := db.DeleteMonitor(r.Context(), monitorID); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to delete monitor")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Monitor deleted",
		})
	}
}

// =============================================================================
// Project handlers
// =============================================================================

func handleGetProjects(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		projects, err := db.GetProjectsByUser(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get projects")
			return
		}

		respondJSON(w, http.StatusOK, projects)
	}
}

func handleCreateProject(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "Name is required")
			return
		}

		project, err := db.CreateProject(r.Context(), userID, req.Name, req.Description)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create project")
			return
		}

		respondJSON(w, http.StatusCreated, project)
	}
}

func handleGetProject(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "projectID")

		project, err := db.GetProjectByID(r.Context(), projectID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				respondError(w, http.StatusNotFound, "Project not found")
				return
			}
			respondError(w, http.StatusInternalServerError, "Failed to get project")
			return
		}

		respondJSON(w, http.StatusOK, project)
	}
}

func handleUpdateProject(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "projectID")

		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := db.UpdateProject(r.Context(), projectID, req.Name, req.Description); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update project")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Project updated",
		})
	}
}

func handleDeleteProject(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "projectID")

		if err := db.DeleteProject(r.Context(), projectID); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to delete project")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Project deleted",
		})
	}
}

// =============================================================================
// Contract handlers
// =============================================================================

func handleGetContracts(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)
		projectID := r.URL.Query().Get("project_id")

		var contracts []*database.Contract
		var err error

		if projectID != "" {
			contracts, err = db.GetContractsByProject(r.Context(), projectID)
		} else {
			contracts, err = db.GetContractsByUser(r.Context(), userID)
		}

		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get contracts")
			return
		}

		respondJSON(w, http.StatusOK, contracts)
	}
}

func handleAddContract(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			ProjectID string `json:"project_id"`
			Address   string `json:"address"`
			Chain     string `json:"chain"`
			Name      string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Address == "" || req.Chain == "" {
			respondError(w, http.StatusBadRequest, "Address and chain are required")
			return
		}

		contract, err := db.CreateContract(r.Context(), req.ProjectID, userID, req.Address, req.Chain, req.Name)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create contract")
			return
		}

		respondJSON(w, http.StatusCreated, contract)
	}
}

func handleDeleteContract(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contractID := chi.URLParam(r, "contractID")

		if err := db.DeleteContract(r.Context(), contractID); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to delete contract")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"message": "Contract deleted",
		})
	}
}

// =============================================================================
// Billing handlers
// =============================================================================

func handleGetSubscription(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get subscription")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"plan":           user.Plan,
			"status":         "active",
			"api_calls_used": user.APICallsUsed,
			"api_calls_reset_at": user.APICallsResetAt,
		})
	}
}

func handleCreateCheckout(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			PriceID    string `json:"price_id"`
			SuccessURL string `json:"success_url"`
			CancelURL  string `json:"cancel_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if cfg.StripeSecretKey == "" {
			respondError(w, http.StatusServiceUnavailable, "Billing not configured")
			return
		}

		// Get user
		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get user")
			return
		}

		// Create Stripe checkout session
		checkoutURL, err := createStripeCheckout(cfg, user, req.PriceID, req.SuccessURL, req.CancelURL)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create checkout: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"url": checkoutURL,
		})
	}
}

func handleCreatePortal(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		var req struct {
			ReturnURL string `json:"return_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if cfg.StripeSecretKey == "" {
			respondError(w, http.StatusServiceUnavailable, "Billing not configured")
			return
		}

		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get user")
			return
		}

		if user.StripeCustomerID == nil {
			respondError(w, http.StatusBadRequest, "No billing account found")
			return
		}

		portalURL, err := createStripePortal(cfg, *user.StripeCustomerID, req.ReturnURL)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create portal: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"url": portalURL,
		})
	}
}

func handleStripeWebhook(cfg *config.Config, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.StripeWebhookSecret == "" {
			respondError(w, http.StatusServiceUnavailable, "Webhook not configured")
			return
		}

		// Process Stripe webhook
		err := processStripeWebhook(r, cfg, db)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Webhook error: %v", err))
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"received": "true",
		})
	}
}

// =============================================================================
// Usage tracking
// =============================================================================

func handleGetUsage(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(string)

		user, err := db.GetUserByID(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get usage")
			return
		}

		// Get limit based on plan
		var limit int
		switch user.Plan {
		case "pro":
			limit = 10000
		case "enterprise":
			limit = 100000
		default:
			limit = 1000
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"api_calls_used":  user.APICallsUsed,
			"api_calls_limit": limit,
			"period_start":    user.APICallsResetAt.AddDate(0, -1, 0).Format("2006-01-02"),
			"period_end":      user.APICallsResetAt.Format("2006-01-02"),
		})
	}
}

// =============================================================================
// Helper functions
// =============================================================================

func generateAccessToken(cfg *config.Config, userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(cfg.JWTExpiration) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isValidTxHash(hash string) bool {
	if len(hash) != 66 {
		return false
	}
	if !strings.HasPrefix(hash, "0x") {
		return false
	}
	for _, c := range hash[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func getRPCURL(cfg *config.Config, chain string) string {
	switch strings.ToLower(chain) {
	case "ethereum", "eth", "mainnet":
		return cfg.EthereumRPCURL
	case "polygon", "matic":
		return cfg.PolygonRPCURL
	case "arbitrum", "arb":
		return cfg.ArbitrumRPCURL
	default:
		return ""
	}
}

// Stripe integration using billing package
func createStripeCheckout(cfg *config.Config, user *database.User, priceID, successURL, cancelURL string) (string, error) {
	client := billing.NewStripeClient(cfg)

	customerID := ""
	if user.StripeCustomerID != nil {
		customerID = *user.StripeCustomerID
	}

	session, err := client.CreateCheckoutSession(
		context.Background(),
		customerID,
		priceID,
		successURL+"?session_id={CHECKOUT_SESSION_ID}",
		cancelURL,
	)
	if err != nil {
		return "", err
	}

	return session.URL, nil
}

func createStripePortal(cfg *config.Config, customerID, returnURL string) (string, error) {
	client := billing.NewStripeClient(cfg)
	return client.CreatePortalSession(context.Background(), customerID, returnURL)
}

func processStripeWebhook(r *http.Request, cfg *config.Config, db *database.DB) error {
	client := billing.NewStripeClient(cfg)

	// Read request body
	body, err := billing.ReadRequestBody(r)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Get signature from header
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	// Verify and parse event
	event, err := client.VerifyWebhook(body, signature)
	if err != nil {
		return fmt.Errorf("webhook verification failed: %w", err)
	}

	// Process event
	return client.ProcessWebhookEvent(r.Context(), db, event)
}
