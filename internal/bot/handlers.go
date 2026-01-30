package bot

import (
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
	if err := b.pollService.RecordVote(v); err != nil {
		return err
	}

	// Update results message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetResults(p.ID)
		if err != nil {
			return err
		}
		results.Poll = p

		html, err := poll.RenderResults(results)
		if err != nil {
			return err
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		_, err = b.bot.Edit(msg, html, tele.ModeHTML)
		if err != nil {
			// Log but don't fail
		}
	}

	return nil
}
