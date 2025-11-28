package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Create a temp config file
	configContent := `
server:
  port: 8080
  environment: production
  jwt_secret: "test-secret"
database:
  url: "postgres://localhost/testdb"
  max_conns: 50
  min_conns: 10
redis:
  url: "redis://localhost:6379"
metrics:
  auto_discover: true
  retention: 720h
  storage: prometheus
anomaly:
  enabled: true
  algorithms:
    - statistical
    - seasonal
  sensitivity: high
  baseline_window: 168h
quality:
  enabled: true
  default_rules: true
  score_threshold: 95.0
schema:
  track_changes: true
  alert_on_breaking: true
alerts:
  evaluation_interval: 30s
  channels:
    slack:
      webhook_url: "https://hooks.slack.com/test"
      channel: "#alerts"
    email:
      smtp_host: "smtp.example.com"
      smtp_port: 587
      from: "alerts@example.com"
      username: "user"
      password: "pass"
    pagerduty:
      service_key: "test-key"
    webhook:
      url: "https://webhook.example.com"
      headers:
        X-Custom: "value"
storage:
  embedded:
    path: "/var/data"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Server
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "production" {
		t.Errorf("expected environment 'production', got '%s'", cfg.Server.Environment)
	}
	if cfg.Server.JWTSecret != "test-secret" {
		t.Errorf("expected jwt_secret 'test-secret', got '%s'", cfg.Server.JWTSecret)
	}

	// Database
	if cfg.Database.URL != "postgres://localhost/testdb" {
		t.Errorf("expected db url, got '%s'", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 50 {
		t.Errorf("expected max_conns 50, got %d", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 10 {
		t.Errorf("expected min_conns 10, got %d", cfg.Database.MinConns)
	}

	// Redis
	if cfg.Redis.URL != "redis://localhost:6379" {
		t.Errorf("expected redis url, got '%s'", cfg.Redis.URL)
	}

	// Metrics
	if !cfg.Metrics.AutoDiscover {
		t.Error("expected auto_discover true")
	}
	if cfg.Metrics.Retention != 720*time.Hour {
		t.Errorf("expected retention 720h, got %v", cfg.Metrics.Retention)
	}
	if cfg.Metrics.StorageType != "prometheus" {
		t.Errorf("expected storage 'prometheus', got '%s'", cfg.Metrics.StorageType)
	}

	// Anomaly
	if !cfg.Anomaly.Enabled {
		t.Error("expected anomaly enabled")
	}
	if len(cfg.Anomaly.Algorithms) != 2 {
		t.Errorf("expected 2 algorithms, got %d", len(cfg.Anomaly.Algorithms))
	}
	if cfg.Anomaly.Sensitivity != "high" {
		t.Errorf("expected sensitivity 'high', got '%s'", cfg.Anomaly.Sensitivity)
	}
	if cfg.Anomaly.BaselineWindow != 168*time.Hour {
		t.Errorf("expected baseline_window 168h, got %v", cfg.Anomaly.BaselineWindow)
	}

	// Quality
	if !cfg.Quality.Enabled {
		t.Error("expected quality enabled")
	}
	if !cfg.Quality.DefaultRules {
		t.Error("expected default_rules true")
	}
	if cfg.Quality.ScoreThreshold != 95.0 {
		t.Errorf("expected score_threshold 95.0, got %f", cfg.Quality.ScoreThreshold)
	}

	// Schema
	if !cfg.Schema.TrackChanges {
		t.Error("expected track_changes true")
	}
	if !cfg.Schema.AlertOnBreaking {
		t.Error("expected alert_on_breaking true")
	}

	// Alerts
	if cfg.Alerts.EvaluationInterval != 30*time.Second {
		t.Errorf("expected evaluation_interval 30s, got %v", cfg.Alerts.EvaluationInterval)
	}

	// Alert Channels
	if cfg.Alerts.Channels.Slack == nil {
		t.Fatal("expected slack config")
	}
	if cfg.Alerts.Channels.Slack.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("expected slack webhook url, got '%s'", cfg.Alerts.Channels.Slack.WebhookURL)
	}
	if cfg.Alerts.Channels.Slack.Channel != "#alerts" {
		t.Errorf("expected slack channel '#alerts', got '%s'", cfg.Alerts.Channels.Slack.Channel)
	}

	if cfg.Alerts.Channels.Email == nil {
		t.Fatal("expected email config")
	}
	if cfg.Alerts.Channels.Email.SMTPHost != "smtp.example.com" {
		t.Errorf("expected smtp host, got '%s'", cfg.Alerts.Channels.Email.SMTPHost)
	}
	if cfg.Alerts.Channels.Email.SMTPPort != 587 {
		t.Errorf("expected smtp port 587, got %d", cfg.Alerts.Channels.Email.SMTPPort)
	}

	if cfg.Alerts.Channels.PagerDuty == nil {
		t.Fatal("expected pagerduty config")
	}
	if cfg.Alerts.Channels.PagerDuty.ServiceKey != "test-key" {
		t.Errorf("expected pagerduty service key, got '%s'", cfg.Alerts.Channels.PagerDuty.ServiceKey)
	}

	if cfg.Alerts.Channels.Webhook == nil {
		t.Fatal("expected webhook config")
	}
	if cfg.Alerts.Channels.Webhook.URL != "https://webhook.example.com" {
		t.Errorf("expected webhook url, got '%s'", cfg.Alerts.Channels.Webhook.URL)
	}
	if cfg.Alerts.Channels.Webhook.Headers["X-Custom"] != "value" {
		t.Errorf("expected custom header, got '%s'", cfg.Alerts.Channels.Webhook.Headers["X-Custom"])
	}

	// Storage
	if cfg.Storage.Embedded == nil {
		t.Fatal("expected embedded storage config")
	}
	if cfg.Storage.Embedded.Path != "/var/data" {
		t.Errorf("expected path '/var/data', got '%s'", cfg.Storage.Embedded.Path)
	}
}

func TestLoadWithEnvExpansion(t *testing.T) {
	configContent := `
