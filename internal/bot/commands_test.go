package bot

import (
	"testing"
	"time"
)

func TestNextWeekday(t *testing.T) {
	// Test from a Wednesday (2024-01-10)
	wed := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		from     time.Time
		target   time.Weekday
		expected time.Time
	}{
		{
			name:     "Wednesday to next Monday",
			from:     wed,
			target:   time.Monday,
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Wednesday to next Saturday",
			from:     wed,
			target:   time.Saturday,
			expected: time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Wednesday to same day (Wednesday)",
			from:     wed,
			target:   time.Wednesday,
			expected: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Monday to Monday (same day)",
			from:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			target:   time.Monday,
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextWeekday(tt.from, tt.target)
			if !result.Equal(tt.expected) {
				t.Errorf("nextWeekday(%v, %v) = %v, want %v",
					tt.from.Format("2006-01-02"), tt.target, result.Format("2006-01-02"), tt.expected.Format("2006-01-02"))
			}
		})
	}
}

func TestNearestGameDay(t *testing.T) {
	tests := []struct {
		name     string
		from     time.Time
		expected time.Time
	}{
		{
			name:     "From Sunday - Monday is closer",
			from:     time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC), // Sunday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:     "From Monday - Monday is today",
			from:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), // Monday
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:     "From Tuesday - Saturday is closer",
			from:     time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), // Tuesday
			expected: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
		},
		{
			name:     "From Wednesday - Saturday is closer",
			from:     time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), // Wednesday
			expected: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
		},
		{
			name:     "From Thursday - Saturday is closer",
			from:     time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC), // Thursday
			expected: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
		},
		{
			name:     "From Friday - Saturday is closer",
			from:     time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC), // Friday
			expected: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
		},
		{
			name:     "From Saturday - Saturday is today",
			from:     time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
			expected: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC), // Saturday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nearestGameDay(tt.from)
			if !result.Equal(tt.expected) {
				t.Errorf("nearestGameDay(%v) = %v, want %v",
					tt.from.Format("2006-01-02 Monday"), result.Format("2006-01-02 Monday"), tt.expected.Format("2006-01-02 Monday"))
			}
		})
	}
}

func TestParseEventDate_ExplicitDate(t *testing.T) {
	date, err := parseEventDate([]string{"2024-03-15"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local)
	if !date.Equal(expected) {
		t.Errorf("parseEventDate([\"2024-03-15\"]) = %v, want %v", date, expected)
	}
}

func TestParseEventDate_InvalidDate(t *testing.T) {
	_, err := parseEventDate([]string{"invalid"})
	if err == nil {
		t.Error("expected error for invalid date, got nil")
	}
}

func TestParseEventDate_DayOfWeek(t *testing.T) {
	// Test that day names are recognized
	dayNames := []string{"monday", "mon", "Mon", "MONDAY", "saturday", "sat", "Sat"}
	for _, name := range dayNames {
		_, err := parseEventDate([]string{name})
		if err != nil {
			t.Errorf("parseEventDate([%q]) unexpected error: %v", name, err)
		}
	}
}

func TestMessageRef(t *testing.T) {
	tests := []struct {
		name   string
		chatID int64
		msgID  int
	}{
		{
			name:   "typical message reference",
			chatID: -1001234567890,
			msgID:  42,
		},
		{
			name:   "zero message ID (chat-only reference)",
			chatID: -1001234567890,
			msgID:  0,
		},
		{
			name:   "positive chat ID (private chat)",
			chatID: 123456789,
			msgID:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageRef(tt.chatID, tt.msgID)

			if msg == nil {
				t.Fatal("MessageRef returned nil")
			}
			if msg.ID != tt.msgID {
				t.Errorf("MessageRef(%d, %d).ID = %d, want %d", tt.chatID, tt.msgID, msg.ID, tt.msgID)
			}
			if msg.Chat == nil {
				t.Fatal("MessageRef().Chat is nil")
			}
			if msg.Chat.ID != tt.chatID {
				t.Errorf("MessageRef(%d, %d).Chat.ID = %d, want %d", tt.chatID, tt.msgID, msg.Chat.ID, tt.chatID)
			}
		})
	}
}
