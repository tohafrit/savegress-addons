package workerpool

import (
	"fmt"
	"runtime"
	"time"
)

// Config for worker pool with enterprise features
type Config struct {
	// Base configuration
	Workers         int           // Number of workers
	QueueSize       int           // Task queue buffer size
	ShutdownTimeout time.Duration // Max wait time for graceful shutdown
	EnableMetrics   bool          // Enable detailed metrics collection
	ErrorHandler    func(error)   // Callback for task errors

	// Enterprise features (optional - enabled based on configuration)
	Telemetry      *TelemetryConfig      // Observability configuration
	Resource       *ResourceConfig       // Resource management
	RateLimit      *RateLimitConfig      // Rate limiting
	Persistence    *PersistenceConfig    // Task persistence
	Retry          *RetryConfig          // Retry configuration
	CircuitBreaker *CircuitBreakerConfig // Circuit breaker
	MultiTenancy   *MultiTenancyConfig   // Multi-tenancy
	Security       *SecurityConfig       // Security features
	Dashboard      *DashboardConfig      // Dashboard
	Alerting       *AlertConfig          // Alerting
	Cost           *CostConfig           // Cost tracking
	Features       *FeatureFlags         // Feature flags
}

// TelemetryConfig configures observability features
type TelemetryConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string

	// Metrics
	MetricsExporter string // "prometheus", "otlp", "statsd"
	MetricsEndpoint string
	MetricsInterval time.Duration

	// Tracing
	TracingExporter string // "jaeger", "zipkin", "otlp"
	TracingEndpoint string
	SamplingRate    float64 // 0.0 to 1.0

	// Logging
	LogLevel    string // "debug", "info", "warn", "error"
	LogExporter string // "stdout", "file", "loki"
	LogSampling LogSamplingConfig
}

// LogSamplingConfig configures log sampling
type LogSamplingConfig struct {
	Initial    int // Log first N occurrences
	Thereafter int // Then log every Nth occurrence
}

// ResourceConfig configures resource limits and throttling
type ResourceConfig struct {
	MaxCPUPercent   float64       // Max CPU usage (0-100)
	CPUThrottle     bool          // Throttle when limit reached
	MaxMemoryMB     int64         // Max memory in MB
	MemoryThrottle  bool          // Throttle when limit reached
	MaxTaskSize     int64         // Max task payload size
	MaxTaskDuration time.Duration // Max execution time per task
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled bool
	Rate    float64 // Tokens per second
	Burst   int     // Max burst size
}

// PersistenceConfig configures task persistence
type PersistenceConfig struct {
	Enabled       bool
	Backend       string // "redis", "disk", "postgres"
	RedisURL      string
	DiskPath      string
	BatchSize     int           // Batch writes for performance
	FlushInterval time.Duration
}

// RetryConfig configures task retry behavior
type RetryConfig struct {
	Enabled      bool
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool // Add randomness to prevent thundering herd
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int           // Open after N failures
	SuccessThreshold int           // Close after N successes
	Timeout          time.Duration // Half-open after timeout
	HalfOpenMaxCalls int           // Max calls in half-open state
}

// MultiTenancyConfig configures multi-tenancy features
type MultiTenancyConfig struct {
	Enabled         bool
	DefaultMaxTasks int     // Default max tasks per tenant
	DefaultCPUQuota float64 // Default CPU quota per tenant
	DefaultMemoryMB int64   // Default memory quota per tenant
}

// TenantConfig configures individual tenant settings
type TenantConfig struct {
	TenantID     string
	MaxWorkers   int     // Max workers for this tenant
	MaxQueueSize int     // Max queue size
	CPUQuota     float64 // CPU percentage (0-100)
	MemoryQuota  int64   // Memory in MB
	RateLimit    float64 // Tasks per second
	Priority     Priority // Tenant priority level
}

// SecurityConfig configures security features
type SecurityConfig struct {
	AuthEnabled       bool
	AuthProvider      string // "jwt", "oauth", "apikey"
	EncryptionEnabled bool
	EncryptionKey     []byte
	AuditLog          bool
}

// DashboardConfig configures real-time dashboard
type DashboardConfig struct {
	Enabled bool
	Port    int
	Path    string // Dashboard HTTP path
}

// AlertConfig configures alerting
type AlertConfig struct {
	Enabled  bool
	Rules    []AlertRule
	Channels []AlertChannelConfig
}

// AlertRule defines an alerting rule
type AlertRule struct {
	Name        string
	Condition   string            // "queue_size > 1000", "error_rate > 0.1"
	Duration    time.Duration     // Must be true for this long
	Severity    string            // "warning", "error", "critical"
	Annotations map[string]string
}

// AlertChannelConfig configures alert channels
type AlertChannelConfig struct {
	Type   string            // "slack", "pagerduty", "webhook"
	Config map[string]string // Channel-specific config
}

// CostConfig configures cost tracking
type CostConfig struct {
	Enabled         bool
	CPUCostPerMs    float64 // Cost per CPU millisecond
	MemoryCostPerMB float64 // Cost per MB-second
	TaskCostBase    float64 // Base cost per task
}

// FeatureFlags controls experimental features
type FeatureFlags struct {
	EnableTracing      bool
	EnablePersistence  bool
	EnableCostTracking bool
	EnableMultiTenancy bool
	ExperimentalFeatures map[string]bool
}

// DefaultConfig returns a configuration with sensible defaults
// All enterprise features are disabled by default (nil pointers)
func DefaultConfig() Config {
	return Config{
		Workers:         runtime.NumCPU(),
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
		EnableMetrics:   true,
		ErrorHandler:    nil,
		// Enterprise features are nil by default (disabled)
	}
}

// NewEnterpriseConfig returns a config with enterprise features enabled
func NewEnterpriseConfig() Config {
	return Config{
		Workers:         runtime.NumCPU(),
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
		EnableMetrics:   true,

		Telemetry: &TelemetryConfig{
			Enabled:         true,
			ServiceName:     "workerpool",
			ServiceVersion:  "1.0.0",
			MetricsExporter: "prometheus",
			MetricsInterval: 10 * time.Second,
			TracingExporter: "jaeger",
			SamplingRate:    0.1,
			LogLevel:        "info",
			LogExporter:     "stdout",
			LogSampling: LogSamplingConfig{
				Initial:    10,
				Thereafter: 100,
			},
		},

		Resource: &ResourceConfig{
			MaxCPUPercent:   80.0,
			CPUThrottle:     true,
			MaxMemoryMB:     1024,
			MemoryThrottle:  true,
			MaxTaskDuration: 5 * time.Minute,
		},

		RateLimit: &RateLimitConfig{
			Enabled: true,
			Rate:    1000.0,
			Burst:   100,
		},

		Retry: &RetryConfig{
			Enabled:      true,
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
		},

		CircuitBreaker: &CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
			HalfOpenMaxCalls: 3,
		},

		Features: &FeatureFlags{
			EnableTracing:        true,
			EnablePersistence:    false,
			EnableCostTracking:   false,
			EnableMultiTenancy:   false,
			ExperimentalFeatures: make(map[string]bool),
		},
	}
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.Workers <= 0 {
		return fmt.Errorf("%w: workers must be > 0, got %d", ErrInvalidConfig, c.Workers)
	}
	if c.QueueSize < 0 {
		return fmt.Errorf("%w: queue size must be >= 0, got %d", ErrInvalidConfig, c.QueueSize)
	}
	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("%w: shutdown timeout must be >= 0, got %v", ErrInvalidConfig, c.ShutdownTimeout)
	}
	return nil
}
