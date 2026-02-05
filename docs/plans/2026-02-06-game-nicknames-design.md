# Game Nicknames Feature Design

## Overview

Add support for game nicknames that differ from Telegram usernames. Players often have in-game names (e.g., "sectris") that differ from their Telegram handles (e.g., "@ak_altera"). This feature maps Telegram identities to game nicknames for display and vote resolution.

## Database Schema

### New Table: `nicknames`

```sql
CREATE TABLE IF NOT EXISTS nicknames (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tg_user_id INTEGER,           -- Telegram user ID (nullable, preferred)
    tg_username TEXT,             -- Telegram username without @ (nullable, fallback)
    game_nick TEXT NOT NULL,      -- Game nickname (e.g., "sectris")
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nicknames_tg_user_id ON nicknames(tg_user_id);
CREATE INDEX IF NOT EXISTS idx_nicknames_tg_username ON nicknames(tg_username);
CREATE INDEX IF NOT EXISTS idx_nicknames_game_nick ON nicknames(game_nick);
```

**Design decisions:**
- `game_nick` is NOT unique — allows multiple nicks per user (misspellings: "sectris", "сектрис")
- Either `tg_user_id` OR `tg_username` must be set (enforced in code)
- For display: query `ORDER BY created_at DESC LIMIT 1` for most recent nick
- For `/vote` lookup: search all game_nicks to find telegram identity

### New Index on `votes`

```sql
CREATE INDEX IF NOT EXISTS idx_votes_tg_username ON votes(tg_username);
```

Enables lookup of user ID by username from voting history.

## Commands

### `/nick` — Manage Nickname Mappings

**Usage:**
```
/nick @username gamenick   — link by telegram username
/nick 123456 gamenick      — link by numeric user ID
```

**Flow:**

1. Parse first argument:
   - Starts with `@` → telegram username
   - Numeric → telegram user ID
   - Otherwise → error

2. If username provided, look up user ID from `votes` table:
   ```sql
   SELECT DISTINCT tg_user_id FROM votes
   WHERE tg_username = ? AND tg_user_id > 0
   LIMIT 1
   ```

3. Check for duplicate before insert:
   - Only insert if exact `(tg_user_id, tg_username, game_nick)` combination doesn't exist

4. Save to `nicknames` table

5. **Backfill active poll votes:**
   - Find active poll for this chat
   - Find votes where:
     - `tg_username` matches (case-insensitive), OR
     - `tg_user_id` matches synthetic ID from any of this user's game nicks
   - Update those votes to use the canonical `tg_user_id`

6. Refresh invitation message if poll exists

### `/vote` — Enhanced Resolution

**Usage:**
```
/vote @username 1    — look up by telegram username
/vote gamenick 1     — look up by game nickname (no @ prefix)
```

**Resolution flow:**

1. **If starts with `@`** (telegram username):
   - Look up in `nicknames` by `tg_username` → get `tg_user_id` if available
   - If no nickname record, use username as-is (current behavior)

2. **If no `@`** (game nickname):
   - Look up in `nicknames` by `game_nick` → get `tg_user_id` and/or `tg_username`
   - If not found → error: "Unknown game nickname"

3. **Store vote with real user ID** (if available), enabling automatic deduplication

### `/results` — Redesigned Admin Info

**Current behavior:** Updates the invitation message

**New behavior:** Shows temporary silent message with detailed voter info

**Template: `results.html`**

```html
📊 <b>Результаты голосования</b>
{{.EventDate | ruDate}}

{{if .At19}}
<b>🕖 19:00 ({{len .At19}}):</b>
{{range .At19}}
• <code>{{.TgID}}</code> {{if .TgUsername}}@{{.TgUsername}}{{end}} {{.TgName}} {{if .Nickname}}→ {{.Nickname}}{{end}}
{{end}}
{{end}}

{{if .At20}}
<b>🕗 20:00 ({{len .At20}}):</b>
{{range .At20}}
• <code>{{.TgID}}</code> {{if .TgUsername}}@{{.TgUsername}}{{end}} {{.TgName}} {{if .Nickname}}→ {{.Nickname}}{{end}}
{{end}}
{{end}}

{{if .ComingLater}}
<b>🕘 21:00+ ({{len .ComingLater}}):</b>
{{range .ComingLater}}
• <code>{{.TgID}}</code> {{if .TgUsername}}@{{.TgUsername}}{{end}} {{.TgName}} {{if .Nickname}}→ {{.Nickname}}{{end}}
{{end}}
{{end}}

{{if .Undecided}}
<b>🤔 Думают ({{len .Undecided}}):</b>
{{range .Undecided}}
• <code>{{.TgID}}</code> {{if .TgUsername}}@{{.TgUsername}}{{end}} {{.TgName}} {{if .Nickname}}→ {{.Nickname}}{{end}}
{{end}}
{{end}}
```