server:
  port: 8080
  jwt_secret: "${TEST_JWT_SECRET}"
database:
  url: "${TEST_DB_URL}"
`

	os.Setenv("TEST_JWT_SECRET", "secret-from-env")
	os.Setenv("TEST_DB_URL", "postgres://env/db")
	defer func() {
		os.Unsetenv("TEST_JWT_SECRET")
		os.Unsetenv("TEST_DB_URL")
	}()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.JWTSecret != "secret-from-env" {
		t.Errorf("expected jwt_secret from env, got '%s'", cfg.Server.JWTSecret)
	}
	if cfg.Database.URL != "postgres://env/db" {
		t.Errorf("expected db url from env, got '%s'", cfg.Database.URL)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	invalidYAML := `
server:
  port: "not-a-number"  # string instead of int, may work
invalid yaml:: content
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromEnv(t *testing.T) {
	cfg := LoadFromEnv()
	if cfg == nil {
		t.Fatal("expected config")
	}

	// Check defaults are set
	if cfg.Server.Port != 3002 {
		t.Errorf("expected default port 3002, got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "development" {
		t.Errorf("expected default environment 'development', got '%s'", cfg.Server.Environment)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("expected default max_conns 25, got %d", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 5 {
		t.Errorf("expected default min_conns 5, got %d", cfg.Database.MinConns)
	}
	if cfg.Metrics.Retention != 30*24*time.Hour {
		t.Errorf("expected default retention 30 days, got %v", cfg.Metrics.Retention)
	}
	if cfg.Metrics.StorageType != "embedded" {
		t.Errorf("expected default storage 'embedded', got '%s'", cfg.Metrics.StorageType)
	}
	if cfg.Anomaly.Sensitivity != "medium" {
		t.Errorf("expected default sensitivity 'medium', got '%s'", cfg.Anomaly.Sensitivity)
	}
	if cfg.Anomaly.BaselineWindow != 7*24*time.Hour {
		t.Errorf("expected default baseline_window 7 days, got %v", cfg.Anomaly.BaselineWindow)
	}
	if cfg.Quality.ScoreThreshold != 90.0 {
		t.Errorf("expected default score_threshold 90.0, got %f", cfg.Quality.ScoreThreshold)
	}
	if cfg.Alerts.EvaluationInterval != time.Minute {
		t.Errorf("expected default evaluation_interval 1m, got %v", cfg.Alerts.EvaluationInterval)
	}
}

func TestLoadFromEnvWithOverrides(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://envdb/test")
	os.Setenv("REDIS_URL", "redis://envredis:6379")
	os.Setenv("JWT_SECRET", "env-secret")
	os.Setenv("SLACK_WEBHOOK_URL", "https://env.slack.webhook")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("SLACK_WEBHOOK_URL")
	}()

	cfg := LoadFromEnv()

	if cfg.Database.URL != "postgres://envdb/test" {
		t.Errorf("expected db url from env, got '%s'", cfg.Database.URL)
	}
	if cfg.Redis.URL != "redis://envredis:6379" {
		t.Errorf("expected redis url from env, got '%s'", cfg.Redis.URL)
	}
	if cfg.Server.JWTSecret != "env-secret" {
		t.Errorf("expected jwt_secret from env, got '%s'", cfg.Server.JWTSecret)
	}
	if cfg.Alerts.Channels.Slack == nil {
		t.Fatal("expected slack config from env")
	}
	if cfg.Alerts.Channels.Slack.WebhookURL != "https://env.slack.webhook" {
		t.Errorf("expected slack webhook from env, got '%s'", cfg.Alerts.Channels.Slack.WebhookURL)
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	setDefaults(cfg)

	if cfg.Server.Port != 3002 {
		t.Errorf("expected port 3002, got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "development" {
		t.Errorf("expected environment 'development', got '%s'", cfg.Server.Environment)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("expected max_conns 25, got %d", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 5 {
		t.Errorf("expected min_conns 5, got %d", cfg.Database.MinConns)
	}
	if cfg.Metrics.Retention != 30*24*time.Hour {
		t.Errorf("expected retention 30 days, got %v", cfg.Metrics.Retention)
	}
	if cfg.Metrics.StorageType != "embedded" {
		t.Errorf("expected storage 'embedded', got '%s'", cfg.Metrics.StorageType)
	}
	if cfg.Anomaly.Sensitivity != "medium" {
		t.Errorf("expected sensitivity 'medium', got '%s'", cfg.Anomaly.Sensitivity)
	}
	if cfg.Anomaly.BaselineWindow != 7*24*time.Hour {
		t.Errorf("expected baseline_window 7 days, got %v", cfg.Anomaly.BaselineWindow)
	}
	if cfg.Quality.ScoreThreshold != 90.0 {
		t.Errorf("expected score_threshold 90.0, got %f", cfg.Quality.ScoreThreshold)
	}
	if cfg.Alerts.EvaluationInterval != time.Minute {
		t.Errorf("expected evaluation_interval 1m, got %v", cfg.Alerts.EvaluationInterval)
	}
}

func TestSetDefaultsDoesNotOverride(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:        9999,
			Environment: "production",
		},
		Database: DatabaseConfig{
			MaxConns: 100,
			MinConns: 20,
		},
		Metrics: MetricsConfig{
			Retention:   1000 * time.Hour,
			StorageType: "prometheus",
		},
		Anomaly: AnomalyConfig{
			Sensitivity:    "high",
			BaselineWindow: 100 * time.Hour,
		},
		Quality: QualityConfig{
			ScoreThreshold: 99.9,
		},
		Alerts: AlertsConfig{
			EvaluationInterval: 5 * time.Minute,
		},
	}

	setDefaults(cfg)

	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999 (not overwritten), got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "production" {
		t.Errorf("expected environment 'production' (not overwritten), got '%s'", cfg.Server.Environment)
	}
	if cfg.Database.MaxConns != 100 {
		t.Errorf("expected max_conns 100 (not overwritten), got %d", cfg.Database.MaxConns)
	}
	if cfg.Metrics.StorageType != "prometheus" {
		t.Errorf("expected storage 'prometheus' (not overwritten), got '%s'", cfg.Metrics.StorageType)
	}
	if cfg.Anomaly.Sensitivity != "high" {
		t.Errorf("expected sensitivity 'high' (not overwritten), got '%s'", cfg.Anomaly.Sensitivity)
	}
	if cfg.Quality.ScoreThreshold != 99.9 {
		t.Errorf("expected score_threshold 99.9 (not overwritten), got %f", cfg.Quality.ScoreThreshold)
	}
}

func TestLoadWithAllStorageTypes(t *testing.T) {
	configContent := `
server:
  port: 8080
storage:
  embedded:
    path: "/var/embedded"
  prometheus:
    url: "http://prometheus:9090"
    remote_write: true
  influxdb:
    url: "http://influxdb:8086"
    token: "test-token"
    org: "test-org"
    bucket: "metrics"
  timescale:
    url: "postgres://timescale/db"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Embedded
	if cfg.Storage.Embedded == nil {
		t.Fatal("expected embedded config")
	}
	if cfg.Storage.Embedded.Path != "/var/embedded" {
		t.Errorf("expected embedded path, got '%s'", cfg.Storage.Embedded.Path)
	}

	// Prometheus
	if cfg.Storage.Prometheus == nil {
		t.Fatal("expected prometheus config")
	}
	if cfg.Storage.Prometheus.URL != "http://prometheus:9090" {
		t.Errorf("expected prometheus url, got '%s'", cfg.Storage.Prometheus.URL)
	}
	if !cfg.Storage.Prometheus.RemoteWrite {
		t.Error("expected remote_write true")
	}

	// InfluxDB
	if cfg.Storage.InfluxDB == nil {
		t.Fatal("expected influxdb config")
	}
	if cfg.Storage.InfluxDB.URL != "http://influxdb:8086" {
		t.Errorf("expected influxdb url, got '%s'", cfg.Storage.InfluxDB.URL)
	}
	if cfg.Storage.InfluxDB.Token != "test-token" {
		t.Errorf("expected influxdb token, got '%s'", cfg.Storage.InfluxDB.Token)
	}
	if cfg.Storage.InfluxDB.Org != "test-org" {
		t.Errorf("expected influxdb org, got '%s'", cfg.Storage.InfluxDB.Org)
	}
	if cfg.Storage.InfluxDB.Bucket != "metrics" {
		t.Errorf("expected influxdb bucket, got '%s'", cfg.Storage.InfluxDB.Bucket)
	}

	// Timescale
	if cfg.Storage.Timescale == nil {
		t.Fatal("expected timescale config")
	}
	if cfg.Storage.Timescale.URL != "postgres://timescale/db" {
		t.Errorf("expected timescale url, got '%s'", cfg.Storage.Timescale.URL)
	}
}

func TestLoadMinimalConfig(t *testing.T) {
	configContent := `
server:
  port: 8080
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// User-specified value
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	// Defaults should be applied
	if cfg.Server.Environment != "development" {
		t.Errorf("expected default environment 'development', got '%s'", cfg.Server.Environment)
	}
	if cfg.Metrics.StorageType != "embedded" {
		t.Errorf("expected default storage 'embedded', got '%s'", cfg.Metrics.StorageType)
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	configContent := ``

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// All defaults should be applied
	if cfg.Server.Port != 3002 {
		t.Errorf("expected default port 3002, got %d", cfg.Server.Port)
	}
}
