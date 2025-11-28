package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for HealthSync
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	FHIR       FHIRConfig       `yaml:"fhir"`
	Compliance ComplianceConfig `yaml:"compliance"`
	Audit      AuditConfig      `yaml:"audit"`
	Consent    ConsentConfig    `yaml:"consent"`
	Encryption EncryptionConfig `yaml:"encryption"`
	Alerts     AlertsConfig     `yaml:"alerts"`
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

// FHIRConfig holds FHIR configuration
type FHIRConfig struct {
	Version           string   `yaml:"version"`
	SupportedResources []string `yaml:"supported_resources"`
	ValidationEnabled bool     `yaml:"validation_enabled"`
	ProfileURL        string   `yaml:"profile_url"`
}

// ComplianceConfig holds HIPAA compliance configuration
type ComplianceConfig struct {
	HIPAAEnabled      bool          `yaml:"hipaa_enabled"`
	MinimumNecessary  bool          `yaml:"minimum_necessary"`
	BreachNotification bool         `yaml:"breach_notification"`
	RetentionPeriod   time.Duration `yaml:"retention_period"`
	PHICategories     []string      `yaml:"phi_categories"`
}

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	Enabled         bool          `yaml:"enabled"`
	RetentionDays   int           `yaml:"retention_days"`
	DetailLevel     string        `yaml:"detail_level"`
	RealTimeAlerts  bool          `yaml:"realtime_alerts"`
	StorageBackend  string        `yaml:"storage_backend"`
}

// ConsentConfig holds consent management configuration
type ConsentConfig struct {
	Required           bool     `yaml:"required"`
	DefaultPolicy      string   `yaml:"default_policy"`
	AllowedPurposes    []string `yaml:"allowed_purposes"`
	ExpirationDays     int      `yaml:"expiration_days"`
	GranularityLevel   string   `yaml:"granularity_level"`
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	AtRestEnabled   bool   `yaml:"at_rest_enabled"`
	InTransitEnabled bool  `yaml:"in_transit_enabled"`
	Algorithm       string `yaml:"algorithm"`
	KeyRotationDays int    `yaml:"key_rotation_days"`
}

// AlertsConfig holds alerting configuration
type AlertsConfig struct {
	Channels AlertChannels `yaml:"channels"`
}

// AlertChannels holds alert channel configurations
type AlertChannels struct {
	Slack SlackConfig `yaml:"slack"`
	Email EmailConfig `yaml:"email"`
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
			Port:        getEnvInt("PORT", 3005),
			Environment: getEnv("ENVIRONMENT", "development"),
			JWTSecret:   getEnv("JWT_SECRET", ""),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", "postgres://healthsync:healthsync@localhost:5432/healthsync"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		FHIR: FHIRConfig{
			Version:           getEnv("FHIR_VERSION", "R4"),
			SupportedResources: []string{"Patient", "Practitioner", "Organization", "Encounter", "Observation", "Condition", "Medication", "MedicationRequest"},
			ValidationEnabled: getEnvBool("FHIR_VALIDATION", true),
		},
		Compliance: ComplianceConfig{
			HIPAAEnabled:       getEnvBool("HIPAA_ENABLED", true),
			MinimumNecessary:   getEnvBool("MINIMUM_NECESSARY", true),
			BreachNotification: getEnvBool("BREACH_NOTIFICATION", true),
			RetentionPeriod:    getEnvDuration("RETENTION_PERIOD", 6*365*24*time.Hour), // 6 years
		},
		Audit: AuditConfig{
			Enabled:        getEnvBool("AUDIT_ENABLED", true),
			RetentionDays:  getEnvInt("AUDIT_RETENTION_DAYS", 2190), // 6 years
			DetailLevel:    getEnv("AUDIT_DETAIL_LEVEL", "full"),
			RealTimeAlerts: getEnvBool("AUDIT_REALTIME_ALERTS", true),
			StorageBackend: getEnv("AUDIT_STORAGE", "database"),
		},
		Consent: ConsentConfig{
			Required:        getEnvBool("CONSENT_REQUIRED", true),
			DefaultPolicy:   getEnv("CONSENT_DEFAULT_POLICY", "deny"),
			AllowedPurposes: []string{"treatment", "payment", "operations", "research"},
			ExpirationDays:  getEnvInt("CONSENT_EXPIRATION_DAYS", 365),
			GranularityLevel: getEnv("CONSENT_GRANULARITY", "resource"),
		},
		Encryption: EncryptionConfig{
			AtRestEnabled:    getEnvBool("ENCRYPTION_AT_REST", true),
			InTransitEnabled: getEnvBool("ENCRYPTION_IN_TRANSIT", true),
			Algorithm:        getEnv("ENCRYPTION_ALGORITHM", "AES-256-GCM"),
			KeyRotationDays:  getEnvInt("KEY_ROTATION_DAYS", 90),
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
