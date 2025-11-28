package edge

import (
	"encoding/json"
	"time"
)

// EdgeNodeState represents the state of an edge node
type EdgeNodeState string

const (
	EdgeNodeStateOnline       EdgeNodeState = "online"
	EdgeNodeStateOffline      EdgeNodeState = "offline"
	EdgeNodeStateConnecting   EdgeNodeState = "connecting"
	EdgeNodeStateError        EdgeNodeState = "error"
	EdgeNodeStateMaintenance  EdgeNodeState = "maintenance"
	EdgeNodeStateProvisioning EdgeNodeState = "provisioning"
)

// EdgeNode represents an edge computing node
type EdgeNode struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Location        string                 `json:"location,omitempty"`
	State           EdgeNodeState          `json:"state"`
	Capabilities    *NodeCapabilities      `json:"capabilities"`
	Configuration   *NodeConfiguration     `json:"configuration"`
	Metrics         *NodeMetrics           `json:"metrics,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
	ConnectedAt     time.Time              `json:"connected_at,omitempty"`
	LastHeartbeat   time.Time              `json:"last_heartbeat"`
	LastSync        time.Time              `json:"last_sync"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// NodeCapabilities represents the capabilities of an edge node
type NodeCapabilities struct {
	CPUCores        int      `json:"cpu_cores"`
	MemoryMB        int      `json:"memory_mb"`
	StorageGB       int      `json:"storage_gb"`
	GPUAvailable    bool     `json:"gpu_available"`
	GPUModel        string   `json:"gpu_model,omitempty"`
	Architecture    string   `json:"architecture"`
	OS              string   `json:"os"`
	ContainerRuntime string  `json:"container_runtime,omitempty"`
	Protocols       []string `json:"protocols"`
	MaxDevices      int      `json:"max_devices"`
}

// NodeConfiguration represents the configuration of an edge node
type NodeConfiguration struct {
	HeartbeatInterval    time.Duration          `json:"heartbeat_interval"`
	SyncInterval         time.Duration          `json:"sync_interval"`
	BufferSize           int                    `json:"buffer_size"`
	RetryPolicy          *RetryPolicy           `json:"retry_policy"`
	Compression          bool                   `json:"compression"`
	Encryption           bool                   `json:"encryption"`
	LocalProcessing      *LocalProcessingConfig `json:"local_processing,omitempty"`
	DataRetention        time.Duration          `json:"data_retention"`
	OfflineMode          *OfflineModeConfig     `json:"offline_mode,omitempty"`
}

// RetryPolicy defines retry behavior for edge communication
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	BackoffFactor  float64       `json:"backoff_factor"`
}

// LocalProcessingConfig defines local processing settings
type LocalProcessingConfig struct {
	Enabled          bool     `json:"enabled"`
	FilterRules      []string `json:"filter_rules,omitempty"`
	AggregationRules []string `json:"aggregation_rules,omitempty"`
	AlertRules       []string `json:"alert_rules,omitempty"`
	MLModels         []string `json:"ml_models,omitempty"`
}

// OfflineModeConfig defines offline mode behavior
type OfflineModeConfig struct {
	Enabled            bool          `json:"enabled"`
	MaxBufferSize      int           `json:"max_buffer_size"`
	PersistToDisk      bool          `json:"persist_to_disk"`
	SyncOnReconnect    bool          `json:"sync_on_reconnect"`
	MaxOfflineDuration time.Duration `json:"max_offline_duration"`
}

// NodeMetrics represents real-time metrics for an edge node
type NodeMetrics struct {
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	DiskUsage       float64   `json:"disk_usage"`
	NetworkInBytes  int64     `json:"network_in_bytes"`
	NetworkOutBytes int64     `json:"network_out_bytes"`
	DeviceCount     int       `json:"device_count"`
	MessageRate     float64   `json:"message_rate"`
	ErrorRate       float64   `json:"error_rate"`
	Latency         float64   `json:"latency_ms"`
	BufferUsage     float64   `json:"buffer_usage"`
	Uptime          time.Duration `json:"uptime"`
	Timestamp       time.Time `json:"timestamp"`
}

// EdgeDevice represents a device connected to an edge node
type EdgeDevice struct {
	ID            string                 `json:"id"`
	NodeID        string                 `json:"node_id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Protocol      string                 `json:"protocol"`
	Address       string                 `json:"address"`
	State         DeviceState            `json:"state"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	LastData      time.Time              `json:"last_data"`
	DataPoints    map[string]DataPoint   `json:"data_points,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// DeviceState represents the state of a connected device
type DeviceState string

const (
	DeviceStateConnected    DeviceState = "connected"
	DeviceStateDisconnected DeviceState = "disconnected"
	DeviceStateError        DeviceState = "error"
	DeviceStateUnknown      DeviceState = "unknown"
)

// DataPoint represents a data point from a device
type DataPoint struct {
	Name      string      `json:"name"`
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`
	Unit      string      `json:"unit,omitempty"`
	Quality   string      `json:"quality"`
	Timestamp time.Time   `json:"timestamp"`
}

