package bot

import (
	"log/slog"
	"os"
	"testing"

	"nuclight.org/consigliere/internal/poll"
)

// mockNicknameService implements the nickname-related methods of poll.Service
// needed for testing nickname enrichment functions.
type mockNicknameService struct {
	*poll.Service
	nicknames map[int64]poll.NicknameInfo    // by user ID
	byName    map[string]poll.NicknameInfo   // by username (lowercase)
}

func newMockNicknameService() *mockNicknameService {
	return &mockNicknameService{
		nicknames: make(map[int64]poll.NicknameInfo),
		byName:    make(map[string]poll.NicknameInfo),
	}
}

func (m *mockNicknameService) addNickname(userID int64, username string, nick string, gender string) {
	info := poll.NicknameInfo{Nick: nick, Gender: gender}
	if userID > 0 {
		m.nicknames[userID] = info
	}
	if username != "" {
		m.byName[poll.NormalizeUsername(username)] = info
	}
}

// mockNicknameRepo implements poll.NicknameRepository for testing
type mockNicknameRepoWithData struct {
	nicknames map[int64]poll.NicknameInfo
	byName    map[string]poll.NicknameInfo
}

func (m *mockNicknameRepoWithData) Create(tgUserID *int64, tgUsername *string, gameNick string, gender string) (bool, error) {
	return true, nil
}

func (m *mockNicknameRepoWithData) FindByGameNick(gameNick string) (*int64, *string, error) {
	return nil, nil, nil
}

func (m *mockNicknameRepoWithData) FindByTgUsername(username string) (string, *int64, error) {
	return "", nil, nil
}

func (m *mockNicknameRepoWithData) FindByTgUserID(userID int64) (string, error) {
	return "", nil
}

func (m *mockNicknameRepoWithData) GetDisplayNick(userID int64, username string) (string, error) {
	if info, ok := m.nicknames[userID]; ok {
		return info.DisplayNick(), nil
	}
	if info, ok := m.byName[poll.NormalizeUsername(username)]; ok {
		return info.DisplayNick(), nil
	}
	return "", nil
}

func (m *mockNicknameRepoWithData) GetDisplayNicksBatch(keys []poll.NicknameLookupKey) (map[int64]poll.NicknameInfo, map[string]poll.NicknameInfo, error) {
	byUserID := make(map[int64]poll.NicknameInfo)
	byUsername := make(map[string]poll.NicknameInfo)

	for _, key := range keys {
		if key.UserID > 0 {
			if info, ok := m.nicknames[key.UserID]; ok {
				byUserID[key.UserID] = info
			}
		}
		if key.Username != "" {
			normalized := poll.NormalizeUsername(key.Username)
			if info, ok := m.byName[normalized]; ok {
				byUsername[normalized] = info
			}
		}
	}
	return byUserID, byUsername, nil
}

func (m *mockNicknameRepoWithData) UpdateUserIDByUsername(username string, userID int64) error {
	return nil
}

func (m *mockNicknameRepoWithData) UpdateUserData(userID int64, username string) error {
	return nil
}

func (m *mockNicknameRepoWithData) GetAllGameNicksForUser(userID int64, username string) ([]string, error) {
	return nil, nil
}

// mockPollRepoForNick implements poll.PollRepository for testing
type mockPollRepoForNick struct {
	polls map[int64]*poll.Poll
}

func (m *mockPollRepoForNick) Create(p *poll.Poll) error                          { return nil }
func (m *mockPollRepoForNick) GetLatestActive(chatID int64) (*poll.Poll, error)   { return nil, nil }
func (m *mockPollRepoForNick) GetLatestCancelled(chatID int64) (*poll.Poll, error) { return nil, nil }
func (m *mockPollRepoForNick) GetLatest(chatID int64) (*poll.Poll, error)          { return nil, nil }
func (m *mockPollRepoForNick) GetByTgPollID(tgPollID string) (*poll.Poll, error)  { return nil, nil }
func (m *mockPollRepoForNick) Update(p *poll.Poll) error                          { return nil }

// mockVoteRepoForNick implements poll.VoteRepository for testing
type mockVoteRepoForNick struct{}

