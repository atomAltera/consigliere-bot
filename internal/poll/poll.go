package poll

import "time"

type Poll struct {
	ID                 int64
	TgChatID           int64
	TgPollID           string
	TgMessageID        int
	// TgResultsMessageID stores the invitation message ID (the persistent message
	// that displays current votes and updates as users vote). Named "results" for
	// historical reasons; /results command sends a separate temporary message.
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
