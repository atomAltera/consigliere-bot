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

	p, err := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreatePoll failed: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected poll to have ID")
	}
	if !p.IsActive {
		t.Error("expected poll to be active")
	}
}

func TestService_GetInvitationResults(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	p, _ := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))

	// Add votes: 19:00, 20:00, 21:00+, decide later
	now := time.Now()
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 1, TgFirstName: "Alice", TgOptionIndex: int(OptionComeAt19), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 2, TgFirstName: "Bob", TgOptionIndex: int(OptionComeAt20), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 3, TgFirstName: "Charlie", TgOptionIndex: int(OptionComeAt21OrLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 4, TgFirstName: "Diana", TgOptionIndex: int(OptionDecideLater), VotedAt: now})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 5, TgFirstName: "Eve", TgOptionIndex: int(OptionNotComing), VotedAt: now})

	results, err := svc.GetInvitationResults(p.ID)
	if err != nil {
		t.Fatalf("GetInvitationResults failed: %v", err)
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
