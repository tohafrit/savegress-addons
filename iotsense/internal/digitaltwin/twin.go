package digitaltwin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TwinManager manages digital twins
type TwinManager struct {
	mu            sync.RWMutex
	twins         map[string]*DigitalTwin
	models        map[string]*TwinModel
	relationships map[string]*Relationship
	eventHandlers []EventHandler
	syncManager   *SyncManager
	simulator     *Simulator
}

// EventHandler handles twin events
type EventHandler func(event *TwinEvent)

// NewTwinManager creates a new twin manager
func NewTwinManager() *TwinManager {
	tm := &TwinManager{
		twins:         make(map[string]*DigitalTwin),
		models:        make(map[string]*TwinModel),
		relationships: make(map[string]*Relationship),
		eventHandlers: make([]EventHandler, 0),
	}
	tm.syncManager = NewSyncManager(tm)
	tm.simulator = NewSimulator(tm)
	return tm
}

// CreateTwin creates a new digital twin
func (tm *TwinManager) CreateTwin(ctx context.Context, twin *DigitalTwin) (*DigitalTwin, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if twin.ID == "" {
		twin.ID = uuid.New().String()
	}

	if twin.PhysicalID == "" {
		return nil, fmt.Errorf("physical ID is required")
	}

	// Validate model if specified
	if twin.Model != nil && twin.Model.ID != "" {
		model, ok := tm.models[twin.Model.ID]
		if !ok {
			return nil, fmt.Errorf("model %s not found", twin.Model.ID)
		}
		twin.Model = model
	}

	// Initialize maps if nil
	if twin.Properties == nil {
		twin.Properties = make(map[string]interface{})
	}
	if twin.Telemetry == nil {
		twin.Telemetry = make(map[string]TelemetryPoint)
	}
	if twin.Attributes == nil {
		twin.Attributes = make(map[string]string)
	}

	now := time.Now()
	twin.CreatedAt = now
	twin.UpdatedAt = now
	twin.LastSync = now

	if twin.State == "" {
		twin.State = TwinStateUnknown
	}

	tm.twins[twin.ID] = twin

	// Emit creation event
	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    twin.ID,
		Type:      TwinEventCreated,
		Timestamp: now,
		Data: map[string]interface{}{
			"name":        twin.Name,
			"type":        twin.Type,
			"physical_id": twin.PhysicalID,
		},
	})

	return twin, nil
}

// GetTwin retrieves a digital twin by ID
func (tm *TwinManager) GetTwin(ctx context.Context, id string) (*DigitalTwin, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	twin, ok := tm.twins[id]
	if !ok {
		return nil, fmt.Errorf("twin %s not found", id)
	}

	return twin, nil
}

// GetTwinByPhysicalID retrieves a digital twin by physical asset ID
func (tm *TwinManager) GetTwinByPhysicalID(ctx context.Context, physicalID string) (*DigitalTwin, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, twin := range tm.twins {
		if twin.PhysicalID == physicalID {
			return twin, nil
		}
	}

	return nil, fmt.Errorf("twin with physical ID %s not found", physicalID)
}

// UpdateTwin updates a digital twin
func (tm *TwinManager) UpdateTwin(ctx context.Context, id string, updates map[string]interface{}) (*DigitalTwin, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	twin, ok := tm.twins[id]
	if !ok {
		return nil, fmt.Errorf("twin %s not found", id)
	}

	// Apply updates to properties
	for key, value := range updates {
		twin.Properties[key] = value
	}

	twin.UpdatedAt = time.Now()

	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    twin.ID,
		Type:      TwinEventPropertyUpdate,
		Timestamp: twin.UpdatedAt,
		Data:      updates,
	})

	return twin, nil
}

// UpdateTelemetry updates telemetry data for a twin
func (tm *TwinManager) UpdateTelemetry(ctx context.Context, id string, telemetry map[string]TelemetryPoint) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	twin, ok := tm.twins[id]
	if !ok {
		return fmt.Errorf("twin %s not found", id)
	}

	for name, point := range telemetry {
		point.Name = name
		if point.Timestamp.IsZero() {
			point.Timestamp = time.Now()
		}
		if point.Quality == "" {
			point.Quality = DataQualityGood
		}
		twin.Telemetry[name] = point
	}

	twin.UpdatedAt = time.Now()
	twin.LastSync = time.Now()

	// Convert telemetry to generic map for event
	telemetryData := make(map[string]interface{})
	for name, point := range telemetry {
		telemetryData[name] = point.Value
	}

	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    twin.ID,
		Type:      TwinEventTelemetryUpdate,
		Timestamp: twin.UpdatedAt,
		Data:      telemetryData,
	})

	return nil
}

