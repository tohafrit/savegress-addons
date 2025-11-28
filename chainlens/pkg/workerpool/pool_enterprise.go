package workerpool

import (
	"context"
	"fmt"
	"time"

	"github.com/tohafrit/worker-pool/workerpool/enterprise/config"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/cost"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/dashboard"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/multitenancy"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/observability"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/persistence"
	"github.com/tohafrit/worker-pool/workerpool/enterprise/resilience"
)

// Enterprise components (embedded in WorkerPool)
type enterpriseComponents struct {
	// Observability
	logger        observability.Logger
	metrics       *observability.MetricsCollector
	tracer        observability.Tracer
	healthChecker *observability.HealthChecker

	// Resilience
	rateLimiter      resilience.RateLimiter
	circuitBreakers  *resilience.CircuitBreakerManager
	resourceMonitor  *resilience.ResourceMonitor
	retryer          *resilience.Retryer

	// Persistence
	persistentQueue *persistence.InMemoryQueue
	dlq             *persistence.DeadLetterQueue

	// Multi-tenancy
	tenantManager *multitenancy.TenantManager

	// Cost tracking
	costTracker *cost.CostTracker

	// Dashboard & Alerting
	dashboard    *dashboard.Dashboard
	alertManager *dashboard.AlertManager

	// Configuration
	configManager *config.ConfigManager
	featureFlags  *config.FeatureFlags
}

// initializeEnterpriseFeatures initializes enterprise components based on config
func (p *WorkerPool) initializeEnterpriseFeatures() {
	p.enterprise = &enterpriseComponents{}

	// Initialize observability if enabled
	if p.config.Telemetry != nil && p.config.Telemetry.Enabled {
		p.initObservability()
	} else {
		// Use no-op implementations
		p.enterprise.logger = &observability.NoOpLogger{}
		p.enterprise.tracer = &observability.NoOpTracer{}
	}

	// Always initialize metrics collector (used internally)
	p.enterprise.metrics = observability.NewMetricsCollector(p.config.QueueSize)
	p.enterprise.healthChecker = observability.NewHealthChecker(p.config.QueueSize)

	// Initialize resilience features
	if p.config.RateLimit != nil && p.config.RateLimit.Enabled {
		p.enterprise.rateLimiter = resilience.NewTokenBucketLimiter(
			p.config.RateLimit.Rate,
			p.config.RateLimit.Burst,
		)
	} else {
		p.enterprise.rateLimiter = &resilience.NoOpRateLimiter{}
	}

	if p.config.CircuitBreaker != nil && p.config.CircuitBreaker.Enabled {
		p.enterprise.circuitBreakers = resilience.NewCircuitBreakerManager(resilience.CircuitBreakerConfig{
			FailureThreshold: p.config.CircuitBreaker.FailureThreshold,
			SuccessThreshold: p.config.CircuitBreaker.SuccessThreshold,
			Timeout:          p.config.CircuitBreaker.Timeout,
			HalfOpenMaxCalls: p.config.CircuitBreaker.HalfOpenMaxCalls,
		})
	}

	if p.config.Resource != nil {
		p.enterprise.resourceMonitor = resilience.NewResourceMonitor(resilience.ResourceConfig{
			MaxCPUPercent:  p.config.Resource.MaxCPUPercent,
			MaxMemoryMB:    p.config.Resource.MaxMemoryMB,
			CPUThrottle:    p.config.Resource.CPUThrottle,
			MemoryThrottle: p.config.Resource.MemoryThrottle,
			OnThrottle: func(resource string) {
				p.enterprise.logger.Warn("resource throttling activated",
					observability.Field{Key: "resource", Value: resource},
				)
			},
		})
		p.enterprise.resourceMonitor.Start()
	}

	if p.config.Retry != nil && p.config.Retry.Enabled {
		p.enterprise.retryer = resilience.NewRetryer(resilience.RetryPolicy{
			MaxRetries:   p.config.Retry.MaxRetries,
			InitialDelay: p.config.Retry.InitialDelay,
			MaxDelay:     p.config.Retry.MaxDelay,
			Multiplier:   p.config.Retry.Multiplier,
			Jitter:       p.config.Retry.Jitter,
		}, nil)
	}

	// Initialize persistence
	if p.config.Persistence != nil && p.config.Persistence.Enabled {
		p.initPersistence()
	}

	// Initialize multi-tenancy
	if p.config.MultiTenancy != nil && p.config.MultiTenancy.Enabled {
		p.enterprise.tenantManager = multitenancy.NewTenantManager(multitenancy.TenantConfig{
			MaxWorkers:   p.config.MultiTenancy.DefaultMaxTasks,
			MaxQueueSize: p.config.MultiTenancy.DefaultMaxTasks,
			CPUQuota:     p.config.MultiTenancy.DefaultCPUQuota,
			MemoryQuota:  p.config.MultiTenancy.DefaultMemoryMB,
		})
	}

	// Initialize cost tracking
	if p.config.Cost != nil && p.config.Cost.Enabled {
		p.enterprise.costTracker = cost.NewCostTracker(cost.CostConfig{
			Enabled:         p.config.Cost.Enabled,
			CPUCostPerMs:    p.config.Cost.CPUCostPerMs,
			MemoryCostPerMB: p.config.Cost.MemoryCostPerMB,
			TaskCostBase:    p.config.Cost.TaskCostBase,
		})
	}

	// Initialize dashboard
	if p.config.Dashboard != nil && p.config.Dashboard.Enabled {
		p.initDashboard()
	}

	// Initialize configuration management
	p.enterprise.configManager = config.NewConfigManager(p.config, "")
	p.enterprise.featureFlags = config.NewFeatureFlags()

	if p.config.Features != nil {
		p.enterprise.featureFlags.Set("tracing", p.config.Features.EnableTracing)
		p.enterprise.featureFlags.Set("persistence", p.config.Features.EnablePersistence)
		p.enterprise.featureFlags.Set("cost_tracking", p.config.Features.EnableCostTracking)
		p.enterprise.featureFlags.Set("multi_tenancy", p.config.Features.EnableMultiTenancy)
	}

	// Mark as started
	if p.enterprise.healthChecker != nil {
		p.enterprise.healthChecker.MarkStarted()
	}
}

