package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type TestConfig struct {
	Workers   int    `json:"workers"`
	QueueSize int    `json:"queue_size"`
	Name      string `json:"name"`
}

func TestNewConfigManager(t *testing.T) {
	cfg := TestConfig{Workers: 4, QueueSize: 100}
	cm := NewConfigManager(cfg, "/tmp/config.json")

	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}

	got := cm.Get().(TestConfig)
	if got.Workers != cfg.Workers {
		t.Errorf("Workers = %d, want %d", got.Workers, cfg.Workers)
	}
}

func TestConfigManager_Get(t *testing.T) {
	cfg := TestConfig{Workers: 8, QueueSize: 200}
	cm := NewConfigManager(cfg, "/tmp/config.json")

	got := cm.Get().(TestConfig)
	if got != cfg {
		t.Errorf("Get() = %+v, want %+v", got, cfg)
	}
}

func TestConfigManager_Update(t *testing.T) {
	cfg := TestConfig{Workers: 4, QueueSize: 100}
	cm := NewConfigManager(cfg, "/tmp/config.json")

	newCfg := TestConfig{Workers: 8, QueueSize: 200}
	err := cm.Update(newCfg)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got := cm.Get().(TestConfig)
	if got.Workers != newCfg.Workers {
		t.Errorf("Workers = %d, want %d", got.Workers, newCfg.Workers)
	}
}

func TestConfigManager_SetOnChange(t *testing.T) {
	cfg := TestConfig{Workers: 4, QueueSize: 100}
	cm := NewConfigManager(cfg, "/tmp/config.json")

	var called bool
	var mu sync.Mutex
	cm.SetOnChange(func(old, new interface{}) {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	newCfg := TestConfig{Workers: 8, QueueSize: 200}
	cm.Update(newCfg)

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !called {
		t.Error("OnChange callback not called")
	}
	mu.Unlock()
}

func TestConfigManager_SaveToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := TestConfig{Workers: 4, QueueSize: 100, Name: "test"}
	cm := NewConfigManager(cfg, path)

	err := cm.SaveToFile()
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and contains correct data
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var loaded TestConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if loaded.Workers != cfg.Workers {
		t.Errorf("Saved Workers = %d, want %d", loaded.Workers, cfg.Workers)
	}
}

func TestConfigManager_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := TestConfig{Workers: 4, QueueSize: 100, Name: "test"}
	data, _ := json.Marshal(cfg)
	os.WriteFile(path, data, 0644)

	// Need to use a pointer to TestConfig so atomic.Value stores same type
	initial := &TestConfig{}
	cm := NewConfigManager(initial, path)

	loaded := &TestConfig{}
	err := cm.LoadFromFile(loaded)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if loaded.Workers != cfg.Workers {
		t.Errorf("Loaded Workers = %d, want %d", loaded.Workers, cfg.Workers)
	}
}

func TestConfigManager_LoadFromFile_NotExists(t *testing.T) {
	cm := NewConfigManager(TestConfig{}, "/nonexistent/path/config.json")

	var loaded TestConfig
	err := cm.LoadFromFile(&loaded)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestConfigManager_LoadFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	os.WriteFile(path, []byte("invalid json"), 0644)

	cm := NewConfigManager(TestConfig{}, path)

	var loaded TestConfig
	err := cm.LoadFromFile(&loaded)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// FeatureFlags tests

func TestNewFeatureFlags(t *testing.T) {
	ff := NewFeatureFlags()
	if ff == nil {
		t.Fatal("NewFeatureFlags returned nil")
	}
}

func TestFeatureFlags_IsEnabled(t *testing.T) {
	ff := NewFeatureFlags()

	// Not set
	if ff.IsEnabled("feature1") {
		t.Error("Unset feature should not be enabled")
	}

	// Enable
	ff.Enable("feature1")
	if !ff.IsEnabled("feature1") {
		t.Error("Enabled feature should be enabled")
	}

	// Disable
	ff.Disable("feature1")
	if ff.IsEnabled("feature1") {
		t.Error("Disabled feature should not be enabled")
	}
}

func TestFeatureFlags_Set(t *testing.T) {
	ff := NewFeatureFlags()

	ff.Set("feature1", true)
	if !ff.IsEnabled("feature1") {
		t.Error("Feature should be enabled after Set(true)")
	}

	ff.Set("feature1", false)
	if ff.IsEnabled("feature1") {
		t.Error("Feature should be disabled after Set(false)")
	}
}

func TestFeatureFlags_GetAll(t *testing.T) {
	ff := NewFeatureFlags()

	ff.Enable("feature1")
	ff.Enable("feature2")
	ff.Disable("feature3")

	all := ff.GetAll()

	if len(all) != 3 {
		t.Errorf("GetAll() len = %d, want 3", len(all))
	}

	if !all["feature1"] {
		t.Error("feature1 should be enabled")
	}
	if !all["feature2"] {
		t.Error("feature2 should be enabled")
	}
	if all["feature3"] {
		t.Error("feature3 should be disabled")
	}
}

func TestFeatureFlags_Concurrent(t *testing.T) {
	ff := NewFeatureFlags()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			feature := "feature"
			ff.Enable(feature)
			ff.IsEnabled(feature)
			ff.Disable(feature)
			ff.GetAll()
		}(i)
	}

	wg.Wait()
}

// Benchmarks

func BenchmarkConfigManager_Get(b *testing.B) {
	cfg := TestConfig{Workers: 4, QueueSize: 100}
	cm := NewConfigManager(cfg, "/tmp/config.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.Get()
	}
}

func BenchmarkFeatureFlags_IsEnabled(b *testing.B) {
	ff := NewFeatureFlags()
	ff.Enable("test_feature")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ff.IsEnabled("test_feature")
	}
}
