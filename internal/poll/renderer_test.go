package poll

import (
	"strings"
	"testing"
	"time"
)

func TestRenderInvitation(t *testing.T) {
	results := &InvitationResults{
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Participants: []*Vote{
			{TgUsername: "alice", TgFirstName: "Alice", TgOptionIndex: int(OptionComeAt19)},
			{TgUsername: "bob", TgFirstName: "Bob", TgOptionIndex: int(OptionComeAt20)},
		},
		ComingLater: []*Vote{
			{TgUsername: "charlie", TgFirstName: "Charlie"},
		},
		Undecided:   []*Vote{},
		IsCancelled: false,
	}

	html, err := RenderInvitation(results)
	if err != nil {
		t.Fatalf("RenderInvitation failed: %v", err)
	}

	if !strings.Contains(html, "Приглашаем на мафию") {
		t.Error("expected invitation header in output")
	}
	if !strings.Contains(html, "Участники (2)") {
		t.Error("expected participant count in output")
	}
	if !strings.Contains(html, "@alice") {
		t.Error("expected alice username in output")
	}
	if !strings.Contains(html, "(19:00)") {
		t.Error("expected time label in output")
	}
	if !strings.Contains(html, "Будут позже") {
		t.Error("expected 'coming later' section in output")
	}
	if !strings.Contains(html, "@charlie") {
		t.Error("expected charlie in coming later section")
	}
}

func TestRenderInvitationCancelled(t *testing.T) {
	results := &InvitationResults{
		EventDate:    time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Participants: []*Vote{},
		ComingLater:  []*Vote{},
		Undecided:    []*Vote{},
		IsCancelled:  true,
	}

	html, err := RenderInvitation(results)
	if err != nil {
		t.Fatalf("RenderInvitation failed: %v", err)
	}

	if !strings.Contains(html, "Мероприятие отменено") {
		t.Error("expected cancellation message in output")
	}
}

func TestRenderTitle(t *testing.T) {
	eventDate := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	title, err := RenderTitle(eventDate)
	if err != nil {
		t.Fatalf("RenderTitle failed: %v", err)
	}

	if !strings.Contains(title, "Мафия") {
		t.Error("expected 'Мафия' in title")
	}
	if !strings.Contains(title, "февраля") {
		t.Error("expected month in title")
	}
}
