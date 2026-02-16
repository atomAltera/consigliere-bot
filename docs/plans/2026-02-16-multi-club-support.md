# Multi-Club Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support multiple clubs (vanmo, tbilissimo) with per-club templates, admin lists, and game day configuration.

**Architecture:** Add `Club` domain type in `internal/poll`, define `ClubConfig` structs with per-club templates/admins/weekdays in `internal/bot`, use middleware to resolve chat→club and enforce permissions. Each render function receives `*template.Template` as a parameter instead of using globals.

**Tech Stack:** Go 1.24.5, telebot v4, SQLite, embed FS

---

### Task 1: Domain layer — Club type and Poll field

**Files:**
- Create: `internal/poll/club.go`
- Modify: `internal/poll/poll.go:5-21`
- Modify: `internal/storage/sqlite.go:90-95` (migrations)
- Modify: `internal/storage/polls.go:44-62,64-74,76-84,86-96,98-108,110-120,122-148`
- Modify: `internal/poll/service.go:86-130`

**Step 1: Create Club type**

Create `internal/poll/club.go`:
```go
package poll

// Club identifies which club a poll belongs to.
type Club string

const (
	ClubVanmo      Club = "vanmo"
	ClubTbilissimo Club = "tbilissimo"
)
```

**Step 2: Add Club field to Poll struct**

In `internal/poll/poll.go`, add `Club` field after `TgChatID`:
```go
type Poll struct {
	ID                    int64
	TgChatID              int64
	Club                  Club
	// ... rest unchanged
}
```

**Step 3: Add DB migration**

In `internal/storage/sqlite.go`, add to `migrations` slice:
```go
`ALTER TABLE polls ADD COLUMN club TEXT NOT NULL DEFAULT ''`,
`UPDATE polls SET club = 'vanmo' WHERE club = ''`,
```

**Step 4: Update storage layer**

In `internal/storage/polls.go`:
- `Create`: Add `club` to INSERT column list and values
- `Update`: Add `club` to SET clause
- `scanPoll`: Add `club` to SELECT column lists in all query methods, scan into `p.Club` (scan as string, cast to `poll.Club`)
- All 4 SELECT queries (`GetLatestActive`, `GetByTgPollID`, `GetLatestCancelled`, `GetLatest`) need `club` added to their column list

**Step 5: Update service CreatePoll**

In `internal/poll/service.go:111-118`, add `Club` parameter:
```go
func (s *Service) CreatePoll(tgChatID int64, eventDate time.Time, club Club) (*CreatePollResult, error) {
```

Set `Club: club` in the Poll struct created at line 111.

**Step 6: Update CreatePoll caller**

In `internal/bot/command_poll.go:25`, pass club:
```go
result, err := b.pollService.CreatePoll(c.Chat().ID, eventDate, /* club from config - will wire in Task 5 */)
```

Temporarily pass `poll.ClubVanmo` as default until Task 5 wires it properly.

**Step 7: Run tests**

Run: `go test ./internal/poll/ ./internal/storage/ ./internal/bot/`
Expect: All pass. The new `club` field defaults to empty string in tests which is acceptable.

**Step 8: Commit**

```
feat: add Club type to poll domain and club column to database
```

---

### Task 2: Template directory restructure

**Files:**
- Move: `internal/bot/templates/*.html` and `*.txt` → `internal/bot/templates/vanmo/`
- Create: `internal/bot/templates/tbilissimo/` (copy of vanmo with name changes)

**Step 1: Create directory structure and move files**

```bash
mkdir -p internal/bot/templates/vanmo internal/bot/templates/tbilissimo
git mv internal/bot/templates/*.html internal/bot/templates/vanmo/
git mv internal/bot/templates/*.txt internal/bot/templates/vanmo/
```

**Step 2: Create tbilissimo copies**

Copy all files from `vanmo/` to `tbilissimo/`. In the tbilissimo copies, replace:
- `VANMO` → `Tbilissimo` (in invitation.html)
- `JOIN BAR` → remains same for now (user will adjust text later)

Only replace the club name in the invitation header line. Leave all other text identical for now — user will adjust manually.

**Step 3: Verify embed still works**

The existing `//go:embed templates/*` directive in `renderer.go` will recursively include subdirectories. Verify this compiles.

**Step 4: Commit**

```
refactor: restructure templates into per-club directories
```

---

### Task 3: Per-club template loading and render function signatures

This task is the largest — template loading, render functions, and all callers must change atomically.

