package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
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
		appLog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		appLog.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	appLog.Info("database initialized")

	// Initialize templates
	if err := bot.InitTemplates(); err != nil {
		appLog.Error("failed to initialize templates", "error", err)
		os.Exit(1)
	}

	// Create repositories
	pollRepo := storage.NewPollRepository(db)
	voteRepo := storage.NewVoteRepository(db)

	// Create service
	pollService := poll.NewService(pollRepo, voteRepo)

	// Create and start bot
	b, err := bot.New(cfg.TelegramToken, pollService, appLog)
	if err != nil {
		appLog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	b.RegisterCommands()
	b.RegisterHandlers()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		appLog.Info("received shutdown signal", "signal", sig.String())
		b.Stop()
	}()

	appLog.Info("bot started, press Ctrl+C to stop")
	b.Start()
	appLog.Info("bot stopped")
}
