package bot

import (
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) isAdmin(chatID int64, userID int64) (bool, error) {
	chat := &tele.Chat{ID: chatID}
	member, err := b.bot.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		return false, err
	}

	return member.Role == tele.Administrator || member.Role == tele.Creator, nil
}

func (b *Bot) AdminOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			isAdmin, err := b.isAdmin(c.Chat().ID, c.Sender().ID)
			if err != nil {
				return err
			}

			if !isAdmin {
				return nil // Silently ignore non-admin commands
			}

			return next(c)
		}
	}
}

func (b *Bot) DeleteCommand() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			// Delete the command message
			if err := c.Delete(); err != nil {
				b.logger.Warn("failed to delete command message",
					"error", err,
					"chat_id", c.Chat().ID,
					"message_id", c.Message().ID,
				)
			}

			return next(c)
		}
	}
}

// HandleErrors is a middleware that properly handles errors from command handlers.
// It logs internal errors and sends appropriate messages to users:
// - UserError: sends the user-friendly message, logs if there's an underlying cause
// - Other errors: sends a generic error message, logs the full error
func (b *Bot) HandleErrors() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			// Log the error if needed
			if ShouldLog(err) {
				b.logger.Error("command error",
					"error", GetLogError(err),
					"chat_id", c.Chat().ID,
					"user_id", c.Sender().ID,
					"command", c.Text(),
				)
			}

			// Send user-friendly message
			userMsg := GetUserMessage(err)
			if sendErr := c.Send(userMsg); sendErr != nil {
				b.logger.Error("failed to send error message to user",
					"error", sendErr,
					"original_error", err,
				)
			}

			// Return nil to prevent telebot from handling the error again
			return nil
		}
	}
}
