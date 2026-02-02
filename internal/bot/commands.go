package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// weekdayMap maps day names (lowercase) to time.Weekday
// Supports both English and Russian day names
var weekdayMap = map[string]time.Weekday{
	// English
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
	// Russian
	"воскресенье": time.Sunday,
	"вс":          time.Sunday,
	"понедельник": time.Monday,
	"пн":          time.Monday,
	"вторник":     time.Tuesday,
	"вт":          time.Tuesday,
	"среда":       time.Wednesday,
	"ср":          time.Wednesday,
	"четверг":     time.Thursday,
	"чт":          time.Thursday,
	"пятница":     time.Friday,
	"пт":          time.Friday,
	"суббота":     time.Saturday,
	"сб":          time.Saturday,
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
	adminGroup.Handle("/restore", b.handleRestore)
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
		return UserErrorf(MsgInvalidDateFormat)
	}

	b.logger.Info("command /poll",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"event_date", eventDate.Format("2006-01-02"),
	)

	// Create poll in database (service checks for existing poll)
	p, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		if errors.Is(err, poll.ErrPollExists) {
			return UserErrorf(MsgPollAlreadyExists)
		}
		return WrapUserError(MsgFailedCreatePoll, err)
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
		return WrapUserError(MsgFailedRenderResults, err)
	}

	invitationMsg, err := b.bot.Send(c.Chat(), invitationHTML, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		return WrapUserError(MsgFailedSendResults, err)
	}

	// Store invitation message ID
	p.TgResultsMessageID = invitationMsg.ID

	// Render poll title from template
	pollTitle, err := RenderPollTitle(eventDate)
	if err != nil {
		// Clean up invitation message on failure
		_ = b.bot.Delete(invitationMsg)
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
	sentMsg, err := b.bot.Send(c.Chat(), telePoll)
	if err != nil {
		// Clean up invitation message on failure
		_ = b.bot.Delete(invitationMsg)
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

// handleResults recreates the invitation message for the latest active poll
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

	// Get invitation data
	results, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		return WrapUserError(MsgFailedGetResults, err)
	}

	results.Poll = p
	results.EventDate = p.EventDate
	results.IsCancelled = !p.IsActive

	// Render invitation as HTML
	html, err := RenderInvitation(results)
	if err != nil {
		return WrapUserError(MsgFailedRenderResults, err)
	}

	// Send new invitation message
	sentMsg, err := c.Bot().Send(c.Chat(), html, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
	if err != nil {
		return WrapUserError(MsgFailedSendResults, err)
	}

	// Delete previous results message only after new one is sent successfully
	oldResultsMessageID := p.TgResultsMessageID

	// Update poll with new results message ID
	p.TgResultsMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSaveResults, err)
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
			b.logger.Warn("failed to delete previous invitation message", "error", err)
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

	// Get active poll
	p, err := b.pollService.GetActivePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
	}

	if p.TgMessageID == 0 {
		return UserErrorf(MsgPollMessageMissing)
	}

	// Unpin all previously pinned messages before pinning the new one
	chat := &tele.Chat{ID: p.TgChatID}
	if err := c.Bot().UnpinAll(chat); err != nil {
		b.logger.Warn("failed to unpin previous messages", "error", err)
	}

	// Pin the poll message (without Silent option to notify all members)
	msg := &tele.Message{
		ID:   p.TgMessageID,
		Chat: chat,
	}

	if err := c.Bot().Pin(msg); err != nil {
		return WrapUserError(MsgFailedPinPoll, err)
	}

	// Update poll status via service
	_, err = b.pollService.SetPinned(c.Chat().ID, true)
	if err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}

