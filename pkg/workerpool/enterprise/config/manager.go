package config

import (
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"
)

// ConfigManager manages hot-reloadable configuration
type ConfigManager struct {
	current   atomic.Value // stores current config
	path      string
	onChange  func(old, new interface{})
	mu        sync.RWMutex
}

// NewConfigManager creates a new config manager
func NewConfigManager(initialConfig interface{}, path string) *ConfigManager {
	cm := &ConfigManager{
		path: path,
	}
	cm.current.Store(initialConfig)
	return cm
}

// SetOnChange sets a callback for config changes
func (cm *ConfigManager) SetOnChange(fn func(old, new interface{})) {
	cm.onChange = fn
}

// Get returns the current configuration
func (cm *ConfigManager) Get() interface{} {
	return cm.current.Load()
}

// Update updates the configuration
func (cm *ConfigManager) Update(newConfig interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldConfig := cm.current.Load()
	cm.current.Store(newConfig)

	// Trigger callback
	if cm.onChange != nil {
		go cm.onChange(oldConfig, newConfig)
	}

	return nil
}

// LoadFromFile loads configuration from a file
func (cm *ConfigManager) LoadFromFile(config interface{}) error {
	data, err := os.ReadFile(cm.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return err
	}

	return cm.Update(config)
}

// SaveToFile saves current configuration to file
func (cm *ConfigManager) SaveToFile() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config := cm.current.Load()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.path, data, 0644)
}

// FeatureFlags controls feature toggles
type FeatureFlags struct {
	flags sync.Map // map[string]bool
}

// NewFeatureFlags creates a new feature flags manager
func NewFeatureFlags() *FeatureFlags {
	return &FeatureFlags{}
}

// IsEnabled checks if a feature is enabled
func (ff *FeatureFlags) IsEnabled(feature string) bool {
	val, ok := ff.flags.Load(feature)
	if !ok {
		return false
	}
	return val.(bool)
}

// Enable enables a feature
func (ff *FeatureFlags) Enable(feature string) {
	ff.flags.Store(feature, true)
}

// Disable disables a feature
func (ff *FeatureFlags) Disable(feature string) {
	ff.flags.Store(feature, false)
}

// Set sets a feature flag
func (ff *FeatureFlags) Set(feature string, enabled bool) {
	ff.flags.Store(feature, enabled)
}

// GetAll returns all feature flags
func (ff *FeatureFlags) GetAll() map[string]bool {
	result := make(map[string]bool)
	ff.flags.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(bool)
		return true
	})
	return result
}
