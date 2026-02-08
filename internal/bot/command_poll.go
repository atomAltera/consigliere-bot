package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handlePoll creates a new poll for the specified date
// Usage: /poll [day|YYYY-MM-DD]
// - No arguments: nearest Monday or Saturday
// - Day name: monday, mon, saturday, sat, etc.
// - Explicit date: YYYY-MM-DD
func (b *Bot) handlePoll(c tele.Context) error {
	eventDate, err := parseEventDate(c.Args())
	if err != nil {
		return UserErrorf(MsgInvalidDateFormat)
	}

	b.logger.Info("command /poll",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"event_date", eventDate.Format("2006-01-02"),
	)

	// Create poll in database (service checks for existing poll)
	result, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		if errors.Is(err, poll.ErrPollExists) {
			return UserErrorf(MsgPollAlreadyExists)
		}
		return WrapUserError(MsgFailedCreatePoll, err)
	}
	p := result.Poll

	// Unpin old poll message if it was replaced and pinned
	if result.ReplacedPoll != nil && result.ReplacedPoll.TgMessageID != 0 {
		if err := c.Bot().Unpin(MessageRef(c.Chat().ID, 0).Chat, result.ReplacedPoll.TgMessageID); err != nil {
			// Non-critical: old message may have been deleted, just log
			b.logger.Warn("failed to unpin replaced poll", "error", err)
		}
	}

	// Helper to rollback poll on failure (deactivate so a new one can be created)
	rollbackPoll := func() {
		p.IsActive = false
		if updateErr := b.pollService.UpdatePoll(p); updateErr != nil {
			b.logger.Error("failed to rollback poll after error", "error", updateErr, "poll_id", p.ID)
		}
	}

	// Send invitation message first (empty participants)
	invitationData := &poll.InvitationData{
		Poll:         p,
		EventDate:    eventDate,
		Participants: []*poll.Vote{},
		ComingLater:  []*poll.Vote{},
		Undecided:    []*poll.Vote{},
		IsCancelled:  false,
	}

	invitationHTML, err := RenderInvitation(invitationData)
	if err != nil {
		rollbackPoll()
		return WrapUserError(MsgFailedRenderResults, err)
	}

	invitationMsg, err := b.SendWithRetry(c.Chat(), invitationHTML, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		rollbackPoll()
		return WrapUserError(MsgFailedSendResults, err)
	}

	// Store invitation message ID
	p.TgInvitationMessageID = invitationMsg.ID

	// Render poll title from template
	pollTitle, err := RenderPollTitle(eventDate)
	if err != nil {
		// Clean up invitation message and rollback poll on failure
		_ = b.bot.Delete(invitationMsg)
		rollbackPoll()
		return WrapUserError(MsgFailedRenderPollTitle, err)
	}

	// Create Telegram poll
	pollOptions := AllOptionLabels()
	telePoll := &tele.Poll{
		Type:            tele.PollRegular,
		Question:        pollTitle,
		MultipleAnswers: false,
		Anonymous:       false,
	}

	// Add options
	telePoll.AddOptions(pollOptions...)

	// Send poll to the chat
	sentMsg, err := b.SendWithRetry(c.Chat(), telePoll)
	if err != nil {
		// Clean up invitation message and rollback poll on failure
		_ = b.bot.Delete(invitationMsg)
		rollbackPoll()
		return WrapUserError(MsgFailedSendPoll, err)
	}

	// Update poll with Telegram IDs
	p.TgPollID = sentMsg.Poll.ID
	p.TgMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePoll, err)
	}

	return nil
}
