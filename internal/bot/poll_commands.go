package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handlePoll creates a new poll for the specified date
// Usage: /poll [day|YYYY-MM-DD]
// - No arguments: nearest Monday or Saturday
// - Day name: monday, mon, saturday, sat, etc.
// - Explicit date: YYYY-MM-DD
func (b *Bot) handlePoll(c tele.Context) error {
	eventDate, err := parseEventDate(c.Args())
	if err != nil {
		return UserErrorf(MsgInvalidDateFormat)
	}

	b.logger.Info("command /poll",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"event_date", eventDate.Format("2006-01-02"),
	)

	// Create poll in database (service checks for existing poll)
	result, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		if errors.Is(err, poll.ErrPollExists) {
			return UserErrorf(MsgPollAlreadyExists)
		}
		return WrapUserError(MsgFailedCreatePoll, err)
	}
	p := result.Poll

	// Unpin old poll message if it was replaced and pinned
	if result.ReplacedPoll != nil && result.ReplacedPoll.TgMessageID != 0 {
		chat := &tele.Chat{ID: c.Chat().ID}
		if err := c.Bot().Unpin(chat, result.ReplacedPoll.TgMessageID); err != nil {
			// Non-critical: old message may have been deleted, just log
			b.logger.Warn("failed to unpin replaced poll", "error", err)
		}
	}

	// Helper to rollback poll on failure (deactivate so a new one can be created)
	rollbackPoll := func() {
		p.IsActive = false
		if updateErr := b.pollService.UpdatePoll(p); updateErr != nil {
			b.logger.Error("failed to rollback poll after error", "error", updateErr, "poll_id", p.ID)
		}
	}

	// Send invitation message first (empty participants)
	invitationData := &poll.InvitationData{
		Poll:         p,
		EventDate:    eventDate,
		Participants: []*poll.Vote{},
		ComingLater:  []*poll.Vote{},
		Undecided:    []*poll.Vote{},
		IsCancelled:  false,
	}

	invitationHTML, err := RenderInvitation(invitationData)
	if err != nil {
		rollbackPoll()
		return WrapUserError(MsgFailedRenderResults, err)
	}

	invitationMsg, err := b.SendWithRetry(c.Chat(), invitationHTML, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		rollbackPoll()
		return WrapUserError(MsgFailedSendResults, err)
	}

	// Store invitation message ID
	p.TgResultsMessageID = invitationMsg.ID

	// Render poll title from template
	pollTitle, err := RenderPollTitle(eventDate)
	if err != nil {
		// Clean up invitation message and rollback poll on failure
		_ = b.bot.Delete(invitationMsg)
		rollbackPoll()
		return WrapUserError(MsgFailedRenderPollTitle, err)
	}

	// Create Telegram poll
	pollOptions := AllOptionLabels()
	telePoll := &tele.Poll{
		Type:            tele.PollRegular,
		Question:        pollTitle,
		MultipleAnswers: false,
		Anonymous:       false,
	}

	// Add options
	telePoll.AddOptions(pollOptions...)

	// Send poll to the chat
	sentMsg, err := b.SendWithRetry(c.Chat(), telePoll)
	if err != nil {
		// Clean up invitation message and rollback poll on failure
		_ = b.bot.Delete(invitationMsg)
		rollbackPoll()
		return WrapUserError(MsgFailedSendPoll, err)
	}

	// Update poll with Telegram IDs
	p.TgPollID = sentMsg.Poll.ID
	p.TgMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePoll, err)
	}

	return nil
}

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

// handleVote manually records a vote for a user
// Usage: /vote @username <option number 1-5>
// Options: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming
func (b *Bot) handleVote(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return UserErrorf(MsgVoteUsage)
	}

	// Parse username (remove @ prefix if present)
	username := strings.TrimPrefix(args[0], "@")
	if username == "" {
		return UserErrorf(MsgInvalidUsername)
	}

	// Parse option number (1-5)
	optionNum, err := strconv.Atoi(args[1])
	if err != nil || optionNum < 1 || optionNum > 5 {
		return UserErrorf(MsgInvalidVoteOption)
	}

	// Convert to 0-indexed option
	optionIndex := optionNum - 1

	b.logger.Info("command /vote",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"target_username", username,
		"option", OptionLabel(poll.OptionKind(optionIndex)),
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

	// Create manual vote with synthetic user ID
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      poll.ManualUserID(username),
		TgUsername:    username,
		TgFirstName:   username, // Use username as first name for display
		TgOptionIndex: optionIndex,
		IsManual:      true,
	}

	if err := b.pollService.RecordVote(v); err != nil {
		return WrapUserError(MsgFailedRecordVote, err)
	}

	// Update invitation message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			return WrapUserError(MsgFailedGetResults, err)
		}
		results.Poll = p
		results.EventDate = p.EventDate
		results.IsCancelled = !p.IsActive

		html, err := RenderInvitation(results)
		if err != nil {
			return WrapUserError(MsgFailedRenderResults, err)
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
			b.logger.Warn("failed to update invitation message", "error", err)
		}
	}

	_, err = b.SendTemporary(c.Chat(), fmt.Sprintf(MsgFmtVoteRecorded, username, OptionLabel(poll.OptionKind(optionIndex))), 0)
	return err
}
