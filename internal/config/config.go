package config

import (
	"fmt"
	"os"
)

type Config struct {
	TelegramToken string
	DBPath        string
	SentryDSN     string
	DevMode       bool
}

func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_API_KEY")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_API_KEY is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("DB_PATH is required")
	}

	sentryDSN := os.Getenv("SENTRY_DSN")
	devMode := os.Getenv("DEV_MODE") == "true"

	return &Config{
		TelegramToken: token,
		DBPath:        dbPath,
		SentryDSN:     sentryDSN,
		DevMode:       devMode,
	}, nil
}
