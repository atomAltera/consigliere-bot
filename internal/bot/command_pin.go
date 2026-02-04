package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handlePin pins the poll message
func (b *Bot) handlePin(c tele.Context) error {
	b.logger.Info("command /pin",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get active poll
	p, err := b.pollService.GetActivePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
	}

	// Check if event date is in the past
	if isPollDatePassed(p.EventDate) {
		return UserErrorf(MsgPollDatePassed)
	}

	if p.TgMessageID == 0 {
		return UserErrorf(MsgPollMessageMissing)
	}

	// Unpin all previously pinned messages before pinning the new one
	chat := &tele.Chat{ID: p.TgChatID}
	if err := c.Bot().UnpinAll(chat); err != nil {
		b.logger.Warn("failed to unpin previous messages", "error", err)
	}

	// Pin the poll message (without Silent option to notify all members)
	msg := &tele.Message{
		ID:   p.TgMessageID,
		Chat: chat,
	}

	if err := c.Bot().Pin(msg); err != nil {
		return WrapUserError(MsgFailedPinPoll, err)
	}

	// Update poll status via service
	_, err = b.pollService.SetPinned(c.Chat().ID, true)
	if err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
