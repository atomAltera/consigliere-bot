package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramToken string
	GroupID       int64
	DBPath        string
}

func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_API_KEY")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_API_KEY is required")
	}

	groupIDStr := os.Getenv("TELEGRAM_GROUP_ID")
	if groupIDStr == "" {
		return nil, fmt.Errorf("TELEGRAM_GROUP_ID is required")
	}
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_GROUP_ID must be a number: %w", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("DB_PATH is required")
	}

	return &Config{
		TelegramToken: token,
		GroupID:       groupID,
		DBPath:        dbPath,
	}, nil
}
