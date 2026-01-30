# Consigliere Bot

Telegram bot for coordinating weekly mafia game events. Posts polls to collect participants, tracks votes, and manages event announcements.

## Features

- **Poll Management**: Create polls for Monday/Saturday game events with time slot options
- **Vote Tracking**: Records all votes in SQLite database with full history
- **Results Display**: Auto-updating results message that reflects vote changes in real-time
- **Admin-Only Commands**: Only chat administrators can control the bot
- **Clean Chat**: Command messages are deleted after execution

## Commands

| Command | Description |
|---------|-------------|
| `/poll [day]` | Create a poll for the specified day. Accepts day names (`monday`, `sat`) or dates (`2024-01-15`). Defaults to nearest Monday or Saturday. |
| `/results` | Post/update the results message for the active poll |
| `/pin` | Pin the poll message and notify all members |
| `/cancel` | Cancel the event and pin a cancellation notice |
| `/vote @user <1-5>` | Manually record a vote for a user (1=19:00, 2=20:00, 3=21:00+, 4=undecided, 5=not coming) |

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
│   ├── bot/             # Telegram bot handlers and middleware
│   ├── config/          # Configuration loading
│   ├── poll/            # Poll domain logic and rendering
│   └── storage/         # SQLite database layer
└── templates/           # HTML templates for results
```

## License

MIT License - see [LICENSE](LICENSE) for details.
