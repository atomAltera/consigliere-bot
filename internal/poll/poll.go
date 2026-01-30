package poll

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusPinned    Status = "pinned"
	StatusCancelled Status = "cancelled"
)

type Poll struct {
	ID                  int64
	TgChatID            int64
	TgPollID            string
	TgMessageID         int
	TgResultsMessageID  int
	EventDate           time.Time
	Status              Status
	CreatedAt           time.Time
}