// EdgeMessage represents a message from the edge
type EdgeMessage struct {
	ID          string                 `json:"id"`
	NodeID      string                 `json:"node_id"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Type        MessageType            `json:"type"`
	Topic       string                 `json:"topic"`
	Payload     json.RawMessage        `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    int                    `json:"priority"`
	Compressed  bool                   `json:"compressed"`
	Encrypted   bool                   `json:"encrypted"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// MessageType represents the type of edge message
type MessageType string

const (
	MessageTypeTelemetry   MessageType = "telemetry"
	MessageTypeAlert       MessageType = "alert"
	MessageTypeEvent       MessageType = "event"
	MessageTypeCommand     MessageType = "command"
	MessageTypeResponse    MessageType = "response"
	MessageTypeHeartbeat   MessageType = "heartbeat"
	MessageTypeConfig      MessageType = "config"
	MessageTypeSync        MessageType = "sync"
)

// EdgeCommand represents a command to send to the edge
type EdgeCommand struct {
	ID          string                 `json:"id"`
	NodeID      string                 `json:"node_id"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Command     string                 `json:"command"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiredAt   time.Time              `json:"expired_at,omitempty"`
}

// CommandResult represents the result of an edge command
type CommandResult struct {
	CommandID   string                 `json:"command_id"`
	NodeID      string                 `json:"node_id"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Status      CommandStatus          `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
}

// CommandStatus represents the status of a command
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusSent      CommandStatus = "sent"
	CommandStatusRunning   CommandStatus = "running"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
	CommandStatusCancelled CommandStatus = "cancelled"
)

// EdgeAlert represents an alert from the edge
type EdgeAlert struct {
	ID          string            `json:"id"`
	NodeID      string            `json:"node_id"`
	DeviceID    string            `json:"device_id,omitempty"`
	Severity    AlertSeverity     `json:"severity"`
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Source      string            `json:"source"`
	Timestamp   time.Time         `json:"timestamp"`
	Acknowledged bool             `json:"acknowledged"`
	Resolved    bool              `json:"resolved"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// EdgeDeployment represents a deployment to edge nodes
type EdgeDeployment struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	TargetNodes    []string               `json:"target_nodes"`
	Modules        []DeploymentModule     `json:"modules"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	Status         DeploymentStatus       `json:"status"`
	Progress       map[string]float64     `json:"progress"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      time.Time              `json:"started_at,omitempty"`
	CompletedAt    time.Time              `json:"completed_at,omitempty"`
}

// DeploymentModule represents a module to deploy
type DeploymentModule struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Image       string                 `json:"image,omitempty"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Resources   *ResourceRequirements  `json:"resources,omitempty"`
}

// ResourceRequirements defines resource requirements for a module
type ResourceRequirements struct {
	CPURequest    string `json:"cpu_request,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	MemoryRequest string `json:"memory_request,omitempty"`
	MemoryLimit   string `json:"memory_limit,omitempty"`
}

// DeploymentStatus represents deployment status
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusInProgress DeploymentStatus = "in_progress"
	DeploymentStatusCompleted  DeploymentStatus = "completed"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back"
)

// SyncRequest represents a sync request from edge
type SyncRequest struct {
	NodeID        string    `json:"node_id"`
	LastSyncTime  time.Time `json:"last_sync_time"`
	MessageCount  int       `json:"message_count"`
	SizeBytes     int64     `json:"size_bytes"`
	Compressed    bool      `json:"compressed"`
}

// SyncResponse represents a sync response to edge
type SyncResponse struct {
	Success       bool      `json:"success"`
	MessagesSynced int      `json:"messages_synced"`
	NextSyncTime  time.Time `json:"next_sync_time"`
	Commands      []EdgeCommand `json:"commands,omitempty"`
	ConfigUpdates map[string]interface{} `json:"config_updates,omitempty"`
}

// EdgeRule represents a rule for edge processing
type EdgeRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Type        RuleType               `json:"type"`
	Condition   string                 `json:"condition"`
	Action      *RuleAction            `json:"action"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RuleType represents the type of edge rule
type RuleType string

const (
	RuleTypeFilter     RuleType = "filter"
	RuleTypeTransform  RuleType = "transform"
	RuleTypeAggregate  RuleType = "aggregate"
	RuleTypeAlert      RuleType = "alert"
	RuleTypeForward    RuleType = "forward"
)

// RuleAction represents the action to take when a rule matches
type RuleAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ProtocolAdapter represents a protocol adapter for device communication
type ProtocolAdapter struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Protocol    string                 `json:"protocol"`
	Version     string                 `json:"version"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
	DeviceCount int                    `json:"device_count"`
}

// SupportedProtocol represents a supported protocol
type SupportedProtocol string

const (
	ProtocolMQTT      SupportedProtocol = "mqtt"
	ProtocolModbus    SupportedProtocol = "modbus"
	ProtocolOPCUA     SupportedProtocol = "opcua"
	ProtocolHTTP      SupportedProtocol = "http"
	ProtocolCoAP      SupportedProtocol = "coap"
	ProtocolBACnet    SupportedProtocol = "bacnet"
	ProtocolProfinet  SupportedProtocol = "profinet"
	ProtocolEthernetIP SupportedProtocol = "ethernet_ip"
)
