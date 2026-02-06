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

// votesToResultsVotersWithCache converts votes to ResultsVoter, filtering by option.
// Uses a pre-populated nickname cache to avoid N+1 queries.
func (b *Bot) votesToResultsVotersWithCache(votes []*poll.Vote, option poll.OptionKind, cache *poll.NicknameCache) []ResultsVoter {
	var result []ResultsVoter
	for _, v := range votes {
		if poll.OptionKind(v.TgOptionIndex) != option {
			continue
		}
		voter := b.voteToResultsVoterWithCache(v, cache)
		result = append(result, voter)
	}
	return result
}

// votesToResultsVotersAllWithCache converts all votes to ResultsVoter.
// Uses a pre-populated nickname cache to avoid N+1 queries.
func (b *Bot) votesToResultsVotersAllWithCache(votes []*poll.Vote, cache *poll.NicknameCache) []ResultsVoter {
	result := make([]ResultsVoter, 0, len(votes))
	for _, v := range votes {
		voter := b.voteToResultsVoterWithCache(v, cache)
		result = append(result, voter)
	}
	return result
}

// voteToResultsVoterWithCache converts a single vote to ResultsVoter using cached nickname.
func (b *Bot) voteToResultsVoterWithCache(v *poll.Vote, cache *poll.NicknameCache) ResultsVoter {
	voter := ResultsVoter{
		TgID:       v.TgUserID,
		TgUsername: v.TgUsername,
		TgName:     v.TgFirstName,
	}

	// Get nickname from cache (nil-safe)
	if cache != nil {
		voter.Nickname = cache.Get(v.TgUserID, v.TgUsername)
	}

	return voter
}
