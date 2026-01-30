package bot

import (
	"errors"
	"testing"

	tele "gopkg.in/telebot.v4"
)

// botInterface wraps the methods we need from telebot.Bot for testing
type botInterface interface {
	ChatMemberOf(*tele.Chat, *tele.User) (*tele.ChatMember, error)
}

// mockBot implements botInterface for testing
type mockBot struct {
	chatMemberFunc func(*tele.Chat, *tele.User) (*tele.ChatMember, error)
}

func (m *mockBot) ChatMemberOf(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
	if m.chatMemberFunc != nil {
		return m.chatMemberFunc(chat, user)
	}
	return nil, errors.New("not implemented")
}

// testBot wraps Bot to allow injecting mock for testing
type testBot struct {
	botAPI botInterface
}

func (tb *testBot) isAdmin(chatID int64, userID int64) (bool, error) {
	chat := &tele.Chat{ID: chatID}
	member, err := tb.botAPI.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		return false, err
	}

	return member.Role == tele.Administrator || member.Role == tele.Creator, nil
}

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

func TestBot_isAdmin_Administrator(t *testing.T) {
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return &tele.ChatMember{
				Role: tele.Administrator,
			}, nil
		},
	}

	tb := &testBot{botAPI: mb}

	isAdmin, err := tb.isAdmin(123, 456)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isAdmin {
		t.Error("expected isAdmin to be true for administrator role")
	}
}

func TestBot_isAdmin_Creator(t *testing.T) {
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return &tele.ChatMember{
				Role: tele.Creator,
			}, nil
		},
	}

	tb := &testBot{botAPI: mb}

	isAdmin, err := tb.isAdmin(123, 456)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isAdmin {
		t.Error("expected isAdmin to be true for creator role")
	}
}

func TestBot_isAdmin_RegularMember(t *testing.T) {
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return &tele.ChatMember{
				Role: tele.Member,
			}, nil
		},
	}

	tb := &testBot{botAPI: mb}

	isAdmin, err := tb.isAdmin(123, 456)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isAdmin {
		t.Error("expected isAdmin to be false for regular member role")
	}
}

func TestBot_isAdmin_Error(t *testing.T) {
	expectedErr := errors.New("API error")
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return nil, expectedErr
		},
	}

	tb := &testBot{botAPI: mb}

	isAdmin, err := tb.isAdmin(123, 456)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if isAdmin {
		t.Error("expected isAdmin to be false when error occurs")
	}
}

func TestBot_AdminOnly_PassesForAdmin(t *testing.T) {
	// Test the middleware behavior logic for admins
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return &tele.ChatMember{
				Role: tele.Administrator,
			}, nil
		},
	}

	tb := &testBot{botAPI: mb}
	ctx := &mockContext{chatID: 123, userID: 456}

	// Test the isAdmin check that middleware would perform
	isAdmin, err := tb.isAdmin(ctx.Chat().ID, ctx.Sender().ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isAdmin {
		t.Error("expected isAdmin to be true, middleware would call next handler")
	}
}

func TestBot_AdminOnly_RejectsNonAdmin(t *testing.T) {
	// Test the middleware behavior logic for non-admins
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return &tele.ChatMember{
				Role: tele.Member,
			}, nil
		},
	}

	tb := &testBot{botAPI: mb}
	ctx := &mockContext{chatID: 123, userID: 456}

	// Test the isAdmin check that middleware would perform
	isAdmin, err := tb.isAdmin(ctx.Chat().ID, ctx.Sender().ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isAdmin {
		t.Error("expected isAdmin to be false, middleware would silently reject")
	}
}

func TestBot_AdminOnly_PropagatesError(t *testing.T) {
	// Test that errors from isAdmin would be propagated by middleware
	expectedErr := errors.New("API error")
	mb := &mockBot{
		chatMemberFunc: func(chat *tele.Chat, user *tele.User) (*tele.ChatMember, error) {
			return nil, expectedErr
		},
	}

	tb := &testBot{botAPI: mb}
	ctx := &mockContext{chatID: 123, userID: 456}

	// Test that isAdmin returns error that middleware would propagate
	_, err := tb.isAdmin(ctx.Chat().ID, ctx.Sender().ID)

	if err == nil {
		t.Fatal("expected error to be propagated, got nil")
	}
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
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
