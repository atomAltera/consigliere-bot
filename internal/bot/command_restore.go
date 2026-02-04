package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleRestore restores the last cancelled poll if it's for today or a future date
func (b *Bot) handleRestore(c tele.Context) error {
	b.logger.Info("command /restore",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Restore poll via service (validates date, marks as active)
	p, err := b.pollService.RestorePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoCancelledPoll) {
			return UserErrorf(MsgNoCancelledPoll)
		}
		if errors.Is(err, poll.ErrPollDatePassed) {
			return UserErrorf(MsgPollDatePassed)
		}
		return WrapUserError(MsgFailedRestorePoll, err)
	}

	// Update invitation message to remove cancellation footer
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get invitation data for restore", "error", err)
		} else {
			results.Poll = p
			results.EventDate = p.EventDate
			results.IsCancelled = false

			html, err := RenderInvitation(results)
			if err != nil {
				b.logger.Warn("failed to render restored invitation", "error", err)
			} else {
				chat := &tele.Chat{ID: p.TgChatID}
				msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to update invitation message", "error", err)
				}
			}
		}
	}

	// Delete cancellation notification message
	if p.TgCancelMessageID != 0 {
		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgCancelMessageID, Chat: chat}
		if err := b.bot.Delete(msg); err != nil {
			b.logger.Warn("failed to delete cancellation message", "error", err)
		}
		p.TgCancelMessageID = 0
	}

	// Get attending votes for members
	votes, err := b.pollService.GetAttendingVotes(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending votes", "error", err)
	}

	// Render and send restore message
	html, err := RenderRestoreMessage(&RestoreData{
		EventDate: p.EventDate,
		Members:   MembersFromVotes(votes),
	})
	if err != nil {
		return WrapUserError(MsgFailedRenderRestore, err)
	}

	restoreMsg, err := b.SendWithRetry(c.Chat(), html, tele.ModeHTML)
	if err != nil {
		return WrapUserError(MsgFailedSendRestore, err)
	}

	// Pin the restore message
	if err := c.Bot().Pin(restoreMsg); err != nil {
		b.logger.Warn("failed to pin restore message", "error", err)
	}

	// Save poll (clear cancel message ID)
	if err := b.pollService.UpdatePoll(p); err != nil {
		b.logger.Warn("failed to update poll after restore", "error", err)
	}

	return nil
}
