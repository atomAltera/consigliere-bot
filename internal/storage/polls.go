package storage

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

// optionsToString converts []OptionKind to comma-separated string
func optionsToString(opts []poll.OptionKind) string {
	if len(opts) == 0 {
		return "0,1,2,3,4" // default
	}
	strs := make([]string, len(opts))
	for i, o := range opts {
		strs[i] = strconv.Itoa(int(o))
	}
	return strings.Join(strs, ",")
}

// parseOptions converts comma-separated string to []OptionKind
func parseOptions(s string) []poll.OptionKind {
	if s == "" {
		return poll.DefaultOptions()
	}
	parts := strings.Split(s, ",")
	opts := make([]poll.OptionKind, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			opts = append(opts, poll.OptionKind(v))
		}
	}
	if len(opts) == 0 {
		return poll.DefaultOptions()
	}
	return opts
}

type PollRepository struct {
	db *DB
}

func NewPollRepository(db *DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) Create(p *poll.Poll) error {
	if len(p.Options) == 0 {
		p.Options = poll.DefaultOptions()
	}
	result, err := r.db.db.Exec(`
		INSERT INTO polls (tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, tg_cancel_message_id, event_date, options, is_active, is_pinned, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.TgChatID, p.TgPollID, p.TgMessageID, p.TgResultsMessageID, p.TgCancelMessageID, p.EventDate, optionsToString(p.Options), p.IsActive, p.IsPinned, time.Now())
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
		SELECT id, tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, tg_cancel_message_id, event_date, options, is_active, is_pinned, created_at
		FROM polls
		WHERE tg_chat_id = ? AND is_active = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, chatID)

	return r.scanPoll(row)
}

func (r *PollRepository) GetByTgPollID(tgPollID string) (*poll.Poll, error) {
	row := r.db.db.QueryRow(`
		SELECT id, tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, tg_cancel_message_id, event_date, options, is_active, is_pinned, created_at
		FROM polls
		WHERE tg_poll_id = ?
	`, tgPollID)

	return r.scanPoll(row)
}

func (r *PollRepository) GetLatestCancelled(chatID int64) (*poll.Poll, error) {
	row := r.db.db.QueryRow(`
		SELECT id, tg_chat_id, tg_poll_id, tg_message_id, tg_results_message_id, tg_cancel_message_id, event_date, options, is_active, is_pinned, created_at
		FROM polls
		WHERE tg_chat_id = ? AND is_active = 0
		ORDER BY created_at DESC
		LIMIT 1
	`, chatID)

	return r.scanPoll(row)
}

func (r *PollRepository) Update(p *poll.Poll) error {
	_, err := r.db.db.Exec(`
		UPDATE polls
		SET tg_poll_id = ?, tg_message_id = ?, tg_results_message_id = ?, tg_cancel_message_id = ?, options = ?, is_active = ?, is_pinned = ?
		WHERE id = ?
	`, p.TgPollID, p.TgMessageID, p.TgResultsMessageID, p.TgCancelMessageID, optionsToString(p.Options), p.IsActive, p.IsPinned, p.ID)
	if err != nil {
		return fmt.Errorf("update poll: %w", err)
	}
	return nil
}

func (r *PollRepository) scanPoll(row *sql.Row) (*poll.Poll, error) {
	var p poll.Poll
	var tgPollID sql.NullString
	var tgMessageID, tgResultsMessageID, tgCancelMessageID sql.NullInt64
	var optionsStr string

	err := row.Scan(
		&p.ID, &p.TgChatID, &tgPollID, &tgMessageID, &tgResultsMessageID, &tgCancelMessageID,
		&p.EventDate, &optionsStr, &p.IsActive, &p.IsPinned, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.TgPollID = tgPollID.String
	p.TgMessageID = int(tgMessageID.Int64)
	p.TgResultsMessageID = int(tgResultsMessageID.Int64)
	p.TgCancelMessageID = int(tgCancelMessageID.Int64)
	p.Options = parseOptions(optionsStr)

	return &p, nil
}
