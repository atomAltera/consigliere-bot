package bot

import (
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

type Bot struct {
	bot         *tele.Bot
	groupID     int64
	pollService *poll.Service
	logger      *slog.Logger
}

func New(token string, groupID int64, pollService *poll.Service, logger *slog.Logger) (*Bot, error) {
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
		groupID:     groupID,
		pollService: pollService,
		logger:      logger,
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

func (b *Bot) GroupID() int64 {
	return b.groupID
}

func (b *Bot) PollService() *poll.Service {
	return b.pollService
}
