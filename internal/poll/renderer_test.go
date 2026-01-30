package poll

import (
	"strings"
	"testing"
	"time"
)

func TestRenderResults(t *testing.T) {
	results := &Results{
		Poll: &Poll{
			EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		TimeSlots: []TimeSlot{
			{Option: OptionComeAt19, Label: "19:00", Voters: []*Vote{
				{TgUsername: "alice", TgFirstName: "Alice"},
			}},
			{Option: OptionComeAt20, Label: "20:00", Voters: []*Vote{}},
			{Option: OptionComeAt21OrLater, Label: "21:00+", Voters: []*Vote{}},
		},
		Undecided:      []*Vote{},
		NotComing:      []*Vote{},
		AttendingCount: 1,
	}

	html, err := RenderResults(results)
	if err != nil {
		t.Fatalf("RenderResults failed: %v", err)
	}

	if !strings.Contains(html, "Coming (1)") {
		t.Error("expected attending count in output")
	}
	if !strings.Contains(html, "@alice") {
		t.Error("expected username in output")
	}
}
