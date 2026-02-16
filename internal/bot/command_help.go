package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"
)

// handleHelp shows the help message with all available commands
func (b *Bot) handleHelp(c tele.Context) error {
	config := getClubConfig(c)

	helpText, err := HelpMessage(config.templates)
	if err != nil {
		return fmt.Errorf("read help template: %w", err)
	}
	_, err = b.SendTemporary(c.Chat(), helpText, 30*time.Second, tele.ModeHTML)
	return err
}
