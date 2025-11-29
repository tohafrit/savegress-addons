package cache

import (
	"testing"
	"time"
)

func TestTTLConstants(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		expected time.Duration
	}{
		{"TTLBlock", TTLBlock, 5 * time.Minute},
		{"TTLTransaction", TTLTransaction, 10 * time.Minute},
		{"TTLAddress", TTLAddress, 2 * time.Minute},
		{"TTLToken", TTLToken, 5 * time.Minute},
		{"TTLGasPrice", TTLGasPrice, 15 * time.Second},
		{"TTLNetworkOverview", TTLNetworkOverview, 30 * time.Second},
		{"TTLChart", TTLChart, 1 * time.Minute},
		{"TTLSearch", TTLSearch, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttl != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.ttl)
			}
		})
	}
}

func TestCacheDisabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}

	cache, err := New(cfg)
	if err != nil {
		t.Fatalf("Expected no error for disabled cache, got %v", err)
	}

	if cache.IsEnabled() {
		t.Error("Expected cache to be disabled")
	}
}

func TestCacheKey(t *testing.T) {
	cache := &Cache{
		keyPrefix: "chainlens",
		enabled:   false,
	}

	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"block", "ethereum", "123"}, "chainlens:block:ethereum:123"},
		{[]string{"tx", "polygon", "0xabc"}, "chainlens:tx:polygon:0xabc"},
		{[]string{"gas", "ethereum"}, "chainlens:gas:ethereum"},
	}

	for _, tt := range tests {
		result := cache.key(tt.parts...)
		if result != tt.expected {
			t.Errorf("key(%v) = %s, expected %s", tt.parts, result, tt.expected)
		}
	}
}

func TestCacheKeyWithDifferentPrefix(t *testing.T) {
	cache := &Cache{
		keyPrefix: "myapp",
		enabled:   false,
	}

	result := cache.key("test", "key")
	expected := "myapp:test:key"

	if result != expected {
		t.Errorf("key() = %s, expected %s", result, expected)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		Host:    "localhost",
		Port:    6379,
		DB:      0,
		Enabled: false,
	}

	if cfg.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", cfg.Host)
	}

	if cfg.Port != 6379 {
		t.Errorf("Expected port 6379, got %d", cfg.Port)
	}
}

func TestRateLimitKey(t *testing.T) {
	cache := &Cache{
		keyPrefix: "chainlens",
		enabled:   false,
	}

	key := cache.RateLimitKey("user123", time.Minute)
	if key == "" {
		t.Error("Expected non-empty rate limit key")
	}

	// Key should contain identifier
	if len(key) < 10 {
		t.Error("Rate limit key seems too short")
	}
}

func TestDisabledCacheOperations(t *testing.T) {
	cache := &Cache{
		enabled: false,
	}

	// All operations should succeed silently when disabled
	if err := cache.Close(); err != nil {
		t.Errorf("Close() should not error when disabled: %v", err)
	}

	exceeded, remaining, err := cache.CheckRateLimit(nil, "test", 100, time.Minute)
	if err != nil {
		t.Errorf("CheckRateLimit() should not error when disabled: %v", err)
	}
	if exceeded {
		t.Error("Rate limit should not be exceeded when disabled")
	}
	if remaining != 100 {
		t.Errorf("Expected remaining 100, got %d", remaining)
	}
}

func TestCacheStatsDisabled(t *testing.T) {
	cache := &Cache{
		enabled: false,
	}

	stats, err := cache.Stats(nil)
	if err != nil {
		t.Errorf("Stats() should not error when disabled: %v", err)
	}

	enabled, ok := stats["enabled"].(bool)
	if !ok || enabled {
		t.Error("Expected enabled=false in stats")
	}
}

func TestNewCacheWithEmptyPrefix(t *testing.T) {
	cfg := &Config{
		Host:      "localhost",
		Port:      6379,
		KeyPrefix: "",
		Enabled:   false,
	}

	cache, err := New(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// When disabled, prefix doesn't matter but should handle empty
	if cache.keyPrefix != "" && cache.enabled {
		t.Error("Prefix should be empty or cache should be disabled")
	}
}