- `<code>` tags make IDs copiable
- Shows: ID, @username (if any), name, game nick (if any)
- Sent as temporary silent message (30 seconds)
- Allows admin to copy user ID for `/nick` command

## Vote Recording Enhancement

When a real Telegram poll vote comes in:

1. Record the vote (existing logic)
2. **Backfill nicknames table:**
   ```sql
   UPDATE nicknames
   SET tg_user_id = ?
   WHERE tg_username = ?
     AND tg_user_id IS NULL
   ```

This progressively enriches `nicknames` with user IDs as users vote.

## Display Logic

### `Member` Struct Updates

```go
// MentionName returns name suitable for @mentions (telegram identity)
func (m Member) MentionName() string {
    if m.TgUsername != "" {
        return "@" + m.TgUsername
    }
    return m.TgName
}

// DisplayName returns name for display (prefers game nick)
func (m Member) DisplayName() string {
    if m.Nickname != "" {
        return m.Nickname
    }
    if m.TgName != "" {
        return m.TgName
    }
    if m.TgUsername != "" {
        return "@" + m.TgUsername
    }
    return ""
}
```

### Template Usage

| Template | Method | Reason |
|----------|--------|--------|
| `invitation.html` | `DisplayName()` | Show game nicks |
| `collected.html` | `DisplayName()` | Show game nicks |
| `call.html` | `MentionName()` | Need clickable @mentions |
| `cancel.html` | `MentionName()` | Need clickable @mentions |
| `restore.html` | `MentionName()` | Need clickable @mentions |
| `results.html` | All fields | Admin info display |

## Nickname Resolution at Query Time

When fetching votes for display, JOIN with nicknames to populate `Member.Nickname`:

```sql
SELECT v.*,
       (SELECT game_nick FROM nicknames
        WHERE (tg_user_id = v.tg_user_id OR tg_username = v.tg_username)
        ORDER BY created_at DESC LIMIT 1) as game_nick
FROM votes v
WHERE ...
```

## Files to Modify

| File | Changes |
|------|---------|
| `internal/storage/sqlite.go` | Add nicknames table and index migrations |
| `internal/storage/nicknames.go` | New file: NicknameRepository |
| `internal/bot/command_nick.go` | New file: `/nick` command handler |
| `internal/bot/command_vote.go` | Add game nick resolution |
| `internal/bot/command_results.go` | Redesign to show admin info |
| `internal/bot/member.go` | Add `MentionName()` method, update `DisplayName()` |
| `internal/bot/renderer.go` | Add `formatMentions()`, `RenderResultsMessage()` |
| `internal/bot/templates/results.html` | New template |
| `internal/bot/templates/call.html` | Use `MentionName()` |
| `internal/bot/templates/cancel.html` | Use `MentionName()` |
| `internal/bot/templates/restore.html` | Use `MentionName()` |
| `internal/bot/templates/help.html` | Add `/nick` and `/results` documentation |
| `internal/bot/handlers.go` | Backfill nicknames on vote |
| `internal/poll/service.go` | Add nickname-aware vote queries |

## Edge Cases

1. **User changes Telegram username:** Nickname linked by user ID still works; username-only links break (admin re-adds)
2. **User has no Telegram username:** Use `/nick 123456 gamenick` with numeric ID
3. **Multiple game nicks for same user:** All resolve to same user; most recent shown in display
4. **Game nick not found in `/vote`:** Return error "Unknown game nickname"
5. **Duplicate nickname record:** Skip insert if exact combination exists
