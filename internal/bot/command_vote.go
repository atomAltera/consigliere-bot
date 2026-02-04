package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

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
