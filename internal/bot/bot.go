package bot

import (
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// DefaultTempMessageDelay is the default delay before deleting temporary messages.
const DefaultTempMessageDelay = 5 * time.Second

type Bot struct {
	bot         *tele.Bot
	pollService *poll.Service
	logger      *slog.Logger
	rateLimiter *rateLimiter
}

func New(token string, pollService *poll.Service, logger *slog.Logger) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot:         b,
		pollService: pollService,
		logger:      logger,
		rateLimiter: newRateLimiter(),
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

// SendTemporary sends a message that will be automatically deleted after the specified delay.
// If delay is 0, DefaultTempMessageDelay is used.
func (b *Bot) SendTemporary(to tele.Recipient, what any, delay time.Duration, opts ...any) (*tele.Message, error) {
	msg, err := b.bot.Send(to, what, opts...)
	if err != nil {
		return nil, err
	}

	if delay == 0 {
		delay = DefaultTempMessageDelay
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
