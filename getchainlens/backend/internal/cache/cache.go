// Package cache provides Redis-based caching for hot data
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Default TTLs for different data types
const (
	TTLBlock          = 5 * time.Minute
	TTLTransaction    = 10 * time.Minute
	TTLAddress        = 2 * time.Minute
	TTLToken          = 5 * time.Minute
	TTLGasPrice       = 15 * time.Second
	TTLNetworkOverview = 30 * time.Second
	TTLChart          = 1 * time.Minute
	TTLSearch         = 5 * time.Minute
)

// Cache provides Redis-based caching operations
type Cache struct {
	client    *redis.Client
	keyPrefix string
	enabled   bool
}

// Config holds cache configuration
type Config struct {
	Host      string
	Port      int
	Password  string
	DB        int
	KeyPrefix string
	Enabled   bool
}

// New creates a new Cache instance
func New(cfg *Config) (*Cache, error) {
	if !cfg.Enabled {
		return &Cache{enabled: false}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "chainlens"
	}

	return &Cache{
		client:    client,
		keyPrefix: prefix,
		enabled:   true,
	}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns whether caching is enabled
func (c *Cache) IsEnabled() bool {
	return c.enabled
}

// key generates a cache key with prefix
func (c *Cache) key(parts ...string) string {
	key := c.keyPrefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	if !c.enabled {
		return redis.Nil
	}

	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set stores a value in cache with TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete removes a value from cache
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if !c.enabled || len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.key(k)
	}

	return c.client.Del(ctx, fullKeys...).Err()
}

// DeletePattern removes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	if !c.enabled {
		return nil
	}

	iter := c.client.Scan(ctx, 0, c.key(pattern), 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if !c.enabled {
		return false, nil
	}

	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

// TTL returns the remaining TTL for a key
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !c.enabled {
		return 0, nil
	}

	return c.client.TTL(ctx, c.key(key)).Result()
}

// Increment increments an integer value
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	if !c.enabled {
		return 0, nil
	}

	return c.client.Incr(ctx, c.key(key)).Result()
}

// IncrementBy increments by a specific amount
func (c *Cache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	if !c.enabled {
		return 0, nil
	}

	return c.client.IncrBy(ctx, c.key(key), value).Result()
}

// SetNX sets a value only if it doesn't exist (for distributed locking)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if !c.enabled {
		return true, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	return c.client.SetNX(ctx, c.key(key), data, ttl).Result()
}

// Block caching

// GetBlock retrieves a block from cache
func (c *Cache) GetBlock(ctx context.Context, network string, blockNumber int64) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("block", network, fmt.Sprintf("%d", blockNumber))).Bytes()
}

// SetBlock stores a block in cache
func (c *Cache) SetBlock(ctx context.Context, network string, blockNumber int64, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("block", network, fmt.Sprintf("%d", blockNumber)), data, TTLBlock).Err()
}

// GetBlockByHash retrieves a block by hash
func (c *Cache) GetBlockByHash(ctx context.Context, network, hash string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("block", network, "hash", hash)).Bytes()
}

// SetBlockByHash stores a block by hash
func (c *Cache) SetBlockByHash(ctx context.Context, network, hash string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("block", network, "hash", hash), data, TTLBlock).Err()
}

// Transaction caching

// GetTransaction retrieves a transaction from cache
func (c *Cache) GetTransaction(ctx context.Context, network, hash string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("tx", network, hash)).Bytes()
}

// SetTransaction stores a transaction in cache
func (c *Cache) SetTransaction(ctx context.Context, network, hash string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("tx", network, hash), data, TTLTransaction).Err()
}

// Address caching

// GetAddress retrieves an address from cache
func (c *Cache) GetAddress(ctx context.Context, network, address string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("addr", network, address)).Bytes()
}

// SetAddress stores an address in cache
func (c *Cache) SetAddress(ctx context.Context, network, address string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("addr", network, address), data, TTLAddress).Err()
}

// InvalidateAddress invalidates address cache
func (c *Cache) InvalidateAddress(ctx context.Context, network, address string) error {
	return c.Delete(ctx, "addr:"+network+":"+address)
}

// Gas price caching

// GetGasPrice retrieves current gas price from cache
func (c *Cache) GetGasPrice(ctx context.Context, network string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("gas", network)).Bytes()
}

// SetGasPrice stores current gas price in cache
func (c *Cache) SetGasPrice(ctx context.Context, network string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("gas", network), data, TTLGasPrice).Err()
}

// Network overview caching

// GetNetworkOverview retrieves network overview from cache
func (c *Cache) GetNetworkOverview(ctx context.Context, network string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("overview", network)).Bytes()
}

// SetNetworkOverview stores network overview in cache
func (c *Cache) SetNetworkOverview(ctx context.Context, network string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("overview", network), data, TTLNetworkOverview).Err()
}

// Token caching

// GetToken retrieves token info from cache
func (c *Cache) GetToken(ctx context.Context, network, address string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("token", network, address)).Bytes()
}

// SetToken stores token info in cache
func (c *Cache) SetToken(ctx context.Context, network, address string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("token", network, address), data, TTLToken).Err()
}

// Chart caching

// GetChart retrieves chart data from cache
func (c *Cache) GetChart(ctx context.Context, network, chartType, period string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("chart", network, chartType, period)).Bytes()
}

// SetChart stores chart data in cache
func (c *Cache) SetChart(ctx context.Context, network, chartType, period string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("chart", network, chartType, period), data, TTLChart).Err()
}

// Search caching

// GetSearchResult retrieves search result from cache
func (c *Cache) GetSearchResult(ctx context.Context, network, query string) ([]byte, error) {
	if !c.enabled {
		return nil, redis.Nil
	}

	return c.client.Get(ctx, c.key("search", network, query)).Bytes()
}

// SetSearchResult stores search result in cache
func (c *Cache) SetSearchResult(ctx context.Context, network, query string, data []byte) error {
	if !c.enabled {
		return nil
	}

	return c.client.Set(ctx, c.key("search", network, query), data, TTLSearch).Err()
}

// Rate limiting helpers

// RateLimitKey returns the rate limit key for an identifier
func (c *Cache) RateLimitKey(identifier string, window time.Duration) string {
	windowStart := time.Now().Truncate(window).Unix()
	return fmt.Sprintf("ratelimit:%s:%d", identifier, windowStart)
}

// CheckRateLimit checks if rate limit is exceeded
func (c *Cache) CheckRateLimit(ctx context.Context, identifier string, limit int64, window time.Duration) (bool, int64, error) {
	if !c.enabled {
		return false, limit, nil
	}

	key := c.key(c.RateLimitKey(identifier, window))

	pipe := c.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := incrCmd.Val()
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return count > limit, remaining, nil
}

// Stats returns cache statistics
func (c *Cache) Stats(ctx context.Context) (map[string]interface{}, error) {
	if !c.enabled {
		return map[string]interface{}{"enabled": false}, nil
	}

	info, err := c.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"enabled": true,
		"keys":    dbSize,
		"info":    info,
	}, nil
}
