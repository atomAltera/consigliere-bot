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
	if err != nil || p == nil {
		return nil // Not our poll or error
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
		optionLabel = poll.OptionKind(optionIndex).Label()
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

	// Update results message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetResults(p.ID)
		if err != nil {
			return fmt.Errorf("get results: %w", err)
		}
		results.Poll = p
		results.Title, err = poll.RenderTitle(p.EventDate)
		if err != nil {
			return fmt.Errorf("render title: %w", err)
		}

		html, err := poll.RenderResults(results)
		if err != nil {
			return fmt.Errorf("render results: %w", err)
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
			// Non-critical: message may have been deleted, just log
			b.logger.Warn("failed to update results message", "error", err)
		}
	}

	return nil
}
