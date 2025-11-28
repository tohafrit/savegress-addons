package models

import (
	"time"
)

// DeviceStatus represents the status of a device
type DeviceStatus string

const (
	DeviceStatusOnline      DeviceStatus = "online"
	DeviceStatusOffline     DeviceStatus = "offline"
	DeviceStatusMaintenance DeviceStatus = "maintenance"
	DeviceStatusError       DeviceStatus = "error"
	DeviceStatusUnknown     DeviceStatus = "unknown"
)

// DeviceType represents the type of IoT device
type DeviceType string

const (
	DeviceTypeSensor     DeviceType = "sensor"
	DeviceTypeActuator   DeviceType = "actuator"
	DeviceTypeGateway    DeviceType = "gateway"
	DeviceTypeController DeviceType = "controller"
	DeviceTypeCamera     DeviceType = "camera"
	DeviceTypeMeter      DeviceType = "meter"
)

// Device represents an IoT device
type Device struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          DeviceType             `json:"type"`
	Model         string                 `json:"model"`
	Manufacturer  string                 `json:"manufacturer"`
	SerialNumber  string                 `json:"serial_number"`
	FirmwareVer   string                 `json:"firmware_version"`
	Status        DeviceStatus           `json:"status"`
	Location      *Location              `json:"location,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Capabilities  []Capability           `json:"capabilities,omitempty"`
	LastSeen      *time.Time             `json:"last_seen,omitempty"`
	LastTelemetry *time.Time             `json:"last_telemetry,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// Location represents a physical location
type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	Floor     string  `json:"floor,omitempty"`
	Room      string  `json:"room,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	Building  string  `json:"building,omitempty"`
	Site      string  `json:"site,omitempty"`
}

// Capability represents a device capability
type Capability struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // telemetry, command, property
	DataType   string                 `json:"data_type"`
	Unit       string                 `json:"unit,omitempty"`
	Readable   bool                   `json:"readable"`
	Writable   bool                   `json:"writable"`
	Min        *float64               `json:"min,omitempty"`
	Max        *float64               `json:"max,omitempty"`
	EnumValues []string               `json:"enum_values,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TelemetryPoint represents a single telemetry data point
type TelemetryPoint struct {
	DeviceID  string                 `json:"device_id"`
	Timestamp time.Time              `json:"timestamp"`
	Metric    string                 `json:"metric"`
	Value     interface{}            `json:"value"`
	Unit      string                 `json:"unit,omitempty"`
	Quality   DataQuality            `json:"quality,omitempty"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DataQuality represents the quality of telemetry data
type DataQuality string

const (
	DataQualityGood      DataQuality = "good"
	DataQualityUncertain DataQuality = "uncertain"
	DataQualityBad       DataQuality = "bad"
	DataQualityStale     DataQuality = "stale"
)

// TelemetryBatch represents a batch of telemetry points
type TelemetryBatch struct {
	DeviceID  string           `json:"device_id"`
	Timestamp time.Time        `json:"timestamp"`
	Points    []TelemetryPoint `json:"points"`
}

// DeviceGroup represents a group of devices
type DeviceGroup struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	ParentID    string            `json:"parent_id,omitempty"`
	DeviceIDs   []string          `json:"device_ids"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Alert represents an IoT alert
type Alert struct {
	ID          string         `json:"id"`
	DeviceID    string         `json:"device_id"`
	RuleID      string         `json:"rule_id"`
	Severity    AlertSeverity  `json:"severity"`
	Type        AlertType      `json:"type"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Status      AlertStatus    `json:"status"`
	TriggerValue interface{}   `json:"trigger_value,omitempty"`
	Threshold   interface{}    `json:"threshold,omitempty"`
	Metric      string         `json:"metric,omitempty"`
	AckedBy     string         `json:"acked_by,omitempty"`
	AckedAt     *time.Time     `json:"acked_at,omitempty"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeThreshold   AlertType = "threshold"
	AlertTypeAnomaly     AlertType = "anomaly"
	AlertTypeOffline     AlertType = "offline"
	AlertTypeBattery     AlertType = "battery"
	AlertTypeGeofence    AlertType = "geofence"
	AlertTypeRule        AlertType = "rule"
	AlertTypeMaintenance AlertType = "maintenance"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusSuppressed   AlertStatus = "suppressed"
)

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	DeviceIDs   []string          `json:"device_ids,omitempty"`
	GroupIDs    []string          `json:"group_ids,omitempty"`
	Metric      string            `json:"metric"`
	Condition   RuleCondition     `json:"condition"`
	Severity    AlertSeverity     `json:"severity"`
	Cooldown    time.Duration     `json:"cooldown"`
	Actions     []AlertAction     `json:"actions,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// RuleCondition represents a condition for an alert rule
type RuleCondition struct {
	Type      string      `json:"type"` // threshold, range, change, absence
	Operator  string      `json:"operator"` // gt, lt, gte, lte, eq, neq
	Value     interface{} `json:"value"`
	Duration  string      `json:"duration,omitempty"`
	Window    string      `json:"window,omitempty"`
	MinValue  interface{} `json:"min_value,omitempty"`
	MaxValue  interface{} `json:"max_value,omitempty"`
}

// AlertAction represents an action to take when an alert fires
type AlertAction struct {
	Type       string                 `json:"type"` // email, webhook, sms, slack, command
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Command represents a command to send to a device
type Command struct {
	ID         string                 `json:"id"`
	DeviceID   string                 `json:"device_id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Status     CommandStatus          `json:"status"`
	Result     interface{}            `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	SentAt     *time.Time             `json:"sent_at,omitempty"`
	AckedAt    *time.Time             `json:"acked_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
}

