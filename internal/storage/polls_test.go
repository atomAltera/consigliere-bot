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
		IsActive:  true,
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
		IsActive:  true,
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

func TestPollRepository_GetLatestActive_PinnedStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	// Create a pinned poll (pinned polls should also be returned by GetLatestActive)
	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		IsActive:  true,
		IsPinned:  true,
	}
	repo.Create(p)

	// Fetch latest should find pinned poll
	latest, err := repo.GetLatestActive(-123456)
	if err != nil {
		t.Fatalf("GetLatestActive failed: %v", err)
	}
	if latest.ID != p.ID {
		t.Errorf("got poll ID %d, want %d", latest.ID, p.ID)
	}
}

func TestPollRepository_GetLatestActive_DifferentChat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	// Create a poll for chat A
	pA := &poll.Poll{
		TgChatID:  -111111,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		IsActive:  true,
	}
	repo.Create(pA)

	// Create a poll for chat B
	pB := &poll.Poll{
		TgChatID:  -222222,
		EventDate: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
		IsActive:  true,
	}
	repo.Create(pB)

	// Fetch latest for chat A should only return chat A's poll
	latest, err := repo.GetLatestActive(-111111)
	if err != nil {
		t.Fatalf("GetLatestActive failed: %v", err)
	}
	if latest.ID != pA.ID {
		t.Errorf("got poll ID %d, want %d", latest.ID, pA.ID)
	}

	// Fetch latest for chat B should only return chat B's poll
	latestB, err := repo.GetLatestActive(-222222)
	if err != nil {
		t.Fatalf("GetLatestActive failed: %v", err)
	}
	if latestB.ID != pB.ID {
		t.Errorf("got poll ID %d, want %d", latestB.ID, pB.ID)
	}
}

func TestPollRepository_GetLatestActive_NoPoll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	// Fetch latest for non-existent chat should return (nil, nil)
	p, err := repo.GetLatestActive(-999999)
	if err != nil {
		t.Fatalf("expected nil error for non-existent chat, got %v", err)
	}
	if p != nil {
		t.Fatal("expected nil poll for non-existent chat")
	}
}
