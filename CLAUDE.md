# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Telegram bot for coordinating weekly mafia game events. Posts polls to collect participants, tracks votes, and manages event announcements.

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

## Key Features

- Bot posts polls for Monday/Saturday events with time slot options (19:00, 20:00, 21:00+, undecided, not coming)
- Only chat admins can use bot commands
- Commands are deleted after execution
- Voting history tracked in SQLite database
- Results message auto-updates when votes change
- Commands: `/poll`, `/results`, `/cancel`, `/pin`, `/restore`, `/vote`, `/call`, `/help`

## Message Handling

- **Error messages auto-delete**: All error messages are automatically deleted after 5 seconds (`DefaultTempMessageDelay` in `bot.go`)
- **Temporary messages**: Use `Bot.SendTemporary()` for notifications that should auto-delete
- **Error handling**: Return `UserErrorf()` or `WrapUserError()` from handlers - the `HandleErrors` middleware sends and auto-deletes the message
- **User vs system errors**: Messages defined in `messages.go` are split into user errors (shown directly) and system errors (hide internal details)

## Important

- When adding, changing, or removing bot commands, always update the `/help` template at `internal/bot/templates/help.html`