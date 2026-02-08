package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleCancel cancels the event and updates the invitation message with cancellation footer
func (b *Bot) handleCancel(c tele.Context) error {
	// Get active poll to check date first
	p, err := b.GetActivePollOrError(c.Chat().ID)
	if err != nil {
		return err
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
		if err := c.Bot().Unpin(MessageRef(p.TgChatID, 0).Chat, p.TgMessageID); err != nil {
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

	sentMsg, err := b.RenderAndSend(c, func() (string, error) {
		return RenderCancelMessage(cancelData)
	}, MsgFailedRenderCancellation, MsgFailedSendCancellation)
	if err != nil {
		return err
	}

	// Save cancel message ID
	p.TgCancelMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
