package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	Env                    string
	AuthServiceURL         string
	AttendanceServiceURL   string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[leave][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:                   getEnv("PORT", "8083"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_attendance?sslmode=disable"),
		Env:                    getEnv("ENV", "development"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		AttendanceServiceURL:   getEnv("ATTENDANCE_SERVICE_URL", "http://localhost:8082"),
	}

	log.Printf("[leave][config] loaded: port=%s env=%s auth=%s attendance=%s",
		cfg.Port, cfg.Env, cfg.AuthServiceURL, cfg.AttendanceServiceURL)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
