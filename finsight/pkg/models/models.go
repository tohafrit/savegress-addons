package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType represents the type of financial transaction
type TransactionType string

const (
	TransactionTypeDebit    TransactionType = "debit"
	TransactionTypeCredit   TransactionType = "credit"
	TransactionTypeTransfer TransactionType = "transfer"
	TransactionTypeRefund   TransactionType = "refund"
	TransactionTypeFee      TransactionType = "fee"
	TransactionTypeInterest TransactionType = "interest"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusReversed  TransactionStatus = "reversed"
	TransactionStatusHeld      TransactionStatus = "held"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID              string            `json:"id"`
	ExternalID      string            `json:"external_id"`
	Type            TransactionType   `json:"type"`
	Status          TransactionStatus `json:"status"`
	Amount          decimal.Decimal   `json:"amount"`
	Currency        string            `json:"currency"`
	SourceAccount   string            `json:"source_account"`
	DestAccount     string            `json:"dest_account,omitempty"`
	Description     string            `json:"description"`
	Category        string            `json:"category"`
	Merchant        *Merchant         `json:"merchant,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	RiskScore       float64           `json:"risk_score"`
	FraudFlags      []string          `json:"fraud_flags,omitempty"`
	ReconcileStatus ReconcileStatus   `json:"reconcile_status"`
	CreatedAt       time.Time         `json:"created_at"`
	ProcessedAt     *time.Time        `json:"processed_at,omitempty"`
	SettledAt       *time.Time        `json:"settled_at,omitempty"`
}

// Merchant represents a merchant in a transaction
type Merchant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	MCC      string `json:"mcc"` // Merchant Category Code
	Country  string `json:"country"`
	City     string `json:"city"`
}

// Account represents a financial account
type Account struct {
	ID            string          `json:"id"`
	AccountNumber string          `json:"account_number"`
	AccountType   AccountType     `json:"account_type"`
	Currency      string          `json:"currency"`
	Balance       decimal.Decimal `json:"balance"`
	AvailableBal  decimal.Decimal `json:"available_balance"`
	HoldAmount    decimal.Decimal `json:"hold_amount"`
	Status        AccountStatus   `json:"status"`
	OwnerID       string          `json:"owner_id"`
	OwnerType     string          `json:"owner_type"` // individual, business
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// AccountType represents the type of account
type AccountType string

const (
	AccountTypeChecking AccountType = "checking"
	AccountTypeSavings  AccountType = "savings"
	AccountTypeCredit   AccountType = "credit"
	AccountTypeLoan     AccountType = "loan"
	AccountTypeInvestment AccountType = "investment"
)

// AccountStatus represents the status of an account
type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusFrozen   AccountStatus = "frozen"
	AccountStatusClosed   AccountStatus = "closed"
	AccountStatusPending  AccountStatus = "pending"
)

// ReconcileStatus represents the reconciliation status
type ReconcileStatus string

const (
	ReconcileStatusPending    ReconcileStatus = "pending"
	ReconcileStatusMatched    ReconcileStatus = "matched"
	ReconcileStatusUnmatched  ReconcileStatus = "unmatched"
	ReconcileStatusException  ReconcileStatus = "exception"
	ReconcileStatusManual     ReconcileStatus = "manual"
)

// FraudAlert represents a fraud detection alert
type FraudAlert struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	AlertType     FraudAlertType  `json:"alert_type"`
	Severity      AlertSeverity   `json:"severity"`
	RiskScore     float64         `json:"risk_score"`
	Indicators    []FraudIndicator `json:"indicators"`
	Status        AlertStatus     `json:"status"`
	AssignedTo    string          `json:"assigned_to,omitempty"`
	Resolution    string          `json:"resolution,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
}

// FraudAlertType represents the type of fraud alert
type FraudAlertType string

const (
	FraudAlertTypeVelocity     FraudAlertType = "velocity"
	FraudAlertTypeAmount       FraudAlertType = "amount_anomaly"
	FraudAlertTypeGeolocation  FraudAlertType = "geolocation"
	FraudAlertTypePattern      FraudAlertType = "pattern"
	FraudAlertTypeDevice       FraudAlertType = "device"
	FraudAlertTypeIdentity     FraudAlertType = "identity"
	FraudAlertTypeMerchant     FraudAlertType = "merchant"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusOpen       AlertStatus = "open"
	AlertStatusInProgress AlertStatus = "in_progress"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusFalsePos   AlertStatus = "false_positive"
	AlertStatusEscalated  AlertStatus = "escalated"
)

// FraudIndicator represents a specific fraud indicator
type FraudIndicator struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ReconciliationBatch represents a reconciliation batch
type ReconciliationBatch struct {
	ID              string            `json:"id"`
	Source          string            `json:"source"`
	Target          string            `json:"target"`
	Status          BatchStatus       `json:"status"`
	TotalRecords    int               `json:"total_records"`
	MatchedRecords  int               `json:"matched_records"`
	UnmatchedRecords int              `json:"unmatched_records"`
	Exceptions      int               `json:"exceptions"`
	StartedAt       time.Time         `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	Summary         *ReconcileSummary `json:"summary,omitempty"`
}

// BatchStatus represents the status of a reconciliation batch
type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "pending"
	BatchStatusRunning    BatchStatus = "running"
	BatchStatusCompleted  BatchStatus = "completed"
	BatchStatusFailed     BatchStatus = "failed"
)

// ReconcileSummary contains reconciliation summary data
type ReconcileSummary struct {
	SourceTotal     decimal.Decimal `json:"source_total"`
	TargetTotal     decimal.Decimal `json:"target_total"`
	Difference      decimal.Decimal `json:"difference"`
	MatchRate       float64         `json:"match_rate"`
	ExceptionAmount decimal.Decimal `json:"exception_amount"`
}

// ReconcileException represents a reconciliation exception
type ReconcileException struct {
	ID              string          `json:"id"`
	BatchID         string          `json:"batch_id"`
	Type            ExceptionType   `json:"type"`
	SourceRecord    *Transaction    `json:"source_record,omitempty"`
	TargetRecord    *Transaction    `json:"target_record,omitempty"`
	AmountDiff      decimal.Decimal `json:"amount_diff"`
	Description     string          `json:"description"`
	Status          ExceptionStatus `json:"status"`
	Resolution      string          `json:"resolution,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
}

