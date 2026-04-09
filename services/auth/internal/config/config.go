package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTExpireMinutes  int
	JWTRefreshHours   int
	Env               string
	NatsURL           string
}

func Load() *Config {
	// Load .env file if present (ignore error if missing)
	if err := godotenv.Load(); err != nil {
		log.Printf("[auth][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8081"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_attendance?sslmode=disable"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 60),
		JWTRefreshHours:  getEnvInt("JWT_REFRESH_HOURS", 168),
		Env:              getEnv("ENV", "development"),
		NatsURL:          getEnv("NATS_URL", ""),
	}

	log.Printf("[auth][config] loaded: port=%s env=%s jwt_expire=%dm refresh=%dh",
		cfg.Port, cfg.Env, cfg.JWTExpireMinutes, cfg.JWTRefreshHours)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("[auth][config] warning: invalid int for %s=%s, using default %d", key, v, fallback)
		return fallback
	}
	return i
}
