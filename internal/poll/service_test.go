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

func (m *mockPollRepo) GetLatest(chatID int64) (*Poll, error) {
	var latest *Poll
	for _, p := range m.polls {
		if p.TgChatID == chatID {
			if latest == nil || p.CreatedAt.After(latest.CreatedAt) {
				latest = p
			}
		}
	}
	return latest, nil
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

func (m *mockVoteRepo) LookupUserIDByUsername(username string) (int64, bool, error) {
	return 0, false, nil
}

func (m *mockVoteRepo) LookupUsernameByUserID(userID int64) (string, bool, error) {
	return "", false, nil
}

func (m *mockVoteRepo) UpdateVotesUserID(pollID int64, oldUserID, newUserID int64, tgUsername string) error {
	return nil
}

func (m *mockVoteRepo) ConsolidateSyntheticVotes(pollID int64, realUserID int64, username string, gameNicks []string) error {
	return nil
}

type mockNicknameRepo struct{}

func (m *mockNicknameRepo) Create(tgUserID *int64, tgUsername *string, gameNick string, gender string) (bool, error) {
	return true, nil
}

func (m *mockNicknameRepo) FindByGameNick(gameNick string) (*int64, *string, error) {
	return nil, nil, nil
}

func (m *mockNicknameRepo) FindByTgUsername(username string) (string, *int64, error) {
	return "", nil, nil
}

func (m *mockNicknameRepo) FindByTgUserID(userID int64) (string, error) {
	return "", nil
}

func (m *mockNicknameRepo) GetDisplayNick(userID int64, username string) (string, error) {
	return "", nil
}

func (m *mockNicknameRepo) UpdateUserIDByUsername(username string, userID int64) error {
	return nil
}

func (m *mockNicknameRepo) UpdateUserData(userID int64, username string) error {
	return nil
}

func (m *mockNicknameRepo) GetAllGameNicksForUser(userID int64, username string) ([]string, error) {
	return nil, nil
}

func (m *mockNicknameRepo) GetDisplayNicksBatch(keys []NicknameLookupKey) (map[int64]NicknameInfo, map[string]NicknameInfo, error) {
	return make(map[int64]NicknameInfo), make(map[string]NicknameInfo), nil
}

func TestService_CreatePoll(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	result, err := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), ClubVanmo)
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
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	result, _ := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), ClubVanmo)
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
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7) // 1 week from now

	// Step 1: Create poll
	result, err := svc.CreatePoll(chatID, futureDate, ClubVanmo)
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
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7) // 1 week from now

	// Step 1: Create poll
	result, err := svc.CreatePoll(chatID, futureDate, ClubVanmo)
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
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7)

	// Create first poll
	_, err := svc.CreatePoll(chatID, futureDate, ClubVanmo)
	if err != nil {
		t.Fatalf("First CreatePoll failed: %v", err)
	}

	// Try to create second poll - should fail
	_, err = svc.CreatePoll(chatID, futureDate, ClubVanmo)
	if err != ErrPollExists {
		t.Errorf("expected ErrPollExists for duplicate poll, got %v", err)
	}
}

// Integration test: GetAttendingVotes and GetUndecidedVotes
func TestIntegration_GetVotes(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	nickRepo := &mockNicknameRepo{}
	svc := NewService(pollRepo, voteRepo, nickRepo)

	chatID := int64(-123456)
	futureDate := time.Now().AddDate(0, 0, 7)

	result, _ := svc.CreatePoll(chatID, futureDate, ClubVanmo)
	poll := result.Poll

	now := time.Now()
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 1, TgUsername: "alice", TgOptionIndex: int(OptionComeAt19), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 2, TgUsername: "bob", TgOptionIndex: int(OptionComeAt21OrLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 3, TgUsername: "charlie", TgOptionIndex: int(OptionDecideLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: poll.ID, TgUserID: 4, TgUsername: "", TgOptionIndex: int(OptionComeAt20), VotedAt: now}) // no username

	// Test GetAttendingVotes
	attending, err := svc.GetAttendingVotes(poll.ID)
	if err != nil {
		t.Fatalf("GetAttendingVotes failed: %v", err)
	}
	if len(attending) != 3 { // alice, bob, and user4 (all attending options)
		t.Errorf("expected 3 attending votes, got %d", len(attending))
	}

	// Test GetUndecidedVotes
	undecided, err := svc.GetUndecidedVotes(poll.ID)
	if err != nil {
		t.Fatalf("GetUndecidedVotes failed: %v", err)
	}
	if len(undecided) != 1 { // charlie
		t.Errorf("expected 1 undecided vote, got %d", len(undecided))
	}
	if undecided[0].TgUsername != "charlie" {
		t.Errorf("expected undecided to be 'charlie', got %s", undecided[0].TgUsername)
	}
}

