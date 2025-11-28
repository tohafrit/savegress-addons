package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Gateway handles edge-to-cloud communication
type Gateway struct {
	mu            sync.RWMutex
	manager       *Manager
	connections   map[string]*NodeConnection
	messageBuffer *MessageBuffer
	config        *GatewayConfig
	stopChan      chan struct{}
	running       bool
}

// NodeConnection represents a connection to an edge node
type NodeConnection struct {
	NodeID       string
	Protocol     string
	Connected    bool
	ConnectedAt  time.Time
	LastActivity time.Time
	MessageCount int64
	ErrorCount   int64
}

// GatewayConfig represents gateway configuration
type GatewayConfig struct {
	Port                 int           `json:"port"`
	TLSEnabled           bool          `json:"tls_enabled"`
	CertFile             string        `json:"cert_file,omitempty"`
	KeyFile              string        `json:"key_file,omitempty"`
	MaxConnections       int           `json:"max_connections"`
	MessageBufferSize    int           `json:"message_buffer_size"`
	ConnectionTimeout    time.Duration `json:"connection_timeout"`
	KeepAliveInterval    time.Duration `json:"keep_alive_interval"`
	AuthRequired         bool          `json:"auth_required"`
	CompressionEnabled   bool          `json:"compression_enabled"`
	RateLimitPerSecond   int           `json:"rate_limit_per_second"`
}

// MessageBuffer buffers messages for processing
type MessageBuffer struct {
	mu       sync.RWMutex
	messages []*EdgeMessage
	maxSize  int
}

// NewGateway creates a new edge gateway
func NewGateway(manager *Manager, config *GatewayConfig) *Gateway {
	if config == nil {
		config = &GatewayConfig{
			Port:               8883,
			TLSEnabled:         true,
			MaxConnections:     1000,
			MessageBufferSize:  10000,
			ConnectionTimeout:  30 * time.Second,
			KeepAliveInterval:  60 * time.Second,
			AuthRequired:       true,
			CompressionEnabled: true,
			RateLimitPerSecond: 1000,
		}
	}

	return &Gateway{
		manager:     manager,
		connections: make(map[string]*NodeConnection),
		messageBuffer: &MessageBuffer{
			messages: make([]*EdgeMessage, 0, config.MessageBufferSize),
			maxSize:  config.MessageBufferSize,
		},
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start starts the gateway
func (g *Gateway) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return fmt.Errorf("gateway already running")
	}
	g.running = true
	g.mu.Unlock()

	// Start connection monitor
	go g.connectionMonitor(ctx)

	// Start message processor
	go g.messageProcessor(ctx)

	// In a real implementation, this would start the actual network listener
	// (MQTT broker, gRPC server, WebSocket server, etc.)

	return nil
}

// Stop stops the gateway
func (g *Gateway) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.running {
		return nil
	}

	close(g.stopChan)
	g.running = false

	// Close all connections
	for nodeID := range g.connections {
		g.disconnectNode(nodeID)
	}

	return nil
}

// HandleNodeConnect handles a node connection
func (g *Gateway) HandleNodeConnect(ctx context.Context, nodeID string, protocol string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.connections) >= g.config.MaxConnections {
		return fmt.Errorf("maximum connections reached")
	}

	conn := &NodeConnection{
		NodeID:       nodeID,
		Protocol:     protocol,
		Connected:    true,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
	}

	g.connections[nodeID] = conn

	// Update node state
	g.manager.UpdateNodeState(ctx, nodeID, EdgeNodeStateOnline)

	return nil
}

// HandleNodeDisconnect handles a node disconnection
func (g *Gateway) HandleNodeDisconnect(ctx context.Context, nodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.disconnectNode(nodeID)
}

// disconnectNode disconnects a node (must be called with lock held)
func (g *Gateway) disconnectNode(nodeID string) error {
	conn, ok := g.connections[nodeID]
	if !ok {
		return fmt.Errorf("node %s not connected", nodeID)
	}

	conn.Connected = false
	delete(g.connections, nodeID)

	return nil
}

// HandleMessage handles an incoming message from an edge node
func (g *Gateway) HandleMessage(ctx context.Context, msg *EdgeMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Update connection activity
	g.mu.Lock()
	if conn, ok := g.connections[msg.NodeID]; ok {
		conn.LastActivity = time.Now()
		conn.MessageCount++
	}
	g.mu.Unlock()

	// Add to buffer
	g.messageBuffer.Add(msg)

	// Process immediately for certain message types
	switch msg.Type {
	case MessageTypeAlert:
		return g.processAlert(ctx, msg)
	case MessageTypeHeartbeat:
		return g.processHeartbeat(ctx, msg)
	}

	return nil
}

// processAlert processes an alert message
func (g *Gateway) processAlert(ctx context.Context, msg *EdgeMessage) error {
	var alert EdgeAlert
	if err := json.Unmarshal(msg.Payload, &alert); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	alert.NodeID = msg.NodeID
	if alert.Timestamp.IsZero() {
		alert.Timestamp = msg.Timestamp
	}

	// Forward to manager
	g.manager.emitAlert(&alert)

	return nil
}

