package main

import (
	"log"
	"log/slog"
	"os"

	"nuclight.org/consigliere/internal/bot"
	"nuclight.org/consigliere/internal/config"
	"nuclight.org/consigliere/internal/poll"
	"nuclight.org/consigliere/internal/storage"
)

// Version and BuildDate are set via ldflags during build
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	// Set up structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting consigliere bot",
		"version", Version,
		"build_date", BuildDate,
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Info("config loaded",
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

	logger.Info("database initialized")

	// Create repositories
	pollRepo := storage.NewPollRepository(db)
	voteRepo := storage.NewVoteRepository(db)

	// Create service
	pollService := poll.NewService(pollRepo, voteRepo)

	// Create and start bot
	b, err := bot.New(cfg.TelegramToken, pollService, logger)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	b.RegisterCommands()
	b.RegisterHandlers()

	b.Start()
}
