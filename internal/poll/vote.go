package poll

import (
	"hash/fnv"
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

// ManualUserID generates a synthetic user ID for manual votes based on username.
// Uses negative IDs to avoid collision with real Telegram user IDs (which are positive).
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
		return "@" + v.TgUsername
	}
	return v.TgFirstName
}
