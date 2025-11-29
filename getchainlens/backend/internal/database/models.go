package database

import (
	"time"
)

// User represents a user account
type User struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	PasswordHash      string     `json:"-"`
	Name              string     `json:"name"`
	Plan              string     `json:"plan"` // free, pro, enterprise
	StripeCustomerID  *string    `json:"-"`
	StripeSubscID     *string    `json:"-"`
	APICallsUsed      int        `json:"api_calls_used"`
	APICallsResetAt   time.Time  `json:"api_calls_reset_at"`
	EmailVerified     bool       `json:"email_verified"`
	EmailVerifyToken  *string    `json:"-"`
	PasswordResetToken *string   `json:"-"`
	PasswordResetExp  *time.Time `json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// License represents a VS Code extension license
type License struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	LicenseKey  string     `json:"license_key"`
	Tier        string     `json:"tier"` // free, pro, enterprise
	Features    []string   `json:"features"`
	MaxDevices  int        `json:"max_devices"`
	ActivatedAt *time.Time `json:"activated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Project represents a user's project
type Project struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Contract represents a monitored smart contract
type Contract struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	UserID      string    `json:"user_id"`
	Address     string    `json:"address"`
	Chain       string    `json:"chain"`
	Name        string    `json:"name"`
	ABI         string    `json:"abi,omitempty"`
	SourceCode  string    `json:"source_code,omitempty"`
	IsVerified  bool      `json:"is_verified"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Monitor represents a contract monitoring configuration
type Monitor struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ContractID   string    `json:"contract_id"`
	Name         string    `json:"name"`
	EventFilters []string  `json:"event_filters"`
	WebhookURL   string    `json:"webhook_url"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Alert represents an alert triggered by a monitor
type Alert struct {
	ID         string    `json:"id"`
	MonitorID  string    `json:"monitor_id"`
	UserID     string    `json:"user_id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	Data       string    `json:"data"` // JSON
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

// AnalysisResult stores contract analysis results
type AnalysisResult struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SourceHash   string    `json:"source_hash"`
	Status       string    `json:"status"`
	IssuesJSON   string    `json:"issues"` // JSON array
	GasJSON      string    `json:"gas_estimates"` // JSON object
	ScoreJSON    string    `json:"score"` // JSON object
	DurationMs   int64     `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// APIUsage tracks API calls for billing
type APIUsage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	Method    string    `json:"method"`
	StatusCode int      `json:"status_code"`
	DurationMs int64    `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
}