**Files:**
- Modify: `internal/bot/renderer.go` (major rewrite of loading + all render functions)
- Modify: `internal/bot/bot.go:201-232` (UpdateInvitationMessage)
- Modify: `internal/bot/command_nick.go:96-110` (RenderInvitationMessageWithNicks)
- Modify: `internal/bot/command_poll.go:60,78`
- Modify: `internal/bot/command_cancel.go:50-52`
- Modify: `internal/bot/command_restore.go:44-49`
- Modify: `internal/bot/command_done.go:148-155`
- Modify: `internal/bot/command_call.go:26-31`
- Modify: `internal/bot/command_results.go:50`
- Modify: `internal/bot/command_refresh.go:26,63,92`
- Modify: `internal/bot/command_help.go:12-14`
- Modify: `internal/bot/renderer_test.go`
- Modify: `cmd/consigliere/main.go:85-88`

**Step 1: Rewrite template loading in renderer.go**

Remove global template variables (`invitationTmpl`, `pollTitleTmpl`, etc.).

Replace `InitTemplates()` with:
```go
// ParseClubTemplates parses all templates for a club from the embedded FS.
// The subdir should be the club directory name under templates/ (e.g., "vanmo").
func ParseClubTemplates(subdir string) (*template.Template, error) {
	clubFS, err := fs.Sub(templateFS, "templates/"+subdir)
	if err != nil {
		return nil, fmt.Errorf("get club template FS %s: %w", subdir, err)
	}
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFS(clubFS, "*.html", "*.txt")
	if err != nil {
		return nil, fmt.Errorf("parse club templates %s: %w", subdir, err)
	}
	return tmpl, nil
}
```

Rename the embedded FS variable from `templates` to `templateFS` to avoid confusion:
```go
//go:embed templates/*
var templateFS embed.FS
```

**Step 2: Update all render functions to accept `*template.Template`**

Each render function gets `tmpl *template.Template` as first parameter and uses `tmpl.ExecuteTemplate`:

```go
func RenderPollTitleMessage(tmpl *template.Template, eventDate time.Time) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "poll_title.txt", eventDate); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderInvitationMessage(tmpl *template.Template, data *poll.InvitationData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "invitation.html", data); err != nil {
		return "", err
	}
	// ... truncation logic unchanged, but re-render also uses tmpl.ExecuteTemplate
	return buf.String(), nil
}

func RenderCancelMessage(tmpl *template.Template, data *CancelData) (string, error) { ... }
func RenderRestoreMessage(tmpl *template.Template, data *RestoreData) (string, error) { ... }
func RenderCallMessage(tmpl *template.Template, data *CallData) (string, error) { ... }
func RenderCollectedMessage(tmpl *template.Template, data *CollectedData) (string, error) { ... }
func RenderResultsMessage(tmpl *template.Template, data *ResultsData) (string, error) { ... }

func HelpMessage(tmpl *template.Template) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "help.html", nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

**Step 3: Update UpdateInvitationMessage to resolve templates**

In `bot.go`, `UpdateInvitationMessage` looks up the club config from the poll's chat ID:
```go
func (b *Bot) UpdateInvitationMessage(p *poll.Poll, isCancelledOverride *bool) bool {
	// ... existing nil check ...

	config := chatRegistry[p.TgChatID]
	if config == nil {
		b.logger.Warn("unknown chat for invitation update", "chat_id", p.TgChatID)
		return false
	}

	// ... get invitation data ...

	html, err := b.RenderInvitationMessageWithNicks(results, config.templates)
	// ... rest unchanged ...
}
```

**Step 4: Update RenderInvitationMessageWithNicks**

In `command_nick.go`:
```go
func (b *Bot) RenderInvitationMessageWithNicks(data *poll.InvitationData, tmpl *template.Template) (string, error) {
	// ... existing cache + enrichment logic ...
	return RenderInvitationMessage(tmpl, data)
}
```

**Step 5: Update all command handlers**

Each handler that calls a render function needs to get templates. For handlers inside the middleware chain, use `getClubConfig(c).templates`. Pattern:

```go
// In each handler:
config := getClubConfig(c)

