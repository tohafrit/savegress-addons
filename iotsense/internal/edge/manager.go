package edge

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager manages edge nodes
type Manager struct {
	mu              sync.RWMutex
	nodes           map[string]*EdgeNode
	devices         map[string]*EdgeDevice
	deployments     map[string]*EdgeDeployment
	rules           map[string]*EdgeRule
	adapters        map[string]*ProtocolAdapter
	messageHandlers []MessageHandler
	alertHandlers   []AlertHandler
	commandQueue    chan *EdgeCommand
	stopChan        chan struct{}
	running         bool
}

// MessageHandler handles incoming edge messages
type MessageHandler func(msg *EdgeMessage)

// AlertHandler handles edge alerts
type AlertHandler func(alert *EdgeAlert)

// NewManager creates a new edge manager
func NewManager() *Manager {
	return &Manager{
		nodes:           make(map[string]*EdgeNode),
		devices:         make(map[string]*EdgeDevice),
		deployments:     make(map[string]*EdgeDeployment),
		rules:           make(map[string]*EdgeRule),
		adapters:        make(map[string]*ProtocolAdapter),
		messageHandlers: make([]MessageHandler, 0),
		alertHandlers:   make([]AlertHandler, 0),
		commandQueue:    make(chan *EdgeCommand, 1000),
		stopChan:        make(chan struct{}),
	}
}

// Start starts the edge manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("manager already running")
	}
	m.running = true
	m.mu.Unlock()

	go m.heartbeatMonitor(ctx)
	go m.commandProcessor(ctx)

	return nil
}

// Stop stops the edge manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopChan)
	m.running = false

	return nil
}

// RegisterNode registers a new edge node
func (m *Manager) RegisterNode(ctx context.Context, node *EdgeNode) (*EdgeNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	// Set defaults
	if node.Configuration == nil {
		node.Configuration = &NodeConfiguration{
			HeartbeatInterval: 30 * time.Second,
			SyncInterval:      60 * time.Second,
			BufferSize:        10000,
			Compression:       true,
			Encryption:        true,
			DataRetention:     24 * time.Hour,
			RetryPolicy: &RetryPolicy{
				MaxRetries:     3,
				InitialBackoff: time.Second,
				MaxBackoff:     30 * time.Second,
				BackoffFactor:  2.0,
			},
		}
	}

	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now
	node.LastHeartbeat = now
	node.ConnectedAt = now

	if node.State == "" {
		node.State = EdgeNodeStateProvisioning
	}

	if node.Metadata == nil {
		node.Metadata = make(map[string]string)
	}

	m.nodes[node.ID] = node

	return node, nil
}

// GetNode retrieves an edge node by ID
func (m *Manager) GetNode(ctx context.Context, id string) (*EdgeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, ok := m.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	return node, nil
}

// ListNodes lists all edge nodes
func (m *Manager) ListNodes(ctx context.Context) []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]*EdgeNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// UpdateNode updates an edge node
func (m *Manager) UpdateNode(ctx context.Context, id string, updates *EdgeNode) (*EdgeNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	if updates.Name != "" {
		node.Name = updates.Name
	}
	if updates.Description != "" {
		node.Description = updates.Description
	}
	if updates.Location != "" {
		node.Location = updates.Location
	}
	if updates.Configuration != nil {
		node.Configuration = updates.Configuration
	}
	if len(updates.Tags) > 0 {
		node.Tags = updates.Tags
	}

	node.UpdatedAt = time.Now()

	return node, nil
}

// DeleteNode deletes an edge node
func (m *Manager) DeleteNode(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.nodes[id]; !ok {
		return fmt.Errorf("node %s not found", id)
	}

	// Remove associated devices
	for deviceID, device := range m.devices {
		if device.NodeID == id {
			delete(m.devices, deviceID)
		}
	}

	delete(m.nodes, id)

	return nil
}

// UpdateNodeState updates the state of an edge node
func (m *Manager) UpdateNodeState(ctx context.Context, id string, state EdgeNodeState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[id]
	if !ok {
		return fmt.Errorf("node %s not found", id)
	}

	node.State = state
	node.UpdatedAt = time.Now()

	return nil
}

// RecordHeartbeat records a heartbeat from an edge node
func (m *Manager) RecordHeartbeat(ctx context.Context, id string, metrics *NodeMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[id]
	if !ok {
		return fmt.Errorf("node %s not found", id)
	}

	node.LastHeartbeat = time.Now()
	node.Metrics = metrics

	if node.State == EdgeNodeStateOffline || node.State == EdgeNodeStateConnecting {
		node.State = EdgeNodeStateOnline
	}

	return nil
}

// heartbeatMonitor monitors node heartbeats
func (m *Manager) heartbeatMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkHeartbeats()
		}
	}
}