func (m *mockVoteRepoForNick) Record(v *poll.Vote) error                     { return nil }
func (m *mockVoteRepoForNick) GetCurrentVotes(pollID int64) ([]*poll.Vote, error) { return nil, nil }
func (m *mockVoteRepoForNick) LookupUserIDByUsername(username string) (int64, bool, error) {
	return 0, false, nil
}
func (m *mockVoteRepoForNick) LookupUsernameByUserID(userID int64) (string, bool, error) {
	return "", false, nil
}
func (m *mockVoteRepoForNick) UpdateVotesUserID(pollID int64, oldUserID, newUserID int64, tgUsername string) error {
	return nil
}
func (m *mockVoteRepoForNick) ConsolidateSyntheticVotes(pollID int64, realUserID int64, username string, gameNicks []string) error {
	return nil
}

// createTestBot creates a Bot with mock services for testing nickname enrichment
func createTestBot(nickRepo *mockNicknameRepoWithData) *Bot {
	pollRepo := &mockPollRepoForNick{polls: make(map[int64]*poll.Poll)}
	voteRepo := &mockVoteRepoForNick{}
	svc := poll.NewService(pollRepo, voteRepo, nickRepo)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	return &Bot{
		pollService: svc,
		logger:      logger,
	}
}

func TestEnrichVotesWithNicknames(t *testing.T) {
	tests := []struct {
		name           string
		votes          []*poll.Vote
		nicknames      map[int64]poll.NicknameInfo
		byUsername     map[string]poll.NicknameInfo
		wantFirstNames []string // Expected TgFirstName after enrichment
		wantUsernames  []string // Expected TgUsername after enrichment
	}{
		{
			name:           "empty votes",
			votes:          []*poll.Vote{},
			nicknames:      map[int64]poll.NicknameInfo{},
			byUsername:     map[string]poll.NicknameInfo{},
			wantFirstNames: []string{},
			wantUsernames:  []string{},
		},
		{
			name: "no nicknames found",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
				{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob"},
			},
			nicknames:      map[int64]poll.NicknameInfo{},
			byUsername:     map[string]poll.NicknameInfo{},
			wantFirstNames: []string{"Alice", "Bob"}, // Unchanged
			wantUsernames:  []string{"alice", "bob"}, // Unchanged
		},
		{
			name: "nickname found by user ID",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
				{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob"},
			},
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "Кринж", Gender: "male"},
			},
			byUsername:     map[string]poll.NicknameInfo{},
			wantFirstNames: []string{"г-н Кринж", "Bob"}, // Alice gets nickname
			wantUsernames:  []string{"", "bob"},          // Alice's username cleared
		},
		{
			name: "nickname found by username",
			votes: []*poll.Vote{
				{TgUserID: 0, TgFirstName: "", TgUsername: "alice"}, // Synthetic vote
				{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob"},
			},
			nicknames: map[int64]poll.NicknameInfo{},
			byUsername: map[string]poll.NicknameInfo{
				"alice": {Nick: "Кринж", Gender: "female"},
			},
			wantFirstNames: []string{"г-ж Кринж", "Bob"},
			wantUsernames:  []string{"", "bob"},
		},
		{
			name: "nickname without gender prefix",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
			},
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "Кринж", Gender: ""}, // No gender
			},
			byUsername:     map[string]poll.NicknameInfo{},
			wantFirstNames: []string{"Кринж"}, // No prefix
			wantUsernames:  []string{""},
		},
		{
			name: "user ID takes priority over username",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
			},
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "ByID", Gender: "male"},
			},
			byUsername: map[string]poll.NicknameInfo{
				"alice": {Nick: "ByUsername", Gender: "female"},
			},
			wantFirstNames: []string{"г-н ByID"}, // User ID wins
			wantUsernames:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nickRepo := &mockNicknameRepoWithData{
				nicknames: tt.nicknames,
				byName:    tt.byUsername,
			}
			bot := createTestBot(nickRepo)

			// Make copies of votes to avoid modifying test data
			votes := make([]*poll.Vote, len(tt.votes))
			for i, v := range tt.votes {
				copy := *v
				votes[i] = &copy
			}

			bot.enrichVotesWithNicknames(votes)

			if len(votes) != len(tt.wantFirstNames) {
				t.Fatalf("got %d votes, want %d", len(votes), len(tt.wantFirstNames))
			}

			for i, v := range votes {
				if v.TgFirstName != tt.wantFirstNames[i] {
					t.Errorf("vote[%d].TgFirstName = %q, want %q", i, v.TgFirstName, tt.wantFirstNames[i])
				}
				if v.TgUsername != tt.wantUsernames[i] {
					t.Errorf("vote[%d].TgUsername = %q, want %q", i, v.TgUsername, tt.wantUsernames[i])
				}
			}
		})
	}
}