// handleCancel cancels the event and updates the invitation message with cancellation footer
func (b *Bot) handleCancel(c tele.Context) error {
	b.logger.Info("command /cancel",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Cancel poll via service (marks as inactive)
	p, err := b.pollService.CancelPoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedCancelPoll, err)
	}

	// Unpin the poll message if it was pinned
	if p.TgMessageID != 0 {
		chat := &tele.Chat{ID: p.TgChatID}
		if err := c.Bot().Unpin(chat, p.TgMessageID); err != nil {
			b.logger.Warn("failed to unpin poll message", "error", err)
		}
	}

	// Update invitation message with cancellation footer
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get invitation data for cancellation", "error", err)
		} else {
			results.Poll = p
			results.EventDate = p.EventDate
			results.IsCancelled = true

			html, err := RenderInvitation(results)
			if err != nil {
				b.logger.Warn("failed to render cancelled invitation", "error", err)
			} else {
				chat := &tele.Chat{ID: p.TgChatID}
				msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to update invitation message with cancellation", "error", err)
				}
			}
		}
	}

	// Post cancellation notification with mentions
	cancelData := &CancelData{EventDate: p.EventDate}

	// Add mentions of attending participants
	usernames, err := b.pollService.GetAttendingUsernames(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending usernames", "error", err)
	} else if len(usernames) > 0 {
		mentions := make([]string, len(usernames))
		for i, u := range usernames {
			mentions[i] = "@" + u
		}
		cancelData.Mentions = strings.Join(mentions, ", ")
	}

	cancellationMsg, err := RenderCancelMessage(cancelData)
	if err != nil {
		return WrapUserError(MsgFailedRenderCancellation, err)
	}

	sentMsg, err := c.Bot().Send(c.Chat(), cancellationMsg)
	if err != nil {
		return WrapUserError(MsgFailedSendCancellation, err)
	}

	// Save cancel message ID
	p.TgCancelMessageID = sentMsg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return WrapUserError(MsgFailedSavePollStatus, err)
	}

	return nil
}

// handleRestore restores the last cancelled poll if it's for today or a future date
func (b *Bot) handleRestore(c tele.Context) error {
	b.logger.Info("command /restore",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Restore poll via service (validates date, marks as active)
	p, err := b.pollService.RestorePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoCancelledPoll) {
			return UserErrorf(MsgNoCancelledPoll)
		}
		if errors.Is(err, poll.ErrPollDatePassed) {
			return UserErrorf(MsgPollDatePassed)
		}
		return WrapUserError(MsgFailedRestorePoll, err)
	}

	// Update invitation message to remove cancellation footer
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get invitation data for restore", "error", err)
		} else {
			results.Poll = p
			results.EventDate = p.EventDate
			results.IsCancelled = false

			html, err := RenderInvitation(results)
			if err != nil {
				b.logger.Warn("failed to render restored invitation", "error", err)
			} else {
				chat := &tele.Chat{ID: p.TgChatID}
				msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to update invitation message", "error", err)
				}
			}
		}
	}

	// Delete cancellation notification message and clear ID
	if p.TgCancelMessageID != 0 {
		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgCancelMessageID, Chat: chat}
		if err := b.bot.Delete(msg); err != nil {
			b.logger.Warn("failed to delete cancellation message", "error", err)
		}
		p.TgCancelMessageID = 0
		if err := b.pollService.UpdatePoll(p); err != nil {
			b.logger.Warn("failed to clear cancel message ID", "error", err)
		}
	}

	return nil
}

// handleVote manually records a vote for a user
// Usage: /vote @username <option number 1-5>
// Options: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming
func (b *Bot) handleVote(c tele.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return UserErrorf(MsgVoteUsage)
	}

	// Parse username (remove @ prefix if present)
	username := strings.TrimPrefix(args[0], "@")
	if username == "" {
		return UserErrorf(MsgInvalidUsername)
	}

	// Parse option number (1-5)
	optionNum, err := strconv.Atoi(args[1])
	if err != nil || optionNum < 1 || optionNum > 5 {
		return UserErrorf(MsgInvalidVoteOption)
	}

	// Convert to 0-indexed option
	optionIndex := optionNum - 1

	b.logger.Info("command /vote",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"target_username", username,
		"option", OptionLabel(poll.OptionKind(optionIndex)),
	)

	// Get active poll
	p, err := b.pollService.GetActivePoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoActivePoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
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
		return WrapUserError(MsgFailedRecordVote, err)
	}

	// Update invitation message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			return WrapUserError(MsgFailedGetResults, err)
		}
		results.Poll = p
		results.EventDate = p.EventDate
		results.IsCancelled = !p.IsActive

		html, err := RenderInvitation(results)
		if err != nil {
			return WrapUserError(MsgFailedRenderResults, err)
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
			b.logger.Warn("failed to update invitation message", "error", err)
		}
	}

	_, err = b.SendTemporary(c.Chat(), fmt.Sprintf(MsgFmtVoteRecorded, username, OptionLabel(poll.OptionKind(optionIndex))), 0)
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
