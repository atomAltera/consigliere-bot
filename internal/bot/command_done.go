package bot

import (
	"errors"

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

// handleDone announces that enough players have been collected for the game
func (b *Bot) handleDone(c tele.Context) error {
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

	// Get votes for 19:00, 20:00, and 21:00+
	data, err := b.pollService.GetCollectedData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	// Determine start time and voter groups
	result := poll.DetermineStartTimeAndVoters(data, minPlayersRequired)
	if !result.EnoughPlayers {
		return UserErrorf(MsgNotEnoughPlayers)
	}

	// Delete old /done message if exists
	if p.TgDoneMessageID != 0 {
		if err := b.bot.Delete(MessageRef(p.TgChatID, p.TgDoneMessageID)); err != nil {
			b.logger.Warn("failed to delete old done message", "error", err)
		}
	}

	// Build a single cache for all voters (more efficient than 2 separate caches)
	cache, err := b.buildNicknameCacheFromVotes(result.MainVoters, result.ComingLater)
	if err != nil {
		b.logger.Warn("failed to build nickname cache for done message", "error", err)
		// Continue without nicknames - membersFromVotesWithCache handles nil cache
	}

	// Render and send collected message
	sentMsg, err := b.RenderAndSend(c, func() (string, error) {
		return RenderCollectedMessage(&CollectedData{
			EventDate:   p.EventDate,
			StartTime:   result.StartTime,
			Members:     b.membersFromVotesWithCache(result.MainVoters, cache),
			ComingLater: b.membersFromVotesWithCache(result.ComingLater, cache),
		})
	}, MsgFailedRenderCollected, MsgFailedSendCollected)
	if err != nil {
		return err
	}

	// Store done message ID
	p.TgDoneMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
