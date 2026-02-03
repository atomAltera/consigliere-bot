package poll

import "time"

type PollRepository interface {
	Create(p *Poll) error
	GetLatestActive(chatID int64) (*Poll, error)
	GetLatestCancelled(chatID int64) (*Poll, error)
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

// CreatePoll creates a new poll for the given chat and event date.
// Returns ErrPollExists if an active poll already exists in this chat with a future event date.
// If an active poll exists but its event date is in the past, it will be deactivated and
// the new poll created. The replaced poll is returned in CreatePollResult.ReplacedPoll.
func (s *Service) CreatePoll(tgChatID int64, eventDate time.Time) (*CreatePollResult, error) {
	// Check if there's already an active poll
	existing, err := s.polls.GetLatestActive(tgChatID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Check if existing poll's event date is in the past
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		eventDay := time.Date(existing.EventDate.Year(), existing.EventDate.Month(), existing.EventDate.Day(), 0, 0, 0, 0, existing.EventDate.Location())

		if !eventDay.Before(today) {
			// Event date is today or future - don't allow new poll
			return nil, ErrPollExists
		}

		// Event date is in the past - deactivate old poll
		existing.IsActive = false
		existing.IsPinned = false
		if err := s.polls.Update(existing); err != nil {
			return nil, err
		}
	}

	p := &Poll{
		TgChatID:  tgChatID,
		EventDate: eventDate,
		Options:   DefaultOptions(),
		IsActive:  true,
		IsPinned:  false,
		CreatedAt: time.Now(),
	}
	if err := s.polls.Create(p); err != nil {
		return nil, err
	}

	result := &CreatePollResult{
		Poll: p,
	}
	if existing != nil {
		result.ReplacedPoll = existing
	}
	return result, nil
}

// GetActivePoll returns the latest active poll for the given chat.
// Returns ErrNoActivePoll if no active poll exists.
func (s *Service) GetActivePoll(tgChatID int64) (*Poll, error) {
	p, err := s.polls.GetLatestActive(tgChatID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNoActivePoll
	}
	return p, nil
}

// CancelPoll cancels the active poll in the given chat.
// Returns ErrNoActivePoll if no active poll exists.
func (s *Service) CancelPoll(tgChatID int64) (*Poll, error) {
	p, err := s.polls.GetLatestActive(tgChatID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNoActivePoll
	}

	p.IsActive = false
	p.IsPinned = false
	if err := s.polls.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// RestorePoll restores the latest cancelled poll in the given chat.
// Returns ErrNoCancelledPoll if no cancelled poll exists.
// Returns ErrPollDatePassed if the poll's event date is in the past.
// Note: TgCancelMessageID is preserved so the handler can delete the message.
func (s *Service) RestorePoll(tgChatID int64) (*Poll, error) {
	p, err := s.polls.GetLatestCancelled(tgChatID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNoCancelledPoll
	}

	// Check if event date is today or future
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eventDay := time.Date(p.EventDate.Year(), p.EventDate.Month(), p.EventDate.Day(), 0, 0, 0, 0, p.EventDate.Location())

	if eventDay.Before(today) {
		return nil, ErrPollDatePassed
	}

	p.IsActive = true
	// Note: Don't clear TgCancelMessageID here - let handler delete the message first
	if err := s.polls.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// SetPinned sets the pinned status for the active poll in the given chat.
// Returns ErrNoActivePoll if no active poll exists.
func (s *Service) SetPinned(tgChatID int64, pinned bool) (*Poll, error) {
	p, err := s.polls.GetLatestActive(tgChatID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNoActivePoll
	}

	p.IsPinned = pinned
	if err := s.polls.Update(p); err != nil {
		return nil, err
	}
	return p, nil
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

// InvitationData holds data for the invitation message template
type InvitationData struct {
	Poll         *Poll
	EventDate    time.Time
	Participants []*Vote // 19:00 and 20:00 voters, ordered by option index then vote time
	ComingLater  []*Vote // 21:00+ voters
	Undecided    []*Vote // "Decide later" voters
	IsCancelled  bool
}

// GetAttendingUsernames returns usernames of all participants who voted to attend
// (19:00, 20:00, or 21:00+). Returns empty slice if no one is attending.
func (s *Service) GetAttendingUsernames(pollID int64) ([]string, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	var usernames []string
	for _, v := range votes {
		if OptionKind(v.TgOptionIndex).IsAttending() && v.TgUsername != "" {
			usernames = append(usernames, v.TgUsername)
		}
	}
	return usernames, nil
}

// GetUndecidedUsernames returns usernames of all participants who voted "decide later".
// Returns empty slice if no one is undecided.
func (s *Service) GetUndecidedUsernames(pollID int64) ([]string, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	var usernames []string
	for _, v := range votes {
		if OptionKind(v.TgOptionIndex) == OptionDecideLater && v.TgUsername != "" {
			usernames = append(usernames, v.TgUsername)
		}
	}
	return usernames, nil
}

// GetInvitationData returns results formatted for the invitation message
func (s *Service) GetInvitationData(pollID int64) (*InvitationData, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	results := &InvitationData{
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
