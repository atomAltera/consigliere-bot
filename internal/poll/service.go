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

type Results struct {
	Poll           *Poll
	Title          string
	TimeSlots      []TimeSlot
	Undecided      []*Vote
	NotComing      []*Vote
	AttendingCount int
}

type TimeSlot struct {
	Option OptionKind
	Label  string
	Voters []*Vote
}

func (s *Service) GetResults(pollID int64) (*Results, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	results := &Results{
		TimeSlots: []TimeSlot{
			{Option: OptionComeAt19, Label: "19:00", Voters: []*Vote{}},
			{Option: OptionComeAt20, Label: "20:00", Voters: []*Vote{}},
			{Option: OptionComeAt21OrLater, Label: "21:00+", Voters: []*Vote{}},
		},
		Undecided: []*Vote{},
		NotComing: []*Vote{},
	}

	for _, v := range votes {
		switch OptionKind(v.TgOptionIndex) {
		case OptionComeAt19:
			results.TimeSlots[0].Voters = append(results.TimeSlots[0].Voters, v)
			results.AttendingCount++
		case OptionComeAt20:
			results.TimeSlots[1].Voters = append(results.TimeSlots[1].Voters, v)
			results.AttendingCount++
		case OptionComeAt21OrLater:
			results.TimeSlots[2].Voters = append(results.TimeSlots[2].Voters, v)
			results.AttendingCount++
		case OptionDecideLater:
			results.Undecided = append(results.Undecided, v)
		case OptionNotComing:
			results.NotComing = append(results.NotComing, v)
		}
	}

	return results, nil
}
