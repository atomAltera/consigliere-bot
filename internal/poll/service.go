package poll

import "time"

type PollRepository interface {
	Create(p *Poll) error
	GetLatestActive(chatID int64) (*Poll, error)
	GetByTgPollID(tgPollID string) (*Poll, error)
	Update(p *Poll) error
}

type VoteRepository interface {
	Record(v *Vote) error
	GetCurrentVotes(pollID int64) ([]*Vote, error)
}

type Service struct {
	polls PollRepository
	votes VoteRepository
}

func NewService(polls PollRepository, votes VoteRepository) *Service {
	return &Service{polls: polls, votes: votes}
}

func (s *Service) CreatePoll(chatID int64, eventDate time.Time) (*Poll, error) {
	p := &Poll{
		TgChatID:  chatID,
		EventDate: eventDate,
		IsActive:  true,
		IsPinned:  false,
		CreatedAt: time.Now(),
	}
	if err := s.polls.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetLatestActivePoll(chatID int64) (*Poll, error) {
	return s.polls.GetLatestActive(chatID)
}

func (s *Service) GetPollByTgPollID(tgPollID string) (*Poll, error) {
	return s.polls.GetByTgPollID(tgPollID)
}

func (s *Service) UpdatePoll(p *Poll) error {
	return s.polls.Update(p)
}

func (s *Service) RecordVote(v *Vote) error {
	return s.votes.Record(v)
}

// InvitationResults holds data for the invitation message template
type InvitationResults struct {
	Poll         *Poll
	EventDate    time.Time
	Participants []*Vote // 19:00 and 20:00 voters, ordered by vote time
	ComingLater  []*Vote // 21:00+ voters
	Undecided    []*Vote // "Decide later" voters
	IsCancelled  bool
}

// GetInvitationResults returns results formatted for the invitation message
func (s *Service) GetInvitationResults(pollID int64) (*InvitationResults, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	results := &InvitationResults{
		Participants: []*Vote{},
		ComingLater:  []*Vote{},
		Undecided:    []*Vote{},
	}

	for _, v := range votes {
		switch OptionKind(v.TgOptionIndex) {
		case OptionComeAt19, OptionComeAt20:
			results.Participants = append(results.Participants, v)
		case OptionComeAt21OrLater:
			results.ComingLater = append(results.ComingLater, v)
		case OptionDecideLater:
			results.Undecided = append(results.Undecided, v)
		// OptionNotComing is not displayed
		}
	}

	return results, nil
}
