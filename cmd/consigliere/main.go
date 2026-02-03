package main

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"

	"nuclight.org/consigliere/internal/bot"
	"nuclight.org/consigliere/internal/config"
	"nuclight.org/consigliere/internal/logger"
	"nuclight.org/consigliere/internal/poll"
	"nuclight.org/consigliere/internal/storage"
)

// Version and BuildDate are set via ldflags during build
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	// Load configuration first (needed for Sentry DSN)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Sentry if DSN is provided
	sentryEnabled := false
	if cfg.SentryDSN != "" {
		env := "production"
		if cfg.DevMode {
			env = "development"
		}
		err = sentry.Init(sentry.ClientOptions{
			ServerName:  "consigliere-bot",
			Dsn:         cfg.SentryDSN,
			Environment: env,
		})
		if err == nil {
			sentryEnabled = true
			defer sentry.Flush(2 * time.Second)
		}
	}

	// Set up structured logger (with or without Sentry)
	var appLog logger.Logger
	if sentryEnabled {
		appLog = logger.NewLoggerWithSentry()
	} else {
		appLog = logger.NewLogger()
	}

	appLog.Info("starting consigliere bot",
		"version", Version,
		"build_date", BuildDate,
		"dev_mode", cfg.DevMode,
		"sentry", sentryEnabled,
	)

	appLog.Info("config loaded",
		"db_path", cfg.DBPath,
	)

	// Initialize database
	db, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	appLog.Info("database initialized")

	// Create repositories
	pollRepo := storage.NewPollRepository(db)
	voteRepo := storage.NewVoteRepository(db)

	// Create service
	pollService := poll.NewService(pollRepo, voteRepo)

	// Create and start bot
	b, err := bot.New(cfg.TelegramToken, pollService, appLog)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	b.RegisterCommands()
	b.RegisterHandlers()

	b.Start()
}
