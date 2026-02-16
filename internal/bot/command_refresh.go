package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// handleRefresh re-renders and updates invitation, done, and cancel messages
// for the latest poll in this chat, regardless of whether it's active or past.
func (b *Bot) handleRefresh(c tele.Context) error {
	// Get latest poll (regardless of status)
	p, err := b.pollService.GetLatestPoll(c.Chat().ID)
	if err != nil {
		if errors.Is(err, poll.ErrNoActivePoll) {
			return UserErrorf(MsgNoPoll)
		}
		return WrapUserError(MsgFailedGetPoll, err)
	}

	var refreshed int

	// Refresh invitation message if exists
	if b.UpdateInvitationMessage(p, nil) {
		refreshed++
	}

	// Refresh done message if exists
	if p.TgDoneMessageID != 0 {
		data, err := b.pollService.GetCollectedData(p.ID)
		if err != nil {
			b.logger.Warn("failed to get collected data for refresh", "error", err)
		} else {
			// Determine start time and voter groups
			// Note: for refresh, we don't check EnoughPlayers since the done message already exists
			result := poll.DetermineStartTimeAndVoters(data, minPlayersRequired)
			startTime := result.StartTime
			mainVoters := result.MainVoters
			comingLater := result.ComingLater

			// Fallback if not enough players (message was created when there were enough)
			if !result.EnoughPlayers {
				startTime = "20:00"
				mainVoters = append(data.Votes19, data.Votes20...)
				comingLater = data.Votes21
			}

			// Build a single cache for all voters (more efficient than 2 separate caches)
			cache, cacheErr := b.buildNicknameCacheFromVotes(mainVoters, comingLater)
			if cacheErr != nil {
				b.logger.Warn("failed to build nickname cache for refresh", "error", cacheErr)
				// Continue without nicknames - membersFromVotesWithCache handles nil cache
			}

			html, err := RenderCollectedMessage(&CollectedData{
				EventDate:   p.EventDate,
				StartTime:   startTime,
				Members:     b.membersFromVotesWithCache(mainVoters, cache),
				ComingLater: b.membersFromVotesWithCache(comingLater, cache),
			})
			if err != nil {
				b.logger.Warn("failed to render collected message for refresh", "error", err)
			} else {
				if _, err = b.bot.Edit(MessageRef(p.TgChatID, p.TgDoneMessageID), html, tele.ModeHTML); err != nil && !isNotModifiedErr(err) {
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
				if _, err = b.bot.Edit(MessageRef(p.TgChatID, p.TgCancelMessageID), html, tele.ModeHTML); err != nil && !isNotModifiedErr(err) {
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

	return nil
}
