package cost

import (
	"strings"
	"testing"
	"time"
)

func TestNewCostTracker(t *testing.T) {
	cfg := CostConfig{
		Enabled:         true,
		CPUCostPerMs:    0.00001,
		MemoryCostPerMB: 0.0001,
		TaskCostBase:    0.001,
	}

	ct := NewCostTracker(cfg)
	if ct == nil {
		t.Fatal("NewCostTracker returned nil")
	}
}

func TestCostTracker_RecordTaskCost(t *testing.T) {
	cfg := CostConfig{
		Enabled:         true,
		CPUCostPerMs:    0.00001,
		MemoryCostPerMB: 0.0001,
		TaskCostBase:    0.001,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024*1024, 1*time.Second)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	costs := ct.GetCostsByTenant("tenant1", start, end)
	if len(costs) != 1 {
		t.Fatalf("Expected 1 cost, got %d", len(costs))
	}

	cost := costs[0]
	if cost.TaskID != "task1" {
		t.Errorf("TaskID = %s, want task1", cost.TaskID)
	}
	if cost.TenantID != "tenant1" {
		t.Errorf("TenantID = %s, want tenant1", cost.TenantID)
	}
	if cost.CPUMillis != 100 {
		t.Errorf("CPUMillis = %d, want 100", cost.CPUMillis)
	}
	if cost.TotalCost <= 0 {
		t.Error("TotalCost should be greater than 0")
	}
}

func TestCostTracker_RecordTaskCost_Disabled(t *testing.T) {
	cfg := CostConfig{
		Enabled: false,
	}

	ct := NewCostTracker(cfg)
	ct.RecordTaskCost("task1", "tenant1", 100, 1024*1024, 1*time.Second)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	costs := ct.GetCostsByTenant("tenant1", start, end)
	if len(costs) != 0 {
		t.Error("Should not record costs when disabled")
	}
}

func TestCostTracker_GetCostsByTenant_Empty(t *testing.T) {
	cfg := CostConfig{Enabled: true}
	ct := NewCostTracker(cfg)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	costs := ct.GetCostsByTenant("nonexistent", start, end)
	if costs == nil {
		t.Fatal("GetCostsByTenant should return empty slice, not nil")
	}
	if len(costs) != 0 {
		t.Errorf("Expected 0 costs, got %d", len(costs))
	}
}

func TestCostTracker_GetCostsByTenant_TimeFilter(t *testing.T) {
	cfg := CostConfig{
		Enabled:      true,
		TaskCostBase: 0.001,
	}

	ct := NewCostTracker(cfg)

	// Record costs at different times
	ct.RecordTaskCost("task1", "tenant1", 100, 1024, 1*time.Second)
	time.Sleep(10 * time.Millisecond)
	ct.RecordTaskCost("task2", "tenant1", 100, 1024, 1*time.Second)

	// Get costs in a narrow window
	start := time.Now().Add(-1 * time.Second)
	end := time.Now().Add(1 * time.Second)

	costs := ct.GetCostsByTenant("tenant1", start, end)
	if len(costs) != 2 {
		t.Errorf("Expected 2 costs in time range, got %d", len(costs))
	}

	// Get costs outside window
	farPast := time.Now().Add(-1 * time.Hour)
	farPastEnd := time.Now().Add(-30 * time.Minute)

	costs = ct.GetCostsByTenant("tenant1", farPast, farPastEnd)
	if len(costs) != 0 {
		t.Errorf("Expected 0 costs outside time range, got %d", len(costs))
	}
}

func TestCostTracker_GetTotalCost(t *testing.T) {
	cfg := CostConfig{
		Enabled:      true,
		TaskCostBase: 0.01,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024, 1*time.Second)
	ct.RecordTaskCost("task2", "tenant1", 100, 1024, 1*time.Second)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	total := ct.GetTotalCost("tenant1", start, end)
	if total <= 0 {
		t.Error("TotalCost should be greater than 0")
	}
}

func TestCostTracker_GenerateBilling(t *testing.T) {
	cfg := CostConfig{
		Enabled:         true,
		CPUCostPerMs:    0.00001,
		MemoryCostPerMB: 0.0001,
		TaskCostBase:    0.001,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024*1024, 1*time.Second)
	ct.RecordTaskCost("task2", "tenant1", 200, 2*1024*1024, 2*time.Second)

	period := BillingPeriod{
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now().Add(1 * time.Hour),
	}

	billing := ct.GenerateBilling("tenant1", period)

	if billing.TenantID != "tenant1" {
		t.Errorf("TenantID = %s, want tenant1", billing.TenantID)
	}
	if billing.TaskCount != 2 {
		t.Errorf("TaskCount = %d, want 2", billing.TaskCount)
	}
	if billing.TotalCost <= 0 {
		t.Error("TotalCost should be greater than 0")
	}
}

