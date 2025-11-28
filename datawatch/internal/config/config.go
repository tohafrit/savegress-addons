package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Anomaly  AnomalyConfig  `yaml:"anomaly"`
	Quality  QualityConfig  `yaml:"quality"`
	Schema   SchemaConfig   `yaml:"schema"`
	Alerts   AlertsConfig   `yaml:"alerts"`
	Storage  StorageConfig  `yaml:"storage"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
	JWTSecret   string `yaml:"jwt_secret"`
}

type DatabaseConfig struct {
	URL             string `yaml:"url"`
	MaxConns        int    `yaml:"max_conns"`
	MinConns        int    `yaml:"min_conns"`
	MaxConnLifetime string `yaml:"max_conn_lifetime"`
}

type RedisConfig struct {
	URL string `yaml:"url"`
}

type MetricsConfig struct {
	AutoDiscover bool          `yaml:"auto_discover"`
	Retention    time.Duration `yaml:"retention"`
	StorageType  string        `yaml:"storage"` // embedded, prometheus, influxdb, timescale
}

type AnomalyConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Algorithms     []string      `yaml:"algorithms"` // statistical, seasonal, ml
	Sensitivity    string        `yaml:"sensitivity"` // low, medium, high
	BaselineWindow time.Duration `yaml:"baseline_window"`
}

type QualityConfig struct {
	Enabled        bool    `yaml:"enabled"`
	DefaultRules   bool    `yaml:"default_rules"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

type SchemaConfig struct {
	TrackChanges    bool `yaml:"track_changes"`
	AlertOnBreaking bool `yaml:"alert_on_breaking"`
}

type AlertsConfig struct {
	EvaluationInterval time.Duration         `yaml:"evaluation_interval"`
	Channels           AlertChannelsConfig   `yaml:"channels"`
}

type AlertChannelsConfig struct {
	Slack     *SlackConfig     `yaml:"slack,omitempty"`
	Email     *EmailConfig     `yaml:"email,omitempty"`
	PagerDuty *PagerDutyConfig `yaml:"pagerduty,omitempty"`
	Webhook   *WebhookConfig   `yaml:"webhook,omitempty"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

type EmailConfig struct {
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	From     string   `yaml:"from"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type PagerDutyConfig struct {
	ServiceKey string `yaml:"service_key"`
}

type WebhookConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

type StorageConfig struct {
	Embedded   *EmbeddedStorageConfig   `yaml:"embedded,omitempty"`
	Prometheus *PrometheusStorageConfig `yaml:"prometheus,omitempty"`
	InfluxDB   *InfluxDBStorageConfig   `yaml:"influxdb,omitempty"`
	Timescale  *TimescaleStorageConfig  `yaml:"timescale,omitempty"`
}

type EmbeddedStorageConfig struct {
	Path string `yaml:"path"`
}

type PrometheusStorageConfig struct {
	URL         string `yaml:"url"`
	RemoteWrite bool   `yaml:"remote_write"`
}

type InfluxDBStorageConfig struct {
	URL    string `yaml:"url"`
	Token  string `yaml:"token"`
	Org    string `yaml:"org"`
	Bucket string `yaml:"bucket"`
}

type TimescaleStorageConfig struct {
	URL string `yaml:"url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	setDefaults(&cfg)
	return &cfg, nil
}

func LoadFromEnv() *Config {
	cfg := &Config{}
	setDefaults(cfg)

	// Override from environment
	if port := os.Getenv("PORT"); port != "" {
		// Parse port
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.Redis.URL = redisURL
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Server.JWTSecret = jwtSecret
	}
	if slackWebhook := os.Getenv("SLACK_WEBHOOK_URL"); slackWebhook != "" {
		if cfg.Alerts.Channels.Slack == nil {
			cfg.Alerts.Channels.Slack = &SlackConfig{}
		}
		cfg.Alerts.Channels.Slack.WebhookURL = slackWebhook
	}

	return cfg
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3002
	}
	if cfg.Server.Environment == "" {
		cfg.Server.Environment = "development"
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 25
	}
	if cfg.Database.MinConns == 0 {
		cfg.Database.MinConns = 5
	}
	if cfg.Metrics.Retention == 0 {
		cfg.Metrics.Retention = 30 * 24 * time.Hour // 30 days
	}
	if cfg.Metrics.StorageType == "" {
		cfg.Metrics.StorageType = "embedded"
	}
	if cfg.Anomaly.Sensitivity == "" {
		cfg.Anomaly.Sensitivity = "medium"
	}
	if cfg.Anomaly.BaselineWindow == 0 {
		cfg.Anomaly.BaselineWindow = 7 * 24 * time.Hour // 7 days
	}
	if cfg.Quality.ScoreThreshold == 0 {
		cfg.Quality.ScoreThreshold = 90.0
	}
	if cfg.Alerts.EvaluationInterval == 0 {
		cfg.Alerts.EvaluationInterval = time.Minute
	}
}
