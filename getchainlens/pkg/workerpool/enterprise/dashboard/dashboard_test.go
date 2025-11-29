package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock metrics provider
type mockMetricsProvider struct {
	metrics interface{}
}

func (m *mockMetricsProvider) GetMetrics() interface{} {
	return m.metrics
}

type testMetrics struct {
	WorkersActive  int `json:"WorkersActive"`
	TasksCompleted int `json:"TasksCompleted"`
}

func TestNewDashboard(t *testing.T) {
	provider := &mockMetricsProvider{
		metrics: testMetrics{WorkersActive: 4, TasksCompleted: 100},
	}

	d := NewDashboard(provider)
	if d == nil {
		t.Fatal("NewDashboard returned nil")
	}
}

func TestNewDashboard_NilProvider(t *testing.T) {
	d := NewDashboard(nil)
	if d == nil {
		t.Fatal("NewDashboard returned nil")
	}
}

func TestDashboard_handleMetrics(t *testing.T) {
	provider := &mockMetricsProvider{
		metrics: testMetrics{WorkersActive: 4, TasksCompleted: 100},
	}

	d := NewDashboard(provider)

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	d.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
	}

	var metrics testMetrics
	if err := json.NewDecoder(w.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if metrics.WorkersActive != 4 {
		t.Errorf("WorkersActive = %d, want 4", metrics.WorkersActive)
	}
	if metrics.TasksCompleted != 100 {
		t.Errorf("TasksCompleted = %d, want 100", metrics.TasksCompleted)
	}
}

func TestDashboard_handleMetrics_NilProvider(t *testing.T) {
	d := NewDashboard(nil)

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	d.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "no metrics provider" {
		t.Errorf("Expected 'no metrics provider' error, got: %v", response)
	}
}

func TestDashboard_serveDashboard(t *testing.T) {
	d := NewDashboard(nil)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	d.serveDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Content-Type = %s, want text/html", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Response body should not be empty")
	}

	// Check HTML contains expected elements
	expectedStrings := []string{
		"Worker Pool Dashboard",
		"Active Workers",
		"Queued Tasks",
		"Completed Tasks",
		"Rejected Tasks",
	}

	for _, expected := range expectedStrings {
		if !containsString(body, expected) {
			t.Errorf("HTML should contain '%s'", expected)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
		(s[:len(substr)] == substr || containsString(s[1:], substr)))
}

func TestDashboard_SendUpdate(t *testing.T) {
	d := NewDashboard(nil)

	// Should not panic even without clients
	d.SendUpdate("test", map[string]string{"key": "value"})
}

func TestDashboard_Stop(t *testing.T) {
	d := NewDashboard(nil)

	// Start goroutines manually (without HTTP server)
	d.wg.Add(2)
	go d.broadcastLoop()
	go d.metricsLoop()

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		d.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stop timed out")
	}
}

func TestDashboardUpdate_JSON(t *testing.T) {
	update := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      "stats",
		Data:      testMetrics{WorkersActive: 4, TasksCompleted: 100},
	}

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DashboardUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Type != "stats" {
		t.Errorf("Type = %s, want stats", decoded.Type)
	}
}

func TestDashboard_broadcastLoop(t *testing.T) {
	d := NewDashboard(nil)

	// Add a client
	clientChan := make(chan DashboardUpdate, 10)
	d.clients.Store("client1", clientChan)

	// Start broadcast loop (must add to wg before starting goroutine)
	d.wg.Add(1)
	go d.broadcastLoop()

	// Send update
	update := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      "test",
		Data:      "test data",
	}

	select {
	case d.broadcast <- update:
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send to broadcast channel")
	}

	// Wait for broadcast
	select {
	case received := <-clientChan:
		if received.Type != "test" {
			t.Errorf("Type = %s, want test", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive broadcast")
	}

	// Stop
	close(d.stopChan)
	d.wg.Wait()
}

func TestDashboard_metricsLoop(t *testing.T) {
	provider := &mockMetricsProvider{
		metrics: testMetrics{WorkersActive: 4, TasksCompleted: 100},
	}

	d := NewDashboard(provider)

	// Start metrics loop (must add to wg before starting goroutine)
	d.wg.Add(1)
	go d.metricsLoop()

	// Wait for at least one update
	select {
	case update := <-d.broadcast:
		if update.Type != "stats" {
			t.Errorf("Type = %s, want stats", update.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("No metrics update received")
	}

	// Stop
	close(d.stopChan)
	d.wg.Wait()
}

// Benchmarks

func BenchmarkDashboard_handleMetrics(b *testing.B) {
	provider := &mockMetricsProvider{
		metrics: testMetrics{WorkersActive: 4, TasksCompleted: 100},
	}

	d := NewDashboard(provider)

	req := httptest.NewRequest("GET", "/api/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		d.handleMetrics(w, req)
	}
}

func BenchmarkDashboard_SendUpdate(b *testing.B) {
	d := NewDashboard(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.SendUpdate("test", i)
	}
}
