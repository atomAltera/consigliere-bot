package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleNick manages game nickname mappings.
// Usage:
//
//	/nick @username gamenick — link by telegram username
//	/nick 123456 gamenick   — link by telegram user ID
func (b *Bot) handleNick(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return UserErrorf(MsgNickUsage)
	}

	identifier := args[0]
	gameNick := args[1]

	var tgUserID *int64
	var tgUsername *string

	if strings.HasPrefix(identifier, "@") {
		// Telegram username
		username := strings.TrimPrefix(identifier, "@")
		if username == "" {
			return UserErrorf(MsgInvalidUsername)
		}
		tgUsername = &username
	} else if id, err := strconv.ParseInt(identifier, 10, 64); err == nil && id > 0 {
		// Numeric user ID
		tgUserID = &id
	} else {
		return UserErrorf(MsgNickUsage)
	}

	b.logger.Info("command /nick",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"tg_user_id", tgUserID,
		"tg_username", tgUsername,
		"game_nick", gameNick,
	)

	// Create the nickname mapping
	created, err := b.pollService.CreateNickname(tgUserID, tgUsername, gameNick)
	if err != nil {
		return WrapUserError(MsgFailedSaveNick, err)
	}

	// Get the resolved user ID for backfill
	var resolvedUserID int64
	if tgUserID != nil {
		resolvedUserID = *tgUserID
	} else if tgUsername != nil {
		// Try to infer user ID from vote history
		if id, found, err := b.pollService.LookupUserIDByUsername(*tgUsername); err == nil && found {
			resolvedUserID = id
		}
	}

	// Backfill votes if we have a user ID
	if resolvedUserID > 0 {
		var username string
		if tgUsername != nil {
			username = *tgUsername
		}
		gameNicks, err := b.pollService.GetAllGameNicksForUser(resolvedUserID, username)
		if err != nil {
			b.logger.Warn("failed to get game nicks for backfill", "error", err)
		} else {
			if err := b.pollService.BackfillVotesForNickname(c.Chat().ID, resolvedUserID, username, gameNicks); err != nil {
				b.logger.Warn("failed to backfill votes", "error", err)
			}
		}
	}

	// Update invitation message if exists (regardless of user ID - can match by username)
	if err := b.refreshInvitationMessage(c.Chat().ID); err != nil {
		b.logger.Warn("failed to refresh invitation after nick", "error", err)
	}

	// Send confirmation
	var msg string
	if created {
		if tgUsername != nil {
			msg = fmt.Sprintf(MsgFmtNickCreated, "@"+*tgUsername, gameNick)
		} else {
			msg = fmt.Sprintf(MsgFmtNickCreatedByID, *tgUserID, gameNick)
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

	if p.TgResultsMessageID == 0 {
		return nil // No invitation message to update
	}

	results, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return err
	}

	results.Poll = p
	results.EventDate = p.EventDate
	results.IsCancelled = !p.IsActive

	html, err := b.RenderInvitationWithNicks(results)
	if err != nil {
		return err
	}

	chat := &tele.Chat{ID: p.TgChatID}
	msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
	_, err = b.bot.Edit(msg, html, tele.ModeHTML)
	return err
}

// membersFromVotesWithNicknames creates Members from Votes with nicknames looked up.
// Preserves both TgUsername and Nickname for display.
func (b *Bot) membersFromVotesWithNicknames(votes []*poll.Vote) []Member {
	members := make([]Member, 0, len(votes))
	for _, v := range votes {
		m := Member{
			TgID:       v.TgUserID,
			TgName:     v.TgFirstName,
			TgUsername: v.TgUsername,
		}
		// Look up nickname
		nick, err := b.pollService.GetDisplayNick(v.TgUserID, v.TgUsername)
		if err != nil {
			b.logger.Warn("failed to get display nick", "error", err, "user_id", v.TgUserID)
		} else {
			m.Nickname = nick
		}
		members = append(members, m)
	}
	return members
}

// enrichVotesWithNicknames looks up game nicknames for votes and sets them.
// Note: This modifies votes in place and clears username for display purposes.
// When a game nickname is found, it takes priority for both regular and manual votes.
func (b *Bot) enrichVotesWithNicknames(votes []*poll.Vote) {
	for _, v := range votes {
		nick, err := b.pollService.GetDisplayNick(v.TgUserID, v.TgUsername)
		if err != nil {
			b.logger.Warn("failed to get display nick", "error", err, "user_id", v.TgUserID)
			continue
		}
		if nick != "" {
			// Store nickname in TgFirstName for display (Vote.DisplayName() will use it)
			v.TgFirstName = nick
			// Clear username so DisplayName() uses TgFirstName (nickname)
			v.TgUsername = ""
		}
	}
}

// RenderInvitationWithNicks renders invitation with nicknames resolved.
func (b *Bot) RenderInvitationWithNicks(data *poll.InvitationData) (string, error) {
	// Enrich votes with nicknames
	b.enrichVotesWithNicknames(data.Participants)
	b.enrichVotesWithNicknames(data.ComingLater)
	b.enrichVotesWithNicknames(data.Undecided)

	return RenderInvitation(data)
}
