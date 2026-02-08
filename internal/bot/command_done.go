package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

const minPlayersRequired = 11

// handleDone announces that enough players have been collected for the game
func (b *Bot) handleDone(c tele.Context) error {
	b.logger.Info("command /done",
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
		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgDoneMessageID, Chat: chat}
		if err := b.bot.Delete(msg); err != nil {
			b.logger.Warn("failed to delete old done message", "error", err)
		}
	}

	// Render and send collected message
	html, err := RenderCollectedMessage(&CollectedData{
		EventDate:   p.EventDate,
		StartTime:   result.StartTime,
		Members:     b.membersFromVotesWithNicknames(result.MainVoters),
		ComingLater: b.membersFromVotesWithNicknames(result.ComingLater),
	})
	if err != nil {
		return WrapUserError(MsgFailedRenderCollected, err)
	}

	sentMsg, err := b.SendWithRetry(c.Chat(), html, tele.ModeHTML)
	if err != nil {
		return WrapUserError(MsgFailedSendCollected, err)
	}

	// Store done message ID
	p.TgDoneMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}
