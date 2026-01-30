package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// RegisterCommands sets up all bot commands with admin-only middleware
func (b *Bot) RegisterCommands() {
	adminGroup := b.bot.Group()
	adminGroup.Use(b.AdminOnly())
	adminGroup.Use(b.DeleteCommand())

	adminGroup.Handle("/poll", b.handlePoll)
	adminGroup.Handle("/results", b.handleResults)
	adminGroup.Handle("/pin", b.handlePin)
	adminGroup.Handle("/cancel", b.handleCancel)
}

// handlePoll creates a new poll for the specified date
// Usage: /poll YYYY-MM-DD
func (b *Bot) handlePoll(c tele.Context) error {
	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /poll YYYY-MM-DD")
	}

	eventDate, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		return c.Send("Invalid date format. Use YYYY-MM-DD")
	}

	b.logger.Info("command /poll",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"event_date", eventDate.Format("2006-01-02"),
	)

	// Create poll in database
	p, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		return c.Send("Failed to create poll. Please try again.")
	}

	// Create Telegram poll
	pollOptions := poll.AllOptions()
	telePoll := &tele.Poll{
		Type:            tele.PollRegular,
		Question:        fmt.Sprintf("Mafia game on %s", eventDate.Format("Monday, January 2")),
		MultipleAnswers: false,
		Anonymous:       false,
	}

	// Add options
	telePoll.AddOptions(pollOptions...)

	// Send poll to the chat
	sentMsg, err := b.bot.Send(c.Chat(), telePoll)
	if err != nil {
		return c.Send(fmt.Sprintf("Error sending poll: %v", err))
	}

	// Update poll with Telegram IDs
	p.TgPollID = sentMsg.Poll.ID
	p.TgMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return c.Send(fmt.Sprintf("Error updating poll: %v", err))
	}

	return nil
}

// handleResults posts the results message for the latest active poll
func (b *Bot) handleResults(c tele.Context) error {
	b.logger.Info("command /results",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get latest active poll
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil || p == nil {
		return c.Send("No active poll found")
	}

	// Get results
	results, err := b.pollService.GetResults(p.ID)
	if err != nil {
		return c.Send(fmt.Sprintf("Error getting results: %v", err))
	}

	results.Poll = p

	// Render results as HTML
	html, err := poll.RenderResults(results)
	if err != nil {
		return c.Send(fmt.Sprintf("Error rendering results: %v", err))
	}

	// Send results message
	sentMsg, err := c.Bot().Send(c.Chat(), html, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		return c.Send(fmt.Sprintf("Error sending results: %v", err))
	}

	// Update poll with results message ID
	p.TgResultsMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return c.Send(fmt.Sprintf("Error updating poll: %v", err))
	}

	return nil
}

// handlePin pins the poll message
func (b *Bot) handlePin(c tele.Context) error {
	b.logger.Info("command /pin",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get latest active poll
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil || p == nil {
		return c.Send("No active poll found")
	}

	if p.TgMessageID == 0 {
		return c.Send("Poll message not found")
	}

	// Pin the poll message (without Silent option to notify all members)
	msg := &tele.Message{
		ID: p.TgMessageID,
		Chat: &tele.Chat{
			ID: p.TgChatID,
		},
	}

	if err := c.Bot().Pin(msg); err != nil {
		return c.Send(fmt.Sprintf("Error pinning message: %v", err))
	}

	// Update poll status
	p.Status = poll.StatusPinned
	if err := b.pollService.UpdatePoll(p); err != nil {
		return c.Send(fmt.Sprintf("Error updating poll: %v", err))
	}

	return c.Send("Poll pinned successfully")
}

// handleCancel cancels the event and deletes the results message
func (b *Bot) handleCancel(c tele.Context) error {
	b.logger.Info("command /cancel",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get latest active poll
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil || p == nil {
		return c.Send("No active poll found")
	}

	// Delete results message if it exists
	if p.TgResultsMessageID != 0 {
		msg := &tele.Message{
			ID: p.TgResultsMessageID,
			Chat: &tele.Chat{
				ID: p.TgChatID,
			},
		}
		if err := c.Bot().Delete(msg); err != nil {
			b.logger.Warn("failed to delete results message", "error", err)
		}
	}

	// Send cancellation message
	cancellationMsg := fmt.Sprintf(
		"Event on %s has been cancelled",
		p.EventDate.Format("Monday, January 2"),
	)
	if _, err := c.Bot().Send(c.Chat(), cancellationMsg); err != nil {
		return c.Send(fmt.Sprintf("Error sending cancellation: %v", err))
	}

	// Update poll status
	p.Status = poll.StatusCancelled
	if err := b.pollService.UpdatePoll(p); err != nil {
		return c.Send(fmt.Sprintf("Error updating poll: %v", err))
	}

	return nil
}
