package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// weekdayMap maps day names (lowercase) to time.Weekday
var weekdayMap = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"sun":       time.Sunday,
	"monday":    time.Monday,
	"mon":       time.Monday,
	"tuesday":   time.Tuesday,
	"tue":       time.Tuesday,
	"wednesday": time.Wednesday,
	"wed":       time.Wednesday,
	"thursday":  time.Thursday,
	"thu":       time.Thursday,
	"friday":    time.Friday,
	"fri":       time.Friday,
	"saturday":  time.Saturday,
	"sat":       time.Saturday,
}

// nextWeekday returns the next occurrence of the given weekday from the reference date.
// If today is the target weekday, it returns today.
func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysUntil := int(target) - int(from.Weekday())
	if daysUntil < 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil)
}

// nearestGameDay returns the nearest Monday or Saturday from the given date.
// If today is Monday or Saturday, returns today.
func nearestGameDay(from time.Time) time.Time {
	nextMon := nextWeekday(from, time.Monday)
	nextSat := nextWeekday(from, time.Saturday)

	// Return whichever is closer
	if nextMon.Before(nextSat) || nextMon.Equal(nextSat) {
		return nextMon
	}
	return nextSat
}

// parseEventDate parses the event date from command arguments.
// Supports:
// - No arguments: nearest Monday or Saturday
// - Day of week name: "monday", "mon", "saturday", "sat", etc.
// - Explicit date: "YYYY-MM-DD"
func parseEventDate(args []string) (time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if len(args) == 0 {
		return nearestGameDay(today), nil
	}

	arg := strings.ToLower(strings.TrimSpace(args[0]))

	// Check if it's a day of week name
	if weekday, ok := weekdayMap[arg]; ok {
		return nextWeekday(today, weekday), nil
	}

	// Try parsing as YYYY-MM-DD
	eventDate, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format. Use day name (e.g., monday, sat) or YYYY-MM-DD")
	}

	return eventDate, nil
}

// RegisterCommands sets up all bot commands with admin-only middleware
func (b *Bot) RegisterCommands() {
	adminGroup := b.bot.Group()
	adminGroup.Use(b.AdminOnly())
	adminGroup.Use(b.DeleteCommand())
	adminGroup.Use(b.HandleErrors())

	adminGroup.Handle("/poll", b.handlePoll)
	adminGroup.Handle("/results", b.handleResults)
	adminGroup.Handle("/pin", b.handlePin)
	adminGroup.Handle("/cancel", b.handleCancel)
	adminGroup.Handle("/vote", b.handleVote)
	adminGroup.Handle("/help", b.handleHelp)
}

