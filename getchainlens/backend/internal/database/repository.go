package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateEmail = errors.New("email already exists")
	ErrInvalidToken   = errors.New("invalid or expired token")
)

// =============================================================================
// User Repository
// =============================================================================

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
	id := generateID("usr")
	now := time.Now().UTC()
	resetAt := now.AddDate(0, 1, 0) // Reset API calls monthly

	query := `
		INSERT INTO users (id, email, password_hash, name, plan, api_calls_used, api_calls_reset_at, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'free', 0, $5, false, $6, $6)
		RETURNING id, email, name, plan, api_calls_used, api_calls_reset_at, email_verified, created_at, updated_at
	`

	user := &User{}
	err := db.pool.QueryRow(ctx, query, id, email, passwordHash, name, resetAt, now).Scan(
		&user.ID, &user.Email, &user.Name, &user.Plan,
		&user.APICallsUsed, &user.APICallsResetAt, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, name, plan, stripe_customer_id, stripe_subscription_id,
		       api_calls_used, api_calls_reset_at, email_verified, created_at, updated_at
		FROM users WHERE id = $1
	`

	user := &User{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Plan,
		&user.StripeCustomerID, &user.StripeSubscID,
		&user.APICallsUsed, &user.APICallsResetAt, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, name, plan, stripe_customer_id, stripe_subscription_id,
		       api_calls_used, api_calls_reset_at, email_verified, created_at, updated_at
		FROM users WHERE email = $1
	`

	user := &User{}
	err := db.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Plan,
		&user.StripeCustomerID, &user.StripeSubscID,
		&user.APICallsUsed, &user.APICallsResetAt, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (db *DB) UpdateUser(ctx context.Context, id, name string) error {
	query := `UPDATE users SET name = $2, updated_at = $3 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id, name, time.Now().UTC())
	return err
}

// UpdateUserPassword updates user password
func (db *DB) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id, passwordHash, time.Now().UTC())
	return err
}

// UpdateUserPlan updates user subscription plan
func (db *DB) UpdateUserPlan(ctx context.Context, id, plan string) error {
	query := `UPDATE users SET plan = $2, updated_at = $3 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id, plan, time.Now().UTC())
	return err
}

