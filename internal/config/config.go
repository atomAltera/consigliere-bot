package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Default configuration values
const (
	DefaultTempMessageDelay = 5 * time.Second
	DefaultPollingTimeout   = 10 * time.Second
)

type Config struct {
	TelegramToken    string
	DBPath           string
	SentryDSN        string
	DevMode          bool
	TempMessageDelay time.Duration
	PollingTimeout   time.Duration
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

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

	tempMessageDelay := DefaultTempMessageDelay
	if val := os.Getenv("TEMP_MESSAGE_DELAY_SECONDS"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			tempMessageDelay = time.Duration(seconds) * time.Second
		}
	}

	pollingTimeout := DefaultPollingTimeout
	if val := os.Getenv("POLLING_TIMEOUT_SECONDS"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			pollingTimeout = time.Duration(seconds) * time.Second
		}
	}

	return &Config{
		TelegramToken:    token,
		DBPath:           dbPath,
		SentryDSN:        sentryDSN,
		DevMode:          devMode,
		TempMessageDelay: tempMessageDelay,
		PollingTimeout:   pollingTimeout,
	}, nil
}