func TestCostTracker_GenerateInvoice(t *testing.T) {
	cfg := CostConfig{
		Enabled:      true,
		TaskCostBase: 0.01,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024, 1*time.Second)

	period := BillingPeriod{
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   time.Now().Add(1 * time.Hour),
	}

	invoice := ct.GenerateInvoice("tenant1", period)

	if invoice.TenantID != "tenant1" {
		t.Errorf("TenantID = %s, want tenant1", invoice.TenantID)
	}
	if !strings.HasPrefix(invoice.InvoiceID, "INV-") {
		t.Errorf("InvoiceID should start with INV-, got %s", invoice.InvoiceID)
	}
	if invoice.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", invoice.Currency)
	}
	if len(invoice.LineItems) != 1 {
		t.Errorf("LineItems len = %d, want 1", len(invoice.LineItems))
	}
}

func TestCostTracker_GetAllTenantCosts(t *testing.T) {
	cfg := CostConfig{
		Enabled:      true,
		TaskCostBase: 0.01,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024, 1*time.Second)
	ct.RecordTaskCost("task2", "tenant2", 100, 1024, 1*time.Second)
	ct.RecordTaskCost("task3", "tenant1", 100, 1024, 1*time.Second)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	allCosts := ct.GetAllTenantCosts(start, end)

	if len(allCosts) != 2 {
		t.Errorf("Expected 2 tenants, got %d", len(allCosts))
	}

	if allCosts["tenant1"] <= 0 {
		t.Error("tenant1 cost should be > 0")
	}
	if allCosts["tenant2"] <= 0 {
		t.Error("tenant2 cost should be > 0")
	}
}

func TestCostTracker_GetCostSummary(t *testing.T) {
	cfg := CostConfig{
		Enabled:         true,
		CPUCostPerMs:    0.00001,
		MemoryCostPerMB: 0.0001,
		TaskCostBase:    0.001,
	}

	ct := NewCostTracker(cfg)

	ct.RecordTaskCost("task1", "tenant1", 100, 1024*1024, 1*time.Second)
	ct.RecordTaskCost("task2", "tenant2", 200, 2*1024*1024, 2*time.Second)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	summary := ct.GetCostSummary(start, end)

	if summary.TotalTasks != 2 {
		t.Errorf("TotalTasks = %d, want 2", summary.TotalTasks)
	}
	if summary.TotalCost <= 0 {
		t.Error("TotalCost should be > 0")
	}
	if summary.CPUCost <= 0 {
		t.Error("CPUCost should be > 0")
	}
	if summary.MemoryCost <= 0 {
		t.Error("MemoryCost should be > 0")
	}
	if summary.BaseCost <= 0 {
		t.Error("BaseCost should be > 0")
	}
	if len(summary.PerTenant) != 2 {
		t.Errorf("PerTenant len = %d, want 2", len(summary.PerTenant))
	}
}

func TestTaskCost_Fields(t *testing.T) {
	cost := TaskCost{
		TaskID:      "task1",
		TenantID:    "tenant1",
		CPUMillis:   100,
		MemoryMBSec: 1.5,
		Duration:    1 * time.Second,
		CPUCost:     0.001,
		MemoryCost:  0.0001,
		BaseCost:    0.01,
		TotalCost:   0.0111,
		Timestamp:   time.Now(),
	}

	if cost.TaskID != "task1" {
		t.Errorf("TaskID = %s, want task1", cost.TaskID)
	}
	if cost.Duration != 1*time.Second {
		t.Errorf("Duration = %v, want 1s", cost.Duration)
	}
}

func TestGenerateInvoiceID(t *testing.T) {
	id1 := generateInvoiceID()
	time.Sleep(1 * time.Second)
	id2 := generateInvoiceID()

	if !strings.HasPrefix(id1, "INV-") {
		t.Errorf("Invoice ID should start with INV-, got %s", id1)
	}
	if id1 == id2 {
		t.Error("Invoice IDs should be unique")
	}
}

// Benchmarks

func BenchmarkCostTracker_RecordTaskCost(b *testing.B) {
	cfg := CostConfig{
		Enabled:         true,
		CPUCostPerMs:    0.00001,
		MemoryCostPerMB: 0.0001,
		TaskCostBase:    0.001,
	}

	ct := NewCostTracker(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct.RecordTaskCost("task", "tenant", 100, 1024*1024, 1*time.Second)
	}
}

func BenchmarkCostTracker_GetTotalCost(b *testing.B) {
	cfg := CostConfig{
		Enabled:      true,
		TaskCostBase: 0.001,
	}

	ct := NewCostTracker(cfg)

	for i := 0; i < 1000; i++ {
		ct.RecordTaskCost("task", "tenant", 100, 1024, 1*time.Second)
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct.GetTotalCost("tenant", start, end)
	}
}
