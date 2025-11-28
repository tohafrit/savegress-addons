package digitaltwin

import (
	"encoding/json"
	"time"
)

// TwinType represents the type of digital twin
type TwinType string

const (
	TwinTypeDevice    TwinType = "device"
	TwinTypeAsset     TwinType = "asset"
	TwinTypeLine      TwinType = "production_line"
	TwinTypeFacility  TwinType = "facility"
	TwinTypeProcess   TwinType = "process"
	TwinTypeComponent TwinType = "component"
)

// TwinState represents the operational state of a twin
type TwinState string

const (
	TwinStateOnline      TwinState = "online"
	TwinStateOffline     TwinState = "offline"
	TwinStateMaintenance TwinState = "maintenance"
	TwinStateDegraded    TwinState = "degraded"
	TwinStateFailed      TwinState = "failed"
	TwinStateUnknown     TwinState = "unknown"
)

// DigitalTwin represents a virtual representation of a physical asset
type DigitalTwin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        TwinType               `json:"type"`
	PhysicalID  string                 `json:"physical_id"`
	Description string                 `json:"description,omitempty"`
	State       TwinState              `json:"state"`
	Properties  map[string]interface{} `json:"properties"`
	Telemetry   map[string]TelemetryPoint `json:"telemetry"`
	Attributes  map[string]string      `json:"attributes"`
	Tags        []string               `json:"tags,omitempty"`
	Parent      string                 `json:"parent,omitempty"`
	Children    []string               `json:"children,omitempty"`
	Model       *TwinModel             `json:"model,omitempty"`
	Location    *Location              `json:"location,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastSync    time.Time              `json:"last_sync"`
}

// TelemetryPoint represents a real-time data point
type TelemetryPoint struct {
	Name      string      `json:"name"`
	Value     interface{} `json:"value"`
	Unit      string      `json:"unit,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Quality   DataQuality `json:"quality"`
	Source    string      `json:"source,omitempty"`
}

// DataQuality represents the quality of telemetry data
type DataQuality string

const (
	DataQualityGood      DataQuality = "good"
	DataQualityBad       DataQuality = "bad"
	DataQualityUncertain DataQuality = "uncertain"
	DataQualityStale     DataQuality = "stale"
)

// TwinModel represents the model/schema definition for a twin
type TwinModel struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Schema      json.RawMessage       `json:"schema"`
	Properties  []PropertyDefinition  `json:"properties"`
	Telemetry   []TelemetryDefinition `json:"telemetry"`
	Commands    []CommandDefinition   `json:"commands"`
	Relationships []RelationshipDef   `json:"relationships,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

// PropertyDefinition defines a property in the twin model
type PropertyDefinition struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name,omitempty"`
	Type        string      `json:"type"`
	Unit        string      `json:"unit,omitempty"`
	Writable    bool        `json:"writable"`
	Default     interface{} `json:"default,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	EnumValues  []string    `json:"enum_values,omitempty"`
}

// TelemetryDefinition defines telemetry in the twin model
type TelemetryDefinition struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name,omitempty"`
	Type        string   `json:"type"`
	Unit        string   `json:"unit,omitempty"`
	SampleRate  string   `json:"sample_rate,omitempty"`
	Aggregation string   `json:"aggregation,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
}

// CommandDefinition defines a command that can be sent to the twin
type CommandDefinition struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Request     *CommandPayload   `json:"request,omitempty"`
	Response    *CommandPayload   `json:"response,omitempty"`
}

// CommandPayload defines the payload for a command
type CommandPayload struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

// RelationshipDef defines relationships between twins
type RelationshipDef struct {
	Name       string   `json:"name"`
	Target     string   `json:"target"`
	MaxCount   int      `json:"max_count,omitempty"`
	Properties []string `json:"properties,omitempty"`
}

// Location represents the physical location of a twin
type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	Floor     string  `json:"floor,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	Building  string  `json:"building,omitempty"`
	Site      string  `json:"site,omitempty"`
}

