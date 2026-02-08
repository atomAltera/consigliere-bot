package bot

import (
	tele "gopkg.in/telebot.v4"
)

// handlePin pins the poll message
func (b *Bot) handlePin(c tele.Context) error {
	// Get active poll (validates event date hasn't passed)
	p, err := b.GetActivePollForAction(c.Chat().ID)
	if err != nil {
		return err
	}

	if p.TgMessageID == 0 {
		return UserErrorf(MsgPollMessageMissing)
	}

	// Unpin all previously pinned messages before pinning the new one
	if err := c.Bot().UnpinAll(MessageRef(p.TgChatID, 0).Chat); err != nil {
		b.logger.Warn("failed to unpin previous messages", "error", err)
	}

	// Pin the poll message (without Silent option to notify all members)
	if err := c.Bot().Pin(MessageRef(p.TgChatID, p.TgMessageID)); err != nil {
		return WrapUserError(MsgFailedPinPoll, err)
	}

	// Update poll status via service
	_, err = b.pollService.SetPinned(c.Chat().ID, true)
	if err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