// Then pass config.templates to render calls:
html, err := RenderInvitationMessage(config.templates, invitationData)
pollTitle, err := RenderPollTitleMessage(config.templates, eventDate)
```

Handlers to update:
- `handlePoll` — `RenderInvitationMessage`, `RenderPollTitleMessage`
- `handleCancel` — `RenderCancelMessage` (via RenderAndSend)
- `handleRestore` — `RenderRestoreMessage` (via RenderAndSend)
- `handleDone` — `RenderCollectedMessage` (via RenderAndSend)
- `handleCall` — `RenderCallMessage` (via RenderAndSend)
- `handleResults` — `RenderResultsMessage`
- `handleRefresh` — `RenderCollectedMessage`, `RenderCancelMessage` (direct calls), plus `UpdateInvitationMessage` (already handled)
- `handleHelp` — `HelpMessage`

**Note on RenderAndSend:** The `RenderAndSend` helper uses a closure `func() (string, error)` for the render function. The closure captures `config.templates` from the handler scope, so `RenderAndSend` itself doesn't need to change.

**Step 6: NOTE — This task depends on Task 4 (ClubConfig)**

`getClubConfig(c)` and `chatRegistry` are defined in Task 4. These two tasks must be implemented together or Task 3 must come after Task 4. The recommended order is: Task 4 first, then Task 3.

**Step 7: Update renderer_test.go**

The `TestMain` function needs to parse vanmo templates:
```go
func TestMain(m *testing.M) {
	var err error
	testTemplates, err = ParseClubTemplates("vanmo")
	if err != nil {
		panic("failed to parse templates: " + err.Error())
	}
	os.Exit(m.Run())
}
```

All test render calls pass `testTemplates` as first arg:
```go
result, err := RenderPollTitleMessage(testTemplates, eventDate)
result, err := RenderInvitationMessage(testTemplates, data)
```

**Step 8: Update main.go**

Remove `bot.InitTemplates()` call. Template parsing happens during `ClubConfig` initialization (see Task 4).

**Step 9: Run tests**

Run: `go test ./...`
Expect: All pass.

**Step 10: Commit**

```
refactor: per-club template loading and render function signatures
```

---

### Task 4: ClubConfig, registry, and middleware

**Files:**
- Create: `internal/bot/clubs.go`
- Modify: `internal/bot/admin.go:61-91` (replace isAdmin + AdminOnly)
- Modify: `internal/bot/commands.go:1-27` (middleware chain)
- Modify: `internal/bot/messages.go` (add chat-not-permitted message)

**Step 1: Create clubs.go**

```go
package bot