// UpdateUserStripe updates Stripe customer info
func (db *DB) UpdateUserStripe(ctx context.Context, id, customerID, subscriptionID string) error {
	query := `UPDATE users SET stripe_customer_id = $2, stripe_subscription_id = $3, updated_at = $4 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id, customerID, subscriptionID, time.Now().UTC())
	return err
}

// GetUserByStripeCustomerID retrieves a user by Stripe customer ID
func (db *DB) GetUserByStripeCustomerID(ctx context.Context, customerID string) (*User, error) {
	query := `
		SELECT id, email, password_hash, name, plan, stripe_customer_id, stripe_subscription_id,
		       api_calls_used, api_calls_reset_at, email_verified, created_at, updated_at
		FROM users WHERE stripe_customer_id = $1
	`

	user := &User{}
	err := db.pool.QueryRow(ctx, query, customerID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Plan,
		&user.StripeCustomerID, &user.StripeSubscID,
		&user.APICallsUsed, &user.APICallsResetAt, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by Stripe customer ID: %w", err)
	}

	return user, nil
}

// IncrementAPIUsage increments API call counter
func (db *DB) IncrementAPIUsage(ctx context.Context, userID string) (int, error) {
	// First check if we need to reset the counter
	var resetAt time.Time
	var currentUsage int

	err := db.pool.QueryRow(ctx,
		`SELECT api_calls_used, api_calls_reset_at FROM users WHERE id = $1`,
		userID,
	).Scan(&currentUsage, &resetAt)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	if now.After(resetAt) {
		// Reset counter
		newResetAt := now.AddDate(0, 1, 0)
		_, err = db.pool.Exec(ctx,
			`UPDATE users SET api_calls_used = 1, api_calls_reset_at = $2, updated_at = $3 WHERE id = $1`,
			userID, newResetAt, now,
		)
		return 1, err
	}

	// Increment counter
	err = db.pool.QueryRow(ctx,
		`UPDATE users SET api_calls_used = api_calls_used + 1, updated_at = $2 WHERE id = $1 RETURNING api_calls_used`,
		userID, now,
	).Scan(&currentUsage)

	return currentUsage, err
}

// =============================================================================
// Refresh Token Repository
// =============================================================================

// CreateRefreshToken creates a new refresh token
func (db *DB) CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) (*RefreshToken, error) {
	id := generateID("rt")
	now := time.Now().UTC()

	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, expires_at, created_at
	`

	rt := &RefreshToken{Token: token}
	err := db.pool.QueryRow(ctx, query, id, userID, token, expiresAt, now).Scan(
		&rt.ID, &rt.UserID, &rt.ExpiresAt, &rt.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return rt, nil
}

// GetRefreshToken retrieves a refresh token
func (db *DB) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM refresh_tokens WHERE token = $1 AND expires_at > $2
	`

	rt := &RefreshToken{}
	err := db.pool.QueryRow(ctx, query, token, time.Now().UTC()).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return rt, nil
}

// DeleteRefreshToken deletes a refresh token
func (db *DB) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	return err
}

// DeleteUserRefreshTokens deletes all refresh tokens for a user
func (db *DB) DeleteUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

// =============================================================================
// License Repository
// =============================================================================

// CreateLicense creates a new license
func (db *DB) CreateLicense(ctx context.Context, userID, tier string, features []string, maxDevices int) (*License, error) {
	id := generateID("lic")
	licenseKey := generateLicenseKey()
	now := time.Now().UTC()

	query := `
		INSERT INTO licenses (id, user_id, license_key, tier, features, max_devices, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, license_key, tier, features, max_devices, created_at
	`

	lic := &License{}
	err := db.pool.QueryRow(ctx, query, id, userID, licenseKey, tier, features, maxDevices, now).Scan(
		&lic.ID, &lic.UserID, &lic.LicenseKey, &lic.Tier, &lic.Features, &lic.MaxDevices, &lic.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create license: %w", err)
	}

	return lic, nil
}

// GetLicenseByKey retrieves a license by key
func (db *DB) GetLicenseByKey(ctx context.Context, licenseKey string) (*License, error) {
	query := `
		SELECT id, user_id, license_key, tier, features, max_devices, activated_at, expires_at, created_at
		FROM licenses WHERE license_key = $1
	`

	lic := &License{}
	err := db.pool.QueryRow(ctx, query, licenseKey).Scan(
		&lic.ID, &lic.UserID, &lic.LicenseKey, &lic.Tier, &lic.Features,
		&lic.MaxDevices, &lic.ActivatedAt, &lic.ExpiresAt, &lic.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	return lic, nil
}

// =============================================================================
// Project Repository
// =============================================================================

// CreateProject creates a new project
func (db *DB) CreateProject(ctx context.Context, userID, name, description string) (*Project, error) {
	id := generateID("prj")
	now := time.Now().UTC()

	query := `
		INSERT INTO projects (id, user_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, user_id, name, description, created_at, updated_at
	`

	project := &Project{}
	err := db.pool.QueryRow(ctx, query, id, userID, name, description, now).Scan(
		&project.ID, &project.UserID, &project.Name, &project.Description,
		&project.CreatedAt, &project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProjectsByUser retrieves all projects for a user
func (db *DB) GetProjectsByUser(ctx context.Context, userID string) ([]*Project, error) {
	query := `
		SELECT id, user_id, name, description, created_at, updated_at
		FROM projects WHERE user_id = $1 ORDER BY created_at DESC
	`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// GetProjectByID retrieves a project by ID
func (db *DB) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	query := `
		SELECT id, user_id, name, description, created_at, updated_at
		FROM projects WHERE id = $1
	`

	project := &Project{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&project.ID, &project.UserID, &project.Name, &project.Description,
		&project.CreatedAt, &project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// UpdateProject updates a project
func (db *DB) UpdateProject(ctx context.Context, id, name, description string) error {
	query := `UPDATE projects SET name = $2, description = $3, updated_at = $4 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id, name, description, time.Now().UTC())
	return err
}

// DeleteProject deletes a project
func (db *DB) DeleteProject(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

// =============================================================================
// Contract Repository
// =============================================================================

// CreateContract creates a new contract
func (db *DB) CreateContract(ctx context.Context, projectID, userID, address, chain, name string) (*Contract, error) {
	id := generateID("ctr")
	now := time.Now().UTC()

	query := `
		INSERT INTO contracts (id, project_id, user_id, address, chain, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, $7)
		RETURNING id, project_id, user_id, address, chain, name, is_verified, created_at, updated_at
	`

	contract := &Contract{}
	err := db.pool.QueryRow(ctx, query, id, projectID, userID, address, chain, name, now).Scan(
		&contract.ID, &contract.ProjectID, &contract.UserID, &contract.Address,
		&contract.Chain, &contract.Name, &contract.IsVerified,
		&contract.CreatedAt, &contract.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}

	return contract, nil
}

// GetContractsByProject retrieves all contracts for a project
func (db *DB) GetContractsByProject(ctx context.Context, projectID string) ([]*Contract, error) {
	query := `
		SELECT id, project_id, user_id, address, chain, name, abi, source_code, is_verified, created_at, updated_at
		FROM contracts WHERE project_id = $1 ORDER BY created_at DESC
	`

	rows, err := db.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*Contract
	for rows.Next() {
		c := &Contract{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Address, &c.Chain, &c.Name,
			&c.ABI, &c.SourceCode, &c.IsVerified, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}

	return contracts, nil
}

// GetContractsByUser retrieves all contracts for a user
func (db *DB) GetContractsByUser(ctx context.Context, userID string) ([]*Contract, error) {
	query := `
		SELECT id, project_id, user_id, address, chain, name, abi, source_code, is_verified, created_at, updated_at
		FROM contracts WHERE user_id = $1 ORDER BY created_at DESC
	`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*Contract
	for rows.Next() {
		c := &Contract{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Address, &c.Chain, &c.Name,
			&c.ABI, &c.SourceCode, &c.IsVerified, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}

	return contracts, nil
}

// DeleteContract deletes a contract
func (db *DB) DeleteContract(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM contracts WHERE id = $1`, id)
	return err
}

// =============================================================================
// Monitor Repository
// =============================================================================

// CreateMonitor creates a new monitor
func (db *DB) CreateMonitor(ctx context.Context, userID, contractID, name, webhookURL string, eventFilters []string) (*Monitor, error) {
	id := generateID("mon")
	now := time.Now().UTC()

	query := `
		INSERT INTO monitors (id, user_id, contract_id, name, event_filters, webhook_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)
		RETURNING id, user_id, contract_id, name, event_filters, webhook_url, is_active, created_at, updated_at
	`

	monitor := &Monitor{}
	err := db.pool.QueryRow(ctx, query, id, userID, contractID, name, eventFilters, webhookURL, now).Scan(
		&monitor.ID, &monitor.UserID, &monitor.ContractID, &monitor.Name,
		&monitor.EventFilters, &monitor.WebhookURL, &monitor.IsActive,
		&monitor.CreatedAt, &monitor.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	return monitor, nil
}

// GetMonitorsByUser retrieves all monitors for a user
func (db *DB) GetMonitorsByUser(ctx context.Context, userID string) ([]*Monitor, error) {
	query := `
		SELECT id, user_id, contract_id, name, event_filters, webhook_url, is_active, created_at, updated_at
		FROM monitors WHERE user_id = $1 ORDER BY created_at DESC
	`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*Monitor
	for rows.Next() {
		m := &Monitor{}
		if err := rows.Scan(&m.ID, &m.UserID, &m.ContractID, &m.Name,
			&m.EventFilters, &m.WebhookURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}

	return monitors, nil
}

// GetMonitorByID retrieves a monitor by ID
func (db *DB) GetMonitorByID(ctx context.Context, id string) (*Monitor, error) {
	query := `
		SELECT id, user_id, contract_id, name, event_filters, webhook_url, is_active, created_at, updated_at
		FROM monitors WHERE id = $1
	`

	m := &Monitor{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.UserID, &m.ContractID, &m.Name,
		&m.EventFilters, &m.WebhookURL, &m.IsActive,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get monitor: %w", err)
	}

	return m, nil
}

// UpdateMonitor updates a monitor
func (db *DB) UpdateMonitor(ctx context.Context, id, name, webhookURL string, eventFilters []string, isActive bool) error {
	query := `
		UPDATE monitors
		SET name = $2, webhook_url = $3, event_filters = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := db.pool.Exec(ctx, query, id, name, webhookURL, eventFilters, isActive, time.Now().UTC())
	return err
}

// DeleteMonitor deletes a monitor
func (db *DB) DeleteMonitor(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, id)
	return err
}

// =============================================================================
// Alert Repository
// =============================================================================

// CreateAlert creates a new alert
func (db *DB) CreateAlert(ctx context.Context, monitorID, userID, alertType, severity, message, data string) (*Alert, error) {
	id := generateID("alt")
	now := time.Now().UTC()

	query := `
		INSERT INTO alerts (id, monitor_id, user_id, type, severity, message, data, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8)
		RETURNING id, monitor_id, user_id, type, severity, message, data, is_read, created_at
	`

	alert := &Alert{}
	err := db.pool.QueryRow(ctx, query, id, monitorID, userID, alertType, severity, message, data, now).Scan(
		&alert.ID, &alert.MonitorID, &alert.UserID, &alert.Type,
		&alert.Severity, &alert.Message, &alert.Data, &alert.IsRead, &alert.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	return alert, nil
}

// GetAlertsByUser retrieves alerts for a user
func (db *DB) GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*Alert, error) {
	query := `
		SELECT id, monitor_id, user_id, type, severity, message, data, is_read, created_at
		FROM alerts WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
	`

	rows, err := db.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		a := &Alert{}
		if err := rows.Scan(&a.ID, &a.MonitorID, &a.UserID, &a.Type,
			&a.Severity, &a.Message, &a.Data, &a.IsRead, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

// MarkAlertRead marks an alert as read
func (db *DB) MarkAlertRead(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `UPDATE alerts SET is_read = true WHERE id = $1`, id)
	return err
}

// =============================================================================
// API Usage Repository
// =============================================================================

// RecordAPIUsage records an API call
func (db *DB) RecordAPIUsage(ctx context.Context, userID, endpoint, method string, statusCode int, durationMs int64) error {
	id := generateID("usg")
	query := `
		INSERT INTO api_usage (id, user_id, endpoint, method, status_code, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.pool.Exec(ctx, query, id, userID, endpoint, method, statusCode, durationMs, time.Now().UTC())
	return err
}

// TrackAPIUsage is an alias for RecordAPIUsage
func (db *DB) TrackAPIUsage(ctx context.Context, userID, endpoint, method string, statusCode int, durationMs int64) error {
	return db.RecordAPIUsage(ctx, userID, endpoint, method, statusCode, durationMs)
}

// ResetAPIUsage resets API usage counter for a user
func (db *DB) ResetAPIUsage(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	newResetAt := now.AddDate(0, 1, 0)
	query := `UPDATE users SET api_calls_used = 0, api_calls_reset_at = $2, updated_at = $3 WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, userID, newResetAt, now)
	return err
}

// GetAPIUsageStats retrieves API usage stats for a user
func (db *DB) GetAPIUsageStats(ctx context.Context, userID string, since time.Time) (int64, error) {
	var count int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_usage WHERE user_id = $1 AND created_at >= $2`,
		userID, since,
	).Scan(&count)
	return count, err
}

// =============================================================================
// Analysis Result Repository
// =============================================================================

// SaveAnalysisResult saves an analysis result
func (db *DB) SaveAnalysisResult(ctx context.Context, userID, sourceHash, status, issuesJSON, gasJSON, scoreJSON string, durationMs int64) (*AnalysisResult, error) {
	id := generateID("anl")
	now := time.Now().UTC()

	query := `
		INSERT INTO analysis_results (id, user_id, source_hash, status, issues, gas_estimates, score, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, source_hash, status, issues, gas_estimates, score, duration_ms, created_at
	`

	result := &AnalysisResult{}
	err := db.pool.QueryRow(ctx, query, id, userID, sourceHash, status, issuesJSON, gasJSON, scoreJSON, durationMs, now).Scan(
		&result.ID, &result.UserID, &result.SourceHash, &result.Status,
		&result.IssuesJSON, &result.GasJSON, &result.ScoreJSON,
		&result.DurationMs, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save analysis result: %w", err)
	}

	return result, nil
}

// =============================================================================
// Helper functions
// =============================================================================

func generateID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func generateLicenseKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	key := hex.EncodeToString(b)
	// Format: XXXX-XXXX-XXXX-XXXX
	return key[0:4] + "-" + key[4:8] + "-" + key[8:12] + "-" + key[12:16]
}

func isDuplicateError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate") || contains(err.Error(), "unique"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
