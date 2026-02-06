package bot

import (
	"errors"
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
	p, err := b.pollService.GetActivePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
	}

	// Get invitation data (votes grouped by option)
	invData, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	// Convert votes to ResultsVoter for detailed display
	resultsData := &ResultsData{
		EventDate:   p.EventDate,
		At19:        b.votesToResultsVoters(invData.Participants, poll.OptionComeAt19),
		At20:        b.votesToResultsVoters(invData.Participants, poll.OptionComeAt20),
		ComingLater: b.votesToResultsVotersAll(invData.ComingLater),
		Undecided:   b.votesToResultsVotersAll(invData.Undecided),
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

// votesToResultsVoters converts votes to ResultsVoter, filtering by option.
func (b *Bot) votesToResultsVoters(votes []*poll.Vote, option poll.OptionKind) []ResultsVoter {
	var result []ResultsVoter
	for _, v := range votes {
		if poll.OptionKind(v.TgOptionIndex) != option {
			continue
		}
		voter := b.voteToResultsVoter(v)
		result = append(result, voter)
	}
	return result
}

// votesToResultsVotersAll converts all votes to ResultsVoter.
func (b *Bot) votesToResultsVotersAll(votes []*poll.Vote) []ResultsVoter {
	result := make([]ResultsVoter, 0, len(votes))
	for _, v := range votes {
		voter := b.voteToResultsVoter(v)
		result = append(result, voter)
	}
	return result
}

// voteToResultsVoter converts a single vote to ResultsVoter with nickname lookup.
func (b *Bot) voteToResultsVoter(v *poll.Vote) ResultsVoter {
	voter := ResultsVoter{
		TgID:       v.TgUserID,
		TgUsername: v.TgUsername,
		TgName:     v.TgFirstName,
	}

	// Look up game nickname
	nick, err := b.pollService.GetDisplayNick(v.TgUserID, v.TgUsername)
	if err != nil {
		b.logger.Warn("failed to get display nick for results", "error", err)
	} else {
		voter.Nickname = nick
	}

	return voter
}
