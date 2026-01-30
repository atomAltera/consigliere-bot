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
