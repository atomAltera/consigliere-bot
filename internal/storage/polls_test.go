package storage

import (
	"os"
	"testing"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	path := "/tmp/test_polls_" + time.Now().Format("20060102150405.000") + ".db"
	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return db, func() {
		db.Close()
		os.Remove(path)
	}
}

func TestPollRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}

	err := repo.Create(p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if p.ID == 0 {
		t.Error("expected poll ID to be set")
	}
}

func TestPollRepository_GetLatestActive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	// Create a poll
	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	repo.Create(p)

	// Fetch latest
	latest, err := repo.GetLatestActive(-123456)
	if err != nil {
		t.Fatalf("GetLatestActive failed: %v", err)
	}
	if latest.ID != p.ID {
		t.Errorf("got poll ID %d, want %d", latest.ID, p.ID)
	}
}
