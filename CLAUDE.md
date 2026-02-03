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

## Key Requirements (from task.md)

- Bot posts polls for Monday/Saturday events with time slot options (19:00, 20:00, 21:00+, undecided, not coming)
- Only chat admins can use bot commands
- Commands should be deleted after execution
- Track voting history in database
- Commands: `/poll <day>`, `/results`, `/cancel`, `/pin`, `/restore`, `/vote`, `/help`
- Results message auto-updates when votes change
- Minimum 11 participants required; cancel event if not met by 5pm
- commit after each change

## Important

- When adding, changing, or removing bot commands, always update the `/help` template at `internal/bot/templates/help.html`