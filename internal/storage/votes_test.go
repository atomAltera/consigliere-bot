package storage

import (
	"testing"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

func TestVoteRepository_RecordAndGetCurrent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pollRepo := NewPollRepository(db)
	voteRepo := NewVoteRepository(db)

	// Create a poll first
	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	pollRepo.Create(p)

	// Record a vote
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgUsername:    "alice",
		TgFirstName:   "Alice",
		TgOptionIndex: 0,
	}
	err := voteRepo.Record(v)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Get current votes
	votes, err := voteRepo.GetCurrentVotes(p.ID)
	if err != nil {
		t.Fatalf("GetCurrentVotes failed: %v", err)
	}

	if len(votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(votes))
	}
	if votes[0].TgUsername != "alice" {
		t.Errorf("expected username alice, got %s", votes[0].TgUsername)
	}
}

func TestVoteRepository_LatestVoteWins(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pollRepo := NewPollRepository(db)
	voteRepo := NewVoteRepository(db)

	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	pollRepo.Create(p)

	// User votes option 0
	voteRepo.Record(&poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgFirstName:   "Alice",
		TgOptionIndex: 0,
	})

	// User changes vote to option 1
	voteRepo.Record(&poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgFirstName:   "Alice",
		TgOptionIndex: 1,
	})

	votes, _ := voteRepo.GetCurrentVotes(p.ID)
	if len(votes) != 1 {
		t.Fatalf("expected 1 current vote, got %d", len(votes))
	}
	if votes[0].TgOptionIndex != 1 {
		t.Errorf("expected option 1, got %d", votes[0].TgOptionIndex)
	}
}