func TestMembersFromVotesWithNicknames(t *testing.T) {
	tests := []struct {
		name          string
		votes         []*poll.Vote
		nicknames     map[int64]poll.NicknameInfo
		byUsername    map[string]poll.NicknameInfo
		wantNicknames []string // Expected Nickname field in Members
		wantTgNames   []string // Expected TgName field (unchanged)
		wantUsernames []string // Expected TgUsername field (unchanged)
	}{
		{
			name:          "empty votes",
			votes:         []*poll.Vote{},
			nicknames:     map[int64]poll.NicknameInfo{},
			byUsername:    map[string]poll.NicknameInfo{},
			wantNicknames: []string{},
			wantTgNames:   []string{},
			wantUsernames: []string{},
		},
		{
			name: "no nicknames found",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
			},
			nicknames:     map[int64]poll.NicknameInfo{},
			byUsername:    map[string]poll.NicknameInfo{},
			wantNicknames: []string{""},      // No nickname
			wantTgNames:   []string{"Alice"}, // Preserved
			wantUsernames: []string{"alice"}, // Preserved
		},
		{
			name: "nickname found - preserves all fields",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
			},
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "Кринж", Gender: "male"},
			},
			byUsername:    map[string]poll.NicknameInfo{},
			wantNicknames: []string{"г-н Кринж"}, // Gets nickname with prefix
			wantTgNames:   []string{"Alice"},     // TgName preserved
			wantUsernames: []string{"alice"},     // Username preserved
		},
		{
			name: "multiple votes with mixed nicknames",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice"},
				{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob"},
				{TgUserID: 3, TgFirstName: "Charlie", TgUsername: "charlie"},
			},
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "Кринж", Gender: "male"},
				3: {Nick: "Чарли", Gender: ""},
			},
			byUsername:    map[string]poll.NicknameInfo{},
			wantNicknames: []string{"г-н Кринж", "", "Чарли"},
			wantTgNames:   []string{"Alice", "Bob", "Charlie"},
			wantUsernames: []string{"alice", "bob", "charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nickRepo := &mockNicknameRepoWithData{
				nicknames: tt.nicknames,
				byName:    tt.byUsername,
			}
			bot := createTestBot(nickRepo)

			members := bot.membersFromVotesWithNicknames(tt.votes)

			if len(members) != len(tt.wantNicknames) {
				t.Fatalf("got %d members, want %d", len(members), len(tt.wantNicknames))
			}

			for i, m := range members {
				if m.Nickname != tt.wantNicknames[i] {
					t.Errorf("member[%d].Nickname = %q, want %q", i, m.Nickname, tt.wantNicknames[i])
				}
				if m.TgName != tt.wantTgNames[i] {
					t.Errorf("member[%d].TgName = %q, want %q", i, m.TgName, tt.wantTgNames[i])
				}
				if m.TgUsername != tt.wantUsernames[i] {
					t.Errorf("member[%d].TgUsername = %q, want %q", i, m.TgUsername, tt.wantUsernames[i])
				}
			}
		})
	}
}

