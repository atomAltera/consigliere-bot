package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleResults recreates the invitation message for the latest active poll
func (b *Bot) handleResults(c tele.Context) error {
	b.logger.Info("command /results",
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

	// Get invitation data
	results, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	results.Poll = p
	results.EventDate = p.EventDate
	results.IsCancelled = !p.IsActive

	// Render invitation as HTML
	html, err := RenderInvitation(results)
	if err != nil {
		return WrapUserError(MsgFailedRenderResults, err)
	}

	// Send new invitation message
	sentMsg, err := c.Bot().Send(c.Chat(), html, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		return WrapUserError(MsgFailedSendResults, err)
	}

	// Delete previous results message only after new one is sent successfully
	oldResultsMessageID := p.TgResultsMessageID

	// Update poll with new results message ID
	p.TgResultsMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSaveResults, err)
	}

	// Now safe to delete the old message
	if oldResultsMessageID != 0 {
		msg := &tele.Message{
			ID: oldResultsMessageID,
			Chat: &tele.Chat{
				ID: p.TgChatID,
			},
		}
		if err := c.Bot().Delete(msg); err != nil {
			b.logger.Warn("failed to delete previous invitation message", "error", err)
		}
	}

	return nil
}
