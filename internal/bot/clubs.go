package bot

import (
	"slices"
	"html/template"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

const UserSectris = 375533758
const UserMamaLama = 319348068
const UserKezlev = 7437375018
const UserFrancuz = 1091792914

const ChatVanmo = -1001857572582
const ChatTbilissimo = -1001446847412
const ChatAntispamTest = -1002544962928
const ChatConsigliereTestTbilissimo = -1003733477508

// FeatureFlags holds per-club feature toggles.
type FeatureFlags struct{}

// ClubConfig holds configuration for a club's chat groups.
type ClubConfig struct {
	Club            poll.Club
	Name            string
	DefaultWeekDays []time.Weekday
	Admins          []int64
	MediaDir        string // subdirectory under media/ for event videos (empty = no video)
	FeatureFlags    FeatureFlags
	templates       *template.Template // unexported, accessed within bot package only
}

var vanmoConfig = &ClubConfig{
	Club:            poll.ClubVanmo,
	Name:            "VANMO",
	DefaultWeekDays: []time.Weekday{time.Monday, time.Saturday},
	Admins: []int64{
		UserSectris,
		UserFrancuz,
	},
	MediaDir: "vanmo",
}

var tbilissimoConfig = &ClubConfig{
	Club:            poll.ClubTbilissimo,
	Name:            "Tbilissimo",
	DefaultWeekDays: []time.Weekday{time.Wednesday, time.Sunday},
	Admins: []int64{
		UserSectris,
		UserMamaLama,
		UserKezlev,
	},
}

// chatRegistry maps Telegram chat IDs to their club configuration.
var chatRegistry = map[int64]*ClubConfig{
	// todo: replace with real chat IDs
	ChatVanmo: vanmoConfig,
	ChatTbilissimo: tbilissimoConfig,
	ChatAntispamTest: vanmoConfig,
	ChatConsigliereTestTbilissimo: tbilissimoConfig,
}

// InitClubTemplates parses templates for all club configs.
// Must be called at startup before handling any messages.
func InitClubTemplates() error {
	var err error
	vanmoConfig.templates, err = ParseClubTemplates("vanmo")
	if err != nil {
		return err
	}
	tbilissimoConfig.templates, err = ParseClubTemplates("tbilissimo")
	if err != nil {
		return err
	}
	return nil
}

// getClubConfig retrieves the ClubConfig stored in the telebot context.
// Must only be called after ResolveClub middleware has run.
func getClubConfig(c tele.Context) *ClubConfig {
	return c.Get("club").(*ClubConfig)
}

// ResolveClub looks up the chat in the registry and stores the ClubConfig in context.
// If the chat is not registered, sends a self-destructing error and stops the chain.
func (b *Bot) ResolveClub() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			config, ok := chatRegistry[c.Chat().ID]
			if !ok {
				b.logger.Warn("unregistered chat attempted to use bot",
					"chat_id", c.Chat().ID,
				)
				_, _ = b.SendTemporary(c.Chat(), MsgChatNotPermitted, 0)
				return nil
			}
			c.Set("club", config)
			return next(c)
		}
	}
}

// ClubAdminOnly checks if the sender is in the club's admin list.
// Non-admins are silently ignored (command is deleted but no error posted).
func (b *Bot) ClubAdminOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			config := getClubConfig(c)
			userID := c.Sender().ID

			if slices.Contains(config.Admins, userID) {
				return next(c)
			}

			b.logger.Warn("unauthorized command attempt",
				"user_id", userID,
				"username", c.Sender().Username,
				"chat_id", c.Chat().ID,
				"club", config.Club,
				"command", c.Text(),
			)
			return nil
		}
	}
}