// CommandStatus represents the status of a command
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusSent      CommandStatus = "sent"
	CommandStatusAcked     CommandStatus = "acknowledged"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusExpired   CommandStatus = "expired"
)

// DeviceEvent represents an event from a device
type DeviceEvent struct {
	ID        string                 `json:"id"`
	DeviceID  string                 `json:"device_id"`
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// AggregatedMetric represents aggregated telemetry data
type AggregatedMetric struct {
	DeviceID  string    `json:"device_id"`
	Metric    string    `json:"metric"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Count     int       `json:"count"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	Sum       float64   `json:"sum"`
	Avg       float64   `json:"avg"`
	StdDev    float64   `json:"std_dev,omitempty"`
	Median    float64   `json:"median,omitempty"`
	P95       float64   `json:"p95,omitempty"`
	P99       float64   `json:"p99,omitempty"`
}

// TimeSeriesQuery represents a query for time series data
type TimeSeriesQuery struct {
	DeviceID    string            `json:"device_id"`
	DeviceIDs   []string          `json:"device_ids,omitempty"`
	Metric      string            `json:"metric"`
	Metrics     []string          `json:"metrics,omitempty"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Aggregation string            `json:"aggregation,omitempty"` // avg, min, max, sum, count
	Interval    string            `json:"interval,omitempty"`    // 1m, 5m, 1h, 1d
	Fill        string            `json:"fill,omitempty"`        // none, null, previous, linear
	Tags        map[string]string `json:"tags,omitempty"`
	Limit       int               `json:"limit,omitempty"`
}

// TimeSeriesResult represents time series query results
type TimeSeriesResult struct {
	DeviceID string            `json:"device_id"`
	Metric   string            `json:"metric"`
	Points   []TimeSeriesPoint `json:"points"`
}

// TimeSeriesPoint represents a single point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
}

// DeviceStats contains statistics about a device
type DeviceStats struct {
	DeviceID         string    `json:"device_id"`
	TotalDataPoints  int64     `json:"total_data_points"`
	DataPointsToday  int64     `json:"data_points_today"`
	LastTelemetry    time.Time `json:"last_telemetry"`
	UptimePercent    float64   `json:"uptime_percent"`
	AlertCount       int       `json:"alert_count"`
	ActiveAlerts     int       `json:"active_alerts"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	MetricsCollected []string  `json:"metrics_collected"`
}

// SystemStats contains overall system statistics
type SystemStats struct {
	TotalDevices      int                `json:"total_devices"`
	OnlineDevices     int                `json:"online_devices"`
	OfflineDevices    int                `json:"offline_devices"`
	TotalDataPoints   int64              `json:"total_data_points"`
	DataPointsPerSec  float64            `json:"data_points_per_sec"`
	ActiveAlerts      int                `json:"active_alerts"`
	DevicesByType     map[string]int     `json:"devices_by_type"`
	DevicesByStatus   map[string]int     `json:"devices_by_status"`
	TopMetrics        []MetricCount      `json:"top_metrics"`
}

// MetricCount represents a metric with its count
type MetricCount struct {
	Metric string `json:"metric"`
	Count  int64  `json:"count"`
}
