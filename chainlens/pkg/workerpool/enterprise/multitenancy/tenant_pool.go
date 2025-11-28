package multitenancy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrQuotaExceeded  = errors.New("tenant quota exceeded")
)

// TenantConfig configures a tenant
type TenantConfig struct {
	TenantID     string
	MaxWorkers   int
	MaxQueueSize int
	CPUQuota     float64 // CPU percentage
	MemoryQuota  int64   // Memory in MB
	RateLimit    float64 // Tasks per second
	Priority     int     // Tenant priority (higher = more important)
}

// ResourceQuota tracks resource usage for a tenant
type ResourceQuota struct {
	cpuUsed    atomic.Int64 // CPU milliseconds
	memoryUsed atomic.Int64 // Memory bytes
	tasksUsed  atomic.Int64 // Task count

	cpuLimit    int64
	memoryLimit int64
	tasksLimit  int64
}

// NewResourceQuota creates a new resource quota
func NewResourceQuota(cpuLimit, memoryLimit, tasksLimit int64) *ResourceQuota {
	return &ResourceQuota{
		cpuLimit:    cpuLimit,
		memoryLimit: memoryLimit,
		tasksLimit:  tasksLimit,
	}
}

// CheckAndReserve checks if resources are available and reserves them
func (rq *ResourceQuota) CheckAndReserve() bool {
	// Check if we have capacity
	if rq.tasksLimit > 0 && rq.tasksUsed.Load() >= rq.tasksLimit {
		return false
	}

	rq.tasksUsed.Add(1)
	return true
}

// Release releases reserved resources
func (rq *ResourceQuota) Release() {
	rq.tasksUsed.Add(-1)
}

// RecordCPU records CPU usage
func (rq *ResourceQuota) RecordCPU(cpuMillis int64) {
	rq.cpuUsed.Add(cpuMillis)
}

// RecordMemory records memory usage
func (rq *ResourceQuota) RecordMemory(bytes int64) {
	rq.memoryUsed.Store(bytes)
}

// GetUsage returns current usage
func (rq *ResourceQuota) GetUsage() QuotaUsage {
	return QuotaUsage{
		CPUUsed:    rq.cpuUsed.Load(),
		MemoryUsed: rq.memoryUsed.Load(),
		TasksUsed:  rq.tasksUsed.Load(),
		CPULimit:   rq.cpuLimit,
		MemoryLimit: rq.memoryLimit,
		TasksLimit: rq.tasksLimit,
	}
}

// QuotaUsage represents quota usage
type QuotaUsage struct {
	CPUUsed     int64
	MemoryUsed  int64
	TasksUsed   int64
	CPULimit    int64
	MemoryLimit int64
	TasksLimit  int64
}

// TenantManager manages multiple tenants
type TenantManager struct {
	tenants       sync.Map // map[string]*TenantInfo
	defaultConfig TenantConfig
	mu            sync.RWMutex
}

// TenantInfo contains tenant information
type TenantInfo struct {
	Config TenantConfig
	Quota  *ResourceQuota
	Stats  *TenantStats
}