// initObservability initializes observability components
func (p *WorkerPool) initObservability() {
	// Logger
	logger := observability.NewDefaultLogger()
	logger.SetLevel(p.parseLogLevel(p.config.Telemetry.LogLevel))
	if p.config.Telemetry.LogSampling.Initial > 0 {
		logger.WithSampling(
			p.config.Telemetry.LogSampling.Initial,
			p.config.Telemetry.LogSampling.Thereafter,
		)
	}
	p.enterprise.logger = logger

	// Tracer
	if p.config.Features != nil && p.config.Features.EnableTracing {
		p.enterprise.tracer = observability.NewSimpleTracer(
			p.config.Telemetry.ServiceName,
			&observability.InMemorySpanExporter{},
		)
	} else {
		p.enterprise.tracer = &observability.NoOpTracer{}
	}

	p.enterprise.logger.Info("worker pool initialized",
		observability.Field{Key: "workers", Value: p.config.Workers},
		observability.Field{Key: "queue_size", Value: p.config.QueueSize},
	)
}

// initPersistence initializes persistence components
func (p *WorkerPool) initPersistence() {
	p.enterprise.persistentQueue = persistence.NewInMemoryQueue(p.config.QueueSize)

	dlqStorage := persistence.NewInMemoryQueue(1000)
	p.enterprise.dlq = persistence.NewDeadLetterQueue(persistence.DLQConfig{
		MaxSize:   1000,
		Retention: 24 * time.Hour,
		Storage:   dlqStorage,
		OnMessage: func(entry *persistence.DLQEntry) {
			p.enterprise.logger.Error("task moved to DLQ",
				observability.Field{Key: "task_id", Value: entry.TaskID},
				observability.Field{Key: "failure_count", Value: entry.FailureCount},
			)
		},
	})
}

// initDashboard initializes dashboard and alerting
func (p *WorkerPool) initDashboard() {
	p.enterprise.dashboard = dashboard.NewDashboard(p)
	p.enterprise.alertManager = dashboard.NewAlertManager(p)

	// Add default alert rules
	p.addDefaultAlertRules()

	// Start dashboard in background
	go func() {
		addr := fmt.Sprintf(":%d", p.config.Dashboard.Port)
		if err := p.enterprise.dashboard.Start(addr); err != nil {
			p.enterprise.logger.Error("dashboard failed to start",
				observability.Field{Key: "error", Value: err},
			)
		}
	}()

	p.enterprise.alertManager.Start()
}

