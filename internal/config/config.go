package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string // development, production

	// Database
	DBPath string

	// JWT
	JWTSecret        string
	JWTExpireMinutes int
	JWTRefreshHours  int

	// Rate Limiting
	RateLimitPerMin int

	// Microsoft OAuth
	MicrosoftClientID     string
	MicrosoftClientSecret string
	MicrosoftRedirectURI  string
	MicrosoftTenantID     string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:             getEnv("PORT", "8080"),
		Env:              getEnv("ENV", "development"),
		DBPath:           getEnv("DB_PATH", "data/smart_attendance.db"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 60),
		JWTRefreshHours:  getEnvInt("JWT_REFRESH_HOURS", 168),
		RateLimitPerMin:  getEnvInt("RATE_LIMIT_PER_MIN", 10),

		MicrosoftClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
		MicrosoftClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
		MicrosoftRedirectURI:  getEnv("MICROSOFT_REDIRECT_URI", "http://localhost:8080/auth/oauth/microsoft/callback"),
		MicrosoftTenantID:     getEnv("MICROSOFT_TENANT_ID", "common"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
