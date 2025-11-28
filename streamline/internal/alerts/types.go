package alerts

import "time"

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeLowStock        AlertType = "low_stock"
	AlertTypeOutOfStock      AlertType = "out_of_stock"
	AlertTypeOverstock       AlertType = "overstock"
	AlertTypePriceChange     AlertType = "price_change"
	AlertTypeSyncError       AlertType = "sync_error"
	AlertTypeOrderAnomaly    AlertType = "order_anomaly"
	AlertTypeChannelDown     AlertType = "channel_down"
	AlertTypeListingError    AlertType = "listing_error"
	AlertTypeInventoryMismatch AlertType = "inventory_mismatch"
	AlertTypeSalesSpike      AlertType = "sales_spike"
	AlertTypeSalesDrop       AlertType = "sales_drop"
	AlertTypeReturnRate      AlertType = "return_rate"
)

// AlertSeverity represents the severity level
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusOpen         AlertStatus = "open"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusSnoozed      AlertStatus = "snoozed"
)

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        AlertType              `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Channels    []string               `json:"channels"` // Channel IDs to apply rule to
	Condition   AlertCondition         `json:"condition"`
	Actions     []AlertAction          `json:"actions"`
	Cooldown    time.Duration          `json:"cooldown"` // Min time between alerts
	Schedule    *AlertSchedule         `json:"schedule,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastFiredAt *time.Time             `json:"last_fired_at,omitempty"`
}

// AlertCondition defines when an alert should trigger
type AlertCondition struct {
	// For inventory alerts
	Threshold      int     `json:"threshold,omitempty"`       // Stock threshold
	PercentChange  float64 `json:"percent_change,omitempty"`  // For price/sales changes

	// For comparison alerts
	Operator       string  `json:"operator,omitempty"`        // lt, gt, eq, lte, gte
	Value          float64 `json:"value,omitempty"`

	// For time-based alerts
	TimeWindow     time.Duration `json:"time_window,omitempty"`

	// For pattern matching
	Pattern        string  `json:"pattern,omitempty"`

	// For multi-condition rules
	And            []AlertCondition `json:"and,omitempty"`
	Or             []AlertCondition `json:"or,omitempty"`
}

// AlertAction defines what happens when an alert triggers
type AlertAction struct {
	Type       ActionType             `json:"type"`
	Target     string                 `json:"target"`      // Channel name, webhook URL, email
	Template   string                 `json:"template,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// ActionType represents the type of alert action
type ActionType string

const (
	ActionTypeSlack     ActionType = "slack"
	ActionTypeEmail     ActionType = "email"
	ActionTypeWebhook   ActionType = "webhook"
	ActionTypeSMS       ActionType = "sms"
	ActionTypePagerDuty ActionType = "pagerduty"
	ActionTypeTeams     ActionType = "teams"
	ActionTypeDiscord   ActionType = "discord"
	ActionTypeAutoReorder ActionType = "auto_reorder"
	ActionTypeAutoDisable ActionType = "auto_disable"
)

// AlertSchedule defines when a rule is active
type AlertSchedule struct {
	Timezone    string   `json:"timezone"`
	ActiveDays  []int    `json:"active_days"`  // 0=Sunday, 6=Saturday
	ActiveStart string   `json:"active_start"` // HH:MM format
	ActiveEnd   string   `json:"active_end"`   // HH:MM format
}

// Alert represents a triggered alert instance
type Alert struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	ChannelID   string                 `json:"channel_id,omitempty"`
	ChannelName string                 `json:"channel_name,omitempty"`
	ProductID   string                 `json:"product_id,omitempty"`
	ProductSKU  string                 `json:"product_sku,omitempty"`
	ProductName string                 `json:"product_name,omitempty"`
	OrderID     string                 `json:"order_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	FiredAt     time.Time              `json:"fired_at"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	AckedBy     string                 `json:"acked_by,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy  string                 `json:"resolved_by,omitempty"`
	SnoozedUntil *time.Time            `json:"snoozed_until,omitempty"`
	Notes       []AlertNote            `json:"notes,omitempty"`
}

// AlertNote represents a note added to an alert
type AlertNote struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationChannel represents a configured notification channel
type NotificationChannel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        ActionType             `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertStats contains alert statistics
type AlertStats struct {
	TotalAlerts    int            `json:"total_alerts"`
	OpenAlerts     int            `json:"open_alerts"`
	AckedAlerts    int            `json:"acked_alerts"`
	ResolvedAlerts int            `json:"resolved_alerts"`
	ByType         map[string]int `json:"by_type"`
	BySeverity     map[string]int `json:"by_severity"`
	ByChannel      map[string]int `json:"by_channel"`
	Last24Hours    int            `json:"last_24_hours"`
	Last7Days      int            `json:"last_7_days"`
}

// InventorySnapshot represents a point-in-time inventory state
type InventorySnapshot struct {
	ProductID   string    `json:"product_id"`
	SKU         string    `json:"sku"`
	ChannelID   string    `json:"channel_id"`
	Quantity    int       `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

// SalesMetric represents sales data for analysis
type SalesMetric struct {
	ProductID   string    `json:"product_id"`
	SKU         string    `json:"sku"`
	ChannelID   string    `json:"channel_id"`
	Units       int       `json:"units"`
	Revenue     float64   `json:"revenue"`
	Period      time.Time `json:"period"` // Start of period
	PeriodType  string    `json:"period_type"` // hourly, daily, weekly
}

// AlertFilter defines filters for querying alerts
type AlertFilter struct {
	Status     AlertStatus   `json:"status,omitempty"`
	Type       AlertType     `json:"type,omitempty"`
	Severity   AlertSeverity `json:"severity,omitempty"`
	ChannelID  string        `json:"channel_id,omitempty"`
	ProductID  string        `json:"product_id,omitempty"`
	RuleID     string        `json:"rule_id,omitempty"`
	StartTime  *time.Time    `json:"start_time,omitempty"`
	EndTime    *time.Time    `json:"end_time,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
}
