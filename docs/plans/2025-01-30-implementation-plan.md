# Consigliere Bot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Telegram bot that posts polls for weekly mafia events, tracks votes, and manages results announcements.

**Architecture:** Standard Go layout with cmd/internal separation. SQLite for persistence, telebot v4 for Telegram API, long polling for updates. Admin-only commands that auto-delete after execution.

**Tech Stack:** Go 1.24, telebot v4, modernc.org/sqlite, html/template

---

## Task 1: Project Setup & Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add dependencies**

Run:
```bash
go get gopkg.in/telebot.v4
go get modernc.org/sqlite
```

**Step 2: Verify dependencies**

Run: `go mod tidy && cat go.mod`
Expected: Both dependencies listed in go.mod

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add telebot v4 and sqlite driver"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1: Write the failing test**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"testing"
)

func TestLoad_AllEnvVarsSet(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_API_KEY", "test-token")
	os.Setenv("TELEGRAM_GROUP_ID", "-123456")
	os.Setenv("DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_API_KEY")
		os.Unsetenv("TELEGRAM_GROUP_ID")
		os.Unsetenv("DB_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TelegramToken != "test-token" {
		t.Errorf("TelegramToken = %q, want %q", cfg.TelegramToken, "test-token")
	}
	if cfg.GroupID != -123456 {
		t.Errorf("GroupID = %d, want %d", cfg.GroupID, -123456)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/test.db")
	}
}

