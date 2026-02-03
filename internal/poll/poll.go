package poll

import "time"

type Poll struct {
	ID                 int64
	TgChatID           int64
	TgPollID           string
	TgMessageID        int
	TgResultsMessageID int
	TgCancelMessageID  int
	EventDate          time.Time
	Options            []OptionKind
	IsActive           bool
	IsPinned           bool
	CreatedAt          time.Time
}

// CreatePollResult contains the result of creating a poll,
// including any poll that was replaced (deactivated due to past event date).
type CreatePollResult struct {
	Poll         *Poll
	ReplacedPoll *Poll // Non-nil if an old poll with past event date was deactivated
}