// TenantStats tracks tenant statistics
type TenantStats struct {
	TasksSubmitted atomic.Int64
	TasksCompleted atomic.Int64
	TasksRejected  atomic.Int64
	TotalCPUTime   atomic.Int64 // milliseconds
	TotalMemory    atomic.Int64 // bytes
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(defaultConfig TenantConfig) *TenantManager {
	return &TenantManager{
		defaultConfig: defaultConfig,
	}
}

// RegisterTenant registers a new tenant
func (tm *TenantManager) RegisterTenant(config TenantConfig) error {
	quota := NewResourceQuota(
		int64(config.CPUQuota),
		config.MemoryQuota*1024*1024, // Convert MB to bytes
		int64(config.MaxQueueSize),
	)

	info := &TenantInfo{
		Config: config,
		Quota:  quota,
		Stats:  &TenantStats{},
	}

	tm.tenants.Store(config.TenantID, info)
	return nil
}

// GetTenant gets tenant information
func (tm *TenantManager) GetTenant(tenantID string) (*TenantInfo, error) {
	infoI, ok := tm.tenants.Load(tenantID)
	if !ok {
		// Create tenant with default config
		defaultInfo := &TenantInfo{
			Config: tm.defaultConfig,
			Quota: NewResourceQuota(
				int64(tm.defaultConfig.CPUQuota),
				tm.defaultConfig.MemoryQuota*1024*1024,
				int64(tm.defaultConfig.MaxQueueSize),
			),
			Stats: &TenantStats{},
		}
		tm.tenants.Store(tenantID, defaultInfo)
		return defaultInfo, nil
	}

	return infoI.(*TenantInfo), nil
}

// CheckQuota checks if tenant has quota available
func (tm *TenantManager) CheckQuota(tenantID string) (bool, error) {
	info, err := tm.GetTenant(tenantID)
	if err != nil {
		return false, err
	}

	return info.Quota.CheckAndReserve(), nil
}

// ReleaseQuota releases tenant quota
func (tm *TenantManager) ReleaseQuota(tenantID string) error {
	info, err := tm.GetTenant(tenantID)
	if err != nil {
		return err
	}

	info.Quota.Release()
	return nil
}

// RecordTaskSubmitted records a task submission for a tenant
func (tm *TenantManager) RecordTaskSubmitted(tenantID string) {
	info, _ := tm.GetTenant(tenantID)
	if info != nil {
		info.Stats.TasksSubmitted.Add(1)
	}
}

// RecordTaskCompleted records a task completion for a tenant
func (tm *TenantManager) RecordTaskCompleted(tenantID string, cpuMillis, memoryBytes int64) {
	info, _ := tm.GetTenant(tenantID)
	if info != nil {
		info.Stats.TasksCompleted.Add(1)
		info.Stats.TotalCPUTime.Add(cpuMillis)
		info.Quota.RecordCPU(cpuMillis)
		info.Quota.RecordMemory(memoryBytes)
	}
}

// RecordTaskRejected records a task rejection for a tenant
func (tm *TenantManager) RecordTaskRejected(tenantID string) {
	info, _ := tm.GetTenant(tenantID)
	if info != nil {
		info.Stats.TasksRejected.Add(1)
	}
}

// GetTenantStats returns statistics for a tenant
func (tm *TenantManager) GetTenantStats(tenantID string) (*TenantStats, error) {
	info, err := tm.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}
	return info.Stats, nil
}

// GetAllTenants returns all tenants
func (tm *TenantManager) GetAllTenants() map[string]*TenantInfo {
	result := make(map[string]*TenantInfo)
	tm.tenants.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*TenantInfo)
		return true
	})
	return result
}

// TenantScheduler implements fair scheduling across tenants
type TenantScheduler struct {
	queues map[int][]*TenantInfo // Priority-based queues
	mu     sync.RWMutex
}

// NewTenantScheduler creates a new tenant scheduler
func NewTenantScheduler() *TenantScheduler {
	return &TenantScheduler{
		queues: make(map[int][]*TenantInfo),
	}
}

// Schedule selects the next tenant to process
func (ts *TenantScheduler) Schedule() *TenantInfo {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check priorities from high to low
	priorities := []int{10, 5, 1, 0} // High, medium, normal, low
	for _, priority := range priorities {
		tenants := ts.queues[priority]
		if len(tenants) == 0 {
			continue
		}

		// Round-robin within priority
		tenant := tenants[0]
		ts.queues[priority] = append(tenants[1:], tenant)
		return tenant
	}

	return nil
}

// AddTenant adds a tenant to the scheduler
func (ts *TenantScheduler) AddTenant(info *TenantInfo) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	priority := info.Config.Priority
	ts.queues[priority] = append(ts.queues[priority], info)
}

// TaskContext contains task execution context with tenant info
type TaskContext struct {
	Context   context.Context
	TenantID  string
	UserID    string
	RequestID string
	SourceIP  string
	UserAgent string
	Token     string
	Claims    map[string]interface{}
}

// NewTaskContext creates a new task context
func NewTaskContext(ctx context.Context, tenantID string) *TaskContext {
	return &TaskContext{
		Context:  ctx,
		TenantID: tenantID,
		Claims:   make(map[string]interface{}),
	}
}
