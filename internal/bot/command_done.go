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

	// Get votes for 19:00 and 20:00
	data, err := b.pollService.GetCollectedData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	count19 := len(data.Votes19)
	count20 := len(data.Votes20)
	totalEarly := count19 + count20

	// Determine start time and which voters to mention
	var startTime string
	var votesToMention []*poll.Vote

	if count19 >= minPlayersRequired {
		// Enough players at 19:00
		startTime = "19:00"
		votesToMention = data.Votes19
	} else if totalEarly >= minPlayersRequired {
		// Combined 19:00 + 20:00 is enough, start at 20:00
		startTime = "20:00"
		votesToMention = append(data.Votes19, data.Votes20...)
	} else {
		// Not enough players
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
		StartTime:   startTime,
		Members:     b.membersFromVotesWithNicknames(votesToMention),
		ComingLater: b.membersFromVotesWithNicknames(data.Votes21),
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
