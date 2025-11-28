package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to be non-nil")
	}

	// Check defaults
	if cfg.Port != 3001 {
		t.Errorf("expected default Port 3001, got %d", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default Environment 'development', got %s", cfg.Environment)
	}

	if cfg.JWTExpiration != 24 {
		t.Errorf("expected default JWTExpiration 24, got %d", cfg.JWTExpiration)
	}

	if cfg.RefreshExpiration != 30 {
		t.Errorf("expected default RefreshExpiration 30, got %d", cfg.RefreshExpiration)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTPPort 587, got %d", cfg.SMTPPort)
	}

	if cfg.FreeAPICallsPerMonth != 1000 {
		t.Errorf("expected default FreeAPICallsPerMonth 1000, got %d", cfg.FreeAPICallsPerMonth)
	}

	if cfg.ProAPICallsPerMonth != 10000 {
		t.Errorf("expected default ProAPICallsPerMonth 10000, got %d", cfg.ProAPICallsPerMonth)
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("DATABASE_URL", "postgres://test:test@testhost:5432/testdb")
	os.Setenv("REDIS_URL", "redis://testredis:6379")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_EXPIRATION_HOURS", "48")
	os.Setenv("REFRESH_EXPIRATION_DAYS", "60")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	os.Setenv("ETHEREUM_RPC_URL", "https://custom-eth.com")
	os.Setenv("POLYGON_RPC_URL", "https://custom-polygon.com")
	os.Setenv("ARBITRUM_RPC_URL", "https://custom-arbitrum.com")
	os.Setenv("ALCHEMY_API_KEY", "alchemy-key")
	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_PORT", "465")
	os.Setenv("SMTP_USER", "user@test.com")
	os.Setenv("SMTP_PASSWORD", "password123")
	os.Setenv("FROM_EMAIL", "test@chainlens.dev")
	os.Setenv("FREE_API_CALLS_PER_MONTH", "2000")
	os.Setenv("PRO_API_CALLS_PER_MONTH", "20000")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_EXPIRATION_HOURS")
		os.Unsetenv("REFRESH_EXPIRATION_DAYS")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
		os.Unsetenv("ETHEREUM_RPC_URL")
		os.Unsetenv("POLYGON_RPC_URL")
		os.Unsetenv("ARBITRUM_RPC_URL")
		os.Unsetenv("ALCHEMY_API_KEY")
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USER")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("FROM_EMAIL")
		os.Unsetenv("FREE_API_CALLS_PER_MONTH")
		os.Unsetenv("PRO_API_CALLS_PER_MONTH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}

	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got %s", cfg.Environment)
	}

	if cfg.DatabaseURL != "postgres://test:test@testhost:5432/testdb" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}

	if cfg.RedisURL != "redis://testredis:6379" {
		t.Errorf("unexpected RedisURL: %s", cfg.RedisURL)
	}

	if cfg.JWTSecret != "test-secret" {
		t.Errorf("unexpected JWTSecret: %s", cfg.JWTSecret)
	}

	if cfg.JWTExpiration != 48 {
		t.Errorf("expected JWTExpiration 48, got %d", cfg.JWTExpiration)
	}

	if cfg.RefreshExpiration != 60 {
		t.Errorf("expected RefreshExpiration 60, got %d", cfg.RefreshExpiration)
	}

	if cfg.StripeSecretKey != "sk_test_123" {
		t.Errorf("unexpected StripeSecretKey: %s", cfg.StripeSecretKey)
	}

	if cfg.StripeWebhookSecret != "whsec_123" {
		t.Errorf("unexpected StripeWebhookSecret: %s", cfg.StripeWebhookSecret)
	}

	if cfg.EthereumRPCURL != "https://custom-eth.com" {
		t.Errorf("unexpected EthereumRPCURL: %s", cfg.EthereumRPCURL)
	}

	if cfg.PolygonRPCURL != "https://custom-polygon.com" {
		t.Errorf("unexpected PolygonRPCURL: %s", cfg.PolygonRPCURL)
	}

	if cfg.ArbitrumRPCURL != "https://custom-arbitrum.com" {
		t.Errorf("unexpected ArbitrumRPCURL: %s", cfg.ArbitrumRPCURL)
	}

	if cfg.AlchemyAPIKey != "alchemy-key" {
		t.Errorf("unexpected AlchemyAPIKey: %s", cfg.AlchemyAPIKey)
	}

	if cfg.SMTPHost != "smtp.test.com" {
		t.Errorf("unexpected SMTPHost: %s", cfg.SMTPHost)
	}

	if cfg.SMTPPort != 465 {
		t.Errorf("expected SMTPPort 465, got %d", cfg.SMTPPort)
	}

	if cfg.SMTPUser != "user@test.com" {
		t.Errorf("unexpected SMTPUser: %s", cfg.SMTPUser)
	}

	if cfg.SMTPPassword != "password123" {
		t.Errorf("unexpected SMTPPassword")
	}

	if cfg.FromEmail != "test@chainlens.dev" {
		t.Errorf("unexpected FromEmail: %s", cfg.FromEmail)
	}

	if cfg.FreeAPICallsPerMonth != 2000 {
		t.Errorf("expected FreeAPICallsPerMonth 2000, got %d", cfg.FreeAPICallsPerMonth)
	}

	if cfg.ProAPICallsPerMonth != 20000 {
		t.Errorf("expected ProAPICallsPerMonth 20000, got %d", cfg.ProAPICallsPerMonth)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with unset env var
	result := getEnv("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("expected 'default', got %s", result)
	}

	// Test with set env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result = getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("expected 'test_value', got %s", result)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test with unset env var
	result := getEnvInt("NONEXISTENT_INT_VAR", 42)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}

	// Test with valid int
	os.Setenv("TEST_INT_VAR", "100")
	defer os.Unsetenv("TEST_INT_VAR")

	result = getEnvInt("TEST_INT_VAR", 42)
	if result != 100 {
		t.Errorf("expected 100, got %d", result)
	}

	// Test with invalid int (should return default)
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")

	result = getEnvInt("TEST_INVALID_INT", 42)
	if result != 42 {
		t.Errorf("expected 42 for invalid int, got %d", result)
	}
}
