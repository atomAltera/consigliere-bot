package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleCancel cancels the event and updates the invitation message with cancellation footer
func (b *Bot) handleCancel(c tele.Context) error {
	b.logger.Info("command /cancel",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get active poll to check date first
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

	// Cancel poll via service (marks as inactive)
	p, err = b.pollService.CancelPoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedCancelPoll, err)
	}

	// Unpin the poll message if it was pinned
	if p.TgMessageID != 0 {
		chat := &tele.Chat{ID: p.TgChatID}
		if err := c.Bot().Unpin(chat, p.TgMessageID); err != nil {
			b.logger.Warn("failed to unpin poll message", "error", err)
		}
	}

	// Update invitation message with cancellation footer
	cancelled := true
	b.UpdateInvitationMessage(p, &cancelled)

	// Post cancellation notification with mentions
	cancelData := &CancelData{EventDate: p.EventDate}

	// Add attending participants as members
	votes, err := b.pollService.GetAttendingVotes(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending votes", "error", err)
	} else {
		cancelData.Members = MembersFromVotes(votes)
	}

	cancellationMsg, err := RenderCancelMessage(cancelData)
	if err != nil {
		return WrapUserError(MsgFailedRenderCancellation, err)
	}

	sentMsg, err := b.SendWithRetry(c.Chat(), cancellationMsg, tele.ModeHTML)
	if err != nil {
		return WrapUserError(MsgFailedSendCancellation, err)
	}

	// Save cancel message ID
	p.TgCancelMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