// processHeartbeat processes a heartbeat message
func (g *Gateway) processHeartbeat(ctx context.Context, msg *EdgeMessage) error {
	var metrics NodeMetrics
	if err := json.Unmarshal(msg.Payload, &metrics); err != nil {
		// Heartbeat without metrics is ok
		return g.manager.RecordHeartbeat(ctx, msg.NodeID, nil)
	}

	return g.manager.RecordHeartbeat(ctx, msg.NodeID, &metrics)
}

// connectionMonitor monitors connections for timeouts
func (g *Gateway) connectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopChan:
			return
		case <-ticker.C:
			g.checkConnections(ctx)
		}
	}
}

// checkConnections checks for stale connections
func (g *Gateway) checkConnections(ctx context.Context) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	timeout := g.config.ConnectionTimeout

	for nodeID, conn := range g.connections {
		if now.Sub(conn.LastActivity) > timeout {
			conn.Connected = false
			delete(g.connections, nodeID)

			// Update node state
			g.manager.UpdateNodeState(ctx, nodeID, EdgeNodeStateOffline)
		}
	}
}

// messageProcessor processes buffered messages
func (g *Gateway) messageProcessor(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopChan:
			return
		case <-ticker.C:
			g.processBatch(ctx)
		}
	}
}

// processBatch processes a batch of messages
func (g *Gateway) processBatch(ctx context.Context) {
	messages := g.messageBuffer.Drain(100)

	for _, msg := range messages {
		// Forward to manager
		g.manager.ProcessMessage(ctx, msg)
	}
}

// SendToNode sends a message to an edge node
func (g *Gateway) SendToNode(ctx context.Context, nodeID string, msg *EdgeMessage) error {
	g.mu.RLock()
	conn, ok := g.connections[nodeID]
	g.mu.RUnlock()

	if !ok || !conn.Connected {
		return fmt.Errorf("node %s not connected", nodeID)
	}

	// In a real implementation, this would send via the appropriate protocol
	// For now, we just log success

	return nil
}

// SendCommand sends a command to an edge node
func (g *Gateway) SendCommand(ctx context.Context, cmd *EdgeCommand) error {
	g.mu.RLock()
	conn, ok := g.connections[cmd.NodeID]
	g.mu.RUnlock()

	if !ok || !conn.Connected {
		return fmt.Errorf("node %s not connected", cmd.NodeID)
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	msg := &EdgeMessage{
		ID:        uuid.New().String(),
		NodeID:    cmd.NodeID,
		DeviceID:  cmd.DeviceID,
		Type:      MessageTypeCommand,
		Topic:     "commands",
		Payload:   payload,
		Timestamp: time.Now(),
		Priority:  cmd.Priority,
	}

	return g.SendToNode(ctx, cmd.NodeID, msg)
}

// BroadcastMessage broadcasts a message to all connected nodes
func (g *Gateway) BroadcastMessage(ctx context.Context, msg *EdgeMessage) error {
	g.mu.RLock()
	nodeIDs := make([]string, 0, len(g.connections))
	for nodeID, conn := range g.connections {
		if conn.Connected {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	g.mu.RUnlock()

	var lastErr error
	for _, nodeID := range nodeIDs {
		msgCopy := *msg
		msgCopy.NodeID = nodeID
		if err := g.SendToNode(ctx, nodeID, &msgCopy); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetConnectionStats returns connection statistics
func (g *Gateway) GetConnectionStats() *ConnectionStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := &ConnectionStats{
		TotalConnections: len(g.connections),
		ActiveConnections: 0,
		MessageCount:     0,
		ErrorCount:       0,
		ConnectionsByProtocol: make(map[string]int),
	}

	for _, conn := range g.connections {
		if conn.Connected {
			stats.ActiveConnections++
		}
		stats.MessageCount += conn.MessageCount
		stats.ErrorCount += conn.ErrorCount
		stats.ConnectionsByProtocol[conn.Protocol]++
	}

	stats.BufferUsage = float64(g.messageBuffer.Len()) / float64(g.messageBuffer.maxSize) * 100

	return stats
}

// ConnectionStats represents gateway connection statistics
type ConnectionStats struct {
	TotalConnections      int            `json:"total_connections"`
	ActiveConnections     int            `json:"active_connections"`
	MessageCount          int64          `json:"message_count"`
	ErrorCount            int64          `json:"error_count"`
	BufferUsage           float64        `json:"buffer_usage"`
	ConnectionsByProtocol map[string]int `json:"connections_by_protocol"`
}

// MessageBuffer methods

// Add adds a message to the buffer
func (b *MessageBuffer) Add(msg *EdgeMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.messages) >= b.maxSize {
		// Drop oldest message
		b.messages = b.messages[1:]
	}

	b.messages = append(b.messages, msg)
}

// Drain drains up to n messages from the buffer
func (b *MessageBuffer) Drain(n int) []*EdgeMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.messages) == 0 {
		return nil
	}

	count := n
	if count > len(b.messages) {
		count = len(b.messages)
	}

	messages := make([]*EdgeMessage, count)
	copy(messages, b.messages[:count])
	b.messages = b.messages[count:]

	return messages
}

// Len returns the number of buffered messages
func (b *MessageBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.messages)
}

// Clear clears the buffer
func (b *MessageBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = b.messages[:0]
}