// handlePoll creates a new poll for the specified date
// Usage: /poll [day|YYYY-MM-DD]
// - No arguments: nearest Monday or Saturday
// - Day name: monday, mon, saturday, sat, etc.
// - Explicit date: YYYY-MM-DD
func (b *Bot) handlePoll(c tele.Context) error {
	eventDate, err := parseEventDate(c.Args())
	if err != nil {
		return UserErrorf("Invalid date format. Use day name (e.g., monday, sat) or YYYY-MM-DD")
	}

	b.logger.Info("command /poll",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"event_date", eventDate.Format("2006-01-02"),
	)

	// Check if there's already an active poll in this chat
	existingPoll, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err == nil && existingPoll != nil {
		return UserErrorf("There is already an active poll in this chat. Cancel it first with /cancel before creating a new one.")
	}

	// Create poll in database
	p, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		return WrapUserError("Failed to create poll. Please try again.", err)
	}

	// Render poll title from template
	pollTitle, err := poll.RenderTitle(eventDate)
	if err != nil {
		return WrapUserError("Failed to render poll title. Please try again.", err)
	}

	// Create Telegram poll
	pollOptions := poll.AllOptions()
	telePoll := &tele.Poll{
		Type:            tele.PollRegular,
		Question:        pollTitle,
		MultipleAnswers: false,
		Anonymous:       false,
	}

	// Add options
	telePoll.AddOptions(pollOptions...)

	// Send poll to the chat
	sentMsg, err := b.bot.Send(c.Chat(), telePoll)
	if err != nil {
		return WrapUserError("Failed to send poll. Please try again.", err)
	}

	// Update poll with Telegram IDs
	p.TgPollID = sentMsg.Poll.ID
	p.TgMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError("Failed to save poll. Please try again.", err)
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
		return UserErrorf("No active poll found")
	}

	// Get results
	results, err := b.pollService.GetResults(p.ID)
	if err != nil {
		return WrapUserError("Failed to get results. Please try again.", err)
	}

	results.Poll = p
	results.Title, err = poll.RenderTitle(p.EventDate)
	if err != nil {
		return WrapUserError("Failed to render title. Please try again.", err)
	}

	// Render results as HTML
	html, err := poll.RenderResults(results)
	if err != nil {
		return WrapUserError("Failed to render results. Please try again.", err)
	}

	// Send results message
	sentMsg, err := c.Bot().Send(c.Chat(), html, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		return WrapUserError("Failed to send results. Please try again.", err)
	}

	// Delete previous results message only after new one is sent successfully
	oldResultsMessageID := p.TgResultsMessageID

	// Update poll with new results message ID
	p.TgResultsMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError("Failed to save results. Please try again.", err)
	}

	// Now safe to delete the old message
	if oldResultsMessageID != 0 {
		msg := &tele.Message{
			ID: oldResultsMessageID,
			Chat: &tele.Chat{
				ID: p.TgChatID,
			},
		}
		if err := c.Bot().Delete(msg); err != nil {
			b.logger.Warn("failed to delete previous results message", "error", err)
		}
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
		return UserErrorf("No active poll found")
	}

	if p.TgMessageID == 0 {
		return UserErrorf("Poll message not found")
	}

	// Unpin all previously pinned messages before pinning the new one
	chat := &tele.Chat{ID: p.TgChatID}
	if err := c.Bot().UnpinAll(chat); err != nil {
		b.logger.Warn("failed to unpin previous messages", "error", err)
	}

	// Pin the poll message (without Silent option to notify all members)
	msg := &tele.Message{
		ID: p.TgMessageID,
		Chat: chat,
	}

	if err := c.Bot().Pin(msg); err != nil {
		return WrapUserError("Failed to pin poll. Please try again.", err)
	}

	// Update poll status
	p.IsPinned = true
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError("Failed to save poll status. Please try again.", err)
	}

	return nil
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
		return UserErrorf("No active poll found")
	}

	// Unpin the poll message if it was pinned
	if p.IsPinned && p.TgMessageID != 0 {
		chat := &tele.Chat{ID: p.TgChatID}
		if err := c.Bot().Unpin(chat, p.TgMessageID); err != nil {
			b.logger.Warn("failed to unpin poll message", "error", err)
		}
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
		"⚠️ Event on %s has been cancelled",
		p.EventDate.Format("Monday, January 2"),
	)
	sentMsg, err := c.Bot().Send(c.Chat(), cancellationMsg)
	if err != nil {
		return WrapUserError("Failed to send cancellation message. Please try again.", err)
	}

	// Pin the cancellation message (without Silent option to notify all members)
	if err := c.Bot().Pin(sentMsg); err != nil {
		b.logger.Warn("failed to pin cancellation message", "error", err)
	}

	// Update poll status - mark as inactive
	p.IsActive = false
	p.IsPinned = false
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError("Failed to save poll status. Please try again.", err)
	}

	return nil
}

// handleVote manually records a vote for a user
// Usage: /vote @username <option number 1-5>
// Options: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming
func (b *Bot) handleVote(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return UserErrorf("Usage: /vote @username <option 1-5>\nOptions: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming")
	}

	// Parse username (remove @ prefix if present)
	username := strings.TrimPrefix(args[0], "@")
	if username == "" {
		return UserErrorf("Invalid username")
	}

	// Parse option number (1-5)
	optionNum, err := strconv.Atoi(args[1])
	if err != nil || optionNum < 1 || optionNum > 5 {
		return UserErrorf("Invalid option. Use 1-5:\n1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming")
	}

	// Convert to 0-indexed option
	optionIndex := optionNum - 1

	b.logger.Info("command /vote",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"target_username", username,
		"option", poll.OptionKind(optionIndex).Label(),
	)

	// Get latest active poll
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil || p == nil {
		return UserErrorf("No active poll found")
	}

	// Create manual vote with synthetic user ID
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      poll.ManualUserID(username),
		TgUsername:    username,
		TgFirstName:   username, // Use username as first name for display
		TgOptionIndex: optionIndex,
		IsManual:      true,
	}

	if err := b.pollService.RecordVote(v); err != nil {
		return WrapUserError("Failed to record vote. Please try again.", err)
	}

	// Update results message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetResults(p.ID)
		if err != nil {
			return WrapUserError("Failed to get results. Please try again.", err)
		}
		results.Poll = p
		results.Title, err = poll.RenderTitle(p.EventDate)
		if err != nil {
			return WrapUserError("Failed to render title. Please try again.", err)
		}

		html, err := poll.RenderResults(results)
		if err != nil {
			return WrapUserError("Failed to render results. Please try again.", err)
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
			b.logger.Warn("failed to update results message", "error", err)
		}
	}

	_, err = b.SendTemporary(c.Chat(), fmt.Sprintf("Recorded vote for %s: %s", username, poll.OptionKind(optionIndex).Label()), 0)
	return err
}

// handleHelp shows the help message with all available commands
func (b *Bot) handleHelp(c tele.Context) error {
	b.logger.Info("command /help",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	_, err := b.SendTemporary(c.Chat(), HelpMessage(), 30*time.Second, tele.ModeHTML)
	return err
}
