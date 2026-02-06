package poll

import (
	"hash/fnv"
	"strings"
	"time"
)

type Vote struct {
	ID            int64
	PollID        int64
	TgUserID      int64
	TgUsername    string
	TgFirstName   string
	TgOptionIndex int
	IsManual      bool
	VotedAt       time.Time
}

// NormalizeUsername normalizes a Telegram username to lowercase.
// Telegram usernames are case-insensitive, so we store them lowercase
// to avoid duplicate rows and lookup misses (e.g., @User vs @user).
func NormalizeUsername(username string) string {
	return strings.ToLower(username)
}

// ManualUserID generates a synthetic user ID for manual votes based on username.
// Uses negative IDs to avoid collision with real Telegram user IDs (which are positive).
// Note: username should be normalized before calling this function for consistency.
func ManualUserID(username string) int64 {
	h := fnv.New64a()
	h.Write([]byte(username))
	// Make it negative to distinguish from real user IDs
	return -int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}

func (v *Vote) OptionKind() OptionKind {
	return OptionKind(v.TgOptionIndex)
}

func (v *Vote) DisplayName() string {
	if v.TgUsername != "" {
		// Manual votes store the provided name in TgUsername,
		// which might be a real name (not a Telegram handle), so don't add @
		if v.IsManual {
			return v.TgUsername
		}
		return "@" + v.TgUsername
	}
	return v.TgFirstName
}

// TimeLabel returns the time label for display (e.g., "19:00", "20:00")
func (v *Vote) TimeLabel() string {
	switch OptionKind(v.TgOptionIndex) {
	case OptionComeAt19:
		return "19:00"
	case OptionComeAt20:
		return "20:00"
	default:
		return ""
	}
}
