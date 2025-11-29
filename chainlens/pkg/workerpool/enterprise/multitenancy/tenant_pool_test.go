package multitenancy

import (
	"context"
	"testing"
)

func TestNewResourceQuota(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 100)

	if rq == nil {
		t.Fatal("NewResourceQuota returned nil")
	}

	usage := rq.GetUsage()
	if usage.CPULimit != 1000 {
		t.Errorf("CPULimit = %d, want 1000", usage.CPULimit)
	}
	if usage.MemoryLimit != 1024*1024 {
		t.Errorf("MemoryLimit = %d, want %d", usage.MemoryLimit, 1024*1024)
	}
	if usage.TasksLimit != 100 {
		t.Errorf("TasksLimit = %d, want 100", usage.TasksLimit)
	}
}

func TestResourceQuota_CheckAndReserve(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 3)

	// Should succeed
	if !rq.CheckAndReserve() {
		t.Error("First reserve should succeed")
	}
	if !rq.CheckAndReserve() {
		t.Error("Second reserve should succeed")
	}
	if !rq.CheckAndReserve() {
		t.Error("Third reserve should succeed")
	}

	// Should fail at limit
	if rq.CheckAndReserve() {
		t.Error("Fourth reserve should fail")
	}

	usage := rq.GetUsage()
	if usage.TasksUsed != 3 {
		t.Errorf("TasksUsed = %d, want 3", usage.TasksUsed)
	}
}

func TestResourceQuota_Release(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 1)

	rq.CheckAndReserve()
	if rq.CheckAndReserve() {
		t.Error("Should be at limit")
	}

	rq.Release()
	if !rq.CheckAndReserve() {
		t.Error("Should have capacity after release")
	}
}

func TestResourceQuota_RecordCPU(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 100)

	rq.RecordCPU(100)
	rq.RecordCPU(50)

	usage := rq.GetUsage()
	if usage.CPUUsed != 150 {
		t.Errorf("CPUUsed = %d, want 150", usage.CPUUsed)
	}
}

func TestResourceQuota_RecordMemory(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 100)

	rq.RecordMemory(512 * 1024)

	usage := rq.GetUsage()
	if usage.MemoryUsed != 512*1024 {
		t.Errorf("MemoryUsed = %d, want %d", usage.MemoryUsed, 512*1024)
	}

	// RecordMemory stores (not adds)
	rq.RecordMemory(256 * 1024)
	usage = rq.GetUsage()
	if usage.MemoryUsed != 256*1024 {
		t.Errorf("MemoryUsed after update = %d, want %d", usage.MemoryUsed, 256*1024)
	}
}

func TestResourceQuota_UnlimitedTasks(t *testing.T) {
	rq := NewResourceQuota(1000, 1024*1024, 0) // 0 = unlimited

	// Should always succeed
	for i := 0; i < 100; i++ {
		if !rq.CheckAndReserve() {
			t.Fatalf("Reserve %d should succeed with unlimited tasks", i)
		}
	}
}

// TenantManager tests

func TestNewTenantManager(t *testing.T) {
	defaultConfig := TenantConfig{
		MaxWorkers:   4,
		MaxQueueSize: 100,
	}

	tm := NewTenantManager(defaultConfig)
	if tm == nil {
		t.Fatal("NewTenantManager returned nil")
	}
}

func TestTenantManager_RegisterTenant(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})

	config := TenantConfig{
		TenantID:     "tenant1",
		MaxWorkers:   8,
		MaxQueueSize: 200,
		CPUQuota:     80,
		MemoryQuota:  1024,
	}

	err := tm.RegisterTenant(config)
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	info, err := tm.GetTenant("tenant1")
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}

	if info.Config.MaxWorkers != 8 {
		t.Errorf("MaxWorkers = %d, want 8", info.Config.MaxWorkers)
	}
}

func TestTenantManager_GetTenant_Default(t *testing.T) {
	defaultConfig := TenantConfig{
		MaxWorkers:   4,
		MaxQueueSize: 100,
	}

	tm := NewTenantManager(defaultConfig)

	// Unregistered tenant gets default config
	info, err := tm.GetTenant("unknown")
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}

	if info.Config.MaxWorkers != defaultConfig.MaxWorkers {
		t.Errorf("MaxWorkers = %d, want %d", info.Config.MaxWorkers, defaultConfig.MaxWorkers)
	}
}

