package poll

import "errors"

var (
	ErrNoActivePoll    = errors.New("no active poll")
	ErrNoCancelledPoll = errors.New("no cancelled poll")
	ErrPollExists      = errors.New("active poll already exists")
	ErrPollDatePassed  = errors.New("poll date has passed")
)
