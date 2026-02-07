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
	LookupUserIDByUsername(username string) (int64, bool, error)
	UpdateVotesUserID(pollID int64, oldUserID, newUserID int64) error
}

// NicknameLookupKey represents a user identifier for batch nickname lookups.
type NicknameLookupKey struct {
	UserID   int64
	Username string
}

// NicknameInfo contains nickname and gender for display.
type NicknameInfo struct {
	Nick   string
	Gender string // "male", "female", or "" for not set
}

// DisplayNick returns the nickname with gender prefix applied.
// Returns empty string if Nick is empty.
func (n NicknameInfo) DisplayNick() string {
	if n.Nick == "" {
		return ""
	}
	prefix := n.GenderPrefix()
	if prefix == "" {
		return n.Nick
	}
	return prefix + " " + n.Nick
}

// GenderPrefix returns the display prefix for the gender.
func (n NicknameInfo) GenderPrefix() string {
	switch n.Gender {
	case "male":
		return "г-н"
	case "female":
		return "г-ж"
	default:
		return ""
	}
}

type NicknameRepository interface {
	Create(tgUserID *int64, tgUsername *string, gameNick string, gender string) (bool, error)
	FindByGameNick(gameNick string) (tgUserID *int64, tgUsername *string, err error)
	FindByTgUsername(username string) (gameNick string, tgUserID *int64, err error)
	FindByTgUserID(userID int64) (gameNick string, err error)
	GetDisplayNick(userID int64, username string) (string, error)
	GetDisplayNicksBatch(keys []NicknameLookupKey) (byUserID map[int64]NicknameInfo, byUsername map[string]NicknameInfo, err error)
	UpdateUserIDByUsername(username string, userID int64) error
	GetAllGameNicksForUser(userID int64, username string) ([]string, error)
}

type Service struct {
	polls     PollRepository
	votes     VoteRepository
	nicknames NicknameRepository
}

