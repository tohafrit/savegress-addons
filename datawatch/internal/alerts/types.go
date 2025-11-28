package alerts

import (
	"time"
)

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeThreshold     AlertType = "threshold"
	AlertTypeAnomaly       AlertType = "anomaly"
	AlertTypeQuality       AlertType = "quality"
	AlertTypeSchema        AlertType = "schema"
	AlertTypeFreshness     AlertType = "freshness"
	AlertTypeError         AlertType = "error"
)

// Severity defines alert severity
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Status defines alert status
type Status string

const (
	StatusOpen         Status = "open"
	StatusAcknowledged Status = "acknowledged"
	StatusResolved     Status = "resolved"
	StatusSnoozed      Status = "snoozed"
)

// ConditionOperator defines comparison operators for alert conditions
type ConditionOperator string

const (
	OpGreaterThan      ConditionOperator = ">"
	OpGreaterThanEqual ConditionOperator = ">="
	OpLessThan         ConditionOperator = "<"
	OpLessThanEqual    ConditionOperator = "<="
	OpEqual            ConditionOperator = "=="
	OpNotEqual         ConditionOperator = "!="
)

// Rule defines an alert rule
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        AlertType              `json:"type"`
	Metric      string                 `json:"metric"`
	Condition   Condition              `json:"condition"`
	Duration    time.Duration          `json:"duration"`
	Severity    Severity               `json:"severity"`
	Channels    []string               `json:"channels"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Condition defines the alert trigger condition
type Condition struct {
	Operator  ConditionOperator `json:"operator"`
	Threshold float64           `json:"threshold"`
	// For relative comparisons
	CompareWith   string  `json:"compare_with,omitempty"` // baseline, previous_hour, same_hour_last_week
	ChangePercent float64 `json:"change_percent,omitempty"`
}

// Alert represents a triggered alert
type Alert struct {
	ID             string                 `json:"id"`
	RuleID         string                 `json:"rule_id"`
	RuleName       string                 `json:"rule_name"`
	Type           AlertType              `json:"type"`
	Severity       Severity               `json:"severity"`
	Status         Status                 `json:"status"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Metric         string                 `json:"metric,omitempty"`
	CurrentValue   float64                `json:"current_value,omitempty"`
	ThresholdValue float64                `json:"threshold_value,omitempty"`
	Labels         map[string]string      `json:"labels,omitempty"`
	Annotations    map[string]string      `json:"annotations,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	FiredAt        time.Time              `json:"fired_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy     string                 `json:"resolved_by,omitempty"`
	SnoozedUntil   *time.Time             `json:"snoozed_until,omitempty"`
	NotifiedAt     []NotificationRecord   `json:"notified_at,omitempty"`
}

// NotificationRecord records a notification attempt
type NotificationRecord struct {
	Channel   string    `json:"channel"`
	SentAt    time.Time `json:"sent_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// Channel represents a notification channel configuration
type Channel struct {
	ID       string                 `json:"id"`
	Type     ChannelType            `json:"type"`
	Name     string                 `json:"name"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
}

// ChannelType defines notification channel types
type ChannelType string

const (
	ChannelTypeSlack     ChannelType = "slack"
	ChannelTypeEmail     ChannelType = "email"
	ChannelTypePagerDuty ChannelType = "pagerduty"
	ChannelTypeWebhook   ChannelType = "webhook"
	ChannelTypeTeams     ChannelType = "teams"
	ChannelTypeDiscord   ChannelType = "discord"
)

// Notification represents a notification to be sent
type Notification struct {
	Alert     *Alert   `json:"alert"`
	Channels  []string `json:"channels"`
	Timestamp time.Time `json:"timestamp"`
}

// EvaluationResult contains the result of evaluating an alert rule
type EvaluationResult struct {
	RuleID     string    `json:"rule_id"`
	Triggered  bool      `json:"triggered"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	EvaluatedAt time.Time `json:"evaluated_at"`
	Error      string    `json:"error,omitempty"`
}

// AlertSummary contains a summary of alerts
type AlertSummary struct {
	TotalAlerts    int            `json:"total_alerts"`
	OpenAlerts     int            `json:"open_alerts"`
	AckedAlerts    int            `json:"acknowledged_alerts"`
	ResolvedAlerts int            `json:"resolved_alerts"`
	BySeverity     map[string]int `json:"by_severity"`
	ByType         map[string]int `json:"by_type"`
	RecentFired    int            `json:"recent_fired_1h"`
}

// Config contains alerting configuration
type Config struct {
	Enabled            bool                  `json:"enabled"`
	EvaluationInterval time.Duration         `json:"evaluation_interval"`
	RetentionDays      int                   `json:"retention_days"`
	Channels           map[string]*Channel   `json:"channels"`
}
