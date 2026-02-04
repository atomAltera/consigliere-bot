package poll

import (
	"testing"
	"time"
)

type mockPollRepo struct {
	polls   map[int64]*Poll
	counter int64
}

func (m *mockPollRepo) Create(p *Poll) error {
	m.counter++
	p.ID = m.counter
	m.polls[p.ID] = p
	return nil
}

func (m *mockPollRepo) GetLatestActive(chatID int64) (*Poll, error) {
	for _, p := range m.polls {
		if p.TgChatID == chatID && p.IsActive {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPollRepo) GetLatestCancelled(chatID int64) (*Poll, error) {
	for _, p := range m.polls {
		if p.TgChatID == chatID && !p.IsActive {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPollRepo) GetByTgPollID(tgPollID string) (*Poll, error) {
	for _, p := range m.polls {
		if p.TgPollID == tgPollID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPollRepo) Update(p *Poll) error {
	m.polls[p.ID] = p
	return nil
}

type mockVoteRepo struct {
	votes []*Vote
}

func (m *mockVoteRepo) Record(v *Vote) error {
	m.votes = append(m.votes, v)
	return nil
}

func (m *mockVoteRepo) GetCurrentVotes(pollID int64) ([]*Vote, error) {
	latest := make(map[int64]*Vote)
	for _, v := range m.votes {
		if v.PollID == pollID && v.TgOptionIndex >= 0 {
			existing, ok := latest[v.TgUserID]
			if !ok || v.VotedAt.After(existing.VotedAt) {
				latest[v.TgUserID] = v
			}
		}
	}
	var result []*Vote
	for _, v := range latest {
		result = append(result, v)
	}
	return result, nil
}

func TestService_CreatePoll(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	result, err := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreatePoll failed: %v", err)
	}
	if result.Poll.ID == 0 {
		t.Error("expected poll to have ID")
	}
	if !result.Poll.IsActive {
		t.Error("expected poll to be active")
	}
}

func TestService_GetInvitationData(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	result, _ := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	p := result.Poll

	// Add votes: 19:00, 20:00, 21:00+, decide later
	now := time.Now()
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 1, TgFirstName: "Alice", TgOptionIndex: int(OptionComeAt19), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 2, TgFirstName: "Bob", TgOptionIndex: int(OptionComeAt20), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 3, TgFirstName: "Charlie", TgOptionIndex: int(OptionComeAt21OrLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 4, TgFirstName: "Diana", TgOptionIndex: int(OptionDecideLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 5, TgFirstName: "Eve", TgOptionIndex: int(OptionNotComing), VotedAt: now})

	results, err := svc.GetInvitationData(p.ID)
	if err != nil {
		t.Fatalf("GetInvitationData failed: %v", err)
	}
	if len(results.Participants) != 2 {
		t.Errorf("expected 2 participants (19:00 + 20:00), got %d", len(results.Participants))
	}
	if len(results.ComingLater) != 1 {
		t.Errorf("expected 1 coming later (21:00+), got %d", len(results.ComingLater))
	}
	if len(results.Undecided) != 1 {
		t.Errorf("expected 1 undecided, got %d", len(results.Undecided))
	}
}

// Integration test: Full poll → vote → results flow
func TestIntegration_PollVoteResultsFlow(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7) // 1 week from now

	// Step 1: Create poll
	result, err := svc.CreatePoll(chatID, futureDate)
	if err != nil {
		t.Fatalf("CreatePoll failed: %v", err)
	}
	poll := result.Poll
	if !poll.IsActive {
		t.Error("poll should be active after creation")
	}

	// Step 2: Record votes
	now := time.Now()
	votes := []*Vote{
		{PollID: poll.ID, TgUserID: 100, TgUsername: "user1", TgFirstName: "User1", TgOptionIndex: int(OptionComeAt19), VotedAt: now},
		{PollID: poll.ID, TgUserID: 101, TgUsername: "user2", TgFirstName: "User2", TgOptionIndex: int(OptionComeAt20), VotedAt: now},
		{PollID: poll.ID, TgUserID: 102, TgUsername: "user3", TgFirstName: "User3", TgOptionIndex: int(OptionDecideLater), VotedAt: now},
	}
	for _, v := range votes {
		if err := svc.RecordVote(v); err != nil {
			t.Fatalf("RecordVote failed: %v", err)
		}
	}

	// Step 3: Get results
	invData, err := svc.GetInvitationData(poll.ID)
	if err != nil {
		t.Fatalf("GetInvitationData failed: %v", err)
	}
	if len(invData.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(invData.Participants))
	}
	if len(invData.Undecided) != 1 {
		t.Errorf("expected 1 undecided, got %d", len(invData.Undecided))
	}

	// Step 4: User changes vote
	voteRepo.Record(&Vote{
		PollID:        poll.ID,
		TgUserID:      102, // user3 changes from "decide later"
		TgUsername:    "user3",
		TgFirstName:   "User3",
		TgOptionIndex: int(OptionComeAt19), // to "19:00"
		VotedAt:       now.Add(time.Minute),
	})

	// Step 5: Verify updated results
	invData, err = svc.GetInvitationData(poll.ID)
	if err != nil {
		t.Fatalf("GetInvitationData after vote change failed: %v", err)
	}
	if len(invData.Participants) != 3 {
		t.Errorf("expected 3 participants after vote change, got %d", len(invData.Participants))
	}
	if len(invData.Undecided) != 0 {
		t.Errorf("expected 0 undecided after vote change, got %d", len(invData.Undecided))
	}
}

// Integration test: Cancel → Restore flow
func TestIntegration_CancelRestoreFlow(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7) // 1 week from now

	// Step 1: Create poll
	result, err := svc.CreatePoll(chatID, futureDate)
	if err != nil {
		t.Fatalf("CreatePoll failed: %v", err)
	}
	if !result.Poll.IsActive {
		t.Error("poll should be active after creation")
	}

	// Step 2: Cancel poll
	cancelled, err := svc.CancelPoll(chatID)
	if err != nil {
		t.Fatalf("CancelPoll failed: %v", err)
	}
	if cancelled.IsActive {
		t.Error("poll should be inactive after cancellation")
	}

	// Step 3: Verify no active poll
	_, err = svc.GetActivePoll(chatID)
	if err != ErrNoActivePoll {
		t.Errorf("expected ErrNoActivePoll, got %v", err)
	}

	// Step 4: Restore poll
	restored, err := svc.RestorePoll(chatID)
	if err != nil {
		t.Fatalf("RestorePoll failed: %v", err)
	}
	if !restored.IsActive {
		t.Error("poll should be active after restore")
	}

	// Step 5: Verify poll is active again
	active, err := svc.GetActivePoll(chatID)
	if err != nil {
		t.Fatalf("GetActivePoll after restore failed: %v", err)
	}
	if active.ID != restored.ID {
		t.Error("restored poll should be the active poll")
	}
}

// Integration test: Duplicate poll prevention
func TestIntegration_DuplicatePollPrevention(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7)

	// Create first poll
	_, err := svc.CreatePoll(chatID, futureDate)
	if err != nil {
		t.Fatalf("First CreatePoll failed: %v", err)
	}

	// Try to create second poll - should fail
	_, err = svc.CreatePoll(chatID, futureDate)
	if err != ErrPollExists {
		t.Errorf("expected ErrPollExists for duplicate poll, got %v", err)
	}
}

