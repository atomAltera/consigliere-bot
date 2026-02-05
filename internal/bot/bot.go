package bot

import (
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
