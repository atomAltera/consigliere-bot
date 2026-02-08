package bot

import (
	"os"
	"strings"
	"testing"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

func TestMain(m *testing.M) {
	// Initialize templates before running tests
	if err := InitTemplates(); err != nil {
		panic("failed to init templates: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestRenderPollTitleMessage(t *testing.T) {
	t.Run("renders Monday date in Russian", func(t *testing.T) {
		// Monday, 15 January 2024
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		result, err := RenderPollTitleMessage(eventDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should contain the emoji, "игры в JOIN BAR", and the Russian date
		if !strings.Contains(result, "🎭") {
			t.Error("expected result to contain 🎭 emoji")
		}
		if !strings.Contains(result, "игры в JOIN BAR") {
			t.Error("expected result to contain 'игры в JOIN BAR'")
		}
		if !strings.Contains(result, "понедельник") {
			t.Error("expected result to contain 'понедельник' (Monday in Russian)")
		}
		if !strings.Contains(result, "15 января") {
			t.Error("expected result to contain '15 января'")
		}
	})

	t.Run("renders Saturday date in Russian", func(t *testing.T) {
		// Saturday, 20 January 2024
		eventDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
		result, err := RenderPollTitleMessage(eventDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "суббота") {
			t.Error("expected result to contain 'суббота' (Saturday in Russian)")
		}
		if !strings.Contains(result, "20 января") {
			t.Error("expected result to contain '20 января'")
		}
	})
}

func TestRenderInvitationMessage(t *testing.T) {
	t.Run("renders empty invitation", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate:    eventDate,
			Participants: nil,
			ComingLater:  nil,
			Undecided:    nil,
			IsCancelled:  false,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should contain header and date
		if !strings.Contains(result, "Приглашаем на игровой вечер") {
			t.Error("expected result to contain invitation header")
		}
		if !strings.Contains(result, "понедельник, 15 января") {
			t.Error("expected result to contain Russian date")
		}
		// Should show "no participants yet"
		if !strings.Contains(result, "пока никого") {
			t.Error("expected result to contain 'пока никого' when no participants")
		}
		// Should NOT contain cancelled notice
		if strings.Contains(result, "отменен") {
			t.Error("expected result NOT to contain cancelled notice when not cancelled")
		}
	})

	t.Run("renders with participants", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate: eventDate,
			Participants: []*poll.Vote{
				{TgUsername: "user1", TgOptionIndex: int(poll.OptionComeAt19)},
				{TgFirstName: "John", TgOptionIndex: int(poll.OptionComeAt20)},
			},
			ComingLater: nil,
			Undecided:   nil,
			IsCancelled: false,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should show participants count
		if !strings.Contains(result, "Участники (2)") {
			t.Error("expected result to show 2 participants")
		}
		// Should show participant names
		if !strings.Contains(result, "@user1") {
			t.Error("expected result to contain @user1")
		}
		if !strings.Contains(result, "John") {
			t.Error("expected result to contain John")
		}
		// Should show time labels
		if !strings.Contains(result, "19:00") {
			t.Error("expected result to contain 19:00 time label")
		}
		if !strings.Contains(result, "20:00") {
			t.Error("expected result to contain 20:00 time label")
		}
	})

	t.Run("renders with coming later", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate:    eventDate,
			Participants: nil,
			ComingLater: []*poll.Vote{
				{TgUsername: "late1", TgOptionIndex: int(poll.OptionComeAt21OrLater)},
				{TgUsername: "late2", TgOptionIndex: int(poll.OptionComeAt21OrLater)},
			},
			Undecided:   nil,
			IsCancelled: false,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Будут позже") {
			t.Error("expected result to contain 'Будут позже' section")
		}
		if !strings.Contains(result, "@late1") {
			t.Error("expected result to contain @late1")
		}
		if !strings.Contains(result, "@late2") {
			t.Error("expected result to contain @late2")
		}
	})

	t.Run("renders with undecided", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate:    eventDate,
			Participants: nil,
			ComingLater:  nil,
			Undecided: []*poll.Vote{
				{TgUsername: "maybe1", TgOptionIndex: int(poll.OptionDecideLater)},
			},
			IsCancelled: false,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Ещё не решили (1)") {
			t.Error("expected result to contain 'Ещё не решили (1)' section")
		}
		if !strings.Contains(result, "@maybe1") {
			t.Error("expected result to contain @maybe1")
		}
	})

	t.Run("renders cancelled event", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate:    eventDate,
			Participants: nil,
			ComingLater:  nil,
			Undecided:    nil,
			IsCancelled:  true,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "❌") {
			t.Error("expected result to contain ❌ emoji when cancelled")
		}
		if !strings.Contains(result, "Игровой вечер отменен") {
			t.Error("expected result to contain cancellation notice")
		}
	})

	t.Run("renders manual vote without @ prefix", func(t *testing.T) {
		eventDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		data := &poll.InvitationData{
			EventDate: eventDate,
			Participants: []*poll.Vote{
				{TgUsername: "ManualPerson", TgOptionIndex: int(poll.OptionComeAt19), IsManual: true},
			},
			ComingLater: nil,
			Undecided:   nil,
			IsCancelled: false,
		}

		result, err := RenderInvitationMessage(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Manual votes should NOT have @ prefix
		if strings.Contains(result, "@ManualPerson") {
			t.Error("expected manual vote NOT to have @ prefix")
		}
		if !strings.Contains(result, "ManualPerson") {
			t.Error("expected result to contain ManualPerson")
		}
	})
}