// Integration test: GetAttendingUsernames and GetUndecidedUsernames
func TestIntegration_GetUsernames(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7)

	result, _ := svc.CreatePoll(chatID, futureDate)
	poll := result.Poll

	now := time.Now()
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 1, TgUsername: "alice", TgOptionIndex: int(OptionComeAt19), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 2, TgUsername: "bob", TgOptionIndex: int(OptionComeAt21OrLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 3, TgUsername: "charlie", TgOptionIndex: int(OptionDecideLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 4, TgUsername: "", TgOptionIndex: int(OptionComeAt20), VotedAt: now}) // no username

	// Test GetAttendingUsernames
	attending, err := svc.GetAttendingUsernames(poll.ID)
	if err != nil {
		t.Fatalf("GetAttendingUsernames failed: %v", err)
	}
	if len(attending) != 2 { // alice and bob (user without username not counted)
		t.Errorf("expected 2 attending usernames, got %d: %v", len(attending), attending)
	}

	// Test GetUndecidedUsernames
	undecided, err := svc.GetUndecidedUsernames(poll.ID)
	if err != nil {
		t.Fatalf("GetUndecidedUsernames failed: %v", err)
	}
	if len(undecided) != 1 { // charlie
		t.Errorf("expected 1 undecided username, got %d: %v", len(undecided), undecided)
	}
	if undecided[0] != "charlie" {
		t.Errorf("expected undecided to be 'charlie', got %s", undecided[0])
	}
}
