package bot

import (
	"errors"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

// UpdateDoneMessage re-renders and updates the done (collected) message for a poll.
// Returns true if the message was successfully updated, false otherwise.
// This is a non-critical operation — errors are logged but not returned.
func (b *Bot) UpdateDoneMessage(p *poll.Poll, config *ClubConfig) bool {
	if p.TgDoneMessageID == 0 {
		return false
	}

	data, err := b.pollService.GetCollectedData(p.ID)
	if err != nil {
		b.logger.Warn("failed to get collected data for refresh", "error", err)
		return false
	}

	// Use saved start time if available, otherwise recalculate
	var startTime string
	var mainVoters, comingLater []*poll.Vote

	if p.StartTime != "" {
		startTime = p.StartTime
		mainVoters, comingLater = poll.SplitVotersByStartTime(data, startTime)
	} else {
		// Fallback for polls created before start_time was saved
		result := poll.DetermineStartTimeAndVoters(data, minPlayersRequired)
		startTime = result.StartTime
		mainVoters = result.MainVoters
		comingLater = result.ComingLater

		if !result.EnoughPlayers {
			startTime = "20:00"
			mainVoters = append(data.Votes19, data.Votes20...)
			comingLater = data.Votes21
		}
	}

	// Build a single cache for all voters (more efficient than 2 separate caches)
	cache, cacheErr := b.buildNicknameCacheFromVotes(mainVoters, comingLater)
	if cacheErr != nil {
		b.logger.Warn("failed to build nickname cache for refresh", "error", cacheErr)
	}

	html, err := RenderCollectedMessage(config.templates, &CollectedData{
		EventDate:   p.EventDate,
		StartTime:   startTime,
		Members:     b.membersFromVotesWithCache(mainVoters, cache),
		ComingLater: b.membersFromVotesWithCache(comingLater, cache),
	})
	if err != nil {
		b.logger.Warn("failed to render collected message for refresh", "error", err)
		return false
	}

	msgRef := MessageRef(p.TgChatID, p.TgDoneMessageID)
	if config.MediaDir != "" {
		// Try caption edit first (video message), fall back to text edit
		// in case this is a pre-existing text message or video send failed.
		_, err = b.bot.EditCaption(msgRef, html, tele.ModeHTML)
		if err != nil && !isNotModifiedErr(err) {
			_, err = b.bot.Edit(msgRef, html, tele.ModeHTML)
		}
	} else {
		_, err = b.bot.Edit(msgRef, html, tele.ModeHTML)
	}
	if err != nil && !isNotModifiedErr(err) {
		b.logger.Warn("failed to refresh done message", "error", err)
		return false
	}
	return true
}

// UpdateCancelMessage re-renders and updates the cancel message for a poll.
// Returns true if the message was successfully updated, false otherwise.
// This is a non-critical operation — errors are logged but not returned.
func (b *Bot) UpdateCancelMessage(p *poll.Poll, config *ClubConfig) bool {
	if p.TgCancelMessageID == 0 {
		return false
	}

	votes, err := b.pollService.GetAttendingVotes(p.ID)
	if err != nil {
		b.logger.Warn("failed to get attending votes for refresh", "error", err)
		return false
	}

	html, err := RenderCancelMessage(config.templates, &CancelData{
		EventDate: p.EventDate,
		Members:   MembersFromVotes(votes),
	})
	if err != nil {
		b.logger.Warn("failed to render cancel message for refresh", "error", err)
		return false
	}

	if _, err = b.bot.Edit(MessageRef(p.TgChatID, p.TgCancelMessageID), html, tele.ModeHTML); err != nil {
		if isNotModifiedErr(err) {
			return true
		}
		b.logger.Warn("failed to refresh cancel message", "error", err)
		return false
	}
	return true
}

// handleRefresh re-renders and updates invitation, done, and cancel messages
// for the latest poll in this chat, regardless of whether it's active or past.
func (b *Bot) handleRefresh(c tele.Context) error {
	config := getClubConfig(c)

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
	if b.UpdateDoneMessage(p, config) {
		refreshed++
	}

	// Refresh cancel message if exists
	if b.UpdateCancelMessage(p, config) {
		refreshed++
	}

	if refreshed == 0 {
		return UserErrorf(MsgPollMessageMissing)
	}

	return nil
}
