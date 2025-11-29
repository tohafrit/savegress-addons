package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chainlens/chainlens/backend/internal/monitor"
)

// CDCBridge bridges on-chain events to off-chain databases via Savegress CDC
type CDCBridge struct {
	configs     map[string]*monitor.CDCSyncConfig
	status      map[string]*monitor.SyncStatus
	monitor     *monitor.ContractMonitor
	cdcClient   CDCClient
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	eventCh     chan *monitor.ContractEvent
}

// CDCClient interface for Savegress CDC integration
type CDCClient interface {
	// PublishEvent publishes an event to the CDC pipeline
	PublishEvent(ctx context.Context, event *CDCEvent) error
	// BatchPublish publishes multiple events
	BatchPublish(ctx context.Context, events []*CDCEvent) error
	// CreateTable creates a table in the target database
	CreateTable(ctx context.Context, database, table string, schema map[string]string) error
}

// CDCEvent represents an event to be published to CDC
type CDCEvent struct {
	Operation string                 `json:"operation"` // INSERT, UPDATE, DELETE
	Database  string                 `json:"database"`
	Table     string                 `json:"table"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewCDCBridge creates a new CDC bridge
func NewCDCBridge(mon *monitor.ContractMonitor, client CDCClient) *CDCBridge {
	bridge := &CDCBridge{
		configs:   make(map[string]*monitor.CDCSyncConfig),
		status:    make(map[string]*monitor.SyncStatus),
		monitor:   mon,
		cdcClient: client,
		stopCh:    make(chan struct{}),
		eventCh:   make(chan *monitor.ContractEvent, 1000),
	}

	// Register event callback with monitor
	mon.SetEventCallback(bridge.handleEvent)

	return bridge
}

// Start starts the CDC bridge
func (b *CDCBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = true
	b.mu.Unlock()

	go b.processEvents(ctx)
	go b.balanceSyncLoop(ctx)

	return nil
}

// Stop stops the CDC bridge
func (b *CDCBridge) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		close(b.stopCh)
		b.running = false
	}
}

func (b *CDCBridge) handleEvent(event *monitor.ContractEvent) {
	select {
	case b.eventCh <- event:
	default:
		// Channel full, drop event
	}
}

func (b *CDCBridge) processEvents(ctx context.Context) {
	batch := make([]*CDCEvent, 0, 100)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return

		case event := <-b.eventCh:
			cdcEvents := b.transformEvent(event)
			batch = append(batch, cdcEvents...)

			if len(batch) >= 100 {
				b.flushBatch(ctx, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				b.flushBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

func (b *CDCBridge) transformEvent(event *monitor.ContractEvent) []*CDCEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var cdcEvents []*CDCEvent

	for _, config := range b.configs {
		if !config.Enabled {
			continue
		}
		if config.Contract != event.ContractAddress || config.ChainID != event.ChainID {
			continue
		}

		for _, mapping := range config.Events {
			if mapping.EventName != event.EventName {
				continue
			}

			data := make(map[string]interface{})

			// Map event arguments to database columns
			for argName, colName := range mapping.FieldMap {
				if val, ok := event.DecodedArgs[argName]; ok {
					data[colName] = val
				}
			}

			// Add standard fields
			data["tx_hash"] = event.TxHash
			data["block_number"] = event.BlockNumber
			data["log_index"] = event.LogIndex
			data["timestamp"] = event.Timestamp

			cdcEvent := &CDCEvent{
				Operation: "INSERT",
				Database:  config.TargetDatabase,
				Table:     mapping.TableName,
				Data:      data,
				Timestamp: event.Timestamp,
				Source:    "chainlens",
				Metadata: map[string]interface{}{
					"chain_id":         event.ChainID,
					"contract_address": event.ContractAddress,
					"event_name":       event.EventName,
				},
			}

			cdcEvents = append(cdcEvents, cdcEvent)
		}
	}

	return cdcEvents
}

func (b *CDCBridge) flushBatch(ctx context.Context, batch []*CDCEvent) {
	if b.cdcClient == nil || len(batch) == 0 {
		return
	}

	if err := b.cdcClient.BatchPublish(ctx, batch); err != nil {
		// Update status with error
		b.mu.Lock()
		for _, config := range b.configs {
			if status, ok := b.status[config.ID]; ok {
				status.ErrorCount++
				status.LastError = err.Error()
			}
		}
		b.mu.Unlock()
	} else {
		// Update status on success
		b.mu.Lock()
		for _, config := range b.configs {
			if status, ok := b.status[config.ID]; ok {
				status.EventsProcessed += int64(len(batch))
				status.LastSyncAt = time.Now()
			}
		}
		b.mu.Unlock()
	}
}

func (b *CDCBridge) balanceSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.syncBalances(ctx)
		}
	}
}

func (b *CDCBridge) syncBalances(ctx context.Context) {
	b.mu.RLock()
	configs := make([]*monitor.CDCSyncConfig, 0)
	for _, c := range b.configs {
		if c.Enabled && c.BalanceSync {
			configs = append(configs, c)
		}
	}
	b.mu.RUnlock()

	for _, config := range configs {
		contract, ok := b.monitor.GetContract(config.Contract, config.ChainID)
		if !ok || contract.Balance == nil {
			continue
		}

		// Create balance sync event
		cdcEvent := &CDCEvent{
			Operation: "UPSERT",
			Database:  config.TargetDatabase,
			Table:     "contract_balances",
			Data: map[string]interface{}{
				"contract_address": config.Contract,
				"chain_id":         config.ChainID,
				"balance":          contract.Balance.String(),
				"updated_at":       time.Now(),
			},
			Timestamp: time.Now(),
			Source:    "chainlens",
		}

		if b.cdcClient != nil {
			b.cdcClient.PublishEvent(ctx, cdcEvent)
		}
	}
}

// AddConfig adds a sync configuration
func (b *CDCBridge) AddConfig(config *monitor.CDCSyncConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if config.ID == "" {
		config.ID = fmt.Sprintf("sync_%d", time.Now().UnixNano())
	}
	config.CreatedAt = time.Now()

	b.configs[config.ID] = config
	b.status[config.ID] = &monitor.SyncStatus{
		ConfigID: config.ID,
		Status:   "running",
	}

	return nil
}

// GetConfig returns a config by ID
func (b *CDCBridge) GetConfig(id string) (*monitor.CDCSyncConfig, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	config, ok := b.configs[id]
	return config, ok
}

// ListConfigs returns all configs
func (b *CDCBridge) ListConfigs() []*monitor.CDCSyncConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()

	configs := make([]*monitor.CDCSyncConfig, 0, len(b.configs))
	for _, c := range b.configs {
		configs = append(configs, c)
	}
	return configs
}

// UpdateConfig updates a config
func (b *CDCBridge) UpdateConfig(config *monitor.CDCSyncConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.configs[config.ID]; !ok {
		return fmt.Errorf("config not found: %s", config.ID)
	}

	b.configs[config.ID] = config
	return nil
}

// DeleteConfig removes a config
func (b *CDCBridge) DeleteConfig(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.configs, id)
	delete(b.status, id)
}

// GetStatus returns sync status for a config
func (b *CDCBridge) GetStatus(configID string) (*monitor.SyncStatus, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	status, ok := b.status[configID]
	return status, ok
}

// GetAllStatus returns status for all configs
func (b *CDCBridge) GetAllStatus() []*monitor.SyncStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	statuses := make([]*monitor.SyncStatus, 0, len(b.status))
	for _, s := range b.status {
		statuses = append(statuses, s)
	}
	return statuses
}

// PauseSync pauses synchronization for a config
func (b *CDCBridge) PauseSync(configID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	config, ok := b.configs[configID]
	if !ok {
		return fmt.Errorf("config not found: %s", configID)
	}

	config.Enabled = false

	if status, ok := b.status[configID]; ok {
		status.Status = "stopped"
	}

	return nil
}

// ResumeSync resumes synchronization for a config
func (b *CDCBridge) ResumeSync(configID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	config, ok := b.configs[configID]
	if !ok {
		return fmt.Errorf("config not found: %s", configID)
	}

	config.Enabled = true

	if status, ok := b.status[configID]; ok {
		status.Status = "running"
	}

	return nil
}

// CreateConfigFromJSON creates a config from JSON
func (b *CDCBridge) CreateConfigFromJSON(data []byte) (*monitor.CDCSyncConfig, error) {
	var config monitor.CDCSyncConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := b.AddConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetBridgeStats returns CDC bridge statistics
func (b *CDCBridge) GetBridgeStats() *BridgeStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := &BridgeStats{
		TotalConfigs:  len(b.configs),
		ActiveConfigs: 0,
	}

	for _, config := range b.configs {
		if config.Enabled {
			stats.ActiveConfigs++
		}
	}

	for _, status := range b.status {
		stats.TotalEventsProcessed += status.EventsProcessed
		stats.TotalErrors += status.ErrorCount
	}

	return stats
}

// BridgeStats contains CDC bridge statistics
type BridgeStats struct {
	TotalConfigs         int   `json:"total_configs"`
	ActiveConfigs        int   `json:"active_configs"`
	TotalEventsProcessed int64 `json:"total_events_processed"`
	TotalErrors          int   `json:"total_errors"`
}

// MockCDCClient is a mock implementation for testing
type MockCDCClient struct {
	events []CDCEvent
	mu     sync.Mutex
}

// NewMockCDCClient creates a mock CDC client
func NewMockCDCClient() *MockCDCClient {
	return &MockCDCClient{
		events: make([]CDCEvent, 0),
	}
}

func (c *MockCDCClient) PublishEvent(ctx context.Context, event *CDCEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, *event)
	return nil
}

func (c *MockCDCClient) BatchPublish(ctx context.Context, events []*CDCEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range events {
		c.events = append(c.events, *e)
	}
	return nil
}

func (c *MockCDCClient) CreateTable(ctx context.Context, database, table string, schema map[string]string) error {
	return nil
}

func (c *MockCDCClient) GetEvents() []CDCEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]CDCEvent{}, c.events...)
}