func TestLoad_MissingToken(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_API_KEY")
	os.Setenv("TELEGRAM_GROUP_ID", "-123456")
	os.Setenv("DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("TELEGRAM_GROUP_ID")
		os.Unsetenv("DB_PATH")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramToken string
	GroupID       int64
	DBPath        string
}

func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_API_KEY")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_API_KEY is required")
	}

	groupIDStr := os.Getenv("TELEGRAM_GROUP_ID")
	if groupIDStr == "" {
		return nil, fmt.Errorf("TELEGRAM_GROUP_ID is required")
	}
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_GROUP_ID must be a number: %w", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("DB_PATH is required")
	}

	return &Config{
		TelegramToken: token,
		GroupID:       groupID,
		DBPath:        dbPath,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package for env var loading"
```

---

## Task 3: Poll Domain Types

**Files:**
- Create: `internal/poll/option.go`
- Test: `internal/poll/option_test.go`

**Step 1: Write the failing test**

Create `internal/poll/option_test.go`:
```go
package poll

import "testing"

func TestOptionKind_Label(t *testing.T) {
	tests := []struct {
		opt  OptionKind
		want string
	}{
		{OptionComeAt19, "Will come at 19:00"},
		{OptionComeAt20, "Will come at 20:00"},
		{OptionComeAt21OrLater, "Will come at 21:00 or later"},
		{OptionDecideLater, "Will decide later"},
		{OptionNotComing, "Will not come"},
	}
	for _, tt := range tests {
		if got := tt.opt.Label(); got != tt.want {
			t.Errorf("OptionKind(%d).Label() = %q, want %q", tt.opt, got, tt.want)
		}
	}
}

func TestOptionKind_IsAttending(t *testing.T) {
	attending := []OptionKind{OptionComeAt19, OptionComeAt20, OptionComeAt21OrLater}
	notAttending := []OptionKind{OptionDecideLater, OptionNotComing}

	for _, opt := range attending {
		if !opt.IsAttending() {
			t.Errorf("OptionKind(%d).IsAttending() = false, want true", opt)
		}
	}
	for _, opt := range notAttending {
		if opt.IsAttending() {
			t.Errorf("OptionKind(%d).IsAttending() = true, want false", opt)
		}
	}
}

func TestAllOptions(t *testing.T) {
	opts := AllOptions()
	if len(opts) != 5 {
		t.Errorf("AllOptions() returned %d options, want 5", len(opts))
	}
	if opts[0] != "Will come at 19:00" {
		t.Errorf("AllOptions()[0] = %q, want %q", opts[0], "Will come at 19:00")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/poll/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/poll/option.go`:
```go
package poll

type OptionKind int

const (
	OptionComeAt19 OptionKind = iota
	OptionComeAt20
	OptionComeAt21OrLater
	OptionDecideLater
	OptionNotComing
	OptionRetracted OptionKind = -1
)

var optionLabels = []string{
	"Will come at 19:00",
	"Will come at 20:00",
	"Will come at 21:00 or later",
	"Will decide later",
	"Will not come",
}

func (o OptionKind) Label() string {
	if o < 0 || int(o) >= len(optionLabels) {
		return "Unknown"
	}
	return optionLabels[o]
}

func (o OptionKind) IsAttending() bool {
	return o >= 0 && o <= OptionComeAt21OrLater
}

func AllOptions() []string {
	return optionLabels
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/poll/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/poll/
git commit -m "feat: add poll option types and helpers"
```

---

## Task 4: Poll Domain Model

**Files:**
- Create: `internal/poll/poll.go`
- Create: `internal/poll/vote.go`

**Step 1: Write poll model**

Create `internal/poll/poll.go`:
```go
package poll

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusPinned    Status = "pinned"
	StatusCancelled Status = "cancelled"
)

type Poll struct {
	ID                  int64
	TgChatID            int64
	TgPollID            string
	TgMessageID         int
	TgResultsMessageID  int
	EventDate           time.Time
	Status              Status
	CreatedAt           time.Time
}
```

**Step 2: Write vote model**

Create `internal/poll/vote.go`:
```go
package poll

import "time"

type Vote struct {
	ID            int64
	PollID        int64
	TgUserID      int64
	TgUsername    string
	TgFirstName   string
	TgOptionIndex int
	VotedAt       time.Time
}

func (v *Vote) OptionKind() OptionKind {
	return OptionKind(v.TgOptionIndex)
}

func (v *Vote) DisplayName() string {
	if v.TgUsername != "" {
		return "@" + v.TgUsername
	}
	return v.TgFirstName
}
```

**Step 3: Run tests**

Run: `go test ./internal/poll/... -v`
Expected: PASS (no new tests, just compiles)

**Step 4: Commit**

```bash
git add internal/poll/
git commit -m "feat: add Poll and Vote domain models"
```

---

## Task 5: SQLite Storage Setup

**Files:**
- Create: `internal/storage/sqlite.go`
- Test: `internal/storage/sqlite_test.go`

**Step 1: Write the failing test**

Create `internal/storage/sqlite_test.go`:
```go
package storage

import (
	"os"
	"testing"
)

func TestNewDB_CreatesFile(t *testing.T) {
	path := "/tmp/test_consigliere.db"
	os.Remove(path)
	defer os.Remove(path)

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestMigrate_CreatesTables(t *testing.T) {
	path := "/tmp/test_consigliere_migrate.db"
	os.Remove(path)
	defer os.Remove(path)

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify tables exist
	var name string
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='polls'").Scan(&name)
	if err != nil {
		t.Errorf("polls table not found: %v", err)
	}

	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='votes'").Scan(&name)
	if err != nil {
		t.Errorf("votes table not found: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/storage/sqlite.go`:
```go
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func NewDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS polls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tg_chat_id INTEGER NOT NULL,
		tg_poll_id TEXT,
		tg_message_id INTEGER,
		tg_results_message_id INTEGER,
		event_date DATE NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		poll_id INTEGER NOT NULL REFERENCES polls(id),
		tg_user_id INTEGER NOT NULL,
		tg_username TEXT,
		tg_first_name TEXT NOT NULL,
		tg_option_index INTEGER NOT NULL,
		voted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON votes(poll_id);
	CREATE INDEX IF NOT EXISTS idx_votes_user_latest ON votes(poll_id, tg_user_id, voted_at DESC);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	return nil
}

func (d *DB) DB() *sql.DB {
	return d.db
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: add SQLite database setup and migrations"
```

---

## Task 6: Poll Repository

**Files:**
- Create: `internal/storage/polls.go`
- Test: `internal/storage/polls_test.go`

**Step 1: Write the failing test**

Create `internal/storage/polls_test.go`:
```go
package storage

import (
	"os"
	"testing"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	path := "/tmp/test_polls_" + time.Now().Format("20060102150405.000") + ".db"
	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return db, func() {
		db.Close()
		os.Remove(path)
	}
}

func TestPollRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}

	err := repo.Create(p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if p.ID == 0 {
		t.Error("expected poll ID to be set")
	}
}

func TestPollRepository_GetLatestActive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPollRepository(db)

	// Create a poll
	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	repo.Create(p)

	// Fetch latest
	latest, err := repo.GetLatestActive(-123456)
	if err != nil {
		t.Fatalf("GetLatestActive failed: %v", err)
	}
	if latest.ID != p.ID {
		t.Errorf("got poll ID %d, want %d", latest.ID, p.ID)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/... -v`
Expected: FAIL (NewPollRepository undefined)

**Step 3: Write minimal implementation**

Create `internal/storage/polls.go`:
```go
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

type PollRepository struct {
	db *DB
}

func NewPollRepository(db *DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) Create(p *poll.Poll) error {
	result, err := r.db.db.Exec(`
		INSERT INTO polls (tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, event_date, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.TgChatID, p.TgPollID, p.TgMessageID, p.TgResultsMessageID, p.EventDate, p.Status, time.Now())
	if err != nil {
		return fmt.Errorf("insert poll: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	p.ID = id
	return nil
}

func (r *PollRepository) GetLatestActive(chatID int64) (*poll.Poll, error) {
	row := r.db.db.QueryRow(`
		SELECT id, tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, event_date, status, created_at
		FROM polls
		WHERE tg_chat_id = ? AND status = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, chatID, poll.StatusActive)

	return r.scanPoll(row)
}

func (r *PollRepository) GetByTgPollID(tgPollID string) (*poll.Poll, error) {
	row := r.db.db.QueryRow(`
		SELECT id, tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, event_date, status, created_at
		FROM polls
		WHERE tg_poll_id = ?
	`, tgPollID)

	return r.scanPoll(row)
}

func (r *PollRepository) Update(p *poll.Poll) error {
	_, err := r.db.db.Exec(`
		UPDATE polls
		SET tg_poll_id = ?, tg_message_id = ?, tg_results_message_id = ?, status = ?
		WHERE id = ?
	`, p.TgPollID, p.TgMessageID, p.TgResultsMessageID, p.Status, p.ID)
	if err != nil {
		return fmt.Errorf("update poll: %w", err)
	}
	return nil
}

func (r *PollRepository) scanPoll(row *sql.Row) (*poll.Poll, error) {
	var p poll.Poll
	var tgPollID, status sql.NullString
	var tgMessageID, tgResultsMessageID sql.NullInt64

	err := row.Scan(
		&p.ID, &p.TgChatID, &tgPollID, &tgMessageID, &tgResultsMessageID,
		&p.EventDate, &status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.TgPollID = tgPollID.String
	p.TgMessageID = int(tgMessageID.Int64)
	p.TgResultsMessageID = int(tgResultsMessageID.Int64)
	p.Status = poll.Status(status.String)

	return &p, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: add poll repository"
```

---

## Task 7: Vote Repository

**Files:**
- Create: `internal/storage/votes.go`
- Test: `internal/storage/votes_test.go`

**Step 1: Write the failing test**

Create `internal/storage/votes_test.go`:
```go
package storage

import (
	"testing"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

func TestVoteRepository_RecordAndGetCurrent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pollRepo := NewPollRepository(db)
	voteRepo := NewVoteRepository(db)

	// Create a poll first
	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	pollRepo.Create(p)

	// Record a vote
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgUsername:    "alice",
		TgFirstName:   "Alice",
		TgOptionIndex: 0,
	}
	err := voteRepo.Record(v)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Get current votes
	votes, err := voteRepo.GetCurrentVotes(p.ID)
	if err != nil {
		t.Fatalf("GetCurrentVotes failed: %v", err)
	}

	if len(votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(votes))
	}
	if votes[0].TgUsername != "alice" {
		t.Errorf("expected username alice, got %s", votes[0].TgUsername)
	}
}

func TestVoteRepository_LatestVoteWins(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pollRepo := NewPollRepository(db)
	voteRepo := NewVoteRepository(db)

	p := &poll.Poll{
		TgChatID:  -123456,
		EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		Status:    poll.StatusActive,
	}
	pollRepo.Create(p)

	// User votes option 0
	voteRepo.Record(&poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgFirstName:   "Alice",
		TgOptionIndex: 0,
	})

	// User changes vote to option 1
	voteRepo.Record(&poll.Vote{
		PollID:        p.ID,
		TgUserID:      111,
		TgFirstName:   "Alice",
		TgOptionIndex: 1,
	})

	votes, _ := voteRepo.GetCurrentVotes(p.ID)
	if len(votes) != 1 {
		t.Fatalf("expected 1 current vote, got %d", len(votes))
	}
	if votes[0].TgOptionIndex != 1 {
		t.Errorf("expected option 1, got %d", votes[0].TgOptionIndex)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/... -v`
Expected: FAIL (NewVoteRepository undefined)

**Step 3: Write minimal implementation**

Create `internal/storage/votes.go`:
```go
package storage

import (
	"fmt"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

type VoteRepository struct {
	db *DB
}

func NewVoteRepository(db *DB) *VoteRepository {
	return &VoteRepository{db: db}
}

func (r *VoteRepository) Record(v *poll.Vote) error {
	result, err := r.db.db.Exec(`
		INSERT INTO votes (poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, voted_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, v.PollID, v.TgUserID, v.TgUsername, v.TgFirstName, v.TgOptionIndex, time.Now())
	if err != nil {
		return fmt.Errorf("insert vote: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	v.ID = id
	return nil
}

func (r *VoteRepository) GetCurrentVotes(pollID int64) ([]*poll.Vote, error) {
	rows, err := r.db.db.Query(`
		WITH ranked AS (
			SELECT
				id, poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, voted_at,
				ROW_NUMBER() OVER (PARTITION BY tg_user_id ORDER BY voted_at DESC) as rn
			FROM votes
			WHERE poll_id = ?
		)
		SELECT id, poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, voted_at
		FROM ranked
		WHERE rn = 1 AND tg_option_index >= 0
		ORDER BY tg_option_index, voted_at
	`, pollID)
	if err != nil {
		return nil, fmt.Errorf("query current votes: %w", err)
	}
	defer rows.Close()

	var votes []*poll.Vote
	for rows.Next() {
		var v poll.Vote
		err := rows.Scan(&v.ID, &v.PollID, &v.TgUserID, &v.TgUsername, &v.TgFirstName, &v.TgOptionIndex, &v.VotedAt)
		if err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		votes = append(votes, &v)
	}

	return votes, rows.Err()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: add vote repository with history support"
```

---

## Task 8: Poll Service

**Files:**
- Create: `internal/poll/service.go`
- Test: `internal/poll/service_test.go`

**Step 1: Write the failing test**

Create `internal/poll/service_test.go`:
```go
package poll

import (
	"testing"
	"time"
)

type mockPollRepo struct {
	polls   map[int64]*Poll
	counter int64
}

func (m *mockPollRepo) Create(p *Poll) error {
	m.counter++
	p.ID = m.counter
	m.polls[p.ID] = p
	return nil
}

func (m *mockPollRepo) GetLatestActive(chatID int64) (*Poll, error) {
	for _, p := range m.polls {
		if p.TgChatID == chatID && p.Status == StatusActive {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPollRepo) GetByTgPollID(tgPollID string) (*Poll, error) {
	for _, p := range m.polls {
		if p.TgPollID == tgPollID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPollRepo) Update(p *Poll) error {
	m.polls[p.ID] = p
	return nil
}

type mockVoteRepo struct {
	votes []*Vote
}

func (m *mockVoteRepo) Record(v *Vote) error {
	m.votes = append(m.votes, v)
	return nil
}

func (m *mockVoteRepo) GetCurrentVotes(pollID int64) ([]*Vote, error) {
	latest := make(map[int64]*Vote)
	for _, v := range m.votes {
		if v.PollID == pollID && v.TgOptionIndex >= 0 {
			existing, ok := latest[v.TgUserID]
			if !ok || v.VotedAt.After(existing.VotedAt) {
				latest[v.TgUserID] = v
			}
		}
	}
	var result []*Vote
	for _, v := range latest {
		result = append(result, v)
	}
	return result, nil
}

func TestService_CreatePoll(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	p, err := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreatePoll failed: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected poll to have ID")
	}
	if p.Status != StatusActive {
		t.Errorf("expected status active, got %s", p.Status)
	}
}

func TestService_GetResults(t *testing.T) {
	pollRepo := &mockPollRepo{polls: make(map[int64]*Poll)}
	voteRepo := &mockVoteRepo{}
	svc := NewService(pollRepo, voteRepo)

	p, _ := svc.CreatePoll(-123456, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))

	// Add votes
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 1, TgFirstName: "Alice", TgOptionIndex: 0, VotedAt: time.Now()})
	voteRepo.Record(&Vote{PollID: p.ID, TgUserID: 2, TgFirstName: "Bob", TgOptionIndex: 1, VotedAt: time.Now()})

	results, err := svc.GetResults(p.ID)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}
	if results.AttendingCount != 2 {
		t.Errorf("expected 2 attending, got %d", results.AttendingCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/poll/... -v`
Expected: FAIL (NewService undefined)

**Step 3: Write minimal implementation**

Create `internal/poll/service.go`:
```go
package poll

import "time"

type PollRepository interface {
	Create(p *Poll) error
	GetLatestActive(chatID int64) (*Poll, error)
	GetByTgPollID(tgPollID string) (*Poll, error)
	Update(p *Poll) error
}

type VoteRepository interface {
	Record(v *Vote) error
	GetCurrentVotes(pollID int64) ([]*Vote, error)
}

type Service struct {
	polls PollRepository
	votes VoteRepository
}

func NewService(polls PollRepository, votes VoteRepository) *Service {
	return &Service{polls: polls, votes: votes}
}

func (s *Service) CreatePoll(chatID int64, eventDate time.Time) (*Poll, error) {
	p := &Poll{
		TgChatID:  chatID,
		EventDate: eventDate,
		Status:    StatusActive,
		CreatedAt: time.Now(),
	}
	if err := s.polls.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetLatestActivePoll(chatID int64) (*Poll, error) {
	return s.polls.GetLatestActive(chatID)
}

func (s *Service) GetPollByTgPollID(tgPollID string) (*Poll, error) {
	return s.polls.GetByTgPollID(tgPollID)
}

func (s *Service) UpdatePoll(p *Poll) error {
	return s.polls.Update(p)
}

func (s *Service) RecordVote(v *Vote) error {
	return s.votes.Record(v)
}

type Results struct {
	Poll           *Poll
	TimeSlots      []TimeSlot
	Undecided      []*Vote
	NotComing      []*Vote
	AttendingCount int
}

type TimeSlot struct {
	Option OptionKind
	Label  string
	Voters []*Vote
}

func (s *Service) GetResults(pollID int64) (*Results, error) {
	votes, err := s.votes.GetCurrentVotes(pollID)
	if err != nil {
		return nil, err
	}

	results := &Results{
		TimeSlots: []TimeSlot{
			{Option: OptionComeAt19, Label: "19:00", Voters: []*Vote{}},
			{Option: OptionComeAt20, Label: "20:00", Voters: []*Vote{}},
			{Option: OptionComeAt21OrLater, Label: "21:00+", Voters: []*Vote{}},
		},
		Undecided: []*Vote{},
		NotComing: []*Vote{},
	}

	for _, v := range votes {
		switch OptionKind(v.TgOptionIndex) {
		case OptionComeAt19:
			results.TimeSlots[0].Voters = append(results.TimeSlots[0].Voters, v)
			results.AttendingCount++
		case OptionComeAt20:
			results.TimeSlots[1].Voters = append(results.TimeSlots[1].Voters, v)
			results.AttendingCount++
		case OptionComeAt21OrLater:
			results.TimeSlots[2].Voters = append(results.TimeSlots[2].Voters, v)
			results.AttendingCount++
		case OptionDecideLater:
			results.Undecided = append(results.Undecided, v)
		case OptionNotComing:
			results.NotComing = append(results.NotComing, v)
		}
	}

	return results, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/poll/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/poll/
git commit -m "feat: add poll service with results aggregation"
```

---

## Task 9: Results Template

**Files:**
- Create: `internal/poll/templates/results.html`
- Create: `internal/poll/renderer.go`
- Test: `internal/poll/renderer_test.go`

**Step 1: Write the failing test**

Create `internal/poll/renderer_test.go`:
```go
package poll

import (
	"strings"
	"testing"
	"time"
)

func TestRenderResults(t *testing.T) {
	results := &Results{
		Poll: &Poll{
			EventDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		TimeSlots: []TimeSlot{
			{Option: OptionComeAt19, Label: "19:00", Voters: []*Vote{
				{TgUsername: "alice", TgFirstName: "Alice"},
			}},
			{Option: OptionComeAt20, Label: "20:00", Voters: []*Vote{}},
			{Option: OptionComeAt21OrLater, Label: "21:00+", Voters: []*Vote{}},
		},
		Undecided:      []*Vote{},
		NotComing:      []*Vote{},
		AttendingCount: 1,
	}

	html, err := RenderResults(results)
	if err != nil {
		t.Fatalf("RenderResults failed: %v", err)
	}

	if !strings.Contains(html, "Coming (1)") {
		t.Error("expected attending count in output")
	}
	if !strings.Contains(html, "@alice") {
		t.Error("expected username in output")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/poll/... -v`
Expected: FAIL (RenderResults undefined)

**Step 3: Create template file**

Create `internal/poll/templates/results.html`:
```html
🎲 <b>Mafia Night — {{.Poll.EventDate.Format "Monday, Jan 2"}}</b>

✅ <b>Coming ({{.AttendingCount}}):</b>
{{- range .TimeSlots}}
{{- if .Voters}}
  {{.Label}}: {{range $i, $v := .Voters}}{{if $i}}, {{end}}{{$v.DisplayName}}{{end}}
{{- end}}
{{- end}}

🤔 <b>Undecided ({{len .Undecided}}):</b>
{{- if .Undecided}}
  {{range $i, $v := .Undecided}}{{if $i}}, {{end}}{{$v.DisplayName}}{{end}}
{{- else}}
  —
{{- end}}

❌ <b>Not coming ({{len .NotComing}}):</b>
{{- if .NotComing}}
  {{range $i, $v := .NotComing}}{{if $i}}, {{end}}{{$v.DisplayName}}{{end}}
{{- else}}
  —
{{- end}}
```

**Step 4: Write renderer implementation**

Create `internal/poll/renderer.go`:
```go
package poll

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templates embed.FS

var resultsTmpl *template.Template

func init() {
	var err error
	resultsTmpl, err = template.ParseFS(templates, "templates/results.html")
	if err != nil {
		panic(err)
	}
}

func RenderResults(results *Results) (string, error) {
	var buf bytes.Buffer
	if err := resultsTmpl.Execute(&buf, results); err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/poll/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/poll/
git commit -m "feat: add HTML results renderer with embedded template"
```

---

## Task 10: Bot Core Setup

**Files:**
- Create: `internal/bot/bot.go`

**Step 1: Write bot core**

Create `internal/bot/bot.go`:
```go
package bot

import (
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

type Bot struct {
	bot         *tele.Bot
	groupID     int64
	pollService *poll.Service
}

func New(token string, groupID int64, pollService *poll.Service) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot:         b,
		groupID:     groupID,
		pollService: pollService,
	}, nil
}

func (b *Bot) Start() {
	log.Println("Bot started")
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

func (b *Bot) Bot() *tele.Bot {
	return b.bot
}

func (b *Bot) GroupID() int64 {
	return b.groupID
}

func (b *Bot) PollService() *poll.Service {
	return b.pollService
}
```

**Step 2: Run build to verify it compiles**

Run: `go build ./internal/bot/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/bot/
git commit -m "feat: add bot core with telebot v4"
```

---

## Task 11: Admin Check

**Files:**
- Create: `internal/bot/admin.go`

**Step 1: Write admin check**

Create `internal/bot/admin.go`:
```go
package bot

import (
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) isAdmin(chatID int64, userID int64) (bool, error) {
	chat := &tele.Chat{ID: chatID}
	member, err := b.bot.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		return false, err
	}

	return member.Role == tele.Administrator || member.Role == tele.Creator, nil
}

func (b *Bot) AdminOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			isAdmin, err := b.isAdmin(c.Chat().ID, c.Sender().ID)
			if err != nil {
				return err
			}

			if !isAdmin {
				return nil // Silently ignore non-admin commands
			}

			return next(c)
		}
	}
}

func (b *Bot) DeleteCommand() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			// Delete the command message
			if err := c.Delete(); err != nil {
				// Log but don't fail - bot might lack delete permission
			}

			return next(c)
		}
	}
}
```

**Step 2: Run build to verify it compiles**

Run: `go build ./internal/bot/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/bot/
git commit -m "feat: add admin check and delete command middleware"
```

---

## Task 12: Bot Commands

**Files:**
- Create: `internal/bot/commands.go`

**Step 1: Write commands**

Create `internal/bot/commands.go`:
```go
package bot

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

func (b *Bot) RegisterCommands() {
	adminGroup := b.bot.Group()
	adminGroup.Use(b.AdminOnly())
	adminGroup.Use(b.DeleteCommand())

	adminGroup.Handle("/poll", b.handlePoll)
	adminGroup.Handle("/results", b.handleResults)
	adminGroup.Handle("/pin", b.handlePin)
	adminGroup.Handle("/cancel", b.handleCancel)
}

func (b *Bot) handlePoll(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Usage: /poll YYYY-MM-DD")
	}

	eventDate, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		return c.Send("Invalid date format. Use YYYY-MM-DD")
	}

	// Create poll in database
	p, err := b.pollService.CreatePoll(c.Chat().ID, eventDate)
	if err != nil {
		return c.Send("Failed to create poll: " + err.Error())
	}

	// Send Telegram poll
	telegramPoll := &tele.Poll{
		Question:       fmt.Sprintf("🎲 Mafia Night — %s", eventDate.Format("Monday, Jan 2")),
		Type:           tele.PollRegular,
		MultipleChoice: false,
		Options:        poll.AllOptions(),
	}

	msg, err := b.bot.Send(c.Chat(), telegramPoll)
	if err != nil {
		return c.Send("Failed to send poll: " + err.Error())
	}

	// Update poll with Telegram IDs
	p.TgPollID = msg.Poll.ID
	p.TgMessageID = msg.ID
	if err := b.pollService.UpdatePoll(p); err != nil {
		return c.Send("Failed to save poll IDs: " + err.Error())
	}

	return nil
}

func (b *Bot) handleResults(c tele.Context) error {
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil {
		return c.Send("Error: " + err.Error())
	}
	if p == nil {
		return c.Send("No active poll found")
	}

	results, err := b.pollService.GetResults(p.ID)
	if err != nil {
		return c.Send("Failed to get results: " + err.Error())
	}
	results.Poll = p

	html, err := poll.RenderResults(results)
	if err != nil {
		return c.Send("Failed to render results: " + err.Error())
	}

	msg, err := b.bot.Send(c.Chat(), html, tele.ModeHTML)
	if err != nil {
		return c.Send("Failed to send results: " + err.Error())
	}

	p.TgResultsMessageID = msg.ID
	b.pollService.UpdatePoll(p)

	return nil
}

func (b *Bot) handlePin(c tele.Context) error {
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil {
		return c.Send("Error: " + err.Error())
	}
	if p == nil {
		return c.Send("No active poll found")
	}

	msg := &tele.Message{ID: p.TgMessageID, Chat: c.Chat()}
	if err := b.bot.Pin(msg, tele.Silent); err != nil {
		return c.Send("Failed to pin: " + err.Error())
	}

	p.Status = poll.StatusPinned
	b.pollService.UpdatePoll(p)

	return nil
}

func (b *Bot) handleCancel(c tele.Context) error {
	p, err := b.pollService.GetLatestActivePoll(c.Chat().ID)
	if err != nil {
		return c.Send("Error: " + err.Error())
	}
	if p == nil {
		return c.Send("No active poll found")
	}

	// Delete results message if exists
	if p.TgResultsMessageID != 0 {
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: c.Chat()}
		b.bot.Delete(msg)
	}

	// Send cancellation
	_, err = b.bot.Send(c.Chat(), "❌ <b>Event cancelled</b>", tele.ModeHTML)
	if err != nil {
		return err
	}

	p.Status = poll.StatusCancelled
	b.pollService.UpdatePoll(p)

	return nil
}
```

**Step 2: Run build to verify it compiles**

Run: `go build ./internal/bot/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/bot/
git commit -m "feat: add /poll, /results, /pin, /cancel commands"
```

---

## Task 13: Poll Answer Handler

**Files:**
- Modify: `internal/bot/bot.go`
- Create: `internal/bot/handlers.go`

**Step 1: Write poll answer handler**

Create `internal/bot/handlers.go`:
```go
package bot

import (
	tele "gopkg.in/telebot.v4"

	"nuclight.org/consigliere/internal/poll"
)

func (b *Bot) RegisterHandlers() {
	b.bot.Handle(tele.OnPollAnswer, b.handlePollAnswer)
}

func (b *Bot) handlePollAnswer(c tele.Context) error {
	answer := c.PollAnswer()
	if answer == nil {
		return nil
	}

	// Find the poll
	p, err := b.pollService.GetPollByTgPollID(answer.PollID)
	if err != nil || p == nil {
		return nil // Not our poll or error
	}

	// Determine option index (-1 if retracted)
	optionIndex := -1
	if len(answer.Options) > 0 {
		optionIndex = answer.Options[0]
	}

	// Record vote
	v := &poll.Vote{
		PollID:        p.ID,
		TgUserID:      answer.Sender.ID,
		TgUsername:    answer.Sender.Username,
		TgFirstName:   answer.Sender.FirstName,
		TgOptionIndex: optionIndex,
	}
	if err := b.pollService.RecordVote(v); err != nil {
		return err
	}

	// Update results message if exists
	if p.TgResultsMessageID != 0 {
		results, err := b.pollService.GetResults(p.ID)
		if err != nil {
			return err
		}
		results.Poll = p

		html, err := poll.RenderResults(results)
		if err != nil {
			return err
		}

		chat := &tele.Chat{ID: p.TgChatID}
		msg := &tele.Message{ID: p.TgResultsMessageID, Chat: chat}
		_, err = b.bot.Edit(msg, html, tele.ModeHTML)
		if err != nil {
			// Log but don't fail
		}
	}

	return nil
}
```

**Step 2: Run build to verify it compiles**

Run: `go build ./internal/bot/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/bot/
git commit -m "feat: add poll answer handler with results auto-update"
```

---

## Task 14: Main Entry Point

**Files:**
- Create: `cmd/consigliere/main.go`

**Step 1: Write main.go**

Create `cmd/consigliere/main.go`:
```go
package main

import (
	"log"

	"nuclight.org/consigliere/internal/bot"
	"nuclight.org/consigliere/internal/config"
	"nuclight.org/consigliere/internal/poll"
	"nuclight.org/consigliere/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create repositories
	pollRepo := storage.NewPollRepository(db)
	voteRepo := storage.NewVoteRepository(db)

	// Create service
	pollService := poll.NewService(pollRepo, voteRepo)

	// Create and start bot
	b, err := bot.New(cfg.TelegramToken, cfg.GroupID, pollService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	b.RegisterCommands()
	b.RegisterHandlers()

	log.Println("Starting bot...")
	b.Start()
}
```

**Step 2: Build the binary**

Run: `go build -o bin/consigliere ./cmd/consigliere`
Expected: Binary created at `bin/consigliere`

**Step 3: Commit**

```bash
git add cmd/
git commit -m "feat: add main entry point"
```

---

## Task 15: Integration Test

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Test build**

Run: `go build -o bin/consigliere ./cmd/consigliere && ls -la bin/`
Expected: Binary exists

**Step 3: Final commit**

```bash
git add -A
git commit -m "chore: complete initial implementation"
```

---

## Summary

15 tasks covering:
1. Project setup
2. Config package
3. Poll domain types
4. Poll/vote models
5. SQLite storage
6. Poll repository
7. Vote repository
8. Poll service
9. Results template
10. Bot core
11. Admin middleware
12. Commands
13. Poll answer handler
14. Main entry point
15. Integration test

Each task is self-contained with tests, implementation, and commit.