// checkHeartbeats checks for missed heartbeats
func (m *Manager) checkHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for _, node := range m.nodes {
		if node.Configuration == nil {
			continue
		}

		// Check if heartbeat is overdue (3x the interval)
		timeout := node.Configuration.HeartbeatInterval * 3
		if now.Sub(node.LastHeartbeat) > timeout && node.State == EdgeNodeStateOnline {
			node.State = EdgeNodeStateOffline

			// Emit alert
			alert := &EdgeAlert{
				ID:        uuid.New().String(),
				NodeID:    node.ID,
				Severity:  AlertSeverityHigh,
				Type:      "node_offline",
				Message:   fmt.Sprintf("Node %s missed heartbeat", node.Name),
				Source:    "heartbeat_monitor",
				Timestamp: now,
			}
			m.emitAlert(alert)
		}
	}
}

// RegisterDevice registers a device connected to an edge node
func (m *Manager) RegisterDevice(ctx context.Context, device *EdgeDevice) (*EdgeDevice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate node exists
	if _, ok := m.nodes[device.NodeID]; !ok {
		return nil, fmt.Errorf("node %s not found", device.NodeID)
	}

	if device.ID == "" {
		device.ID = uuid.New().String()
	}

	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now

	if device.State == "" {
		device.State = DeviceStateUnknown
	}

	if device.DataPoints == nil {
		device.DataPoints = make(map[string]DataPoint)
	}

	m.devices[device.ID] = device

	// Update node device count
	if node, ok := m.nodes[device.NodeID]; ok && node.Metrics != nil {
		node.Metrics.DeviceCount++
	}

	return device, nil
}

// GetDevice retrieves a device by ID
func (m *Manager) GetDevice(ctx context.Context, id string) (*EdgeDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, ok := m.devices[id]
	if !ok {
		return nil, fmt.Errorf("device %s not found", id)
	}

	return device, nil
}

// ListDevices lists devices, optionally filtered by node
func (m *Manager) ListDevices(ctx context.Context, nodeID string) []*EdgeDevice {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]*EdgeDevice, 0)
	for _, device := range m.devices {
		if nodeID == "" || device.NodeID == nodeID {
			devices = append(devices, device)
		}
	}

	return devices
}

// UpdateDeviceData updates data points for a device
func (m *Manager) UpdateDeviceData(ctx context.Context, deviceID string, dataPoints map[string]DataPoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, ok := m.devices[deviceID]
	if !ok {
		return fmt.Errorf("device %s not found", deviceID)
	}

	for name, point := range dataPoints {
		point.Name = name
		if point.Timestamp.IsZero() {
			point.Timestamp = time.Now()
		}
		device.DataPoints[name] = point
	}

	device.LastData = time.Now()
	device.UpdatedAt = time.Now()

	return nil
}

// SendCommand sends a command to an edge node
func (m *Manager) SendCommand(ctx context.Context, cmd *EdgeCommand) (*CommandResult, error) {
	m.mu.RLock()
	node, ok := m.nodes[cmd.NodeID]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("node %s not found", cmd.NodeID)
	}
	m.mu.RUnlock()

	if node.State != EdgeNodeStateOnline {
		return nil, fmt.Errorf("node %s is not online", cmd.NodeID)
	}

	if cmd.ID == "" {
		cmd.ID = uuid.New().String()
	}
	cmd.CreatedAt = time.Now()

	// Queue command
	select {
	case m.commandQueue <- cmd:
	default:
		return nil, fmt.Errorf("command queue full")
	}

	// In a real implementation, this would wait for the response
	// For now, return a pending result
	return &CommandResult{
		CommandID: cmd.ID,
		NodeID:    cmd.NodeID,
		DeviceID:  cmd.DeviceID,
		Status:    CommandStatusSent,
		StartedAt: time.Now(),
	}, nil
}

// commandProcessor processes outgoing commands
func (m *Manager) commandProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case cmd := <-m.commandQueue:
			// In a real implementation, this would send the command to the edge node
			// via MQTT, gRPC, or another protocol
			_ = cmd
		}
	}
}

// ProcessMessage processes an incoming edge message
func (m *Manager) ProcessMessage(ctx context.Context, msg *EdgeMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Update node last heartbeat if it's a heartbeat message
	if msg.Type == MessageTypeHeartbeat {
		m.RecordHeartbeat(ctx, msg.NodeID, nil)
	}

	// Emit to handlers
	for _, handler := range m.messageHandlers {
		go handler(msg)
	}

	return nil
}

// RegisterMessageHandler registers a message handler
func (m *Manager) RegisterMessageHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageHandlers = append(m.messageHandlers, handler)
}

// RegisterAlertHandler registers an alert handler
func (m *Manager) RegisterAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertHandlers = append(m.alertHandlers, handler)
}

