package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port        int
	Environment string

	// Database
	DatabaseURL string
	RedisURL    string

	// Auth
	JWTSecret          string
	JWTExpiration      int // hours
	RefreshExpiration  int // days

	// Stripe
	StripeSecretKey     string
	StripeWebhookSecret string

	// Blockchain
	EthereumRPCURL   string
	PolygonRPCURL    string
	ArbitrumRPCURL   string
	AlchemyAPIKey    string

	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string

	// Limits
	FreeAPICallsPerMonth int
	ProAPICallsPerMonth  int

	// Profiling
	EnablePprof bool
	PprofPort   int
}

func Load() (*Config, error) {
	return &Config{
		// Server
		Port:        getEnvInt("PORT", 3001),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://chainlens:chainlens@localhost:5432/chainlens?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),

		// Auth
		JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration:     getEnvInt("JWT_EXPIRATION_HOURS", 24),
		RefreshExpiration: getEnvInt("REFRESH_EXPIRATION_DAYS", 30),

		// Stripe
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		// Blockchain
		EthereumRPCURL: getEnv("ETHEREUM_RPC_URL", "https://eth.llamarpc.com"),
		PolygonRPCURL:  getEnv("POLYGON_RPC_URL", "https://polygon-rpc.com"),
		ArbitrumRPCURL: getEnv("ARBITRUM_RPC_URL", "https://arb1.arbitrum.io/rpc"),
		AlchemyAPIKey:  getEnv("ALCHEMY_API_KEY", ""),

		// Email
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@chainlens.dev"),

		// Limits
		FreeAPICallsPerMonth: getEnvInt("FREE_API_CALLS_PER_MONTH", 1000),
		ProAPICallsPerMonth:  getEnvInt("PRO_API_CALLS_PER_MONTH", 10000),

		// Profiling
		EnablePprof: getEnvBool("ENABLE_PPROF", false),
		PprofPort:   getEnvInt("PPROF_PORT", 6060),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
