package bot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

func (b *Bot) RegisterHandlers() {
	b.bot.Handle(tele.OnPollAnswer, b.handlePollAnswer)
}

func (b *Bot) handlePollAnswer(c tele.Context) error {
	answer := c.PollAnswer()
	if answer == nil {
		return nil
	}

	// Find the poll
	p, err := b.pollService.GetPollByTgPollID(answer.PollID)
	if err != nil {
		b.logger.Error("failed to lookup poll", "poll_id", answer.PollID, "error", err)
		return nil
	}
	if p == nil {
		return nil // Not our poll
	}

	// Determine option index (-1 if retracted)
	optionIndex := -1
	if len(answer.Options) > 0 {
		optionIndex = answer.Options[0]
	}

	// Record vote
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      answer.Sender.ID,
		TgUsername:    answer.Sender.Username,
		TgFirstName:   answer.Sender.FirstName,
		TgOptionIndex: optionIndex,
	}

	optionLabel := "retracted"
	if optionIndex >= 0 {
		optionLabel = OptionLabel(poll.OptionKind(optionIndex))
	}

	b.logger.Info("vote recorded",
		"user_id", answer.Sender.ID,
		"username", answer.Sender.Username,
		"poll_id", p.ID,
		"option", optionLabel,
	)

	if err := b.pollService.RecordVote(v); err != nil {
		return fmt.Errorf("record vote: %w", err)
	}

	// Ensure data consistency: update nicknames and consolidate synthetic votes
	if err := b.pollService.EnsureUserDataConsistency(p.TgChatID, answer.Sender.ID, answer.Sender.Username); err != nil {
		b.logger.Warn("failed to ensure user data consistency", "error", err)
	}

	// Update invitation message if exists
	b.UpdateInvitationMessage(p, nil)

	return nil
}
