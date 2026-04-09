package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	DatabaseURL          string
	Env                  string
	AuthServiceURL       string
	AttendanceServiceURL string
	LeaveServiceURL      string
	OrgServiceURL        string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[analytics][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:                 getEnv("PORT", "8084"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smart_attendance?sslmode=disable"),
		Env:                  getEnv("ENV", "development"),
		AuthServiceURL:       getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		AttendanceServiceURL: getEnv("ATTENDANCE_SERVICE_URL", "http://localhost:8082"),
		LeaveServiceURL:      getEnv("LEAVE_SERVICE_URL", "http://localhost:8083"),
		OrgServiceURL:        getEnv("ORG_SERVICE_URL", "http://localhost:8085"),
	}

	log.Printf("[analytics][config] loaded: port=%s env=%s auth=%s attendance=%s leave=%s org=%s",
		cfg.Port, cfg.Env, cfg.AuthServiceURL, cfg.AttendanceServiceURL, cfg.LeaveServiceURL, cfg.OrgServiceURL)

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
