package bot

import (
	"errors"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleNick manages game nickname mappings.
// Usage:
//
//	/nick @username gamenick — link by telegram username
//	/nick @username "nick with spaces" — quoted nickname
//	/nick @username gamenick м — with gender (м/ж/m/f/д)
//	/nick 123456 gamenick   — link by telegram user ID
func (b *Bot) handleNick(c tele.Context) error {
	// Join all args back together to parse with our custom parser
	// that handles quotes properly
	input := strings.Join(c.Args(), " ")
	if input == "" {
		return UserErrorf(MsgNickUsage)
	}

	args, err := ParseNickArgs(input)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "invalid gender") {
			return UserErrorf(MsgInvalidGender)
		}
		return UserErrorf(MsgNickUsage)
	}

	b.logger.Info("command /nick",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"tg_user_id", args.TgUserID,
		"tg_username", args.TgUsername,
		"game_nick", args.Nickname,
		"gender", args.Gender.String(),
	)

	// Create the nickname mapping
	created, err := b.pollService.CreateNickname(args.TgUserID, args.TgUsername, args.Nickname, args.Gender.String())
	if err != nil {
		return WrapUserError(MsgFailedSaveNick, err)
	}

	// Resolve user ID for data consistency
	var resolvedUserID int64
	var username string
	if args.TgUserID != nil {
		resolvedUserID = *args.TgUserID
	} else if args.TgUsername != nil {
		// Try to infer user ID from vote history
		if id, found, err := b.pollService.LookupUserIDByUsername(*args.TgUsername); err == nil && found {
			resolvedUserID = id
		}
	}
	if args.TgUsername != nil {
		username = *args.TgUsername
	}

	// Ensure data consistency: update nicknames and consolidate synthetic votes
	if resolvedUserID > 0 {
		if err := b.pollService.EnsureUserDataConsistency(c.Chat().ID, resolvedUserID, username); err != nil {
			b.logger.Warn("failed to ensure user data consistency", "error", err)
		}
	}

	// Update invitation message if exists (regardless of user ID - can match by username)
	if err := b.refreshInvitationMessage(c.Chat().ID); err != nil {
		b.logger.Warn("failed to refresh invitation after nick", "error", err)
	}

	// Send confirmation
	var msg string
	if created {
		if args.TgUsername != nil {
			msg = fmt.Sprintf(MsgFmtNickCreated, "@"+*args.TgUsername, args.Nickname)
		} else {
			msg = fmt.Sprintf(MsgFmtNickCreatedByID, *args.TgUserID, args.Nickname)
		}
	} else {
		msg = MsgNickDuplicate
	}

	_, err = b.SendTemporary(c.Chat(), msg, 0)
	return err
}

// refreshInvitationMessage updates the invitation message if an active poll exists.
func (b *Bot) refreshInvitationMessage(chatID int64) error {
	p, err := b.pollService.GetActivePoll(chatID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return nil // No active poll is fine
		}
		return err // Propagate real errors
	}

	if p.TgInvitationMessageID == 0 {
		return nil // No invitation message to update
	}

	results, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return err
	}

	p.PopulateInvitationData(results)

	html, err := b.RenderInvitationWithNicks(results)
	if err != nil {
		return err
	}

	_, err = b.bot.Edit(MessageRef(p.TgChatID, p.TgInvitationMessageID), html, tele.ModeHTML)
	return err
}


// RenderInvitationWithNicks renders invitation with nicknames resolved.
func (b *Bot) RenderInvitationWithNicks(data *poll.InvitationData) (string, error) {
	// Build a single cache for all vote lists (more efficient than 3 separate caches)
	cache, err := b.buildNicknameCacheFromVotes(data.Participants, data.ComingLater, data.Undecided)
	if err != nil {
		b.logger.Warn("failed to build nickname cache for invitation", "error", err)
		// Continue without nicknames - enrichVotesWithCache handles nil cache
	}

	// Enrich votes with nicknames using shared cache
	b.enrichVotesWithCache(data.Participants, cache)
	b.enrichVotesWithCache(data.ComingLater, cache)
	b.enrichVotesWithCache(data.Undecided, cache)

	return RenderInvitation(data)
}
