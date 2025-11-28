package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for FinSight
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Redis         RedisConfig         `yaml:"redis"`
	Transactions  TransactionsConfig  `yaml:"transactions"`
	Fraud         FraudConfig         `yaml:"fraud"`
	Reconciliation ReconciliationConfig `yaml:"reconciliation"`
	Reporting     ReportingConfig     `yaml:"reporting"`
	Compliance    ComplianceConfig    `yaml:"compliance"`
	Alerts        AlertsConfig        `yaml:"alerts"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
	JWTSecret   string `yaml:"jwt_secret"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL      string `yaml:"url"`
	MaxConns int    `yaml:"max_conns"`
	MinConns int    `yaml:"min_conns"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL string `yaml:"url"`
}

// TransactionsConfig holds transaction processing configuration
type TransactionsConfig struct {
	BatchSize        int           `yaml:"batch_size"`
	ProcessInterval  time.Duration `yaml:"process_interval"`
	RetentionDays    int           `yaml:"retention_days"`
	CategorizationEnabled bool     `yaml:"categorization_enabled"`
}

// FraudConfig holds fraud detection configuration
type FraudConfig struct {
	Enabled           bool          `yaml:"enabled"`
	RealTimeScoring   bool          `yaml:"realtime_scoring"`
	ScoreThreshold    float64       `yaml:"score_threshold"`
	VelocityWindow    time.Duration `yaml:"velocity_window"`
	MaxDailyAmount    float64       `yaml:"max_daily_amount"`
	MaxSingleAmount   float64       `yaml:"max_single_amount"`
	GeofencingEnabled bool          `yaml:"geofencing_enabled"`
	MLModelPath       string        `yaml:"ml_model_path"`
}

// ReconciliationConfig holds reconciliation configuration
type ReconciliationConfig struct {
	AutoReconcile    bool          `yaml:"auto_reconcile"`
	MatchTolerance   float64       `yaml:"match_tolerance"`
	DateTolerance    time.Duration `yaml:"date_tolerance"`
	BatchSize        int           `yaml:"batch_size"`
	ScheduleCron     string        `yaml:"schedule_cron"`
}

// ReportingConfig holds reporting configuration
type ReportingConfig struct {
	Enabled          bool     `yaml:"enabled"`
	StoragePath      string   `yaml:"storage_path"`
	RetentionDays    int      `yaml:"retention_days"`
	DefaultFormats   []string `yaml:"default_formats"`
	ScheduledReports []ScheduledReport `yaml:"scheduled_reports"`
}

// ScheduledReport represents a scheduled report configuration
type ScheduledReport struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Period   string `yaml:"period"`
	Schedule string `yaml:"schedule"`
	Format   string `yaml:"format"`
	Recipients []string `yaml:"recipients"`
}

// ComplianceConfig holds compliance configuration
type ComplianceConfig struct {
	AMLEnabled       bool     `yaml:"aml_enabled"`
	KYCRequired      bool     `yaml:"kyc_required"`
	SARThreshold     float64  `yaml:"sar_threshold"`
	CTRThreshold     float64  `yaml:"ctr_threshold"`
	WatchlistEnabled bool     `yaml:"watchlist_enabled"`
	AuditLogRetention int     `yaml:"audit_log_retention"`
}

// AlertsConfig holds alerting configuration
type AlertsConfig struct {
	Channels AlertChannels `yaml:"channels"`
}

// AlertChannels holds alert channel configurations
type AlertChannels struct {
	Slack SlackConfig `yaml:"slack"`
	Email EmailConfig `yaml:"email"`
	PagerDuty PagerDutyConfig `yaml:"pagerduty"`
}

// SlackConfig holds Slack configuration
type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost   string   `yaml:"smtp_host"`
	SMTPPort   int      `yaml:"smtp_port"`
	From       string   `yaml:"from"`
	Recipients []string `yaml:"recipients"`
}

// PagerDutyConfig holds PagerDuty configuration
type PagerDutyConfig struct {
	ServiceKey string `yaml:"service_key"`
	Enabled    bool   `yaml:"enabled"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        getEnvInt("PORT", 3004),
			Environment: getEnv("ENVIRONMENT", "development"),
			JWTSecret:   getEnv("JWT_SECRET", ""),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", "postgres://finsight:finsight@localhost:5432/finsight"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		Transactions: TransactionsConfig{
			BatchSize:             getEnvInt("TXN_BATCH_SIZE", 1000),
			ProcessInterval:       getEnvDuration("TXN_PROCESS_INTERVAL", 5*time.Second),
			RetentionDays:         getEnvInt("TXN_RETENTION_DAYS", 365),
			CategorizationEnabled: getEnvBool("TXN_CATEGORIZATION", true),
		},
		Fraud: FraudConfig{
			Enabled:           getEnvBool("FRAUD_ENABLED", true),
			RealTimeScoring:   getEnvBool("FRAUD_REALTIME", true),
			ScoreThreshold:    getEnvFloat("FRAUD_THRESHOLD", 0.7),
			VelocityWindow:    getEnvDuration("FRAUD_VELOCITY_WINDOW", 1*time.Hour),
			MaxDailyAmount:    getEnvFloat("FRAUD_MAX_DAILY", 10000),
			MaxSingleAmount:   getEnvFloat("FRAUD_MAX_SINGLE", 5000),
			GeofencingEnabled: getEnvBool("FRAUD_GEOFENCING", true),
		},
		Reconciliation: ReconciliationConfig{
			AutoReconcile:  getEnvBool("RECON_AUTO", true),
			MatchTolerance: getEnvFloat("RECON_TOLERANCE", 0.01),
			DateTolerance:  getEnvDuration("RECON_DATE_TOLERANCE", 24*time.Hour),
			BatchSize:      getEnvInt("RECON_BATCH_SIZE", 5000),
			ScheduleCron:   getEnv("RECON_SCHEDULE", "0 2 * * *"),
		},
		Reporting: ReportingConfig{
			Enabled:        getEnvBool("REPORTING_ENABLED", true),
			StoragePath:    getEnv("REPORTING_PATH", "/var/lib/finsight/reports"),
			RetentionDays:  getEnvInt("REPORTING_RETENTION", 90),
			DefaultFormats: []string{"pdf", "csv", "xlsx"},
		},
		Compliance: ComplianceConfig{
			AMLEnabled:        getEnvBool("COMPLIANCE_AML", true),
			KYCRequired:       getEnvBool("COMPLIANCE_KYC", true),
			SARThreshold:      getEnvFloat("COMPLIANCE_SAR_THRESHOLD", 5000),
			CTRThreshold:      getEnvFloat("COMPLIANCE_CTR_THRESHOLD", 10000),
			WatchlistEnabled:  getEnvBool("COMPLIANCE_WATCHLIST", true),
			AuditLogRetention: getEnvInt("COMPLIANCE_AUDIT_RETENTION", 730),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
