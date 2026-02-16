package bot

import (
	tele "gopkg.in/telebot.v4"
)

// handleCall sends a message mentioning all undecided voters
func (b *Bot) handleCall(c tele.Context) error {
	config := getClubConfig(c)

	// Get active poll (validates event date hasn't passed)
	p, err := b.GetActivePollForAction(c.Chat().ID)
	if err != nil {
		return err
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
	_, err = b.RenderAndSend(c, func() (string, error) {
		return RenderCallMessage(config.templates, &CallData{
			EventDate: p.EventDate,
			Members:   MembersFromVotes(votes),
		})
	}, MsgFailedRenderCall, MsgFailedSendCall)
	return err
}