// UpdateState updates the state of a twin
func (tm *TwinManager) UpdateState(ctx context.Context, id string, state TwinState) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	twin, ok := tm.twins[id]
	if !ok {
		return fmt.Errorf("twin %s not found", id)
	}

	oldState := twin.State
	twin.State = state
	twin.UpdatedAt = time.Now()

	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    twin.ID,
		Type:      TwinEventStateChanged,
		Timestamp: twin.UpdatedAt,
		Data: map[string]interface{}{
			"old_state": oldState,
			"new_state": state,
		},
	})

	return nil
}

// DeleteTwin deletes a digital twin
func (tm *TwinManager) DeleteTwin(ctx context.Context, id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	twin, ok := tm.twins[id]
	if !ok {
		return fmt.Errorf("twin %s not found", id)
	}

	// Remove relationships involving this twin
	for relID, rel := range tm.relationships {
		if rel.SourceID == id || rel.TargetID == id {
			delete(tm.relationships, relID)
		}
	}

	// Update parent's children list
	if twin.Parent != "" {
		if parent, ok := tm.twins[twin.Parent]; ok {
			newChildren := make([]string, 0)
			for _, childID := range parent.Children {
				if childID != id {
					newChildren = append(newChildren, childID)
				}
			}
			parent.Children = newChildren
		}
	}

	delete(tm.twins, id)

	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    id,
		Type:      TwinEventDeleted,
		Timestamp: time.Now(),
	})

	return nil
}

