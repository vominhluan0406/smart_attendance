package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	AuthServiceURL       string
	AttendanceServiceURL string
	LeaveServiceURL      string
	AnalyticsServiceURL  string
	OrgServiceURL        string
	JWTSecret            string
	RateLimitPerMin      int
	CORSOrigin           string
	Env                  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[gateway][config] no .env file found, using environment variables")
	}

	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		AuthServiceURL:       getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		AttendanceServiceURL: getEnv("ATTENDANCE_SERVICE_URL", "http://localhost:8082"),
		LeaveServiceURL:      getEnv("LEAVE_SERVICE_URL", "http://localhost:8083"),
		AnalyticsServiceURL:  getEnv("ANALYTICS_SERVICE_URL", "http://localhost:8084"),
		OrgServiceURL:        getEnv("ORG_SERVICE_URL", "http://localhost:8085"),
		JWTSecret:            getEnv("JWT_SECRET", "change-me-in-production"),
		RateLimitPerMin:      getEnvInt("RATE_LIMIT_PER_MIN", 60),
		CORSOrigin:           getEnv("CORS_ORIGIN", "http://localhost:3000"),
		Env:                  getEnv("ENV", "development"),
	}

	log.Printf("[gateway][config] loaded: port=%s env=%s cors=%s rate_limit=%d/min",
		cfg.Port, cfg.Env, cfg.CORSOrigin, cfg.RateLimitPerMin)
	log.Printf("[gateway][config] services: auth=%s attendance=%s leave=%s analytics=%s org=%s",
		cfg.AuthServiceURL, cfg.AttendanceServiceURL, cfg.LeaveServiceURL,
		cfg.AnalyticsServiceURL, cfg.OrgServiceURL)

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
		log.Printf("[gateway][config] warning: invalid int for %s=%s, using default %d", key, v, fallback)
		return fallback
	}
	return i
}