// emitAlert emits an alert to all handlers
func (m *Manager) emitAlert(alert *EdgeAlert) {
	for _, handler := range m.alertHandlers {
		go handler(alert)
	}
}

// CreateDeployment creates a new deployment
func (m *Manager) CreateDeployment(ctx context.Context, deployment *EdgeDeployment) (*EdgeDeployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if deployment.ID == "" {
		deployment.ID = uuid.New().String()
	}

	// Validate target nodes exist
	for _, nodeID := range deployment.TargetNodes {
		if _, ok := m.nodes[nodeID]; !ok {
			return nil, fmt.Errorf("target node %s not found", nodeID)
		}
	}

	deployment.Status = DeploymentStatusPending
	deployment.Progress = make(map[string]float64)
	deployment.CreatedAt = time.Now()

	for _, nodeID := range deployment.TargetNodes {
		deployment.Progress[nodeID] = 0
	}

	m.deployments[deployment.ID] = deployment

	return deployment, nil
}

// StartDeployment starts a deployment
func (m *Manager) StartDeployment(ctx context.Context, deploymentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deployment, ok := m.deployments[deploymentID]
	if !ok {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	if deployment.Status != DeploymentStatusPending {
		return fmt.Errorf("deployment already started")
	}

	deployment.Status = DeploymentStatusInProgress
	deployment.StartedAt = time.Now()

	// In a real implementation, this would orchestrate the deployment
	// to all target nodes

	return nil
}

// GetDeployment retrieves a deployment
func (m *Manager) GetDeployment(ctx context.Context, id string) (*EdgeDeployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deployment, ok := m.deployments[id]
	if !ok {
		return nil, fmt.Errorf("deployment %s not found", id)
	}

	return deployment, nil
}

// CreateRule creates an edge processing rule
func (m *Manager) CreateRule(ctx context.Context, rule *EdgeRule) (*EdgeRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	rule.CreatedAt = time.Now()
	m.rules[rule.ID] = rule

	return rule, nil
}

// GetRules retrieves all rules
func (m *Manager) GetRules(ctx context.Context) []*EdgeRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*EdgeRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}

	return rules
}

// RegisterAdapter registers a protocol adapter
func (m *Manager) RegisterAdapter(ctx context.Context, adapter *ProtocolAdapter) (*ProtocolAdapter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if adapter.ID == "" {
		adapter.ID = uuid.New().String()
	}

	m.adapters[adapter.ID] = adapter

	return adapter, nil
}

// GetAdapters retrieves all protocol adapters
func (m *Manager) GetAdapters(ctx context.Context) []*ProtocolAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapters := make([]*ProtocolAdapter, 0, len(m.adapters))
	for _, adapter := range m.adapters {
		adapters = append(adapters, adapter)
	}

	return adapters
}

// GetNodeStats returns statistics for an edge node
func (m *Manager) GetNodeStats(ctx context.Context, nodeID string) (*NodeStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, ok := m.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	deviceCount := 0
	for _, device := range m.devices {
		if device.NodeID == nodeID {
			deviceCount++
		}
	}

	stats := &NodeStats{
		NodeID:      nodeID,
		State:       node.State,
		DeviceCount: deviceCount,
		Uptime:      time.Since(node.ConnectedAt),
		LastSync:    node.LastSync,
	}

	if node.Metrics != nil {
		stats.CPUUsage = node.Metrics.CPUUsage
		stats.MemoryUsage = node.Metrics.MemoryUsage
		stats.MessageRate = node.Metrics.MessageRate
	}

	return stats, nil
}

// NodeStats represents statistics for a node
type NodeStats struct {
	NodeID      string        `json:"node_id"`
	State       EdgeNodeState `json:"state"`
	DeviceCount int           `json:"device_count"`
	CPUUsage    float64       `json:"cpu_usage"`
	MemoryUsage float64       `json:"memory_usage"`
	MessageRate float64       `json:"message_rate"`
	Uptime      time.Duration `json:"uptime"`
	LastSync    time.Time     `json:"last_sync"`
}

// HandleSync handles a sync request from an edge node
func (m *Manager) HandleSync(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[req.NodeID]
	if !ok {
		return nil, fmt.Errorf("node %s not found", req.NodeID)
	}

	node.LastSync = time.Now()

	// In a real implementation, this would:
	// 1. Process incoming messages from the edge
	// 2. Queue any pending commands for the node
	// 3. Send configuration updates if needed

	response := &SyncResponse{
		Success:        true,
		MessagesSynced: req.MessageCount,
		NextSyncTime:   time.Now().Add(node.Configuration.SyncInterval),
		Commands:       make([]EdgeCommand, 0),
		ConfigUpdates:  make(map[string]interface{}),
	}

	return response, nil
}
