package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	ServerPort    string
	JWTPrivateKey string // path to RSA private key PEM (used by auth endpoint)
	JWTPublicKey  string // path to RSA public key PEM (used by auth middleware)
	WebhookURL    string // destination URL for status-change notifications; empty disables webhooks
	WebhookSecret string // HMAC-SHA256 signing secret for the X-Webhook-Signature header
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        getEnvOrDefault("DB_PORT", "5432"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		DBSSLMode:     getEnvOrDefault("DB_SSLMODE", "disable"),
		ServerPort:    getEnvOrDefault("SERVER_PORT", "8080"),
		JWTPrivateKey: getEnvOrDefault("JWT_PRIVATE_KEY_PATH", "keys/private.pem"),
		JWTPublicKey:  getEnvOrDefault("JWT_PUBLIC_KEY_PATH", "keys/public.pem"),
		WebhookURL:    os.Getenv("WEBHOOK_URL"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DB_HOST":     c.DBHost,
		"DB_USER":     c.DBUser,
		"DB_PASSWORD": c.DBPassword,
		"DB_NAME":     c.DBName,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("variável de ambiente obrigatória não definida: %s", key)
		}
	}

	if c.WebhookURL != "" {
		u, err := url.Parse(c.WebhookURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("WEBHOOK_URL inválida: deve ser uma URL http(s) completa")
		}
	}

	return nil
}

// WebhookEnabled reports whether a webhook destination is configured.
func (c *Config) WebhookEnabled() bool {
	return c.WebhookURL != ""
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
