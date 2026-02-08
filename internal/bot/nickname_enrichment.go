package bot

import (
	"nuclight.org/consigliere/internal/poll"
)

// enrichVotesWithCache looks up game nicknames and modifies votes in place.
// When a game nickname is found, TgFirstName is set to the display nick (with gender prefix)
// and TgUsername is cleared so Vote.DisplayName() returns the nickname.
// If cache is nil, creates one internally (less efficient for multiple call sites).
func (b *Bot) enrichVotesWithCache(votes []*poll.Vote, cache *poll.NicknameCache) {
	if len(votes) == 0 {
		return
	}

	// Create cache if not provided
	if cache == nil {
		var err error
		cache, err = b.pollService.NewNicknameCacheFromVotes(votes)
		if err != nil {
			b.logger.Warn("failed to batch fetch nicknames for enrichment", "error", err)
			return
		}
	}

	for _, v := range votes {
		// GetDisplayNick returns nickname with gender prefix (e.g., "г-н Кринж")
		nick := cache.GetDisplayNick(v.TgUserID, v.TgUsername)
		if nick != "" {
			// Store nickname in TgFirstName for display (Vote.DisplayName() will use it)
			v.TgFirstName = nick
			// Clear username so DisplayName() uses TgFirstName (nickname)
			v.TgUsername = ""
		}
	}
}

// membersFromVotesWithCache creates Members from Votes with nicknames looked up.
// Preserves both TgUsername and Nickname for display.
// If cache is nil, creates one internally (less efficient for multiple call sites).
func (b *Bot) membersFromVotesWithCache(votes []*poll.Vote, cache *poll.NicknameCache) []Member {
	members := make([]Member, 0, len(votes))

	// Create cache if not provided
	if cache == nil {
		var err error
		cache, err = b.pollService.NewNicknameCacheFromVotes(votes)
		if err != nil {
			b.logger.Warn("failed to batch fetch nicknames", "error", err)
			// Fallback: return members without nicknames
			for _, v := range votes {
				members = append(members, Member{
					TgID:       v.TgUserID,
					TgName:     v.TgFirstName,
					TgUsername: v.TgUsername,
				})
			}
			return members
		}
	}

	for _, v := range votes {
		m := Member{
			TgID:       v.TgUserID,
			TgName:     v.TgFirstName,
			TgUsername: v.TgUsername,
			Nickname:   cache.GetDisplayNick(v.TgUserID, v.TgUsername),
		}
		members = append(members, m)
	}
	return members
}

// votesToResultsVotersWithCache converts votes to ResultsVoter, filtering by option.
// Uses a pre-populated nickname cache to avoid N+1 queries.
// Note: Uses cache.Get() which returns nick WITHOUT gender prefix (for admin display).
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

// buildNicknameCacheFromVotes creates a single cache from multiple vote slices.
// This is more efficient than creating separate caches for each slice.
func (b *Bot) buildNicknameCacheFromVotes(voteLists ...[]*poll.Vote) (*poll.NicknameCache, error) {
	// Count total votes for pre-allocation
	total := 0
	for _, votes := range voteLists {
		total += len(votes)
	}

	// Combine all votes
	allVotes := make([]*poll.Vote, 0, total)
	for _, votes := range voteLists {
		allVotes = append(allVotes, votes...)
	}

	return b.pollService.NewNicknameCacheFromVotes(allVotes)
}

// Backwards-compatible wrappers that create cache internally

// enrichVotesWithNicknames looks up game nicknames for votes and sets them.
// Note: This modifies votes in place and clears username for display purposes.
// Deprecated: Use enrichVotesWithCache with a pre-built cache for better efficiency.
func (b *Bot) enrichVotesWithNicknames(votes []*poll.Vote) {
	b.enrichVotesWithCache(votes, nil)
}

// membersFromVotesWithNicknames creates Members from Votes with nicknames looked up.
// Deprecated: Use membersFromVotesWithCache with a pre-built cache for better efficiency.
func (b *Bot) membersFromVotesWithNicknames(votes []*poll.Vote) []Member {
	return b.membersFromVotesWithCache(votes, nil)
}
