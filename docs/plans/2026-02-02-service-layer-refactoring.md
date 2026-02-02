# Service Layer Refactoring Design

## Overview
Refactor to move business logic from command handlers to poll service. Handlers become thin controllers, service contains all business logic.

## Package Responsibilities

### `internal/poll/` - Domain/Business
- `poll.go` - Poll struct (with `Options []OptionKind`)
- `vote.go` - Vote struct
- `option.go` - OptionKind constants only (no labels)
- `service.go` - Business logic methods
- `errors.go` - Business errors

### `internal/bot/` - Presentation/Telegram
- `commands.go` - Thin handlers
- `handlers.go` - Poll answer handler
- `renderer.go` - Template rendering (moved from poll/)
- `templates/` - HTML/text templates (moved from poll/)
- `messages.go` - All user-facing text
- `options.go` - Option labels mapping

## Service API

```go
func (s *Service) CreatePoll(tgChatID int64, eventDate time.Time) (*Poll, error)
func (s *Service) SetPollMessageIDs(pollID int64, tgResultsMessageID, tgMessageID int) error
func (s *Service) CancelPoll(tgChatID int64) (*Poll, error)
func (s *Service) SetCancelMessageID(pollID int64, tgCancelMessageID int) error
func (s *Service) RestorePoll(tgChatID int64) (*Poll, error)
func (s *Service) RecordVote(pollID int64, tgUserID int64, tgUsername, tgFirstName string, option OptionKind) (*InvitationData, error)
func (s *Service) GetInvitationData(pollID int64) (*InvitationData, error)
func (s *Service) GetActivePoll(tgChatID int64) (*Poll, error)
func (s *Service) SetPinned(tgChatID int64, pinned bool) (*Poll, error)
```

## Business Errors

```go
var (
    ErrNoActivePoll    = errors.New("no active poll")
    ErrNoCancelledPoll = errors.New("no cancelled poll")
    ErrPollExists      = errors.New("active poll already exists")
    ErrPollDatePassed  = errors.New("poll date has passed")
)
```

## Poll Struct Update

```go
type Poll struct {
    ID                 int64
    TgChatID           int64
    TgPollID           string
    TgMessageID        int
    TgResultsMessageID int
    TgCancelMessageID  int
    EventDate          time.Time
    Options            []OptionKind  // NEW: enabled options for this poll
    IsActive           bool
    IsPinned           bool
    CreatedAt          time.Time
}
```

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS polls (
    -- existing fields...
    options TEXT NOT NULL DEFAULT '0,1,2,3,4',  -- comma-separated OptionKind values
);
```

## Files to Move
- `poll/renderer.go` → `bot/renderer.go`
- `poll/templates/` → `bot/templates/`
- Option labels from `poll/option.go` → `bot/options.go`
