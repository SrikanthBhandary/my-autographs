// Package config centralises every piece of environment-based configuration
// the app needs: database connection, S3/object storage credentials, JWT
// signing secret, and server settings. Load() is called once in main.go.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	DB       DBConfig
	S3       S3Config
	SMTP     SMTPConfig
	JWT      JWTConfig
	ShareURL string // base URL used to build shareable links, e.g. https://myapp.com/submit
}

type ServerConfig struct {
	Port string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN builds the Postgres connection string used by database/sql + lib/pq.
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type S3Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	// Endpoint lets you point at an S3-compatible provider (Cloudflare R2,
	// MinIO, Backblaze B2, etc). Leave empty to use real AWS S3.
	Endpoint       string
	UsePathStyle   bool // required by most non-AWS S3-compatible providers
}

type JWTConfig struct {
	Secret     string
	ExpiryHrs  int
}

type SMTPConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

// Load reads configuration from environment variables. In local dev these
// are typically populated from a .env file (see .env.example) via a tool
// like `godotenv` or by exporting them in your shell / docker-compose.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "autograph"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		S3: S3Config{
			Bucket:       getEnv("S3_BUCKET", ""),
			Region:       getEnv("S3_REGION", "us-east-1"),
			AccessKey:    getEnv("S3_ACCESS_KEY", ""),
			SecretKey:    getEnv("S3_SECRET_KEY", ""),
			Endpoint:     getEnv("S3_ENDPOINT", ""), // e.g. https://<accountid>.r2.cloudflarestorage.com
			UsePathStyle: getEnvBool("S3_USE_PATH_STYLE", false),
		},
		JWT: JWTConfig{
			Secret:    getEnv("JWT_SECRET", ""),
			ExpiryHrs: getEnvInt("JWT_EXPIRY_HOURS", 72),
		},
		SMTP: SMTPConfig{
			// Empty host = notifications are silently skipped (logged, not sent).
			// Point at Mailpit for local dev (see docker-compose.yml) or a real
			// provider's SMTP relay in production.
			Host:      getEnv("SMTP_HOST", ""),
			Port:      getEnv("SMTP_PORT", "1025"),
			Username:  getEnv("SMTP_USERNAME", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", "noreply@keepsake.local"),
			FromName:  getEnv("SMTP_FROM_NAME", "Keepsake"),
		},
		ShareURL: getEnv("SHARE_BASE_URL", "http://localhost:5173/submit"),
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required (set it in your .env)")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
