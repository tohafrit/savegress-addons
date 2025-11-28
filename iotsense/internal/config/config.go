package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for IoTSense
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
	Devices   DevicesConfig   `yaml:"devices"`
	Alerts    AlertsConfig    `yaml:"alerts"`
	Storage   StorageConfig   `yaml:"storage"`
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

// TelemetryConfig holds telemetry ingestion configuration
type TelemetryConfig struct {
	BatchSize       int           `yaml:"batch_size"`
	FlushInterval   time.Duration `yaml:"flush_interval"`
	BufferSize      int           `yaml:"buffer_size"`
	MaxDataAge      time.Duration `yaml:"max_data_age"`
	CompressionEnabled bool       `yaml:"compression_enabled"`
	ValidationEnabled  bool       `yaml:"validation_enabled"`
}

// DevicesConfig holds device management configuration
type DevicesConfig struct {
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	OfflineThreshold  time.Duration `yaml:"offline_threshold"`
	MaxDevices        int           `yaml:"max_devices"`
	AutoDiscovery     bool          `yaml:"auto_discovery"`
}

// AlertsConfig holds alerting configuration
type AlertsConfig struct {
	Enabled         bool          `yaml:"enabled"`
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`
	RetentionDays   int           `yaml:"retention_days"`
	MaxActiveAlerts int           `yaml:"max_active_alerts"`
	Channels        AlertChannels `yaml:"channels"`
}

// AlertChannels holds alert channel configurations
type AlertChannels struct {
	Slack   SlackConfig   `yaml:"slack"`
	Email   EmailConfig   `yaml:"email"`
	Webhook WebhookConfig `yaml:"webhook"`
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

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

// StorageConfig holds time-series storage configuration
type StorageConfig struct {
	Type            string        `yaml:"type"` // memory, timescaledb, influxdb
	RetentionPeriod time.Duration `yaml:"retention_period"`
	DownsampleAfter time.Duration `yaml:"downsample_after"`
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
			Port:        getEnvInt("PORT", 3006),
			Environment: getEnv("ENVIRONMENT", "development"),
			JWTSecret:   getEnv("JWT_SECRET", ""),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", "postgres://iotsense:iotsense@localhost:5432/iotsense"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		Telemetry: TelemetryConfig{
			BatchSize:          getEnvInt("TELEMETRY_BATCH_SIZE", 1000),
			FlushInterval:      getEnvDuration("TELEMETRY_FLUSH_INTERVAL", 5*time.Second),
			BufferSize:         getEnvInt("TELEMETRY_BUFFER_SIZE", 100000),
			MaxDataAge:         getEnvDuration("TELEMETRY_MAX_AGE", 24*time.Hour),
			CompressionEnabled: getEnvBool("TELEMETRY_COMPRESSION", true),
			ValidationEnabled:  getEnvBool("TELEMETRY_VALIDATION", true),
		},
		Devices: DevicesConfig{
			HeartbeatInterval: getEnvDuration("DEVICE_HEARTBEAT", 30*time.Second),
			OfflineThreshold:  getEnvDuration("DEVICE_OFFLINE_THRESHOLD", 5*time.Minute),
			MaxDevices:        getEnvInt("MAX_DEVICES", 10000),
			AutoDiscovery:     getEnvBool("AUTO_DISCOVERY", true),
		},
		Alerts: AlertsConfig{
			Enabled:            getEnvBool("ALERTS_ENABLED", true),
			EvaluationInterval: getEnvDuration("ALERT_EVAL_INTERVAL", 10*time.Second),
			RetentionDays:      getEnvInt("ALERT_RETENTION_DAYS", 90),
			MaxActiveAlerts:    getEnvInt("MAX_ACTIVE_ALERTS", 1000),
		},
		Storage: StorageConfig{
			Type:            getEnv("STORAGE_TYPE", "memory"),
			RetentionPeriod: getEnvDuration("STORAGE_RETENTION", 7*24*time.Hour),
			DownsampleAfter: getEnvDuration("STORAGE_DOWNSAMPLE", 24*time.Hour),
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
