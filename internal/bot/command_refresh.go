package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleRefresh re-renders and updates invitation, done, and cancel messages
// for the latest poll in this chat, regardless of whether it's active or past.
func (b *Bot) handleRefresh(c tele.Context) error {
	b.logger.Info("command /refresh",
		"user_id", c.Sender().ID,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)

	// Get latest poll (regardless of status)
	p, err := b.pollService.GetLatestPoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoPoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
	}

	chat := &tele.Chat{ID: p.TgChatID}
	var refreshed int

	// Refresh invitation message if exists
	if p.TgInvitationMessageID != 0 {
		results, err := b.pollService.GetInvitationData(p.ID)
		if err != nil {
			return WrapUserError(MsgFailedGetResults, err)
		}

		results.Poll = p
		results.EventDate = p.EventDate
		results.IsCancelled = !p.IsActive

		html, err := b.RenderInvitationWithNicks(results)
		if err != nil {
			return WrapUserError(MsgFailedRenderResults, err)
		}

		msg := &tele.Message{ID: p.TgInvitationMessageID, Chat: chat}
		if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
			b.logger.Warn("failed to refresh invitation message", "error", err)
		} else {
			refreshed++
		}
	}

	// Refresh done message if exists
	if p.TgDoneMessageID != 0 {
		data, err := b.pollService.GetCollectedData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get collected data for refresh", "error", err)
		} else {
			count19 := len(data.Votes19)
			count20 := len(data.Votes20)
			totalEarly := count19 + count20

			var startTime string
			var votesToMention []*poll.Vote
			var comingLater []*poll.Vote

			if count19 >= minPlayersRequired {
				startTime = "19:00"
				votesToMention = data.Votes19
				comingLater = append(data.Votes20, data.Votes21...)
			} else if totalEarly >= minPlayersRequired {
				startTime = "20:00"
				votesToMention = append(data.Votes19, data.Votes20...)
				comingLater = data.Votes21
			} else {
				// Not enough players - use all early voters and coming later
				startTime = "20:00"
				votesToMention = append(data.Votes19, data.Votes20...)
				comingLater = data.Votes21
			}

			html, err := RenderCollectedMessage(&CollectedData{
				EventDate:   p.EventDate,
				StartTime:   startTime,
				Members:     b.membersFromVotesWithNicknames(votesToMention),
				ComingLater: b.membersFromVotesWithNicknames(comingLater),
			})
			if err != nil {
				b.logger.Warn("failed to render collected message for refresh", "error", err)
			} else {
				msg := &tele.Message{ID: p.TgDoneMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to refresh done message", "error", err)
				} else {
					refreshed++
				}
			}
		}
	}

	// Refresh cancel message if exists
	if p.TgCancelMessageID != 0 {
		votes, err := b.pollService.GetAttendingVotes(p.ID)
		if err != nil {
			b.logger.Warn("failed to get attending votes for refresh", "error", err)
		} else {
			cancelData := &CancelData{
				EventDate: p.EventDate,
				Members:   MembersFromVotes(votes),
			}

			html, err := RenderCancelMessage(cancelData)
			if err != nil {
				b.logger.Warn("failed to render cancel message for refresh", "error", err)
			} else {
				msg := &tele.Message{ID: p.TgCancelMessageID, Chat: chat}
				if _, err = b.bot.Edit(msg, html, tele.ModeHTML); err != nil {
					b.logger.Warn("failed to refresh cancel message", "error", err)
				} else {
					refreshed++
				}
			}
		}
	}

	if refreshed == 0 {
		return UserErrorf(MsgPollMessageMissing)
	}

	_, err = b.SendTemporary(c.Chat(), "Сообщения обновлены", 0)
	return err
}
