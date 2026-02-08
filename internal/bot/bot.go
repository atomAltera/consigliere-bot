package bot

import (
	"errors"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/config"
	"nuclight.org/consigliere/internal/poll"
)

// Retry configuration
const (
	MaxRetries        = 3
	InitialRetryDelay = 100 * time.Millisecond
)

type Bot struct {
	bot              *tele.Bot
	pollService      *poll.Service
	logger           *slog.Logger
	rateLimiter      *rateLimiter
	tempMessageDelay time.Duration
}

func New(cfg *config.Config, pollService *poll.Service, logger *slog.Logger) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.TelegramToken,
		Poller: &tele.LongPoller{Timeout: cfg.PollingTimeout},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot:              b,
		pollService:      pollService,
		logger:           logger,
		rateLimiter:      newRateLimiter(),
		tempMessageDelay: cfg.TempMessageDelay,
	}, nil
}

func (b *Bot) Start() {
	b.logger.Info("bot started")
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

// MessageRef creates a Telegram message reference for use with Edit, Delete, Pin etc.
// This avoids repetitive construction of chat and message structs throughout the codebase.
func MessageRef(chatID int64, msgID int) *tele.Message {
	return &tele.Message{
		ID:   msgID,
		Chat: &tele.Chat{ID: chatID},
	}
}

func (b *Bot) Bot() *tele.Bot {
	return b.bot
}

func (b *Bot) PollService() *poll.Service {
	return b.pollService
}

func (b *Bot) Logger() *slog.Logger {
	return b.logger
}

// SendWithRetry sends a message with exponential backoff retry on failure.
func (b *Bot) SendWithRetry(to tele.Recipient, what any, opts ...any) (*tele.Message, error) {
	var msg *tele.Message
	var err error
	delay := InitialRetryDelay

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		msg, err = b.bot.Send(to, what, opts...)
		if err == nil {
			return msg, nil
		}

		if attempt < MaxRetries {
			b.logger.Warn("telegram send failed, retrying",
				"error", err,
				"attempt", attempt+1,
				"max_retries", MaxRetries,
				"next_delay", delay,
			)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return nil, err
}

// SendTemporary sends a message that will be automatically deleted after the specified delay.
// If delay is 0, the configured TempMessageDelay is used.
// Messages are sent silently (without notification) to avoid disturbing chat members.
func (b *Bot) SendTemporary(to tele.Recipient, what any, delay time.Duration, opts ...any) (*tele.Message, error) {
	// Ensure silent notification for temporary messages.
	// If SendOptions is provided, set DisableNotification directly to avoid being overwritten.
	// Otherwise, prepend tele.Silent flag.
	hasSendOptions := false
	for _, opt := range opts {
		if sendOpts, ok := opt.(*tele.SendOptions); ok {
			sendOpts.DisableNotification = true
			hasSendOptions = true
		}
	}
	if !hasSendOptions {
		opts = append([]any{tele.Silent}, opts...)
	}
	msg, err := b.SendWithRetry(to, what, opts...)
	if err != nil {
		return nil, err
	}

	if delay == 0 {
		delay = b.tempMessageDelay
	}

	go func() {
		time.Sleep(delay)
		if err := b.bot.Delete(msg); err != nil {
			b.logger.Warn("failed to delete temporary message",
				"error", err,
				"chat_id", msg.Chat.ID,
				"message_id", msg.ID,
			)
		}
	}()

	return msg, nil
}

// GetActivePollOrError retrieves the active poll for the given chat.
// Returns a user-friendly error if no active poll exists or if retrieval fails.
func (b *Bot) GetActivePollOrError(chatID int64) (*poll.Poll, error) {
	p, err := b.pollService.GetActivePoll(chatID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return nil, UserErrorf(MsgNoActivePoll)
		}
		return nil, WrapUserError(MsgFailedGetPoll, err)
	}
	return p, nil
}

// GetActivePollForAction retrieves the active poll and validates that its event date
// has not passed. Use this for handlers that perform actions on the poll (cancel, vote, etc.).
// For read-only operations that should work even after the event date, use GetActivePollOrError.
func (b *Bot) GetActivePollForAction(chatID int64) (*poll.Poll, error) {
	p, err := b.GetActivePollOrError(chatID)
	if err != nil {
		return nil, err
	}

	if isPollDatePassed(p.EventDate) {
		return nil, UserErrorf(MsgPollDatePassed)
	}

	return p, nil
}

// RenderAndSend renders a message using the provided render function and sends it to the chat.
// Returns the sent message (for cases that need the message ID) and any error.
// Errors are wrapped with appropriate user-facing messages.
func (b *Bot) RenderAndSend(c tele.Context, renderFunc func() (string, error), renderErrMsg, sendErrMsg string) (*tele.Message, error) {
	html, err := renderFunc()
	if err != nil {
		return nil, WrapUserError(renderErrMsg, err)
	}

	msg, err := b.SendWithRetry(c.Chat(), html, tele.ModeHTML)
	if err != nil {
		return nil, WrapUserError(sendErrMsg, err)
	}

	return msg, nil
}

// UpdateInvitationMessage updates the invitation message for a poll if it exists.
// If isCancelledOverride is provided, it overrides the default IsCancelled value (which is !p.IsActive).
// Returns true if the message was successfully updated, false otherwise.
// This is a non-critical operation - errors are logged but not returned since the message may have been deleted.
func (b *Bot) UpdateInvitationMessage(p *poll.Poll, isCancelledOverride *bool) bool {
	if p.TgInvitationMessageID == 0 {
		return false
	}

	results, err := b.pollService.GetInvitationData(p.ID)
	if err != nil {
		b.logger.Warn("failed to get invitation data", "error", err, "poll_id", p.ID)
		return false
	}

	p.PopulateInvitationData(results)
	if isCancelledOverride != nil {
		results.IsCancelled = *isCancelledOverride
	}

	html, err := b.RenderInvitationWithNicks(results)
	if err != nil {
		b.logger.Warn("failed to render invitation", "error", err, "poll_id", p.ID)
		return false
	}

	if _, err = b.bot.Edit(MessageRef(p.TgChatID, p.TgInvitationMessageID), html, tele.ModeHTML); err != nil {
		b.logger.Warn("failed to update invitation message", "error", err, "poll_id", p.ID)
		return false
	}

	return true
}