// addDefaultAlertRules adds default alerting rules
func (p *WorkerPool) addDefaultAlertRules() {
	p.enterprise.alertManager.AddRule(dashboard.AlertRule{
		Name: "high_queue_size",
		Condition: func(metrics interface{}) bool {
			if m, ok := metrics.(observability.Metrics); ok {
				return m.QueueSize > p.config.QueueSize*9/10
			}
			return false
		},
		Duration: 1 * time.Minute,
		Severity: dashboard.SeverityWarning,
		Annotations: map[string]string{
			"description": "Queue size is above 90% capacity",
		},
	})
}

// GetMetrics implements MetricsProvider interface for dashboard
func (p *WorkerPool) GetMetrics() interface{} {
	if p.enterprise != nil && p.enterprise.metrics != nil {
		return p.enterprise.metrics.GetMetrics()
	}
	return nil
}

// SubmitWithTenant submits a task for a specific tenant (multi-tenancy feature)
func (p *WorkerPool) SubmitWithTenant(ctx context.Context, tenantID string, fn func() error) error {
	if p.enterprise == nil || p.enterprise.tenantManager == nil {
		// Fall back to regular submit
		return p.SubmitWithContext(ctx, fn)
	}

	// Check tenant quota
	allowed, err := p.enterprise.tenantManager.CheckQuota(tenantID)
	if err != nil {
		return err
	}
	if !allowed {
		p.enterprise.tenantManager.RecordTaskRejected(tenantID)
		return multitenancy.ErrQuotaExceeded
	}

	p.enterprise.tenantManager.RecordTaskSubmitted(tenantID)

	wrappedFn := func() error {
		defer p.enterprise.tenantManager.ReleaseQuota(tenantID)

		start := time.Now()
		err := fn()
		duration := time.Since(start)

		p.enterprise.tenantManager.RecordTaskCompleted(tenantID, int64(duration.Milliseconds()), 0)

		if p.enterprise.costTracker != nil {
			p.enterprise.costTracker.RecordTaskCost("", tenantID, int64(duration.Milliseconds()), 0, duration)
		}

		return err
	}

	return p.SubmitWithContext(ctx, wrappedFn)
}

// GetHealthChecker returns the health checker (if enabled)
func (p *WorkerPool) GetHealthChecker() *observability.HealthChecker {
	if p.enterprise != nil {
		return p.enterprise.healthChecker
	}
	return nil
}

// GetDashboard returns the dashboard (if enabled)
func (p *WorkerPool) GetDashboard() *dashboard.Dashboard {
	if p.enterprise != nil {
		return p.enterprise.dashboard
	}
	return nil
}

// GetCostTracker returns the cost tracker (if enabled)
func (p *WorkerPool) GetCostTracker() *cost.CostTracker {
	if p.enterprise != nil {
		return p.enterprise.costTracker
	}
	return nil
}

// parseLogLevel parses log level string
func (p *WorkerPool) parseLogLevel(level string) observability.LogLevel {
	switch level {
	case "debug":
		return observability.DebugLevel
	case "info":
		return observability.InfoLevel
	case "warn":
		return observability.WarnLevel
	case "error":
		return observability.ErrorLevel
	default:
		return observability.InfoLevel
	}
}

// stopEnterpriseComponents stops all enterprise components
func (p *WorkerPool) stopEnterpriseComponents() {
	if p.enterprise == nil {
		return
	}

	if p.enterprise.resourceMonitor != nil {
		p.enterprise.resourceMonitor.Stop()
	}

	if p.enterprise.dashboard != nil {
		p.enterprise.dashboard.Stop()
	}

	if p.enterprise.alertManager != nil {
		p.enterprise.alertManager.Stop()
	}

	if p.enterprise.healthChecker != nil {
		p.enterprise.healthChecker.MarkStopped()
	}

	if p.enterprise.logger != nil {
		p.enterprise.logger.Info("worker pool stopped")
	}
}
