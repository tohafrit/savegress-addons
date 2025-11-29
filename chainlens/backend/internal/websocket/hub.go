// Package websocket provides real-time updates via WebSocket connections
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType constants for WebSocket messages
const (
	TypeSubscribe     = "subscribe"
	TypeUnsubscribe   = "unsubscribe"
	TypeNewBlock      = "new_block"
	TypeNewTx         = "new_tx"
	TypePendingTx     = "pending_tx"
	TypeTokenTransfer = "token_transfer"
	TypeNFTTransfer   = "nft_transfer"
	TypeGasPrice      = "gas_price"
	TypeAddressUpdate = "address_update"
	TypeError         = "error"
	TypePong          = "pong"
)

// SubscriptionType constants
const (
	SubNewBlocks      = "blocks"
	SubNewTxs         = "transactions"
	SubPendingTxs     = "pending"
	SubTokenTransfers = "token_transfers"
	SubNFTTransfers   = "nft_transfers"
	SubGasPrice       = "gas_price"
	SubAddress        = "address"
)

// Message represents a WebSocket message
type Message struct {
	Type      string          `json:"type"`
	Network   string          `json:"network,omitempty"`
	Channel   string          `json:"channel,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Error     string          `json:"error,omitempty"`
}

// Subscription represents a client subscription
type Subscription struct {
	Type    string
	Network string
	Filter  string // address, token address, etc.
}

// Client represents a connected WebSocket client
type Client struct {
	ID            string
	conn          *websocket.Conn
	hub           *Hub
	send          chan []byte
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool // channel -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
	stopCh     chan struct{}
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	Channel string
	Message *Message
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		stopCh:     make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case <-h.stopCh:
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.removeClient(client)
		case msg := <-h.broadcast:
			h.broadcastToChannel(msg)
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	close(h.stopCh)
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		close(client.send)
		client.conn.Close()
	}
}

// removeClient removes a client from all channels
func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from all channels
		for channel := range client.subscriptions {
			if clients, ok := h.channels[channel]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.channels, channel)
				}
			}
		}
	}
}

// broadcastToChannel sends a message to all clients subscribed to a channel
func (h *Hub) broadcastToChannel(msg *BroadcastMessage) {
	h.mu.RLock()
	clients, ok := h.channels[msg.Channel]
	h.mu.RUnlock()

	if !ok {
		return
	}

	data, err := json.Marshal(msg.Message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- data:
		default:
			// Client buffer full, skip
		}
	}
}

// Subscribe subscribes a client to a channel
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true

	client.mu.Lock()
	client.subscriptions[channel] = true
	client.mu.Unlock()
}

// Unsubscribe unsubscribes a client from a channel
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	client.mu.Lock()
	delete(client.subscriptions, channel)
	client.mu.Unlock()
}

// Broadcast sends a message to a channel
func (h *Hub) Broadcast(channel string, msg *Message) {
	msg.Timestamp = time.Now().UTC()
	h.broadcast <- &BroadcastMessage{
		Channel: channel,
		Message: msg,
	}
}

// BroadcastNewBlock broadcasts a new block to subscribers
func (h *Hub) BroadcastNewBlock(network string, blockData interface{}) {
	data, _ := json.Marshal(blockData)
	h.Broadcast(channelKey(SubNewBlocks, network), &Message{
		Type:    TypeNewBlock,
		Network: network,
		Channel: SubNewBlocks,
		Data:    data,
	})
}

// BroadcastNewTransaction broadcasts a new transaction
func (h *Hub) BroadcastNewTransaction(network string, txData interface{}) {
	data, _ := json.Marshal(txData)
	h.Broadcast(channelKey(SubNewTxs, network), &Message{
		Type:    TypeNewTx,
		Network: network,
		Channel: SubNewTxs,
		Data:    data,
	})
}

// BroadcastTokenTransfer broadcasts a token transfer
func (h *Hub) BroadcastTokenTransfer(network, tokenAddress string, transferData interface{}) {
	data, _ := json.Marshal(transferData)

	// Broadcast to general token transfers channel
	h.Broadcast(channelKey(SubTokenTransfers, network), &Message{
		Type:    TypeTokenTransfer,
		Network: network,
		Channel: SubTokenTransfers,
		Data:    data,
	})

	// Broadcast to specific token channel
	h.Broadcast(channelKey(SubTokenTransfers, network, tokenAddress), &Message{
		Type:    TypeTokenTransfer,
		Network: network,
		Channel: SubTokenTransfers,
		Data:    data,
	})
}

// BroadcastNFTTransfer broadcasts an NFT transfer
func (h *Hub) BroadcastNFTTransfer(network, collectionAddress string, transferData interface{}) {
	data, _ := json.Marshal(transferData)

	h.Broadcast(channelKey(SubNFTTransfers, network), &Message{
		Type:    TypeNFTTransfer,
		Network: network,
		Channel: SubNFTTransfers,
		Data:    data,
	})

	h.Broadcast(channelKey(SubNFTTransfers, network, collectionAddress), &Message{
		Type:    TypeNFTTransfer,
		Network: network,
		Channel: SubNFTTransfers,
		Data:    data,
	})
}

// BroadcastGasPrice broadcasts updated gas prices
func (h *Hub) BroadcastGasPrice(network string, gasPriceData interface{}) {
	data, _ := json.Marshal(gasPriceData)
	h.Broadcast(channelKey(SubGasPrice, network), &Message{
		Type:    TypeGasPrice,
		Network: network,
		Channel: SubGasPrice,
		Data:    data,
	})
}

// BroadcastAddressUpdate broadcasts address activity
func (h *Hub) BroadcastAddressUpdate(network, address string, updateData interface{}) {
	data, _ := json.Marshal(updateData)
	h.Broadcast(channelKey(SubAddress, network, address), &Message{
		Type:    TypeAddressUpdate,
		Network: network,
		Channel: SubAddress,
		Data:    data,
	})
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channelStats := make(map[string]int)
	for channel, clients := range h.channels {
		channelStats[channel] = len(clients)
	}

	return map[string]interface{}{
		"total_clients":   len(h.clients),
		"total_channels":  len(h.channels),
		"channel_clients": channelStats,
	}
}

// NewClient creates a new client
func NewClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	return &Client{
		ID:            id,
		conn:          conn,
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			c.handleMessage(message)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(data []byte) {
	var msg struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		Network string `json:"network"`
		Filter  string `json:"filter"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("invalid message format")
		return
	}

	switch msg.Type {
	case TypeSubscribe:
		channel := channelKey(msg.Channel, msg.Network, msg.Filter)
		c.hub.Subscribe(c, channel)
		c.sendAck("subscribed", channel)

	case TypeUnsubscribe:
		channel := channelKey(msg.Channel, msg.Network, msg.Filter)
		c.hub.Unsubscribe(c, channel)
		c.sendAck("unsubscribed", channel)

	case "ping":
		c.sendPong()

	default:
		c.sendError("unknown message type")
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(errMsg string) {
	msg := &Message{
		Type:      TypeError,
		Error:     errMsg,
		Timestamp: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendAck sends an acknowledgment message
func (c *Client) sendAck(action, channel string) {
	msg := map[string]interface{}{
		"type":      "ack",
		"action":    action,
		"channel":   channel,
		"timestamp": time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendPong sends a pong response
func (c *Client) sendPong() {
	msg := &Message{
		Type:      TypePong,
		Timestamp: time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// channelKey generates a channel key
func channelKey(parts ...string) string {
	key := ""
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 && key != "" {
			key += ":"
		}
		key += part
	}
	return key
}