func TestVotesToResultsVotersWithCache(t *testing.T) {
	tests := []struct {
		name          string
		votes         []*poll.Vote
		option        poll.OptionKind
		nicknames     map[int64]poll.NicknameInfo
		wantCount     int
		wantNicknames []string // Expected Nickname field
	}{
		{
			name:          "empty votes",
			votes:         []*poll.Vote{},
			option:        poll.OptionComeAt19,
			nicknames:     map[int64]poll.NicknameInfo{},
			wantCount:     0,
			wantNicknames: []string{},
		},
		{
			name: "filters by option",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgOptionIndex: int(poll.OptionComeAt19)},
				{TgUserID: 2, TgFirstName: "Bob", TgOptionIndex: int(poll.OptionComeAt20)},
				{TgUserID: 3, TgFirstName: "Charlie", TgOptionIndex: int(poll.OptionComeAt19)},
			},
			option:        poll.OptionComeAt19,
			nicknames:     map[int64]poll.NicknameInfo{},
			wantCount:     2, // Only Alice and Charlie
			wantNicknames: []string{"", ""},
		},
		{
			name: "includes nicknames from cache",
			votes: []*poll.Vote{
				{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice", TgOptionIndex: int(poll.OptionComeAt19)},
				{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob", TgOptionIndex: int(poll.OptionComeAt19)},
			},
			option: poll.OptionComeAt19,
			nicknames: map[int64]poll.NicknameInfo{
				1: {Nick: "Кринж", Gender: "male"},
			},
			wantCount:     2,
			wantNicknames: []string{"Кринж", ""}, // Note: Get() returns nick without prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nickRepo := &mockNicknameRepoWithData{
				nicknames: tt.nicknames,
				byName:    make(map[string]poll.NicknameInfo),
			}
			bot := createTestBot(nickRepo)

			// Create cache from votes
			cache, err := bot.pollService.NewNicknameCacheFromVotes(tt.votes)
			if err != nil {
				t.Fatalf("NewNicknameCacheFromVotes failed: %v", err)
			}

			result := bot.votesToResultsVotersWithCache(tt.votes, tt.option, cache)

			if len(result) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(result), tt.wantCount)
			}

			for i, r := range result {
				if i < len(tt.wantNicknames) && r.Nickname != tt.wantNicknames[i] {
					t.Errorf("result[%d].Nickname = %q, want %q", i, r.Nickname, tt.wantNicknames[i])
				}
			}
		})
	}
}

func TestVotesToResultsVotersAllWithCache(t *testing.T) {
	nickRepo := &mockNicknameRepoWithData{
		nicknames: map[int64]poll.NicknameInfo{
			1: {Nick: "Кринж", Gender: "male"},
		},
		byName: make(map[string]poll.NicknameInfo),
	}
	bot := createTestBot(nickRepo)

	votes := []*poll.Vote{
		{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice", TgOptionIndex: int(poll.OptionComeAt19)},
		{TgUserID: 2, TgFirstName: "Bob", TgUsername: "bob", TgOptionIndex: int(poll.OptionComeAt20)},
	}

	cache, err := bot.pollService.NewNicknameCacheFromVotes(votes)
	if err != nil {
		t.Fatalf("NewNicknameCacheFromVotes failed: %v", err)
	}

	result := bot.votesToResultsVotersAllWithCache(votes, cache)

	if len(result) != 2 {
		t.Fatalf("got %d results, want 2", len(result))
	}

	// Should include all votes regardless of option
	if result[0].Nickname != "Кринж" {
		t.Errorf("result[0].Nickname = %q, want %q", result[0].Nickname, "Кринж")
	}
	if result[1].Nickname != "" {
		t.Errorf("result[1].Nickname = %q, want empty", result[1].Nickname)
	}
}

func TestVotesToResultsVotersWithNilCache(t *testing.T) {
	nickRepo := &mockNicknameRepoWithData{
		nicknames: map[int64]poll.NicknameInfo{
			1: {Nick: "Кринж", Gender: "male"},
		},
		byName: make(map[string]poll.NicknameInfo),
	}
	bot := createTestBot(nickRepo)

	votes := []*poll.Vote{
		{TgUserID: 1, TgFirstName: "Alice", TgUsername: "alice", TgOptionIndex: int(poll.OptionComeAt19)},
	}

	// Pass nil cache - should still work but without nicknames
	result := bot.votesToResultsVotersWithCache(votes, poll.OptionComeAt19, nil)

	if len(result) != 1 {
		t.Fatalf("got %d results, want 1", len(result))
	}

	// Nickname should be empty when cache is nil
	if result[0].Nickname != "" {
		t.Errorf("result[0].Nickname = %q, want empty (nil cache)", result[0].Nickname)
	}
}
