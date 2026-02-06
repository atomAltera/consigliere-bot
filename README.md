# Consigliere Bot

Telegram bot for coordinating weekly mafia game events. Posts polls to collect participants, tracks votes, and manages event announcements.

## Features

- **Poll Management**: Create polls for Monday/Saturday game events with time slot options
- **Vote Tracking**: Records all votes in SQLite database with full history
- **Invitation Message**: Auto-updating message that reflects vote changes in real-time
- **Game Nicknames**: Link Telegram users to game nicknames for display
- **Admin-Only Commands**: Only chat administrators can control the bot
- **Clean Chat**: Command messages are deleted after execution

## Commands

| Command | Description |
|---------|-------------|
| `/poll [day]` | Create a poll for the specified day. Accepts day names (`monday`, `sat`) or dates (`2024-01-15`). Defaults to nearest Monday or Saturday. |
| `/results` | Show detailed voter info (Telegram ID, username, name, game nick). Auto-deletes after 30 seconds. |
| `/pin` | Pin the poll message and notify all members |
| `/cancel` | Cancel the event and notify participants |
| `/restore` | Restore the last cancelled poll (if event date hasn't passed) |
| `/vote <name> <1-5>` | Manually record a vote by @username or game nickname |
| `/nick <telegram> <gamenick>` | Link a Telegram user (@username or ID) to a game nickname |
| `/call` | Mention all undecided voters to remind them to vote |
| `/done` | Announce that enough players (11+) have been collected |
| `/help` | Show help message with all commands |

## Poll Options

1. Will come at 19:00
2. Will come at 20:00
3. Will come at 21:00 or later
4. Will decide later
5. Will not come

## Installation

### Prerequisites

- Go 1.24+
- Telegram Bot Token (from [@BotFather](https://t.me/BotFather))

### Build

```bash
go build -o bin/consigliere ./cmd/consigliere
```

### Configuration

Set the following environment variables:

| Variable | Description |
|----------|-------------|
| `TELEGRAM_BOT_API_KEY` | Your Telegram bot token |
| `DB_PATH` | Path to SQLite database file |

### Run

```bash
export TELEGRAM_BOT_API_KEY="your-bot-token"
export DB_PATH="./consigliere.db"
./bin/consigliere
```

## Development

```bash
# Run tests
go test ./...

# Run directly
go run ./cmd/consigliere
```

## Project Structure

```
.
├── cmd/consigliere/     # Application entry point
├── internal/
│   ├── bot/             # Telegram bot handlers, middleware, and templates
│   ├── config/          # Configuration loading
│   ├── logger/          # Structured logging setup
│   ├── poll/            # Poll domain logic
│   └── storage/         # SQLite database layer
```

## License

MIT License - see [LICENSE](LICENSE) for details.
