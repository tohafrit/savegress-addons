package aml

import (
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// SARStatus represents the status of a Suspicious Activity Report
type SARStatus string

const (
	SARStatusDraft     SARStatus = "draft"
	SARStatusPending   SARStatus = "pending_review"
	SARStatusApproved  SARStatus = "approved"
	SARStatusFiled     SARStatus = "filed"
	SARStatusRejected  SARStatus = "rejected"
)

// CTRStatus represents the status of a Currency Transaction Report
type CTRStatus string

const (
	CTRStatusPending  CTRStatus = "pending"
	CTRStatusFiled    CTRStatus = "filed"
	CTRStatusExempt   CTRStatus = "exempt"
)

// RiskLevel represents a customer risk level
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// WatchlistType represents the type of watchlist
type WatchlistType string

const (
	WatchlistTypeOFAC     WatchlistType = "ofac"      // Office of Foreign Assets Control
	WatchlistTypeSDN      WatchlistType = "sdn"       // Specially Designated Nationals
	WatchlistTypePEP      WatchlistType = "pep"       // Politically Exposed Persons
	WatchlistTypeEU       WatchlistType = "eu"        // European Union sanctions
	WatchlistTypeUN       WatchlistType = "un"        // United Nations sanctions
	WatchlistTypeInternal WatchlistType = "internal"  // Internal watchlist
)

// SuspiciousActivityReport represents a SAR filing
type SuspiciousActivityReport struct {
	ID                 string                 `json:"id"`
	Status             SARStatus              `json:"status"`
	FilingType         string                 `json:"filing_type"` // initial, continuing, joint
	SubjectType        string                 `json:"subject_type"` // individual, entity
	Subject            *SARSubject            `json:"subject"`
	SuspiciousActivity *SuspiciousActivity    `json:"suspicious_activity"`
	Narrative          string                 `json:"narrative"`
	Transactions       []string               `json:"transaction_ids"`
	TotalAmount        decimal.Decimal        `json:"total_amount"`
	DateRange          *DateRange             `json:"date_range"`
	FilingInstitution  *FilingInstitution     `json:"filing_institution"`
	PreparedBy         string                 `json:"prepared_by"`
	ReviewedBy         string                 `json:"reviewed_by,omitempty"`
	ApprovedBy         string                 `json:"approved_by,omitempty"`
	BSAIdentifier      string                 `json:"bsa_identifier,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	FiledAt            *time.Time             `json:"filed_at,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// SARSubject represents the subject of a SAR
type SARSubject struct {
	Type               string   `json:"type"` // individual, entity
	Name               string   `json:"name"`
	AlternateNames     []string `json:"alternate_names,omitempty"`
	TIN                string   `json:"tin,omitempty"` // Tax Identification Number
	SSN                string   `json:"ssn,omitempty"` // Social Security Number (masked)
	DateOfBirth        string   `json:"date_of_birth,omitempty"`
	Address            *Address `json:"address,omitempty"`
	AccountNumbers     []string `json:"account_numbers"`
	IDType             string   `json:"id_type,omitempty"`
	IDNumber           string   `json:"id_number,omitempty"`
	IDCountry          string   `json:"id_country,omitempty"`
	RelationshipStatus string   `json:"relationship_status"` // current, former
	Occupation         string   `json:"occupation,omitempty"`
	PhoneNumbers       []string `json:"phone_numbers,omitempty"`
	EmailAddresses     []string `json:"email_addresses,omitempty"`
}

// Address represents a physical address
type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// SuspiciousActivity describes the suspicious activity
type SuspiciousActivity struct {
	Categories          []string `json:"categories"` // structuring, money_laundering, etc.
	InstrumentTypes     []string `json:"instrument_types"` // cash, wire, check, etc.
	ProductTypes        []string `json:"product_types"` // account types involved
	PaymentMechanisms   []string `json:"payment_mechanisms,omitempty"`
	SuspectedViolations []string `json:"suspected_violations,omitempty"`
	LawEnforcement      bool     `json:"law_enforcement_contacted"`
	LEAgency            string   `json:"le_agency,omitempty"`
	LEContactDate       string   `json:"le_contact_date,omitempty"`
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// FilingInstitution represents the institution filing the SAR
type FilingInstitution struct {
	Name          string   `json:"name"`
	TIN           string   `json:"tin"`
	IDType        string   `json:"id_type"` // rssd, fdic, etc.
	IDNumber      string   `json:"id_number"`
	Address       *Address `json:"address"`
	ContactName   string   `json:"contact_name"`
	ContactPhone  string   `json:"contact_phone"`
	ContactEmail  string   `json:"contact_email,omitempty"`
}

// CurrencyTransactionReport represents a CTR filing
type CurrencyTransactionReport struct {
	ID                string                 `json:"id"`
	Status            CTRStatus              `json:"status"`
	TransactionDate   time.Time              `json:"transaction_date"`
	Transactions      []CTRTransaction       `json:"transactions"`
	TotalCashIn       decimal.Decimal        `json:"total_cash_in"`
	TotalCashOut      decimal.Decimal        `json:"total_cash_out"`
	Persons           []CTRPerson            `json:"persons"`
	FilingInstitution *FilingInstitution     `json:"filing_institution"`
	PreparedBy        string                 `json:"prepared_by"`
	BSAIdentifier     string                 `json:"bsa_identifier,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	FiledAt           *time.Time             `json:"filed_at,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// CTRTransaction represents a transaction in a CTR
type CTRTransaction struct {
	TransactionID   string          `json:"transaction_id"`
	Type            string          `json:"type"` // cash_in, cash_out
	Amount          decimal.Decimal `json:"amount"`
	AccountNumber   string          `json:"account_number,omitempty"`
	ForeignCurrency bool            `json:"foreign_currency"`
	CurrencyCode    string          `json:"currency_code,omitempty"`
}

// CTRPerson represents a person involved in a CTR
type CTRPerson struct {
	Role           string   `json:"role"` // conductor, beneficiary
	Name           string   `json:"name"`
	DateOfBirth    string   `json:"date_of_birth,omitempty"`
	SSN            string   `json:"ssn,omitempty"`
	Address        *Address `json:"address,omitempty"`
	IDType         string   `json:"id_type"`
	IDNumber       string   `json:"id_number"`
	IDState        string   `json:"id_state,omitempty"`
	IDCountry      string   `json:"id_country"`
	Occupation     string   `json:"occupation,omitempty"`
	AccountNumbers []string `json:"account_numbers,omitempty"`
}

// CustomerRiskProfile represents a customer's AML risk profile
type CustomerRiskProfile struct {
	CustomerID       string                 `json:"customer_id"`
	CustomerType     string                 `json:"customer_type"` // individual, business
	RiskLevel        RiskLevel              `json:"risk_level"`
	RiskScore        float64                `json:"risk_score"`
	RiskFactors      []RiskFactor           `json:"risk_factors"`
	KYCStatus        string                 `json:"kyc_status"`
	KYCDate          *time.Time             `json:"kyc_date,omitempty"`
	KYCNextReview    *time.Time             `json:"kyc_next_review,omitempty"`
	WatchlistMatches []WatchlistMatch       `json:"watchlist_matches,omitempty"`
	PEPStatus        bool                   `json:"pep_status"`
	HighRiskCountry  bool                   `json:"high_risk_country"`
	HighRiskIndustry bool                   `json:"high_risk_industry"`
	CashIntensive    bool                   `json:"cash_intensive"`
	TransactionProfile *TransactionProfile  `json:"transaction_profile,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// RiskFactor represents a specific risk factor
type RiskFactor struct {
	Category    string  `json:"category"`
	Factor      string  `json:"factor"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description,omitempty"`
}

// WatchlistMatch represents a match against a watchlist
type WatchlistMatch struct {
	ID             string        `json:"id"`
	WatchlistType  WatchlistType `json:"watchlist_type"`
	MatchedName    string        `json:"matched_name"`
	MatchScore     float64       `json:"match_score"`
	MatchType      string        `json:"match_type"` // exact, fuzzy, alias
	ListEntryID    string        `json:"list_entry_id,omitempty"`
	Status         string        `json:"status"` // pending_review, confirmed, false_positive
	ReviewedBy     string        `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time    `json:"reviewed_at,omitempty"`
	MatchedAt      time.Time     `json:"matched_at"`
}

// TransactionProfile represents a customer's typical transaction pattern
type TransactionProfile struct {
	AverageMonthlyVolume  decimal.Decimal `json:"average_monthly_volume"`
	AverageTransactionSize decimal.Decimal `json:"average_transaction_size"`
	ExpectedMonthlyVolume decimal.Decimal `json:"expected_monthly_volume"`
	TypicalCountries      []string        `json:"typical_countries"`
	TypicalCounterparties []string        `json:"typical_counterparties"`
	CashActivityExpected  bool            `json:"cash_activity_expected"`
	WireActivityExpected  bool            `json:"wire_activity_expected"`
	InternationalExpected bool            `json:"international_expected"`
}

// AMLCase represents an AML investigation case
type AMLCase struct {
	ID              string                 `json:"id"`
	CaseNumber      string                 `json:"case_number"`
	Status          CaseStatus             `json:"status"`
	Priority        string                 `json:"priority"` // low, medium, high, critical
	CustomerID      string                 `json:"customer_id"`
	Type            string                 `json:"type"` // alert_driven, manual, periodic_review
	AssignedTo      string                 `json:"assigned_to,omitempty"`
	Alerts          []string               `json:"alert_ids"`
	Transactions    []string               `json:"transaction_ids"`
	Timeline        []CaseEvent            `json:"timeline"`
	Findings        string                 `json:"findings,omitempty"`
	Recommendation  string                 `json:"recommendation,omitempty"`
	SARRequired     bool                   `json:"sar_required"`
	SARID           string                 `json:"sar_id,omitempty"`
	ClosureReason   string                 `json:"closure_reason,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	ClosedAt        *time.Time             `json:"closed_at,omitempty"`
	DueDate         *time.Time             `json:"due_date,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// CaseStatus represents the status of an AML case
type CaseStatus string

const (
	CaseStatusOpen       CaseStatus = "open"
	CaseStatusInProgress CaseStatus = "in_progress"
	CaseStatusPending    CaseStatus = "pending_review"
	CaseStatusEscalated  CaseStatus = "escalated"
	CaseStatusClosed     CaseStatus = "closed"
)

// CaseEvent represents an event in a case timeline
type CaseEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Actor       string                 `json:"actor"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// AMLAlert represents an alert generated by the AML system
type AMLAlert struct {
	ID           string                 `json:"id"`
	CustomerID   string                 `json:"customer_id"`
	AlertType    AMLAlertType           `json:"alert_type"`
	Severity     models.AlertSeverity   `json:"severity"`
	Status       models.AlertStatus     `json:"status"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	RiskScore    float64                `json:"risk_score"`
	Indicators   []AlertIndicator       `json:"indicators"`
	Transactions []string               `json:"transaction_ids"`
	CaseID       string                 `json:"case_id,omitempty"`
	AssignedTo   string                 `json:"assigned_to,omitempty"`
	Resolution   string                 `json:"resolution,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AMLAlertType represents the type of AML alert
type AMLAlertType string

const (
	AlertTypeStructuring       AMLAlertType = "structuring"
	AlertTypeUnusualVolume     AMLAlertType = "unusual_volume"
	AlertTypeHighRiskCountry   AMLAlertType = "high_risk_country"
	AlertTypeWatchlistMatch    AMLAlertType = "watchlist_match"
	AlertTypePEP               AMLAlertType = "pep"
	AlertTypeRapidMovement     AMLAlertType = "rapid_movement"
	AlertTypeRoundAmounts      AMLAlertType = "round_amounts"
	AlertTypeUnusualPattern    AMLAlertType = "unusual_pattern"
	AlertTypeCTRThreshold      AMLAlertType = "ctr_threshold"
	AlertTypeKYCExpiring       AMLAlertType = "kyc_expiring"
)

// AlertIndicator represents a specific indicator that triggered an alert
type AlertIndicator struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Score       float64                `json:"score"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
}

// AMLStats contains AML statistics
type AMLStats struct {
	TotalAlerts       int            `json:"total_alerts"`
	OpenAlerts        int            `json:"open_alerts"`
	TotalCases        int            `json:"total_cases"`
	OpenCases         int            `json:"open_cases"`
	SARsFiled         int            `json:"sars_filed"`
	CTRsFiled         int            `json:"ctrs_filed"`
	WatchlistMatches  int            `json:"watchlist_matches"`
	HighRiskCustomers int            `json:"high_risk_customers"`
	AlertsByType      map[string]int `json:"alerts_by_type"`
	CasesByStatus     map[string]int `json:"cases_by_status"`
	Last30Days        struct {
		NewAlerts     int `json:"new_alerts"`
		ResolvedAlerts int `json:"resolved_alerts"`
		NewCases      int `json:"new_cases"`
		ClosedCases   int `json:"closed_cases"`
		SARsFiled     int `json:"sars_filed"`
	} `json:"last_30_days"`
}
