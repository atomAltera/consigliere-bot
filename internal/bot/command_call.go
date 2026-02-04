package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

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

	// Get undecided votes
	votes, err := b.pollService.GetUndecidedVotes(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetUndecided, err)
	}

	if len(votes) == 0 {
		return UserErrorf(MsgNoUndecidedVoters)
	}

	// Render and send call message
	html, err := RenderCallMessage(&CallData{
		EventDate: p.EventDate,
		Members:   MembersFromVotes(votes),
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
