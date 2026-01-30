# Consigliere Bot Architecture Design

## Overview

Telegram bot for coordinating weekly mafia game events. Posts polls to collect participants, tracks votes, and manages event announcements.

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go 1.24 | Already set up |
| Database | SQLite (modernc.org/sqlite) | No CGO, simple deployment, sufficient scale |
| Telegram | telebot v4 (gopkg.in/telebot.v4) | Clean API, good poll support |
| Config | Environment variables | Simple, 12-factor style |
| Updates | Long polling | No public URL needed |

## Project Structure

```
consigliere-bot/
├── cmd/
│   └── consigliere/
│       └── main.go           # Entry point, wiring
├── internal/
│   ├── bot/
│   │   ├── bot.go            # Bot setup, middleware
│   │   ├── commands.go       # /poll, /results, /cancel, /pin handlers
│   │   └── admin.go          # Admin check via Telegram API
│   ├── poll/
│   │   ├── poll.go           # Poll domain types
│   │   ├── option.go         # OptionKind enum and mapping
│   │   ├── service.go        # Business logic
│   │   └── templates/
│   │       └── results.html  # Results message template
│   ├── storage/
│   │   ├── sqlite.go         # SQLite connection setup
│   │   ├── polls.go          # Poll repository
│   │   └── votes.go          # Vote repository
│   └── config/
│       └── config.go         # Load from env vars
├── .envrc
├── .gitignore
├── go.mod
└── go.sum
```

## Configuration

Environment variables (`.envrc`):

```bash
export TELEGRAM_BOT_API_KEY="your-bot-token"
export TELEGRAM_GROUP_ID="-123456789"
export DB_PATH="./data/consigliere.db"
```

## Database Schema

### Naming Convention

Fields containing values from Telegram use `tg_` prefix. Internal fields have no prefix.

### Tables

```sql
CREATE TABLE polls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tg_chat_id INTEGER NOT NULL,
    tg_poll_id TEXT,
    tg_message_id INTEGER,
    tg_results_message_id INTEGER,
    event_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'pinned', 'cancelled'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    poll_id INTEGER NOT NULL REFERENCES polls(id),
    tg_user_id INTEGER NOT NULL,
    tg_username TEXT,
    tg_first_name TEXT NOT NULL,
    tg_option_index INTEGER NOT NULL,  -- -1 for retracted vote
    voted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_votes_poll_id ON votes(poll_id);
CREATE INDEX idx_votes_user_latest ON votes(poll_id, tg_user_id, voted_at DESC);
```

### Vote History

Full vote history is preserved. Each vote/change INSERTs a new row. Current votes are queried by selecting the latest vote per user:

```sql
WITH ranked AS (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY tg_user_id ORDER BY voted_at DESC) as rn
    FROM votes WHERE poll_id = ?
)
SELECT * FROM ranked WHERE rn = 1 AND tg_option_index >= 0;
```

## Domain Types

### Option Kinds

```go
type OptionKind int

const (
    OptionComeAt19 OptionKind = iota      // "Will come at 19:00"
    OptionComeAt20                         // "Will come at 20:00"
    OptionComeAt21OrLater                  // "Will come at 21:00 or later"
    OptionDecideLater                      // "Will decide later"
    OptionNotComing                        // "Will not come"
)

func (o OptionKind) IsAttending() bool {
    return o <= OptionComeAt21OrLater  // First 3 options = attending
}
```

## Command Flows

### /poll <date>

1. Check sender is admin (Telegram API call)
2. Delete command message
3. Create poll record in DB (status=active)
4. Send Telegram poll to chat
5. Update DB with tg_poll_id, tg_message_id

### /results

1. Check sender is admin
2. Delete command message
3. Get latest active poll from DB
4. Query current votes (latest per user)
5. Render results template (HTML)
6. Send message, store tg_results_message_id

### /pin

1. Check sender is admin
2. Delete command message
3. Pin the poll message (notify_all=true)
4. Update poll status to 'pinned'

### /cancel

1. Check sender is admin
2. Delete command message
3. Delete results message if exists
4. Send cancellation message
5. Update poll status to 'cancelled'

### PollAnswer Event

1. Find poll by tg_poll_id
2. INSERT vote record with timestamp
3. If results message exists, update it

## Results Message Format

Rendered via Go `html/template`, sent with Telegram HTML parse mode:

```
🎲 <b>Mafia Night — Saturday, Feb 1</b>

✅ <b>Coming (13):</b>
  19:00: @alice, @bob, @carol
  20:00: @dave, @eve, @frank, @grace
  21:00+: @henry, @ivan, @judy, @kate, @leo, @mike

🤔 <b>Undecided (2):</b>
  @nancy, @oscar

❌ <b>Not coming (3):</b>
  @pat, @quinn, @rose
```

## Error Handling

| Scenario | Handling |
|----------|----------|
| Poll already exists for date | Return error, don't create duplicate |
| No active poll | Commands return "No active poll" |
| User retracts vote | INSERT with tg_option_index = -1 |
| Bot lacks permissions | Log error, don't crash |
| DB errors | Log and return user-friendly error |

## Startup Sequence

```go
func main() {
    cfg := config.Load()
    db := storage.NewSQLite(cfg.DBPath)
    db.Migrate()

    pollRepo := storage.NewPollRepository(db)
    voteRepo := storage.NewVoteRepository(db)
    pollService := poll.NewService(pollRepo, voteRepo)

    bot := bot.New(cfg.TelegramToken, cfg.GroupID, pollService)
    bot.RegisterCommands()
    bot.RegisterPollHandler()
    bot.Start()
}
```
