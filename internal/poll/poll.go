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
	IsActive           bool
	IsPinned           bool
	CreatedAt          time.Time
}