import (
	"html/template"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

// FeatureFlags holds per-club feature toggles.
type FeatureFlags struct{}

// ClubConfig holds configuration for a club's chat groups.
type ClubConfig struct {
	Club            poll.Club
	Name            string
	DefaultWeekDays []time.Weekday
	Admins          []int64
	FeatureFlags    FeatureFlags
	templates       *template.Template // unexported, accessed within bot package only
}

var vanmoConfig = &ClubConfig{
	Club:            poll.ClubVanmo,
	Name:            "VANMO",
	DefaultWeekDays: []time.Weekday{time.Monday, time.Saturday},
	Admins: []int64{
		// todo: replace with real admin user IDs
		100,
		101,
		102,
	},
}

var tbilissimoConfig = &ClubConfig{
	Club:            poll.ClubTbilissimo,
	Name:            "Tbilissimo",
	DefaultWeekDays: []time.Weekday{time.Monday, time.Saturday},
	Admins: []int64{
		// todo: replace with real admin user IDs
		200,
		201,
	},
}

// chatRegistry maps Telegram chat IDs to their club configuration.
var chatRegistry = map[int64]*ClubConfig{
	// todo: replace with real chat IDs
	-10: vanmoConfig,  // vanmo main
	-20: vanmoConfig,  // vanmo test
	-30: tbilissimoConfig, // tbilissimo main
	-40: tbilissimoConfig, // tbilissimo test
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
func getClubConfig(c interface{ Get(string) any }) *ClubConfig {
	return c.Get("club").(*ClubConfig)
}
```

**Step 2: Add ResolveClub middleware**

In `admin.go` (or `clubs.go`), add:
```go
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
```

**Step 3: Replace AdminOnly with ClubAdminOnly**

In `admin.go`, replace `AdminOnly()`:
```go
// ClubAdminOnly checks if the sender is in the club's admin list.
// Non-admins are silently ignored (command is deleted but no error posted).
func (b *Bot) ClubAdminOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			config := getClubConfig(c)
			userID := c.Sender().ID

			for _, adminID := range config.Admins {
				if adminID == userID {
					return next(c)
				}
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
```

Remove `isAdmin()` function and old `AdminOnly()` middleware.

**Step 4: Update middleware chain in commands.go**

```go
func (b *Bot) RegisterCommands() {
	adminGroup := b.bot.Group()
	adminGroup.Use(b.HandleErrors())
	adminGroup.Use(b.ResolveClub())
	adminGroup.Use(b.RateLimit())
	adminGroup.Use(b.DeleteCommand())
	adminGroup.Use(b.ClubAdminOnly())
	adminGroup.Use(b.LogCommand())
	// ... handlers unchanged ...
}
```

**Step 5: Add message constant**

In `messages.go`:
```go
MsgChatNotPermitted = "Этот чат не зарегистрирован для использования бота"
```

**Step 6: Update main.go**

Replace `bot.InitTemplates()` with `bot.InitClubTemplates()`:
```go
if err := bot.InitClubTemplates(); err != nil {
	appLog.Error("failed to initialize club templates", "error", err)
	os.Exit(1)
}
```

**Step 7: Run tests**

Run: `go test ./...`
Expect: All pass.

**Step 8: Commit**

```
feat: add ClubConfig registry with per-club admins and templates
```

---

### Task 5: Wire ClubConfig through command handlers

**Files:**
- Modify: `internal/bot/command_poll.go` (templates + club field)
- Modify: `internal/bot/command_cancel.go` (templates)
- Modify: `internal/bot/command_restore.go` (templates)
- Modify: `internal/bot/command_done.go` (templates)
- Modify: `internal/bot/command_call.go` (templates)
- Modify: `internal/bot/command_results.go` (templates)
- Modify: `internal/bot/command_refresh.go` (templates)
- Modify: `internal/bot/command_help.go` (templates)
- Modify: `internal/bot/command_nick.go` (templates)

**Step 1: Update handlePoll**

At the top of `handlePoll`, add:
```go
config := getClubConfig(c)
```

Change the render calls:
```go
invitationHTML, err := RenderInvitationMessage(config.templates, invitationData)
pollTitle, err := RenderPollTitleMessage(config.templates, eventDate)
```

Change `CreatePoll` call to pass club:
```go
result, err := b.pollService.CreatePoll(c.Chat().ID, eventDate, config.Club)
```

**Step 2: Update handleCancel**

Add `config := getClubConfig(c)` at top. Update RenderAndSend closure:
```go
sentMsg, err := b.RenderAndSend(c, func() (string, error) {
	return RenderCancelMessage(config.templates, cancelData)
}, ...)
```

**Step 3: Update handleRestore**

Add `config := getClubConfig(c)` at top. Update RenderAndSend closure:
```go
_, err = b.RenderAndSend(c, func() (string, error) {
	return RenderRestoreMessage(config.templates, &RestoreData{...})
}, ...)
```

**Step 4: Update handleDone**

Add `config := getClubConfig(c)` at top. Update RenderAndSend closure:
```go
sentMsg, err := b.RenderAndSend(c, func() (string, error) {
	return RenderCollectedMessage(config.templates, &CollectedData{...})
}, ...)
```

**Step 5: Update handleCall**

Add `config := getClubConfig(c)` at top. Update RenderAndSend closure:
```go
_, err = b.RenderAndSend(c, func() (string, error) {
	return RenderCallMessage(config.templates, &CallData{...})
}, ...)
```

**Step 6: Update handleResults**

Add `config := getClubConfig(c)` at top. Change:
```go
html, err := RenderResultsMessage(config.templates, resultsData)
```

**Step 7: Update handleRefresh**

Add `config := getClubConfig(c)` at top. Update all direct render calls in the refresh logic:
```go
html, err := RenderCollectedMessage(config.templates, &CollectedData{...})
html, err := RenderCancelMessage(config.templates, cancelData)
```

`UpdateInvitationMessage` already handles its own template lookup (Task 3, Step 3).

**Step 8: Update handleHelp**

```go
func (b *Bot) handleHelp(c tele.Context) error {
	config := getClubConfig(c)
	helpText, err := HelpMessage(config.templates)
	if err != nil {
		return fmt.Errorf("render help template: %w", err)
	}
	_, err = b.SendTemporary(c.Chat(), helpText, 30*time.Second, tele.ModeHTML)
	return err
}
```

**Step 9: Run tests**

Run: `go test ./...`
Expect: All pass.

**Step 10: Commit**

```
feat: wire ClubConfig through all command handlers
```

---

### Task 6: Parameterize date parsing

**Files:**
- Modify: `internal/bot/date_parsing.go:56-65,81-103`
- Modify: `internal/bot/command_poll.go:17`
- Modify: `internal/bot/commands_test.go` (if needed)
- Modify: `internal/bot/date_parsing_test.go`

**Step 1: Update nearestGameDay signature**

```go
// nearestGameDay returns the nearest game day from the given date,
// choosing the closest upcoming day from the provided weekdays list.
func nearestGameDay(from time.Time, weekdays []time.Weekday) time.Time {
	if len(weekdays) == 0 {
		return from
	}

	nearest := nextWeekday(from, weekdays[0])
	for _, wd := range weekdays[1:] {
		candidate := nextWeekday(from, wd)
		if candidate.Before(nearest) {
			nearest = candidate
		}
	}
	return nearest
}
```

**Step 2: Update parseEventDate to accept weekdays**

```go
func parseEventDate(args []string, defaultWeekDays []time.Weekday) (time.Time, error) {
	// ...
	if len(args) == 0 {
		return nearestGameDay(today, defaultWeekDays), nil
	}
	// ... rest unchanged ...
}
```

**Step 3: Update handlePoll call**

```go
eventDate, err := parseEventDate(c.Args(), config.DefaultWeekDays)
```

**Step 4: Update tests**

In `date_parsing_test.go`:
- `TestNearestGameDay`: Pass `[]time.Weekday{time.Monday, time.Saturday}` to match current behavior
- `TestParseEventDate_*` tests: Pass weekday slice

Add a new test for custom weekdays:
```go
func TestNearestGameDay_CustomWeekdays(t *testing.T) {
	// Wednesday
	wed := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	// With only Friday as game day
	result := nearestGameDay(wed, []time.Weekday{time.Friday})
	expected := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}
```

**Step 5: Run tests**

Run: `go test ./internal/bot/`
Expect: All pass.

**Step 6: Commit**

```
feat: parameterize game day weekdays from ClubConfig
```

---

### Task 7: Update tests for new admin/middleware logic

**Files:**
- Modify: `internal/bot/admin_test.go` (rewrite for list-based admin check)

**Step 1: Rewrite admin tests**

Remove all `mockBot`/`testBot` scaffolding that tests Telegram API admin checks. Replace with tests for the new list-based approach:

```go
func TestClubAdminOnly_AllowsAdmin(t *testing.T) {
	// Test that user ID in Admins list is allowed
}

func TestClubAdminOnly_RejectsNonAdmin(t *testing.T) {
	// Test that user ID NOT in Admins list is rejected silently
}

func TestResolveClub_KnownChat(t *testing.T) {
	// Test that known chat ID resolves config
}

func TestResolveClub_UnknownChat(t *testing.T) {
	// Test that unknown chat ID sends error and stops chain
}
```

**Step 2: Run tests**

Run: `go test ./internal/bot/`
Expect: All pass.

**Step 3: Commit**

```
test: update admin and middleware tests for list-based permissions
```

---

### Task 8: Final verification and cleanup

**Step 1: Run full test suite**

Run: `go test ./...`
Expect: All pass.

**Step 2: Run go vet and build**

```bash
go vet ./...
go build -o bin/consigliere ./cmd/consigliere
```

**Step 3: Verify gopls diagnostics**

Use `go_diagnostics` to check for any issues.

**Step 4: Review diff**

Review all changes to ensure no logic was broken. Pay special attention to:
- `UpdateInvitationMessage` — resolves templates correctly
- `handlePollAnswer` — works without middleware (looks up chatRegistry directly via UpdateInvitationMessage)
- `handleRefresh` — direct render calls use correct templates
- Migration — idempotent (ALTER + UPDATE pattern)

**Step 5: Commit any remaining fixes**

```
chore: final cleanup for multi-club support
```

---

## Implementation Order

Tasks 1-8 must be implemented in order. The critical dependency chain is:

```
Task 1 (domain) → Task 2 (template dirs) → Task 4 (ClubConfig) → Task 3 (render sigs) → Task 5 (handlers) → Task 6 (dates) → Task 7 (tests) → Task 8 (verify)
```

**Note:** Tasks 3 and 4 are tightly coupled — ClubConfig must exist before render functions can receive templates from it. Implement Task 4 before Task 3, or do them in the same commit.

## Key Risk Areas

1. **Template names in ExecuteTemplate**: After switching from `tmpl.Execute()` to `tmpl.ExecuteTemplate(&buf, "invitation.html", data)`, the template name must match the filename exactly. Verify with tests.

2. **UpdateInvitationMessage**: Called from 7+ places. Now resolves templates via `chatRegistry[p.TgChatID]`. If a poll exists for a chat that's removed from registry, it logs a warning and returns false (non-critical).

3. **handlePollAnswer**: Not in middleware chain, doesn't have ClubConfig in context. But it only calls `UpdateInvitationMessage` which self-resolves templates. No other template usage needed.

4. **Invitation truncation loop**: The re-render inside `RenderInvitationMessage`'s truncation loop must also use `tmpl.ExecuteTemplate`.
