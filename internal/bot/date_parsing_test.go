package bot

import (
	"testing"
	"time"
)

func TestIsPollDatePassed(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name      string
		eventDate time.Time
		want      bool
	}{
		{
			name:      "yesterday's date has passed",
			eventDate: today.AddDate(0, 0, -1),
			want:      true,
		},
		{
			name:      "last week's date has passed",
			eventDate: today.AddDate(0, 0, -7),
			want:      true,
		},
		{
			name:      "today's date has not passed",
			eventDate: today,
			want:      false,
		},
		{
			name:      "tomorrow's date has not passed",
			eventDate: today.AddDate(0, 0, 1),
			want:      false,
		},
		{
			name:      "next week's date has not passed",
			eventDate: today.AddDate(0, 0, 7),
			want:      false,
		},
		{
			name:      "today with time (morning) has not passed",
			eventDate: time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()),
			want:      false,
		},
		{
			name:      "today with time (evening) has not passed",
			eventDate: time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
			want:      false,
		},
		{
			name:      "yesterday with time (evening) has passed",
			eventDate: time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, now.Location()),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPollDatePassed(tt.eventDate)
			if got != tt.want {
				t.Errorf("isPollDatePassed(%v) = %v, want %v", tt.eventDate, got, tt.want)
			}
		})
	}
}
