package bot

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
	tele "gopkg.in/telebot.v4"
)

// Rate limiting constants
const (
	RateLimitPerSecond = 1   // requests per second per user
	RateLimitBurst     = 3   // burst allowance
)

// rateLimiter manages per-user rate limiters
type rateLimiter struct {
	mu       sync.Mutex
	limiters map[int64]*rate.Limiter
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		limiters: make(map[int64]*rate.Limiter),
	}
}

func (r *rateLimiter) getLimiter(userID int64) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(RateLimitPerSecond), RateLimitBurst)
		r.limiters[userID] = limiter
	}
	return limiter
}

// RateLimit is a middleware that limits command frequency per user.
// Returns silently if rate limit is exceeded.
func (b *Bot) RateLimit() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			limiter := b.rateLimiter.getLimiter(c.Sender().ID)
			if !limiter.Allow() {
				b.logger.Warn("rate limit exceeded",
					"user_id", c.Sender().ID,
					"username", c.Sender().Username,
					"chat_id", c.Chat().ID,
					"command", c.Text(),
				)
				return nil // Silently drop rate-limited requests
			}
			return next(c)
		}
	}
}

func (b *Bot) isAdmin(chatID int64, userID int64) (bool, error) {
	member, err := b.bot.ChatMemberOf(MessageRef(chatID, 0).Chat, &tele.User{ID: userID})
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
				b.logger.Info("unauthorized command attempt",
					"user_id", c.Sender().ID,
					"username", c.Sender().Username,
					"chat_id", c.Chat().ID,
					"command", c.Text(),
				)
				return nil // Ignore non-admin commands
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

// extractCommand extracts the command name from text like "/poll@bot arg1 arg2" -> "poll"
func extractCommand(text string) string {
	if text == "" {
		return ""
	}

	// Remove leading slash if present
	if text[0] == '/' {
		text = text[1:]
	}

	// Get first word (command with possible @bot suffix)
	firstWord := text
	if idx := indexAny(text, " \t"); idx != -1 {
		firstWord = text[:idx]
	}

	// Remove @bot suffix if present
	if idx := indexAny(firstWord, "@"); idx != -1 {
		firstWord = firstWord[:idx]
	}

	return firstWord
}

// indexAny returns the index of the first occurrence of any character in chars, or -1 if not found.
func indexAny(s, chars string) int {
	for i, c := range s {
		for _, ch := range chars {
			if c == ch {
				return i
			}
		}
	}
	return -1
}

// LogCommand is a middleware that logs all executed commands with common context.
// Should be placed after AdminOnly() to only log authorized commands.
func (b *Bot) LogCommand() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			b.logger.Info("command",
				"name", extractCommand(c.Text()),
				"user_id", c.Sender().ID,
				"username", c.Sender().Username,
				"chat_id", c.Chat().ID,
			)
			return next(c)
		}
	}
}

// HandleErrors is a middleware that properly handles errors from command handlers.
// It logs internal errors and sends appropriate messages to users:
// - UserError: sends the user-friendly message (temporarily), logs if there's an underlying cause
// - Other errors: sends a generic error message (temporarily), logs the full error
// Error messages are automatically deleted after the configured TempMessageDelay.
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

			// Send user-friendly message (temporary, silent to avoid disturbing chat members)
			userMsg := GetUserMessage(err)
			msg, sendErr := b.SendWithRetry(c.Chat(), userMsg, tele.Silent)
			if sendErr != nil {
				b.logger.Error("failed to send error message to user",
					"error", sendErr,
					"original_error", err,
				)
			} else {
				// Delete the error message after a delay
				go func() {
					time.Sleep(b.tempMessageDelay)
					if delErr := b.bot.Delete(msg); delErr != nil {
						b.logger.Warn("failed to delete temporary error message",
							"error", delErr,
							"chat_id", msg.Chat.ID,
							"message_id", msg.ID,
						)
					}
				}()
			}

			// Return nil to prevent telebot from handling the error again
			return nil
		}
	}
}
