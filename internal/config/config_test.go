package config

import (
	"os"
	"testing"
)

func TestLoad_AllEnvVarsSet(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_API_KEY", "test-token")
	os.Setenv("DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_API_KEY")
		os.Unsetenv("DB_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TelegramToken != "test-token" {
		t.Errorf("TelegramToken = %q, want %q", cfg.TelegramToken, "test-token")
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/test.db")
	}
}

func TestLoad_MissingToken(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_API_KEY")
	os.Setenv("DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("DB_PATH")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}