func TestTenantManager_CheckQuota(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})

	config := TenantConfig{
		TenantID:     "tenant1",
		MaxQueueSize: 2,
	}
	tm.RegisterTenant(config)

	// First two should succeed
	ok, err := tm.CheckQuota("tenant1")
	if err != nil {
		t.Fatalf("CheckQuota failed: %v", err)
	}
	if !ok {
		t.Error("First check should succeed")
	}

	ok, _ = tm.CheckQuota("tenant1")
	if !ok {
		t.Error("Second check should succeed")
	}

	// Third should fail
	ok, _ = tm.CheckQuota("tenant1")
	if ok {
		t.Error("Third check should fail")
	}
}

func TestTenantManager_ReleaseQuota(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})

	config := TenantConfig{
		TenantID:     "tenant1",
		MaxQueueSize: 1,
	}
	tm.RegisterTenant(config)

	tm.CheckQuota("tenant1")
	ok, _ := tm.CheckQuota("tenant1")
	if ok {
		t.Error("Should be at limit")
	}

	err := tm.ReleaseQuota("tenant1")
	if err != nil {
		t.Fatalf("ReleaseQuota failed: %v", err)
	}

	ok, _ = tm.CheckQuota("tenant1")
	if !ok {
		t.Error("Should have capacity after release")
	}
}

func TestTenantManager_RecordTaskSubmitted(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant1"})

	tm.RecordTaskSubmitted("tenant1")
	tm.RecordTaskSubmitted("tenant1")

	stats, err := tm.GetTenantStats("tenant1")
	if err != nil {
		t.Fatalf("GetTenantStats failed: %v", err)
	}

	if stats.TasksSubmitted.Load() != 2 {
		t.Errorf("TasksSubmitted = %d, want 2", stats.TasksSubmitted.Load())
	}
}

func TestTenantManager_RecordTaskCompleted(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant1"})

	tm.RecordTaskCompleted("tenant1", 100, 1024*1024)

	stats, err := tm.GetTenantStats("tenant1")
	if err != nil {
		t.Fatalf("GetTenantStats failed: %v", err)
	}

	if stats.TasksCompleted.Load() != 1 {
		t.Errorf("TasksCompleted = %d, want 1", stats.TasksCompleted.Load())
	}
	if stats.TotalCPUTime.Load() != 100 {
		t.Errorf("TotalCPUTime = %d, want 100", stats.TotalCPUTime.Load())
	}
}

func TestTenantManager_RecordTaskRejected(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant1"})

	tm.RecordTaskRejected("tenant1")

	stats, err := tm.GetTenantStats("tenant1")
	if err != nil {
		t.Fatalf("GetTenantStats failed: %v", err)
	}

	if stats.TasksRejected.Load() != 1 {
		t.Errorf("TasksRejected = %d, want 1", stats.TasksRejected.Load())
	}
}

func TestTenantManager_GetAllTenants(t *testing.T) {
	tm := NewTenantManager(TenantConfig{})

	tm.RegisterTenant(TenantConfig{TenantID: "tenant1"})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant2"})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant3"})

	all := tm.GetAllTenants()
	if len(all) != 3 {
		t.Errorf("GetAllTenants len = %d, want 3", len(all))
	}

	if _, ok := all["tenant1"]; !ok {
		t.Error("tenant1 should be in result")
	}
	if _, ok := all["tenant2"]; !ok {
		t.Error("tenant2 should be in result")
	}
}

// TenantScheduler tests

func TestNewTenantScheduler(t *testing.T) {
	ts := NewTenantScheduler()
	if ts == nil {
		t.Fatal("NewTenantScheduler returned nil")
	}
}

func TestTenantScheduler_AddTenant(t *testing.T) {
	ts := NewTenantScheduler()

	info := &TenantInfo{
		Config: TenantConfig{
			TenantID: "tenant1",
			Priority: 5,
		},
	}

	ts.AddTenant(info)

	// Should be able to schedule
	scheduled := ts.Schedule()
	if scheduled == nil {
		t.Fatal("Schedule should return tenant after AddTenant")
	}
	if scheduled.Config.TenantID != "tenant1" {
		t.Errorf("TenantID = %s, want tenant1", scheduled.Config.TenantID)
	}
}

func TestTenantScheduler_Schedule_Empty(t *testing.T) {
	ts := NewTenantScheduler()

	scheduled := ts.Schedule()
	if scheduled != nil {
		t.Error("Schedule on empty scheduler should return nil")
	}
}

