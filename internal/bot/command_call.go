package bot

import (
	"errors"
	"strings"

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
