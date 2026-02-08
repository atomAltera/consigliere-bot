package bot

import (
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleResults shows detailed voter information for admins.
// Displays telegram IDs (copiable), usernames, names, and game nicknames.
// Sent as a temporary silent message.
func (b *Bot) handleResults(c tele.Context) error {
	b.logger.Info("command /results",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get active poll
	p, err := b.GetActivePollOrError(c.Chat().ID)
	if err != nil {
		return err
	}

	// Get invitation data (votes grouped by option)
	invData, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	// Collect all votes for batch nickname lookup
	allVotes := make([]*poll.Vote, 0, len(invData.Participants)+len(invData.ComingLater)+len(invData.Undecided))
	allVotes = append(allVotes, invData.Participants...)
	allVotes = append(allVotes, invData.ComingLater...)
	allVotes = append(allVotes, invData.Undecided...)

	// Pre-fetch all nicknames in one batch
	cache, err := b.pollService.NewNicknameCacheFromVotes(allVotes)
	if err != nil {
		b.logger.Warn("failed to batch fetch nicknames for results", "error", err)
		// Continue without nicknames - cache will be nil
	}

	// Convert votes to ResultsVoter for detailed display
	resultsData := &ResultsData{
		EventDate:   p.EventDate,
		At19:        b.votesToResultsVotersWithCache(invData.Participants, poll.OptionComeAt19, cache),
		At20:        b.votesToResultsVotersWithCache(invData.Participants, poll.OptionComeAt20, cache),
		ComingLater: b.votesToResultsVotersAllWithCache(invData.ComingLater, cache),
		Undecided:   b.votesToResultsVotersAllWithCache(invData.Undecided, cache),
	}

	// Render results message
	html, err := RenderResultsMessage(resultsData)
	if err != nil {
		return WrapUserError(MsgFailedRenderResults, err)
	}

	// Send as temporary silent message (30 seconds to allow copying IDs)
	_, err = b.SendTemporary(c.Chat(), html, 30*time.Second, tele.ModeHTML)
	return err
}

