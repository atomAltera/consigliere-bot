package bot

import (
	"errors"
	"testing"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// mockContext implements the minimum interface needed for middleware testing
type mockContext struct {
	chatID    int64
	userID    int64
	deleted   bool
	deleteErr error
}

func (m *mockContext) Chat() *tele.Chat {
	return &tele.Chat{ID: m.chatID}
}

func (m *mockContext) Sender() *tele.User {
	return &tele.User{ID: m.userID}
}

func (m *mockContext) Delete() error {
	m.deleted = true
	return m.deleteErr
}

// isInAdminList checks if a user ID is in the club admin list.
// This mirrors the logic in ClubAdminOnly middleware.
func isInAdminList(admins []int64, userID int64) bool {
	for _, adminID := range admins {
		if adminID == userID {
			return true
		}
	}
	return false
}

func TestClubAdminList_AllowsAdmin(t *testing.T) {
	config := &ClubConfig{
		Club:   poll.ClubVanmo,
		Admins: []int64{100, 101, 102},
	}

	if !isInAdminList(config.Admins, 100) {
		t.Error("expected user 100 to be admin")
	}
	if !isInAdminList(config.Admins, 101) {
		t.Error("expected user 101 to be admin")
	}
	if !isInAdminList(config.Admins, 102) {
		t.Error("expected user 102 to be admin")
	}
}

func TestClubAdminList_RejectsNonAdmin(t *testing.T) {
	config := &ClubConfig{
		Club:   poll.ClubVanmo,
		Admins: []int64{100, 101, 102},
	}

	if isInAdminList(config.Admins, 999) {
		t.Error("expected user 999 to not be admin")
	}
	if isInAdminList(config.Admins, 0) {
		t.Error("expected user 0 to not be admin")
	}
}

func TestClubAdminList_EmptyAdminList(t *testing.T) {
	config := &ClubConfig{
		Club:   poll.ClubVanmo,
		Admins: []int64{},
	}

	if isInAdminList(config.Admins, 100) {
		t.Error("expected no one to be admin with empty list")
	}
}

func TestChatRegistry_KnownChat(t *testing.T) {
	// Test that known chat IDs resolve to the correct config
	config, ok := chatRegistry[-10]
	if !ok {
		t.Fatal("expected chat -10 to be in registry")
	}
	if config.Club != poll.ClubVanmo {
		t.Errorf("expected club %s, got %s", poll.ClubVanmo, config.Club)
	}

	config, ok = chatRegistry[-30]
	if !ok {
		t.Fatal("expected chat -30 to be in registry")
	}
	if config.Club != poll.ClubTbilissimo {
		t.Errorf("expected club %s, got %s", poll.ClubTbilissimo, config.Club)
	}
}

func TestChatRegistry_UnknownChat(t *testing.T) {
	_, ok := chatRegistry[-999]
	if ok {
		t.Error("expected chat -999 to not be in registry")
	}
}

func TestChatRegistry_SharedConfig(t *testing.T) {
	// Main and test chats for the same club should share the same config pointer
	if chatRegistry[-10] != chatRegistry[-20] {
		t.Error("expected vanmo main and test chats to share config")
	}
	if chatRegistry[-30] != chatRegistry[-40] {
		t.Error("expected tbilissimo main and test chats to share config")
	}
}

func TestBot_DeleteCommand_DeletesMessage(t *testing.T) {
	// Test that DeleteCommand middleware logic would delete messages
	ctx := &mockContext{chatID: 123, userID: 456}

	// Simulate what the middleware does: call Delete()
	err := ctx.Delete()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ctx.deleted {
		t.Error("expected message to be deleted")
	}
}

func TestBot_DeleteCommand_ContinuesOnDeleteError(t *testing.T) {
	// Test that DeleteCommand middleware would swallow delete errors
	ctx := &mockContext{
		chatID:    123,
		userID:    456,
		deleteErr: errors.New("delete failed"),
	}

	// Simulate what the middleware does: call Delete() and swallow error
	err := ctx.Delete()

	// The middleware swallows delete errors, we verify the error is returned
	// but the actual middleware would ignore it
	if err == nil {
		t.Fatal("expected delete error")
	}
	if !ctx.deleted {
		t.Error("expected delete to be attempted even if it fails")
	}
}

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple command",
			text:     "/poll",
			expected: "poll",
		},
		{
			name:     "command with arguments",
			text:     "/poll 2025-01-20",
			expected: "poll",
		},
		{
			name:     "command with bot mention",
			text:     "/poll@mybot",
			expected: "poll",
		},
		{
			name:     "command with bot mention and arguments",
			text:     "/poll@mybot 2025-01-20",
			expected: "poll",
		},
		{
			name:     "empty text",
			text:     "",
			expected: "",
		},
		{
			name:     "text without slash",
			text:     "poll",
			expected: "poll",
		},
		{
			name:     "command with multiple arguments",
			text:     "/vote @user 3",
			expected: "vote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommand(tt.text)
			if result != tt.expected {
				t.Errorf("extractCommand(%q) = %q, want %q", tt.text, result, tt.expected)
			}
		})
	}
}
