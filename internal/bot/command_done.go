package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

const minPlayersRequired = 11

// uncancelPoll silently restores a cancelled poll: marks it active, deletes the
// cancel message from chat, and updates the invitation to remove the cancellation footer.
func (b *Bot) uncancelPoll(chatID int64) (*poll.Poll, error) {
	p, err := b.pollService.RestorePoll(chatID)
	if err != nil {
		if errors.Is(err, poll.ErrNoCancelledPoll) {
			return nil, UserErrorf(MsgNoActivePoll)
		}
		if errors.Is(err, poll.ErrPollDatePassed) {
			return nil, UserErrorf(MsgPollDatePassed)
		}
		return nil, WrapUserError(MsgFailedRestorePoll, err)
	}

	// Delete cancellation message from chat
	if p.TgCancelMessageID != 0 {
		if err := b.bot.Delete(MessageRef(p.TgChatID, p.TgCancelMessageID)); err != nil {
			b.logger.Warn("failed to delete cancellation message", "error", err)
		}
		p.TgCancelMessageID = 0
	}

	// Update invitation message to remove cancellation footer
	notCancelled := false
	b.UpdateInvitationMessage(p, &notCancelled)

	if err := b.pollService.UpdatePoll(p); err != nil {
		b.logger.Warn("failed to update poll after uncancel", "error", err)
	}

	return p, nil
}

// parseStartTime parses a start time argument.
// Accepts: "19" → "19:00", "20" → "20:00", or "HH:MM" format.
// Returns formatted "HH:MM" string or error if invalid.
func parseStartTime(arg string) (string, error) {
	var hours, minutes int
	var err error

	if parts := strings.SplitN(arg, ":", 2); len(parts) == 2 {
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return "", fmt.Errorf("invalid hours: %s", parts[0])
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("invalid minutes: %s", parts[1])
		}
	} else {
		hours, err = strconv.Atoi(arg)
		if err != nil {
			return "", fmt.Errorf("invalid time: %s", arg)
		}
		minutes = 0
	}

	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
		return "", fmt.Errorf("time out of range: %02d:%02d", hours, minutes)
	}

	return fmt.Sprintf("%d:%02d", hours, minutes), nil
}

// handleDone announces that enough players have been collected for the game
func (b *Bot) handleDone(c tele.Context) error {
	// Parse optional start time argument
	var overrideTime string
	if args := c.Args(); len(args) > 0 {
		t, err := parseStartTime(args[0])
		if err != nil {
			return UserErrorf(MsgInvalidStartTime)
		}
		overrideTime = t
	}

	// Get active poll, or silently uncancel a cancelled poll
	chatID := c.Chat().ID
	p, err := b.pollService.GetActivePoll(chatID)
	if err != nil {
		if !errors.Is(err, poll.ErrNoActivePoll) {
			return WrapUserError(MsgFailedGetPoll, err)
		}
		p, err = b.uncancelPoll(chatID)
		if err != nil {
			return err
		}
	}
	if isPollDatePassed(p.EventDate) {
		return UserErrorf(MsgPollDatePassed)
	}

	// Get votes categorized by time slot
	data, err := b.pollService.GetCollectedData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	var startTime string
	var mainVoters, laterVoters []*poll.Vote

	if overrideTime != "" {
		// Manual mode: use provided time, split voters accordingly, no player count check
		startTime = overrideTime
		mainVoters, laterVoters = poll.SplitVotersByStartTime(data, startTime)
	} else {
		// Auto mode: determine start time from vote counts
		result := poll.DetermineStartTimeAndVoters(data, minPlayersRequired)
		if !result.EnoughPlayers {
			return UserErrorf(MsgNotEnoughPlayers)
		}
		startTime = result.StartTime
		mainVoters = result.MainVoters
		laterVoters = result.ComingLater
	}

	// Build nickname cache and convert to members
	cache, err := b.buildNicknameCacheFromVotes(mainVoters, laterVoters)
	if err != nil {
		b.logger.Warn("failed to build nickname cache for done message", "error", err)
	}
	members := b.membersFromVotesWithCache(mainVoters, cache)
	comingLater := b.membersFromVotesWithCache(laterVoters, cache)

	// Delete old /done message if exists
	if p.TgDoneMessageID != 0 {
		if err := b.bot.Delete(MessageRef(p.TgChatID, p.TgDoneMessageID)); err != nil {
			b.logger.Warn("failed to delete old done message", "error", err)
		}
	}

	// Render and send collected message
	sentMsg, err := b.RenderAndSend(c, func() (string, error) {
		return RenderCollectedMessage(&CollectedData{
			EventDate:   p.EventDate,
			StartTime:   startTime,
			Members:     members,
			ComingLater: comingLater,
		})
	}, MsgFailedRenderCollected, MsgFailedSendCollected)
	if err != nil {
		return err
	}

	// Store done message ID and start time
	p.TgDoneMessageID = sentMsg.ID
	p.StartTime = startTime
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
