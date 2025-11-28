package cost

import (
	"sync"
	"time"
)

// TaskCost represents the cost of a single task
type TaskCost struct {
	TaskID      string
	TenantID    string
	CPUMillis   int64
	MemoryMBSec float64
	Duration    time.Duration
	CPUCost     float64
	MemoryCost  float64
	BaseCost    float64
	TotalCost   float64
	Timestamp   time.Time
}

// CostConfig configures cost tracking
type CostConfig struct {
	Enabled         bool
	CPUCostPerMs    float64 // Cost per CPU millisecond
	MemoryCostPerMB float64 // Cost per MB-second
	TaskCostBase    float64 // Base cost per task
}

// CostTracker tracks and aggregates costs
type CostTracker struct {
	config CostConfig
	costs  sync.Map // map[string][]TaskCost (keyed by tenantID)
	mu     sync.RWMutex
}

// NewCostTracker creates a new cost tracker
func NewCostTracker(config CostConfig) *CostTracker {
	return &CostTracker{
		config: config,
	}
}

// RecordTaskCost records the cost of a task
func (ct *CostTracker) RecordTaskCost(taskID, tenantID string, cpuMillis int64, memoryBytes int64, duration time.Duration) {
	if !ct.config.Enabled {
		return
	}

	memoryMBSec := float64(memoryBytes) / 1024 / 1024 * duration.Seconds()

	cost := TaskCost{
		TaskID:      taskID,
		TenantID:    tenantID,
		CPUMillis:   cpuMillis,
		MemoryMBSec: memoryMBSec,
		Duration:    duration,
		CPUCost:     float64(cpuMillis) * ct.config.CPUCostPerMs,
		MemoryCost:  memoryMBSec * ct.config.MemoryCostPerMB,
		BaseCost:    ct.config.TaskCostBase,
		Timestamp:   time.Now(),
	}

	cost.TotalCost = cost.CPUCost + cost.MemoryCost + cost.BaseCost

	// Store cost
	ct.addCost(tenantID, cost)
}

// addCost adds a cost entry for a tenant
func (ct *CostTracker) addCost(tenantID string, cost TaskCost) {
	costsI, _ := ct.costs.LoadOrStore(tenantID, &[]TaskCost{})
	costs := costsI.(*[]TaskCost)

	ct.mu.Lock()
	defer ct.mu.Unlock()

	*costs = append(*costs, cost)
}

// GetCostsByTenant returns costs for a specific tenant in a time range
func (ct *CostTracker) GetCostsByTenant(tenantID string, start, end time.Time) []TaskCost {
	costsI, ok := ct.costs.Load(tenantID)
	if !ok {
		return []TaskCost{}
	}

	costs := costsI.(*[]TaskCost)

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var filtered []TaskCost
	for _, cost := range *costs {
		if cost.Timestamp.After(start) && cost.Timestamp.Before(end) {
			filtered = append(filtered, cost)
		}
	}

	return filtered
}

// GetTotalCost calculates total cost for a tenant in a time range
func (ct *CostTracker) GetTotalCost(tenantID string, start, end time.Time) float64 {
	costs := ct.GetCostsByTenant(tenantID, start, end)

	var total float64
	for _, cost := range costs {
		total += cost.TotalCost
	}

	return total
}

// GenerateBilling generates billing information for a tenant
func (ct *CostTracker) GenerateBilling(tenantID string, period BillingPeriod) *TenantBilling {
	costs := ct.GetCostsByTenant(tenantID, period.StartDate, period.EndDate)

	billing := &TenantBilling{
		TenantID:  tenantID,
		Period:    period,
		TaskCount: int64(len(costs)),
		Breakdown: make(map[string]float64),
	}

	for _, cost := range costs {
		billing.TotalCost += cost.TotalCost
		billing.CPUCost += cost.CPUCost
		billing.MemoryCost += cost.MemoryCost

		// Breakdown by task type (simplified - could use actual task types)
		billing.Breakdown["tasks"] += cost.TotalCost
	}

	return billing
}

// GenerateInvoice generates an invoice for a tenant
func (ct *CostTracker) GenerateInvoice(tenantID string, period BillingPeriod) *Invoice {
	billing := ct.GenerateBilling(tenantID, period)

	invoice := &Invoice{
		InvoiceID:   generateInvoiceID(),
		TenantID:    tenantID,
		Period:      period,
		LineItems:   ct.GetCostsByTenant(tenantID, period.StartDate, period.EndDate),
		TotalAmount: billing.TotalCost,
		Currency:    "USD",
		GeneratedAt: time.Now(),
	}

	return invoice
}

// BillingPeriod represents a billing period
type BillingPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

// TenantBilling represents billing for a tenant
type TenantBilling struct {
	TenantID   string
	Period     BillingPeriod
	TaskCount  int64
	TotalCost  float64
	CPUCost    float64
	MemoryCost float64
	Breakdown  map[string]float64
}

// Invoice represents an invoice
type Invoice struct {
	InvoiceID   string
	TenantID    string
	Period      BillingPeriod
	LineItems   []TaskCost
	TotalAmount float64
	Currency    string
	GeneratedAt time.Time
}

// GetAllTenantCosts returns costs for all tenants
func (ct *CostTracker) GetAllTenantCosts(start, end time.Time) map[string]float64 {
	result := make(map[string]float64)

	ct.costs.Range(func(key, value interface{}) bool {
		tenantID := key.(string)
		total := ct.GetTotalCost(tenantID, start, end)
		result[tenantID] = total
		return true
	})

	return result
}

// CostSummary provides a summary of costs
type CostSummary struct {
	TotalTasks  int64
	TotalCost   float64
	CPUCost     float64
	MemoryCost  float64
	BaseCost    float64
	PerTenant   map[string]float64
}

// GetCostSummary returns a cost summary for a time period
func (ct *CostTracker) GetCostSummary(start, end time.Time) CostSummary {
	summary := CostSummary{
		PerTenant: make(map[string]float64),
	}

	ct.costs.Range(func(key, value interface{}) bool {
		tenantID := key.(string)
		costs := ct.GetCostsByTenant(tenantID, start, end)

		var tenantTotal float64
		for _, cost := range costs {
			summary.TotalTasks++
			summary.TotalCost += cost.TotalCost
			summary.CPUCost += cost.CPUCost
			summary.MemoryCost += cost.MemoryCost
			summary.BaseCost += cost.BaseCost
			tenantTotal += cost.TotalCost
		}

		summary.PerTenant[tenantID] = tenantTotal
		return true
	})

	return summary
}

// Helper function to generate invoice ID
func generateInvoiceID() string {
	return "INV-" + time.Now().Format("20060102-150405")
}