// QueryTwins queries digital twins
func (tm *TwinManager) QueryTwins(ctx context.Context, query *TwinQuery) ([]*DigitalTwin, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	results := make([]*DigitalTwin, 0)

	for _, twin := range tm.twins {
		if tm.matchesFilter(twin, query.Filter) {
			results = append(results, twin)
		}
	}

	// Apply offset and limit
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// matchesFilter checks if a twin matches the query filter
func (tm *TwinManager) matchesFilter(twin *DigitalTwin, filter map[string]interface{}) bool {
	if filter == nil {
		return true
	}

	for key, value := range filter {
		switch key {
		case "type":
			if twin.Type != TwinType(value.(string)) {
				return false
			}
		case "state":
			if twin.State != TwinState(value.(string)) {
				return false
			}
		case "parent":
			if twin.Parent != value.(string) {
				return false
			}
		case "name":
			if twin.Name != value.(string) {
				return false
			}
		default:
			// Check properties
			if propValue, ok := twin.Properties[key]; ok {
				if propValue != value {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

// CreateModel creates a new twin model
func (tm *TwinManager) CreateModel(ctx context.Context, model *TwinModel) (*TwinModel, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if model.ID == "" {
		model.ID = uuid.New().String()
	}

	model.CreatedAt = time.Now()
	tm.models[model.ID] = model

	return model, nil
}

// GetModel retrieves a twin model by ID
func (tm *TwinManager) GetModel(ctx context.Context, id string) (*TwinModel, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	model, ok := tm.models[id]
	if !ok {
		return nil, fmt.Errorf("model %s not found", id)
	}

	return model, nil
}

// CreateRelationship creates a relationship between twins
func (tm *TwinManager) CreateRelationship(ctx context.Context, rel *Relationship) (*Relationship, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Validate source and target exist
	if _, ok := tm.twins[rel.SourceID]; !ok {
		return nil, fmt.Errorf("source twin %s not found", rel.SourceID)
	}
	if _, ok := tm.twins[rel.TargetID]; !ok {
		return nil, fmt.Errorf("target twin %s not found", rel.TargetID)
	}

	if rel.ID == "" {
		rel.ID = uuid.New().String()
	}
	rel.CreatedAt = time.Now()

	tm.relationships[rel.ID] = rel

	return rel, nil
}

// GetRelationships gets relationships for a twin
func (tm *TwinManager) GetRelationships(ctx context.Context, twinID string) ([]*Relationship, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	results := make([]*Relationship, 0)

	for _, rel := range tm.relationships {
		if rel.SourceID == twinID || rel.TargetID == twinID {
			results = append(results, rel)
		}
	}

	return results, nil
}

// ExecuteCommand executes a command on a twin
func (tm *TwinManager) ExecuteCommand(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	tm.mu.RLock()
	twin, ok := tm.twins[req.TwinID]
	if !ok {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("twin %s not found", req.TwinID)
	}
	tm.mu.RUnlock()

	response := &CommandResponse{
		RequestID:   uuid.New().String(),
		TwinID:      req.TwinID,
		CommandName: req.CommandName,
		Status:      CommandStatusRunning,
		StartedAt:   time.Now(),
	}

	// Validate command exists in model
	if twin.Model != nil {
		commandFound := false
		for _, cmd := range twin.Model.Commands {
			if cmd.Name == req.CommandName {
				commandFound = true
				break
			}
		}
		if !commandFound {
			response.Status = CommandStatusFailed
			response.Error = fmt.Sprintf("command %s not found in model", req.CommandName)
			response.CompletedAt = time.Now()
			return response, nil
		}
	}

	// Execute command (in real implementation, this would send to the physical device)
	// For now, we simulate success
	response.Status = CommandStatusCompleted
	response.CompletedAt = time.Now()
	response.Result = map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Command %s executed successfully", req.CommandName),
	}

	tm.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    req.TwinID,
		Type:      TwinEventCommandExecuted,
		Timestamp: response.CompletedAt,
		Data: map[string]interface{}{
			"command":    req.CommandName,
			"request_id": response.RequestID,
			"status":     response.Status,
		},
	})

	return response, nil
}

// GetChildren gets all children of a twin
func (tm *TwinManager) GetChildren(ctx context.Context, twinID string) ([]*DigitalTwin, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	parent, ok := tm.twins[twinID]
	if !ok {
		return nil, fmt.Errorf("twin %s not found", twinID)
	}

	children := make([]*DigitalTwin, 0, len(parent.Children))
	for _, childID := range parent.Children {
		if child, ok := tm.twins[childID]; ok {
			children = append(children, child)
		}
	}

	return children, nil
}

// SetParent sets the parent of a twin
func (tm *TwinManager) SetParent(ctx context.Context, twinID, parentID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	twin, ok := tm.twins[twinID]
	if !ok {
		return fmt.Errorf("twin %s not found", twinID)
	}

	parent, ok := tm.twins[parentID]
	if !ok {
		return fmt.Errorf("parent twin %s not found", parentID)
	}

	// Remove from old parent if exists
	if twin.Parent != "" {
		if oldParent, ok := tm.twins[twin.Parent]; ok {
			newChildren := make([]string, 0)
			for _, childID := range oldParent.Children {
				if childID != twinID {
					newChildren = append(newChildren, childID)
				}
			}
			oldParent.Children = newChildren
		}
	}

	// Set new parent
	twin.Parent = parentID
	parent.Children = append(parent.Children, twinID)

	return nil
}

// RegisterEventHandler registers an event handler
func (tm *TwinManager) RegisterEventHandler(handler EventHandler) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.eventHandlers = append(tm.eventHandlers, handler)
}

// emitEvent emits an event to all handlers
func (tm *TwinManager) emitEvent(event *TwinEvent) {
	for _, handler := range tm.eventHandlers {
		go handler(event)
	}
}

// GetSyncManager returns the sync manager
func (tm *TwinManager) GetSyncManager() *SyncManager {
	return tm.syncManager
}

// GetSimulator returns the simulator
func (tm *TwinManager) GetSimulator() *Simulator {
	return tm.simulator
}

// ExportTwin exports a twin to JSON
func (tm *TwinManager) ExportTwin(ctx context.Context, id string) ([]byte, error) {
	twin, err := tm.GetTwin(ctx, id)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(twin, "", "  ")
}

// ImportTwin imports a twin from JSON
func (tm *TwinManager) ImportTwin(ctx context.Context, data []byte) (*DigitalTwin, error) {
	var twin DigitalTwin
	if err := json.Unmarshal(data, &twin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twin: %w", err)
	}

	// Clear ID to create new
	twin.ID = ""

	return tm.CreateTwin(ctx, &twin)
}

// GetAllTwins returns all twins
func (tm *TwinManager) GetAllTwins(ctx context.Context) []*DigitalTwin {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	twins := make([]*DigitalTwin, 0, len(tm.twins))
	for _, twin := range tm.twins {
		twins = append(twins, twin)
	}

	return twins
}

// GetTwinHierarchy gets the full hierarchy starting from a root twin
func (tm *TwinManager) GetTwinHierarchy(ctx context.Context, rootID string) (*TwinHierarchy, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	root, ok := tm.twins[rootID]
	if !ok {
		return nil, fmt.Errorf("twin %s not found", rootID)
	}

	return tm.buildHierarchy(root), nil
}

// TwinHierarchy represents a hierarchical view of twins
type TwinHierarchy struct {
	Twin     *DigitalTwin    `json:"twin"`
	Children []*TwinHierarchy `json:"children,omitempty"`
}

// buildHierarchy recursively builds the hierarchy
func (tm *TwinManager) buildHierarchy(twin *DigitalTwin) *TwinHierarchy {
	hierarchy := &TwinHierarchy{
		Twin:     twin,
		Children: make([]*TwinHierarchy, 0),
	}

	for _, childID := range twin.Children {
		if child, ok := tm.twins[childID]; ok {
			hierarchy.Children = append(hierarchy.Children, tm.buildHierarchy(child))
		}
	}

	return hierarchy
}
