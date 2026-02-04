package bot

import (
	"fmt"
	"strings"
	"time"
)

// weekdayMap maps day names (lowercase) to time.Weekday
// Supports both English and Russian day names
var weekdayMap = map[string]time.Weekday{
	// English
	"sunday":    time.Sunday,
	"sun":       time.Sunday,
	"monday":    time.Monday,
	"mon":       time.Monday,
	"tuesday":   time.Tuesday,
	"tue":       time.Tuesday,
	"wednesday": time.Wednesday,
	"wed":       time.Wednesday,
	"thursday":  time.Thursday,
	"thu":       time.Thursday,
	"friday":    time.Friday,
	"fri":       time.Friday,
	"saturday":  time.Saturday,
	"sat":       time.Saturday,
	// Russian
	"воскресенье": time.Sunday,
	"вс":          time.Sunday,
	"понедельник": time.Monday,
	"пн":          time.Monday,
	"вторник":     time.Tuesday,
	"вт":          time.Tuesday,
	"среда":       time.Wednesday,
	"ср":          time.Wednesday,
	"четверг":     time.Thursday,
	"чт":          time.Thursday,
	"пятница":     time.Friday,
	"пт":          time.Friday,
	"суббота":     time.Saturday,
	"сб":          time.Saturday,
}

// nextWeekday returns the next occurrence of the given weekday from the reference date.
// If today is the target weekday, it returns today.
func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysUntil := int(target) - int(from.Weekday())
	if daysUntil < 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil)
}

// nearestGameDay returns the nearest Monday or Saturday from the given date.
// If today is Monday or Saturday, returns today.
func nearestGameDay(from time.Time) time.Time {
	nextMon := nextWeekday(from, time.Monday)
	nextSat := nextWeekday(from, time.Saturday)

	// Return whichever is closer
	if nextMon.Before(nextSat) || nextMon.Equal(nextSat) {
		return nextMon
	}
	return nextSat
}

// isPollDatePassed checks if the poll's event date is before today.
// Returns true if the event date is in the past.
func isPollDatePassed(eventDate time.Time) bool {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eventDay := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, eventDate.Location())
	return eventDay.Before(today)
}

// parseEventDate parses the event date from command arguments.
// Supports:
// - No arguments: nearest Monday or Saturday
// - Day of week name: "monday", "mon", "saturday", "sat", etc.
// - Explicit date: "YYYY-MM-DD"
func parseEventDate(args []string) (time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if len(args) == 0 {
		return nearestGameDay(today), nil
	}

	arg := strings.ToLower(strings.TrimSpace(args[0]))

	// Check if it's a day of week name
	if weekday, ok := weekdayMap[arg]; ok {
		return nextWeekday(today, weekday), nil
	}

	// Try parsing as YYYY-MM-DD (use local timezone for consistency with day name parsing)
	eventDate, err := time.ParseInLocation("2006-01-02", args[0], time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format. Use day name (e.g., monday, sat) or YYYY-MM-DD")
	}

	return eventDate, nil
}