// ExceptionType represents the type of reconciliation exception
type ExceptionType string

const (
	ExceptionTypeMissing    ExceptionType = "missing"
	ExceptionTypeDuplicate  ExceptionType = "duplicate"
	ExceptionTypeAmountDiff ExceptionType = "amount_diff"
	ExceptionTypeDateDiff   ExceptionType = "date_diff"
	ExceptionTypeOther      ExceptionType = "other"
)

// ExceptionStatus represents the status of an exception
type ExceptionStatus string

const (
	ExceptionStatusOpen     ExceptionStatus = "open"
	ExceptionStatusResolved ExceptionStatus = "resolved"
	ExceptionStatusWriteOff ExceptionStatus = "write_off"
)

// FinancialReport represents a financial report
type FinancialReport struct {
	ID          string          `json:"id"`
	Type        ReportType      `json:"type"`
	Period      ReportPeriod    `json:"period"`
	StartDate   time.Time       `json:"start_date"`
	EndDate     time.Time       `json:"end_date"`
	Status      ReportStatus    `json:"status"`
	Data        *ReportData     `json:"data,omitempty"`
	GeneratedAt *time.Time      `json:"generated_at,omitempty"`
	ExportURL   string          `json:"export_url,omitempty"`
}

// ReportType represents the type of report
type ReportType string

const (
	ReportTypeTransaction   ReportType = "transaction"
	ReportTypeCashFlow      ReportType = "cash_flow"
	ReportTypeBalanceSheet  ReportType = "balance_sheet"
	ReportTypeProfitLoss    ReportType = "profit_loss"
	ReportTypeReconciliation ReportType = "reconciliation"
	ReportTypeFraud         ReportType = "fraud"
	ReportTypeCustom        ReportType = "custom"
)

// ReportPeriod represents the period of a report
type ReportPeriod string

const (
	ReportPeriodDaily   ReportPeriod = "daily"
	ReportPeriodWeekly  ReportPeriod = "weekly"
	ReportPeriodMonthly ReportPeriod = "monthly"
	ReportPeriodQuarterly ReportPeriod = "quarterly"
	ReportPeriodYearly  ReportPeriod = "yearly"
	ReportPeriodCustom  ReportPeriod = "custom"
)

// ReportStatus represents the status of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusGenerating ReportStatus = "generating"
	ReportStatusCompleted ReportStatus = "completed"
	ReportStatusFailed    ReportStatus = "failed"
)

// ReportData contains the actual report data
type ReportData struct {
	TotalTransactions   int                     `json:"total_transactions"`
	TotalVolume         decimal.Decimal         `json:"total_volume"`
	NetFlow             decimal.Decimal         `json:"net_flow"`
	ByType              map[string]TypeSummary  `json:"by_type"`
	ByCategory          map[string]decimal.Decimal `json:"by_category"`
	ByStatus            map[string]int          `json:"by_status"`
	DailyBreakdown      []DailySummary          `json:"daily_breakdown,omitempty"`
	TopMerchants        []MerchantSummary       `json:"top_merchants,omitempty"`
	FraudMetrics        *FraudMetrics           `json:"fraud_metrics,omitempty"`
}

// TypeSummary contains summary by transaction type
type TypeSummary struct {
	Count   int             `json:"count"`
	Volume  decimal.Decimal `json:"volume"`
	Average decimal.Decimal `json:"average"`
}

// DailySummary contains daily summary data
type DailySummary struct {
	Date         time.Time       `json:"date"`
	Transactions int             `json:"transactions"`
	Volume       decimal.Decimal `json:"volume"`
	Credits      decimal.Decimal `json:"credits"`
	Debits       decimal.Decimal `json:"debits"`
}

// MerchantSummary contains merchant summary data
type MerchantSummary struct {
	MerchantID   string          `json:"merchant_id"`
	MerchantName string          `json:"merchant_name"`
	Transactions int             `json:"transactions"`
	Volume       decimal.Decimal `json:"volume"`
}

// FraudMetrics contains fraud-related metrics
type FraudMetrics struct {
	TotalAlerts      int             `json:"total_alerts"`
	OpenAlerts       int             `json:"open_alerts"`
	ResolvedAlerts   int             `json:"resolved_alerts"`
	FalsePositives   int             `json:"false_positives"`
	BlockedAmount    decimal.Decimal `json:"blocked_amount"`
	DetectionRate    float64         `json:"detection_rate"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          string                 `json:"id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	ActorID     string                 `json:"actor_id"`
	ActorType   string                 `json:"actor_type"`
	Changes     map[string]interface{} `json:"changes,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ComplianceRule represents a compliance rule
type ComplianceRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RuleType    string   `json:"rule_type"`
	Conditions  []RuleCondition `json:"conditions"`
	Actions     []RuleAction    `json:"actions"`
	Enabled     bool     `json:"enabled"`
	Priority    int      `json:"priority"`
}

// RuleCondition represents a condition in a compliance rule
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// RuleAction represents an action in a compliance rule
type RuleAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}
