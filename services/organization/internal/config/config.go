package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	Env         string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[org][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8085"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_attendance?sslmode=disable"),
		Env:         getEnv("ENV", "development"),
	}

	log.Printf("[org][config] loaded: port=%s env=%s", cfg.Port, cfg.Env)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
