# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Telegram bot for coordinating weekly mafia game events. Supports multiple clubs, each with its own chat, admins, templates, and media. Posts polls to collect participants, tracks votes, and manages event announcements.

## Build Commands

```bash
go build -o bin/consigliere ./cmd/consigliere    # Build binary
go test ./...                                     # Run all tests
go test -v ./internal/poll                        # Run tests for specific package
go run ./cmd/consigliere                          # Run directly
```

## Module

- Package: `nuclight.org/consigliere`
- Go version: 1.24.5

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_API_KEY` | Yes | Telegram bot token from @BotFather |
| `DB_PATH` | Yes | Path to SQLite database file |
| `SENTRY_DSN` | No | Sentry DSN for error tracking |
| `DEV_MODE` | No | Set to `true` for development environment |
| `TEMP_MESSAGE_DELAY_SECONDS` | No | Delay before auto-deleting temp messages (default: 5) |
| `POLLING_TIMEOUT_SECONDS` | No | Telegram long polling timeout (default: 10) |

Configuration can also be loaded from a `.env` file in the project root.

## Architecture

### Multi-Club Support

The bot serves multiple clubs from a single instance. Each club has its own:
- **ClubConfig** (`internal/bot/clubs.go`): club enum, name, game day weekdays, admin list, media directory, feature flags, parsed templates
- **Chat registry**: maps Telegram chat IDs to ClubConfig
- **Templates**: per-club directory under `internal/bot/templates/<club>/` (e.g., `vanmo/`, `tbilissimo/`)
- **Media**: per-club embedded videos under `internal/bot/media/<club>/` (e.g., `vanmo/monday.mp4`)

Middleware pipeline: `DeleteCommand` → `HandleErrors` → `ResolveClub` → `ClubAdminOnly` → handler

### Key Packages

| Package | Purpose |
|---------|---------|
| `cmd/consigliere` | Entry point, config loading, bot setup |
| `internal/bot` | Telegram handlers, middleware, templates, media, rendering |
| `internal/poll` | Domain logic: polls, votes, clubs, time slot calculations |
| `internal/storage` | SQLite database layer |
| `internal/config` | Environment variable loading |
| `internal/logger` | Structured logging setup |

## Key Features

- Polls with time slot options (19:00, 20:00, 21:00+, undecided, not coming)
- Game day weekdays configurable per club (VANMO: Mon/Sat, Tbilissimo: Wed/Sun)
- Per-club admin permissions — only club admins can use bot commands
- Commands are deleted after execution
- Voting history tracked in SQLite database
- Results message auto-updates when votes change
- `/done` sends event video with caption for clubs with media (falls back to text on failure)
- Commands: `/poll`, `/results`, `/cancel`, `/pin`, `/restore`, `/vote`, `/nick`, `/call`, `/done`, `/refresh`, `/help`

## Message Handling

- **Error messages auto-delete**: All error messages are automatically deleted after a configurable delay (`tempMessageDelay` in `bot.go`)
- **Temporary messages**: Use `Bot.SendTemporary()` for notifications that should auto-delete
- **Error handling**: Return `UserErrorf()` or `WrapUserError()` from handlers — the `HandleErrors` middleware sends and auto-deletes the message
- **User vs system errors**: Messages defined in `messages.go` are split into user errors (shown directly) and system errors (hide internal details)
- **Video messages**: When editing a video message caption, use `bot.EditCaption()` instead of `bot.Edit()` (Telegram's `editMessageText` fails on video messages)

## Code Review

Use `codex review` for non-interactive code review:

```bash
codex review --uncommitted                    # Review all uncommitted changes
codex review --commit HEAD                    # Review the last commit
codex review --base main                      # Review changes against main branch
```

## Important

- When adding, changing, or removing bot commands, always update the `/help` template in **each** club's template directory: `internal/bot/templates/<club>/help.html`
- Templates and media are embedded via `go:embed` — changes require a rebuild
- When adding a new club, update `clubs.go` (config + registry + `InitClubTemplates`) and create a template directory under `templates/`
