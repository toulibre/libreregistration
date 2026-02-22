package config

import (
	"os"
)

type Config struct {
	Port           string
	DatabaseDriver string
	DatabasePath   string
	DatabaseURL    string
	SessionSecret  string
	CSRFKey        string
	BaseURL        string
	AdminUsername   string
	AdminPassword  string
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPassword   string
	SMTPFrom       string
	UploadDir      string
}

func Load() *Config {
	return &Config{
		Port:           envOr("PORT", "8080"),
		DatabaseDriver: envOr("DATABASE_DRIVER", "sqlite"),
		DatabasePath:   envOr("DATABASE_PATH", "libreregistration.db"),
		DatabaseURL:    envOr("DATABASE_URL", ""),
		SessionSecret: envOr("SESSION_SECRET", "change-me-in-production-32chars!"),
		CSRFKey:       envOr("CSRF_KEY", "change-me-csrf-key-32-chars!!!!"),
		BaseURL:       envOr("BASE_URL", "http://localhost:8080"),
		AdminUsername: envOr("ADMIN_USERNAME", ""),
		AdminPassword: envOr("ADMIN_PASSWORD", ""),
		SMTPHost:      envOr("SMTP_HOST", ""),
		SMTPPort:      envOr("SMTP_PORT", "587"),
		SMTPUser:      envOr("SMTP_USER", ""),
		SMTPPassword:  envOr("SMTP_PASSWORD", ""),
		SMTPFrom:      envOr("SMTP_FROM", ""),
		UploadDir:     envOr("UPLOAD_DIR", "uploads"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
