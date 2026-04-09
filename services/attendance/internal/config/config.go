package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	Env            string
	AuthServiceURL string
	OrgServiceURL  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[attendance][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:           getEnv("PORT", "8082"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_attendance?sslmode=disable"),
		Env:            getEnv("ENV", "development"),
		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		OrgServiceURL:  getEnv("ORG_SERVICE_URL", "http://localhost:8085"),
	}

	log.Printf("[attendance][config] loaded: port=%s env=%s auth=%s org=%s",
		cfg.Port, cfg.Env, cfg.AuthServiceURL, cfg.OrgServiceURL)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
