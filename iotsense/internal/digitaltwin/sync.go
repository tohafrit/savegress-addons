package digitaltwin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SyncManager handles synchronization between digital twins and physical assets
type SyncManager struct {
	mu            sync.RWMutex
	manager       *TwinManager
	syncStatus    map[string]*SyncStatus
	subscriptions map[string][]SyncSubscription
	syncInterval  time.Duration
	stopChan      chan struct{}
	running       bool
}

// SyncSubscription represents a subscription to sync updates
type SyncSubscription struct {
	ID       string
	TwinID   string
	Callback func(status *SyncStatus)
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	TwinID           string        `json:"twin_id"`
	SyncInterval     time.Duration `json:"sync_interval"`
	TelemetryFields  []string      `json:"telemetry_fields,omitempty"`
	PropertyFields   []string      `json:"property_fields,omitempty"`
	SyncMode         SyncMode      `json:"sync_mode"`
	ConflictStrategy string        `json:"conflict_strategy"`
}

// SyncMode represents the synchronization mode
type SyncMode string

const (
	SyncModeRealtime    SyncMode = "realtime"
	SyncModePolling     SyncMode = "polling"
	SyncModeOnDemand    SyncMode = "on_demand"
	SyncModeBidirectional SyncMode = "bidirectional"
)

// NewSyncManager creates a new sync manager
func NewSyncManager(manager *TwinManager) *SyncManager {
	return &SyncManager{
		manager:       manager,
		syncStatus:    make(map[string]*SyncStatus),
		subscriptions: make(map[string][]SyncSubscription),
		syncInterval:  30 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the sync manager
func (sm *SyncManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return fmt.Errorf("sync manager already running")
	}
	sm.running = true
	sm.mu.Unlock()

	go sm.syncLoop(ctx)

	return nil
}

// Stop stops the sync manager
func (sm *SyncManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	close(sm.stopChan)
	sm.running = false

	return nil
}

// syncLoop periodically syncs all twins
func (sm *SyncManager) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.syncAllTwins(ctx)
		}
	}
}

// syncAllTwins syncs all registered twins
func (sm *SyncManager) syncAllTwins(ctx context.Context) {
	twins := sm.manager.GetAllTwins(ctx)

	for _, twin := range twins {
		if err := sm.SyncTwin(ctx, twin.ID); err != nil {
			// Log error but continue with other twins
			continue
		}
	}
}

// SyncTwin synchronizes a specific twin with its physical asset
func (sm *SyncManager) SyncTwin(ctx context.Context, twinID string) error {
	twin, err := sm.manager.GetTwin(ctx, twinID)
	if err != nil {
		return err
	}

	startTime := time.Now()

	// In a real implementation, this would:
	// 1. Connect to the physical device/asset
	// 2. Fetch latest telemetry and state
	// 3. Update the digital twin
	// 4. Optionally push changes from twin to physical (bidirectional)

	// Simulate sync process
	status := &SyncStatus{
		TwinID:         twinID,
		PhysicalID:     twin.PhysicalID,
		LastSync:       time.Now(),
		SyncState:      "completed",
		PendingUpdates: 0,
		Latency:        time.Since(startTime),
	}

	sm.mu.Lock()
	sm.syncStatus[twinID] = status
	sm.mu.Unlock()

	// Notify subscribers
	sm.notifySubscribers(twinID, status)

	// Emit sync event
	sm.manager.emitEvent(&TwinEvent{
		ID:        uuid.New().String(),
		TwinID:    twinID,
		Type:      TwinEventSyncCompleted,
		Timestamp: status.LastSync,
		Data: map[string]interface{}{
			"latency_ms": status.Latency.Milliseconds(),
			"state":      status.SyncState,
		},
	})

	return nil
}

// GetSyncStatus gets the sync status for a twin
func (sm *SyncManager) GetSyncStatus(twinID string) (*SyncStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status, ok := sm.syncStatus[twinID]
	if !ok {
		return nil, fmt.Errorf("no sync status for twin %s", twinID)
	}

	return status, nil
}

// Subscribe subscribes to sync updates for a twin
func (sm *SyncManager) Subscribe(twinID string, callback func(status *SyncStatus)) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub := SyncSubscription{
		ID:       uuid.New().String(),
		TwinID:   twinID,
		Callback: callback,
	}

	sm.subscriptions[twinID] = append(sm.subscriptions[twinID], sub)

	return sub.ID
}

// Unsubscribe removes a subscription
func (sm *SyncManager) Unsubscribe(twinID, subscriptionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	subs := sm.subscriptions[twinID]
	newSubs := make([]SyncSubscription, 0)

	for _, sub := range subs {
		if sub.ID != subscriptionID {
			newSubs = append(newSubs, sub)
		}
	}

	sm.subscriptions[twinID] = newSubs
}

// notifySubscribers notifies all subscribers of a sync status update
func (sm *SyncManager) notifySubscribers(twinID string, status *SyncStatus) {
	sm.mu.RLock()
	subs := sm.subscriptions[twinID]
	sm.mu.RUnlock()

	for _, sub := range subs {
		go sub.Callback(status)
	}
}

// SetSyncInterval sets the sync interval
func (sm *SyncManager) SetSyncInterval(interval time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.syncInterval = interval
}

// ForceSyncAll forces synchronization of all twins
func (sm *SyncManager) ForceSyncAll(ctx context.Context) error {
	twins := sm.manager.GetAllTwins(ctx)
	var lastErr error

	for _, twin := range twins {
		if err := sm.SyncTwin(ctx, twin.ID); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetAllSyncStatus gets sync status for all twins
func (sm *SyncManager) GetAllSyncStatus() map[string]*SyncStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*SyncStatus)
	for id, status := range sm.syncStatus {
		result[id] = status
	}

	return result
}

// PushToPhysical pushes twin state changes to the physical asset
func (sm *SyncManager) PushToPhysical(ctx context.Context, twinID string, changes map[string]interface{}) error {
	twin, err := sm.manager.GetTwin(ctx, twinID)
	if err != nil {
		return err
	}

	// In a real implementation, this would:
	// 1. Connect to the physical device
	// 2. Validate the changes are allowed
	// 3. Send commands to update the physical asset
	// 4. Wait for confirmation
	// 5. Update sync status

	// For now, simulate success
	status := &SyncStatus{
		TwinID:         twinID,
		PhysicalID:     twin.PhysicalID,
		LastSync:       time.Now(),
		SyncState:      "push_completed",
		PendingUpdates: 0,
		Latency:        10 * time.Millisecond,
	}

	sm.mu.Lock()
	sm.syncStatus[twinID] = status
	sm.mu.Unlock()

	sm.notifySubscribers(twinID, status)

	return nil
}

// RecordSyncError records a sync error
func (sm *SyncManager) RecordSyncError(twinID string, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	status, ok := sm.syncStatus[twinID]
	if !ok {
		status = &SyncStatus{
			TwinID:     twinID,
			SyncErrors: make([]string, 0),
		}
		sm.syncStatus[twinID] = status
	}

	status.SyncState = "error"
	status.SyncErrors = append(status.SyncErrors, err.Error())

	// Keep only last 10 errors
	if len(status.SyncErrors) > 10 {
		status.SyncErrors = status.SyncErrors[len(status.SyncErrors)-10:]
	}
}

// ClearSyncErrors clears sync errors for a twin
func (sm *SyncManager) ClearSyncErrors(twinID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if status, ok := sm.syncStatus[twinID]; ok {
		status.SyncErrors = nil
	}
}