// Relationship represents a connection between twins
type Relationship struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// TwinQuery represents a query for digital twins
type TwinQuery struct {
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Select     []string               `json:"select,omitempty"`
	OrderBy    string                 `json:"order_by,omitempty"`
	Descending bool                   `json:"descending,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
}

// SimulationConfig represents configuration for twin simulation
type SimulationConfig struct {
	TwinID       string                 `json:"twin_id"`
	Duration     time.Duration          `json:"duration"`
	TimeStep     time.Duration          `json:"time_step"`
	Scenarios    []SimulationScenario   `json:"scenarios"`
	InitialState map[string]interface{} `json:"initial_state,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// SimulationScenario represents a scenario to simulate
type SimulationScenario struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	StartTime   time.Duration          `json:"start_time"`
	Events      []SimulationEvent      `json:"events"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
}

// SimulationEvent represents an event in a simulation
type SimulationEvent struct {
	Type       string                 `json:"type"`
	Timestamp  time.Duration          `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
}

// SimulationResult represents the result of a simulation
type SimulationResult struct {
	TwinID      string                   `json:"twin_id"`
	StartTime   time.Time                `json:"start_time"`
	EndTime     time.Time                `json:"end_time"`
	Duration    time.Duration            `json:"duration"`
	TimeSteps   int                      `json:"time_steps"`
	StateHistory []StateSnapshot         `json:"state_history"`
	Metrics     map[string]float64       `json:"metrics"`
	Alerts      []SimulationAlert        `json:"alerts,omitempty"`
	Summary     string                   `json:"summary,omitempty"`
}

// StateSnapshot represents the state at a point in time
type StateSnapshot struct {
	Timestamp  time.Duration          `json:"timestamp"`
	State      TwinState              `json:"state"`
	Properties map[string]interface{} `json:"properties"`
	Telemetry  map[string]interface{} `json:"telemetry"`
}

// SimulationAlert represents an alert generated during simulation
type SimulationAlert struct {
	Timestamp   time.Duration `json:"timestamp"`
	Severity    string        `json:"severity"`
	Message     string        `json:"message"`
	Property    string        `json:"property,omitempty"`
	Threshold   float64       `json:"threshold,omitempty"`
	ActualValue float64       `json:"actual_value,omitempty"`
}

// TwinEvent represents an event in the twin's lifecycle
type TwinEvent struct {
	ID        string                 `json:"id"`
	TwinID    string                 `json:"twin_id"`
	Type      TwinEventType          `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Source    string                 `json:"source,omitempty"`
}

// TwinEventType represents the type of twin event
type TwinEventType string

const (
	TwinEventCreated         TwinEventType = "created"
	TwinEventUpdated         TwinEventType = "updated"
	TwinEventDeleted         TwinEventType = "deleted"
	TwinEventStateChanged    TwinEventType = "state_changed"
	TwinEventTelemetryUpdate TwinEventType = "telemetry_update"
	TwinEventPropertyUpdate  TwinEventType = "property_update"
	TwinEventCommandExecuted TwinEventType = "command_executed"
	TwinEventAlertRaised     TwinEventType = "alert_raised"
	TwinEventSyncCompleted   TwinEventType = "sync_completed"
)

// CommandRequest represents a request to execute a command on a twin
type CommandRequest struct {
	TwinID      string                 `json:"twin_id"`
	CommandName string                 `json:"command_name"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Async       bool                   `json:"async"`
}

// CommandResponse represents the response from a command execution
type CommandResponse struct {
	RequestID   string                 `json:"request_id"`
	TwinID      string                 `json:"twin_id"`
	CommandName string                 `json:"command_name"`
	Status      CommandStatus          `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
}

// CommandStatus represents the status of a command execution
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusRunning   CommandStatus = "running"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
)

// SyncStatus represents the synchronization status between twin and physical asset
type SyncStatus struct {
	TwinID        string    `json:"twin_id"`
	PhysicalID    string    `json:"physical_id"`
	LastSync      time.Time `json:"last_sync"`
	SyncState     string    `json:"sync_state"`
	PendingUpdates int      `json:"pending_updates"`
	SyncErrors    []string  `json:"sync_errors,omitempty"`
	Latency       time.Duration `json:"latency"`
}
