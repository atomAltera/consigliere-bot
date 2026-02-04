package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"
)

// handleHelp shows the help message with all available commands
func (b *Bot) handleHelp(c tele.Context) error {
	b.logger.Info("command /help",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	helpText, err := HelpMessage()
	if err != nil {
		return fmt.Errorf("read help template: %w", err)
	}
	_, err = b.SendTemporary(c.Chat(), helpText, 30*time.Second, tele.ModeHTML)
	return err
}