func TestPoll_PopulateInvitationData(t *testing.T) {
	eventDate := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		isActive       bool
		wantCancelled  bool
	}{
		{
			name:          "active poll sets IsCancelled to false",
			isActive:      true,
			wantCancelled: false,
		},
		{
			name:          "inactive poll sets IsCancelled to true",
			isActive:      false,
			wantCancelled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Poll{
				ID:        1,
				TgChatID:  -123456,
				EventDate: eventDate,
				IsActive:  tt.isActive,
			}

			data := &InvitationData{
				Participants: []*Vote{{TgUserID: 1}},
				ComingLater:  []*Vote{{TgUserID: 2}},
				Undecided:    []*Vote{{TgUserID: 3}},
			}

			p.PopulateInvitationData(data)

			if data.Poll != p {
				t.Error("expected data.Poll to be set to the poll")
			}
			if !data.EventDate.Equal(eventDate) {
				t.Errorf("expected EventDate = %v, got %v", eventDate, data.EventDate)
			}
			if data.IsCancelled != tt.wantCancelled {
				t.Errorf("expected IsCancelled = %v, got %v", tt.wantCancelled, data.IsCancelled)
			}
			// Verify existing data is preserved
			if len(data.Participants) != 1 {
				t.Error("expected Participants to be preserved")
			}
			if len(data.ComingLater) != 1 {
				t.Error("expected ComingLater to be preserved")
			}
			if len(data.Undecided) != 1 {
				t.Error("expected Undecided to be preserved")
			}
		})
	}
}

func TestDetermineStartTimeAndVoters(t *testing.T) {
	// Helper to create N votes
	makeVotes := func(n int) []*Vote {
		votes := make([]*Vote, n)
		for i := range votes {
			votes[i] = &Vote{TgUserID: int64(i + 1)}
		}
		return votes
	}

	tests := []struct {
		name           string
		votes19        []*Vote
		votes20        []*Vote
		votes21        []*Vote
		minPlayers     int
		wantStartTime  string
		wantMainCount  int
		wantLaterCount int
		wantEnough     bool
	}{
		{
			name:           "enough at 19:00 only",
			votes19:        makeVotes(11),
			votes20:        makeVotes(3),
			votes21:        makeVotes(2),
			minPlayers:     11,
			wantStartTime:  "19:00",
			wantMainCount:  11, // only 19:00 voters
			wantLaterCount: 5,  // 20:00 + 21:00
			wantEnough:     true,
		},
		{
			name:           "not enough at 19:00, enough combined",
			votes19:        makeVotes(5),
			votes20:        makeVotes(7),
			votes21:        makeVotes(2),
			minPlayers:     11,
			wantStartTime:  "20:00",
			wantMainCount:  12, // 19:00 + 20:00
			wantLaterCount: 2,  // only 21:00
			wantEnough:     true,
		},
		{
			name:           "not enough players total",
			votes19:        makeVotes(3),
			votes20:        makeVotes(4),
			votes21:        makeVotes(2),
			minPlayers:     11,
			wantStartTime:  "",
			wantMainCount:  0,
			wantLaterCount: 0,
			wantEnough:     false,
		},
		{
			name:           "exact minimum at 19:00",
			votes19:        makeVotes(11),
			votes20:        makeVotes(0),
			votes21:        makeVotes(0),
			minPlayers:     11,
			wantStartTime:  "19:00",
			wantMainCount:  11,
			wantLaterCount: 0,
			wantEnough:     true,
		},
		{
			name:           "exact minimum combined",
			votes19:        makeVotes(5),
			votes20:        makeVotes(6),
			votes21:        makeVotes(0),
			minPlayers:     11,
			wantStartTime:  "20:00",
			wantMainCount:  11,
			wantLaterCount: 0,
			wantEnough:     true,
		},
		{
			name:           "no votes at all",
			votes19:        nil,
			votes20:        nil,
			votes21:        nil,
			minPlayers:     11,
			wantStartTime:  "",
			wantMainCount:  0,
			wantLaterCount: 0,
			wantEnough:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &CollectedData{
				Votes19: tt.votes19,
				Votes20: tt.votes20,
				Votes21: tt.votes21,
			}

			result := DetermineStartTimeAndVoters(data, tt.minPlayers)

			if result.EnoughPlayers != tt.wantEnough {
				t.Errorf("EnoughPlayers = %v, want %v", result.EnoughPlayers, tt.wantEnough)
			}
			if result.StartTime != tt.wantStartTime {
				t.Errorf("StartTime = %q, want %q", result.StartTime, tt.wantStartTime)
			}
			if len(result.MainVoters) != tt.wantMainCount {
				t.Errorf("len(MainVoters) = %d, want %d", len(result.MainVoters), tt.wantMainCount)
			}
			if len(result.ComingLater) != tt.wantLaterCount {
				t.Errorf("len(ComingLater) = %d, want %d", len(result.ComingLater), tt.wantLaterCount)
			}
		})
	}
}