func NewService(polls PollRepository, votes VoteRepository, nicknames NicknameRepository) *Service {
	return &Service{polls: polls, votes: votes, nicknames: nicknames}
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

// CreateNickname creates a new nickname mapping.
// If tgUsername is provided, attempts to look up the user ID from voting history.
// Gender should be "male", "female", or empty string for not set.
// Returns true if created, false if duplicate.
func (s *Service) CreateNickname(tgUserID *int64, tgUsername *string, gameNick string, gender string) (bool, error) {
	// If username provided but no user ID, try to look up from votes
	if tgUserID == nil && tgUsername != nil {
		if userID, found, err := s.votes.LookupUserIDByUsername(*tgUsername); err != nil {
			return false, err
		} else if found {
			tgUserID = &userID
		}
	}

	return s.nicknames.Create(tgUserID, tgUsername, gameNick, gender)
}

// ResolveVoteIdentifier resolves a vote identifier to user information.
// If identifier starts with @, treats it as telegram username.
// Otherwise, treats it as a game nickname.
// Returns: userID (0 if unknown), username, displayName, error
func (s *Service) ResolveVoteIdentifier(identifier string) (int64, string, string, error) {
	if len(identifier) > 0 && identifier[0] == '@' {
		// Telegram username
		username := identifier[1:]
		nick, userID, err := s.nicknames.FindByTgUsername(username)
		if err != nil {
			return 0, "", "", err
		}
		if userID != nil {
			// Have both user ID and nickname
			displayName := nick
			if displayName == "" {
				displayName = username
			}
			return *userID, username, displayName, nil
		}
		// No nickname record, try to get user ID from votes
		if id, found, err := s.votes.LookupUserIDByUsername(username); err != nil {
			return 0, "", "", err
		} else if found {
			return id, username, username, nil
		}
		// Unknown user, use synthetic ID
		return ManualUserID(username), username, username, nil
	}

	// Game nickname - look up in nicknames table
	userID, username, err := s.nicknames.FindByGameNick(identifier)
	if err != nil {
		return 0, "", "", err
	}
	if userID == nil && username == nil {
		// Not found in nicknames - use synthetic ID with game nick as display name
		return ManualUserID(identifier), "", identifier, nil
	}

	// Found a match
	if userID != nil {
		var uname string
		if username != nil {
			uname = *username
		}
		return *userID, uname, identifier, nil
	}

	// Have username but no user ID - try votes lookup
	if id, found, err := s.votes.LookupUserIDByUsername(*username); err != nil {
		return 0, "", "", err
	} else if found {
		return id, *username, identifier, nil
	}
	return ManualUserID(*username), *username, identifier, nil
}

// BackfillVotesForNickname updates votes in the active poll to use the canonical user ID.
// Called after creating a nickname to consolidate votes.
func (s *Service) BackfillVotesForNickname(chatID int64, tgUserID int64, tgUsername string, gameNicks []string) error {
	// Get active poll
	p, err := s.polls.GetLatestActive(chatID)
	if err != nil || p == nil {
		return err // No active poll, nothing to backfill
	}

	// Update votes by username (only for this poll)
	if tgUsername != "" {
		// Find synthetic ID for username and update
		syntheticID := ManualUserID(tgUsername)
		if err := s.votes.UpdateVotesUserID(p.ID, syntheticID, tgUserID); err != nil {
			return err
		}
	}

	// Update votes by game nicks (only for this poll)
	for _, nick := range gameNicks {
		syntheticID := ManualUserID(nick)
		if err := s.votes.UpdateVotesUserID(p.ID, syntheticID, tgUserID); err != nil {
			return err
		}
	}

	return nil
}

// BackfillNicknameUserID updates nickname records when we learn a user's ID from their vote.
func (s *Service) BackfillNicknameUserID(username string, userID int64) error {
	return s.nicknames.UpdateUserIDByUsername(username, userID)
}

// GetDisplayNick returns the game nickname for a user, if one exists.
func (s *Service) GetDisplayNick(userID int64, username string) (string, error) {
	return s.nicknames.GetDisplayNick(userID, username)
}

// GetAllGameNicksForUser returns all game nicks for a user.
func (s *Service) GetAllGameNicksForUser(userID int64, username string) ([]string, error) {
	return s.nicknames.GetAllGameNicksForUser(userID, username)
}

// NicknameCache provides cached nickname lookups for a batch of users.
// Use NewNicknameCache to create and populate the cache.
type NicknameCache struct {
	byUserID   map[int64]NicknameInfo
	byUsername map[string]NicknameInfo
}

// Get returns the nickname for a user, checking user ID first, then username.
// Username is normalized to lowercase for lookup.
// Returns empty string if no nickname found.
func (c *NicknameCache) Get(userID int64, username string) string {
	info := c.GetInfo(userID, username)
	return info.Nick
}

// GetInfo returns the NicknameInfo for a user, checking user ID first, then username.
// Username is normalized to lowercase for lookup.
// Returns empty NicknameInfo if not found.
func (c *NicknameCache) GetInfo(userID int64, username string) NicknameInfo {
	if userID > 0 {
		if info, ok := c.byUserID[userID]; ok {
			return info
		}
	}
	if username != "" {
		if info, ok := c.byUsername[NormalizeUsername(username)]; ok {
			return info
		}
	}
	return NicknameInfo{}
}

// GetDisplayNick returns the nickname with gender prefix for a user.
// Returns empty string if no nickname found.
func (c *NicknameCache) GetDisplayNick(userID int64, username string) string {
	info := c.GetInfo(userID, username)
	return info.DisplayNick()
}

// NewNicknameCache creates a cache pre-populated with nicknames for the given keys.
func (s *Service) NewNicknameCache(keys []NicknameLookupKey) (*NicknameCache, error) {
	byUserID, byUsername, err := s.nicknames.GetDisplayNicksBatch(keys)
	if err != nil {
		return nil, err
	}
	return &NicknameCache{
		byUserID:   byUserID,
		byUsername: byUsername,
	}, nil
}

// NewNicknameCacheFromVotes creates a cache for the users in the given votes.
func (s *Service) NewNicknameCacheFromVotes(votes []*Vote) (*NicknameCache, error) {
	keys := make([]NicknameLookupKey, len(votes))
	for i, v := range votes {
		keys[i] = NicknameLookupKey{
			UserID:   v.TgUserID,
			Username: v.TgUsername,
		}
	}
	return s.NewNicknameCache(keys)
}

// LookupUserIDByUsername finds a user ID from vote history by username.
func (s *Service) LookupUserIDByUsername(username string) (int64, bool, error) {
	return s.votes.LookupUserIDByUsername(username)
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

// GetAttendingVotes returns all votes from participants who voted to attend
// (19:00, 20:00, or 21:00+). Returns empty slice if no one is attending.
func (s *Service) GetAttendingVotes(pollID int64) ([]*Vote, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	var attending []*Vote
	for _, v := range votes {
		if OptionKind(v.TgOptionIndex).IsAttending() {
			attending = append(attending, v)
		}
	}
	return attending, nil
}

// GetUndecidedVotes returns all votes from participants who voted "decide later".
// Returns empty slice if no one is undecided.
func (s *Service) GetUndecidedVotes(pollID int64) ([]*Vote, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	var undecided []*Vote
	for _, v := range votes {
		if OptionKind(v.TgOptionIndex) == OptionDecideLater {
			undecided = append(undecided, v)
		}
	}
	return undecided, nil
}

// CollectedData holds data for the /done command (collected enough players)
type CollectedData struct {
	Votes19 []*Vote // voters for 19:00
	Votes20 []*Vote // voters for 20:00
	Votes21 []*Vote // voters for 21:00+
}

// GetCollectedData returns votes for 19:00, 20:00, and 21:00+ options
func (s *Service) GetCollectedData(pollID int64) (*CollectedData, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	data := &CollectedData{
		Votes19: []*Vote{},
		Votes20: []*Vote{},
		Votes21: []*Vote{},
	}

	for _, v := range votes {
		switch OptionKind(v.TgOptionIndex) {
		case OptionComeAt19:
			data.Votes19 = append(data.Votes19, v)
		case OptionComeAt20:
			data.Votes20 = append(data.Votes20, v)
		case OptionComeAt21OrLater:
			data.Votes21 = append(data.Votes21, v)
		}
	}

	return data, nil
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
