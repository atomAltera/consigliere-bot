package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get invitation data for cancellation", "error", err)
		} else {
			results.Poll = p
			results.EventDate = p.EventDate
			results.IsCancelled = true

			html, err := RenderInvitation(results)
			if err != nil {
				b.logger.Warn("failed to render cancelled invitation", "error", err)
			} else {
				chat := &tele.Chat{ID: p.TgChatID}
				msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to update invitation message with cancellation", "error", err)
				}
			}
		}
	}

	// Post cancellation notification with mentions
	cancelData := &CancelData{EventDate: p.EventDate}

	// Add mentions of attending participants
	usernames, err := b.pollService.GetAttendingUsernames(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending usernames", "error", err)
	} else if len(usernames) > 0 {
		mentions := make([]string, len(usernames))
		for i, u := range usernames {
			mentions[i] = "@" + u
		}
		cancelData.Mentions = strings.Join(mentions, ", ")
	}

	cancellationMsg, err := RenderCancelMessage(cancelData)
	if err != nil {
		return WrapUserError(MsgFailedRenderCancellation, err)
	}

	sentMsg, err := c.Bot().Send(c.Chat(), cancellationMsg)
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

	// Get attending usernames for mentions
	usernames, err := b.pollService.GetAttendingUsernames(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending usernames", "error", err)
	}

	// Build mentions string
	var mentions string
	if len(usernames) > 0 {
		mentionList := make([]string, len(usernames))
		for i, u := range usernames {
			mentionList[i] = "@" + u
		}
		mentions = strings.Join(mentionList, " ")
	}

	// Render and send restore message
	html, err := RenderRestoreMessage(&RestoreData{
		EventDate: p.EventDate,
		Mentions:  mentions,
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

// handleCall sends a message mentioning all undecided voters
func (b *Bot) handleCall(c tele.Context) error {
	b.logger.Info("command /call",
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

	// Get undecided usernames
	usernames, err := b.pollService.GetUndecidedUsernames(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetUndecided, err)
	}

	if len(usernames) == 0 {
		return UserErrorf(MsgNoUndecidedVoters)
	}

	// Build mentions string
	mentions := make([]string, len(usernames))
	for i, u := range usernames {
		mentions[i] = "@" + u
	}

	// Render and send call message
	html, err := RenderCallMessage(&CallData{
		EventDate: p.EventDate,
		Mentions:  strings.Join(mentions, " "),
	})
	if err != nil {
		return WrapUserError(MsgFailedRenderCall, err)
	}

	_, err = b.SendWithRetry(c.Chat(), html, tele.ModeHTML)
	if err != nil {
		return WrapUserError(MsgFailedSendCall, err)
	}

	return nil
}

// handleHelp shows the help message with all available commands
func (b *Bot) handleHelp(c tele.Context) error {
	b.logger.Info("command /help",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	helpText, err := HelpMessage()
	if err != nil {
		return fmt.Errorf("read help template: %w", err)
	}
	_, err = b.SendTemporary(c.Chat(), helpText, 30*time.Second, tele.ModeHTML)
	return err
}