func TestTenantScheduler_Schedule_Priority(t *testing.T) {
	ts := NewTenantScheduler()

	// Add tenants with different priorities
	lowPriority := &TenantInfo{
		Config: TenantConfig{TenantID: "low", Priority: 0},
	}
	highPriority := &TenantInfo{
		Config: TenantConfig{TenantID: "high", Priority: 10},
	}

	ts.AddTenant(lowPriority)
	ts.AddTenant(highPriority)

	// High priority should be scheduled first
	scheduled := ts.Schedule()
	if scheduled.Config.TenantID != "high" {
		t.Errorf("Expected high priority first, got %s", scheduled.Config.TenantID)
	}
}

func TestTenantScheduler_Schedule_RoundRobin(t *testing.T) {
	ts := NewTenantScheduler()

	// Add tenants with same priority
	tenant1 := &TenantInfo{
		Config: TenantConfig{TenantID: "tenant1", Priority: 5},
	}
	tenant2 := &TenantInfo{
		Config: TenantConfig{TenantID: "tenant2", Priority: 5},
	}

	ts.AddTenant(tenant1)
	ts.AddTenant(tenant2)

	// Should round-robin
	first := ts.Schedule()
	second := ts.Schedule()
	third := ts.Schedule()

	if first.Config.TenantID == second.Config.TenantID {
		t.Error("Round-robin should alternate tenants")
	}
	if first.Config.TenantID != third.Config.TenantID {
		t.Error("Round-robin should cycle back")
	}
}

// TaskContext tests

func TestNewTaskContext(t *testing.T) {
	ctx := context.Background()
	tc := NewTaskContext(ctx, "tenant1")

	if tc == nil {
		t.Fatal("NewTaskContext returned nil")
	}
	if tc.TenantID != "tenant1" {
		t.Errorf("TenantID = %s, want tenant1", tc.TenantID)
	}
	if tc.Context != ctx {
		t.Error("Context should be set")
	}
	if tc.Claims == nil {
		t.Error("Claims should be initialized")
	}
}

func TestTaskContext_Fields(t *testing.T) {
	ctx := context.Background()
	tc := NewTaskContext(ctx, "tenant1")

	tc.UserID = "user123"
	tc.RequestID = "req456"
	tc.SourceIP = "192.168.1.1"
	tc.UserAgent = "TestAgent/1.0"
	tc.Token = "token123"
	tc.Claims["role"] = "admin"

	if tc.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", tc.UserID)
	}
	if tc.RequestID != "req456" {
		t.Errorf("RequestID = %s, want req456", tc.RequestID)
	}
	if tc.SourceIP != "192.168.1.1" {
		t.Errorf("SourceIP = %s, want 192.168.1.1", tc.SourceIP)
	}
	if tc.Claims["role"] != "admin" {
		t.Errorf("Claims[role] = %v, want admin", tc.Claims["role"])
	}
}

// Error variables test

func TestErrTenantNotFound(t *testing.T) {
	if ErrTenantNotFound == nil {
		t.Error("ErrTenantNotFound should not be nil")
	}
	if ErrTenantNotFound.Error() != "tenant not found" {
		t.Errorf("ErrTenantNotFound message = %s", ErrTenantNotFound.Error())
	}
}

func TestErrQuotaExceeded(t *testing.T) {
	if ErrQuotaExceeded == nil {
		t.Error("ErrQuotaExceeded should not be nil")
	}
	if ErrQuotaExceeded.Error() != "tenant quota exceeded" {
		t.Errorf("ErrQuotaExceeded message = %s", ErrQuotaExceeded.Error())
	}
}

// Benchmarks

func BenchmarkResourceQuota_CheckAndReserve(b *testing.B) {
	rq := NewResourceQuota(1000, 1024*1024, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rq.CheckAndReserve()
		rq.Release()
	}
}

func BenchmarkTenantManager_CheckQuota(b *testing.B) {
	tm := NewTenantManager(TenantConfig{MaxQueueSize: 0})
	tm.RegisterTenant(TenantConfig{TenantID: "tenant1", MaxQueueSize: 0})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.CheckQuota("tenant1")
		tm.ReleaseQuota("tenant1")
	}
}

func BenchmarkTenantScheduler_Schedule(b *testing.B) {
	ts := NewTenantScheduler()
	for i := 0; i < 10; i++ {
		ts.AddTenant(&TenantInfo{
			Config: TenantConfig{TenantID: "tenant", Priority: 5},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.Schedule()
	}
}
