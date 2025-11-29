package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"TypeSubscribe", TypeSubscribe, "subscribe"},
		{"TypeUnsubscribe", TypeUnsubscribe, "unsubscribe"},
		{"TypeNewBlock", TypeNewBlock, "new_block"},
		{"TypeNewTx", TypeNewTx, "new_tx"},
		{"TypePendingTx", TypePendingTx, "pending_tx"},
		{"TypeTokenTransfer", TypeTokenTransfer, "token_transfer"},
		{"TypeNFTTransfer", TypeNFTTransfer, "nft_transfer"},
		{"TypeGasPrice", TypeGasPrice, "gas_price"},
		{"TypeAddressUpdate", TypeAddressUpdate, "address_update"},
		{"TypeError", TypeError, "error"},
		{"TypePong", TypePong, "pong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestSubscriptionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"SubNewBlocks", SubNewBlocks, "blocks"},
		{"SubNewTxs", SubNewTxs, "transactions"},
		{"SubPendingTxs", SubPendingTxs, "pending"},
		{"SubTokenTransfers", SubTokenTransfers, "token_transfers"},
		{"SubNFTTransfers", SubNFTTransfers, "nft_transfers"},
		{"SubGasPrice", SubGasPrice, "gas_price"},
		{"SubAddress", SubAddress, "address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("Expected non-nil hub")
	}

	if hub.clients == nil {
		t.Error("Expected initialized clients map")
	}

	if hub.channels == nil {
		t.Error("Expected initialized channels map")
	}

	if hub.register == nil {
		t.Error("Expected initialized register channel")
	}

	if hub.unregister == nil {
		t.Error("Expected initialized unregister channel")
	}

	if hub.broadcast == nil {
		t.Error("Expected initialized broadcast channel")
	}

	if hub.stopCh == nil {
		t.Error("Expected initialized stopCh channel")
	}
}

func TestChannelKey(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"blocks", "ethereum"}, "blocks:ethereum"},
		{[]string{"transactions", "polygon"}, "transactions:polygon"},
		{[]string{"address", "ethereum", "0x123"}, "address:ethereum:0x123"},
		{[]string{"gas_price", "ethereum", ""}, "gas_price:ethereum"},
		{[]string{""}, ""},
		{[]string{"single"}, "single"},
	}

	for _, tt := range tests {
		result := channelKey(tt.parts...)
		if result != tt.expected {
			t.Errorf("channelKey(%v) = %s, expected %s", tt.parts, result, tt.expected)
		}
	}
}

func TestMessageSerialization(t *testing.T) {
	msg := &Message{
		Type:      TypeNewBlock,
		Network:   "ethereum",
		Channel:   SubNewBlocks,
		Data:      json.RawMessage(`{"number": 18000000}`),
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, decoded.Type)
	}

	if decoded.Network != msg.Network {
		t.Errorf("Expected network %s, got %s", msg.Network, decoded.Network)
	}

	if decoded.Channel != msg.Channel {
		t.Errorf("Expected channel %s, got %s", msg.Channel, decoded.Channel)
	}
}

func TestErrorMessage(t *testing.T) {
	msg := &Message{
		Type:      TypeError,
		Error:     "invalid subscription",
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal error message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error message: %v", err)
	}

	if decoded.Error != "invalid subscription" {
		t.Errorf("Expected error 'invalid subscription', got %s", decoded.Error)
	}
}

func TestSubscription(t *testing.T) {
	sub := Subscription{
		Type:    SubNewBlocks,
		Network: "ethereum",
		Filter:  "",
	}

	if sub.Type != SubNewBlocks {
		t.Errorf("Expected type %s, got %s", SubNewBlocks, sub.Type)
	}

	if sub.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", sub.Network)
	}
}

func TestBroadcastMessage(t *testing.T) {
	msg := &Message{
		Type:      TypeGasPrice,
		Network:   "ethereum",
		Data:      json.RawMessage(`{"slow": 20, "fast": 50}`),
		Timestamp: time.Now().UTC(),
	}

	broadcast := &BroadcastMessage{
		Channel: "gas_price:ethereum",
		Message: msg,
	}

	if broadcast.Channel != "gas_price:ethereum" {
		t.Errorf("Expected channel 'gas_price:ethereum', got %s", broadcast.Channel)
	}

	if broadcast.Message.Type != TypeGasPrice {
		t.Errorf("Expected type %s, got %s", TypeGasPrice, broadcast.Message.Type)
	}
}

func TestHubGetStats(t *testing.T) {
	hub := NewHub()

	stats := hub.GetStats()

	totalClients, ok := stats["total_clients"].(int)
	if !ok {
		t.Error("Expected total_clients in stats")
	}
	if totalClients != 0 {
		t.Errorf("Expected 0 clients, got %d", totalClients)
	}

	totalChannels, ok := stats["total_channels"].(int)
	if !ok {
		t.Error("Expected total_channels in stats")
	}
	if totalChannels != 0 {
		t.Errorf("Expected 0 channels, got %d", totalChannels)
	}
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	hub := NewHub()

	// Create a mock client
	client := &Client{
		ID:            "test-client",
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	// Add client to hub
	hub.clients[client] = true

	// Subscribe to channel
	channel := "blocks:ethereum"
	hub.Subscribe(client, channel)

	// Check subscription
	if _, ok := hub.channels[channel]; !ok {
		t.Error("Expected channel to exist")
	}

	if !client.subscriptions[channel] {
		t.Error("Expected client to be subscribed")
	}

	// Unsubscribe
	hub.Unsubscribe(client, channel)

	if client.subscriptions[channel] {
		t.Error("Expected client to be unsubscribed")
	}

	// Channel should be removed if no subscribers
	if _, ok := hub.channels[channel]; ok {
		t.Error("Expected empty channel to be removed")
	}
}

func TestClientSubscriptions(t *testing.T) {
	client := &Client{
		ID:            "test-client",
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	client.subscriptions["blocks:ethereum"] = true
	client.subscriptions["transactions:ethereum"] = true

	if len(client.subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(client.subscriptions))
	}

	if !client.subscriptions["blocks:ethereum"] {
		t.Error("Expected blocks:ethereum subscription")
	}
}
