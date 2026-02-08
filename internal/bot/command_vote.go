package bot

import (
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleVote manually records a vote for a user
// Usage:
//
//	/vote @username <option 1-5> — vote by telegram username
//	/vote gamenick <option 1-5>  — vote by game nickname (no @ prefix)
//
// Options: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming
func (b *Bot) handleVote(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return UserErrorf(MsgVoteUsage)
	}

	identifier := args[0]
	if identifier == "" {
		return UserErrorf(MsgInvalidUsername)
	}

	// Parse option number (1-5)
	optionNum, err := strconv.Atoi(args[1])
	if err != nil || optionNum < 1 || optionNum > 5 {
		return UserErrorf(MsgInvalidVoteOption)
	}

	// Convert to 0-indexed option
	optionIndex := optionNum - 1

	// Resolve the identifier to user info
	userID, username, displayName, err := b.pollService.ResolveVoteIdentifier(identifier)
	if err != nil {
		return WrapUserError(MsgFailedRecordVote, err)
	}

	b.logger.Info("vote parameters",
		"identifier", identifier,
		"resolved_user_id", userID,
		"resolved_username", username,
		"display_name", displayName,
		"option", OptionLabel(poll.OptionKind(optionIndex)),
	)

	// Get active poll
	p, err := b.GetActivePollOrError(c.Chat().ID)
	if err != nil {
		return err
	}

	// Check if event date is in the past
	if isPollDatePassed(p.EventDate) {
		return UserErrorf(MsgPollDatePassed)
	}

	// Create vote with resolved user ID
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      userID,
		TgUsername:    username,
		TgFirstName:   displayName,
		TgOptionIndex: optionIndex,
		IsManual:      true,
	}

	if err := b.pollService.RecordVote(v); err != nil {
		return WrapUserError(MsgFailedRecordVote, err)
	}

	// Ensure data consistency if we have a real user ID
	if userID > 0 {
		if err := b.pollService.EnsureUserDataConsistency(c.Chat().ID, userID, username); err != nil {
			b.logger.Warn("failed to ensure user data consistency", "error", err)
		}
	}

	// Update invitation message if exists
	b.UpdateInvitationMessage(p, nil)

	_, err = b.SendTemporary(c.Chat(), fmt.Sprintf(MsgFmtVoteRecorded, displayName, OptionLabel(poll.OptionKind(optionIndex))), 0)
	return err
}
