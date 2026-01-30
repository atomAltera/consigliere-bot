package main

import (
	"log"

	"nuclight.org/consigliere/internal/bot"
	"nuclight.org/consigliere/internal/config"
	"nuclight.org/consigliere/internal/poll"
	"nuclight.org/consigliere/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create repositories
	pollRepo := storage.NewPollRepository(db)
	voteRepo := storage.NewVoteRepository(db)

	// Create service
	pollService := poll.NewService(pollRepo, voteRepo)

	// Create and start bot
	b, err := bot.New(cfg.TelegramToken, cfg.GroupID, pollService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	b.RegisterCommands()
	b.RegisterHandlers()

	log.Println("Starting bot...")
	b.Start()
}
