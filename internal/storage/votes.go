package storage

import (
	"database/sql"
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

// Record inserts a new vote record.
// Username is normalized to lowercase before storing.
func (r *VoteRepository) Record(v *poll.Vote) error {
	// Normalize username for storage
	normalizedUsername := poll.NormalizeUsername(v.TgUsername)

	result, err := r.db.db.Exec(`
		INSERT INTO votes (poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, is_manual, voted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, v.PollID, v.TgUserID, normalizedUsername, v.TgFirstName, v.TgOptionIndex, v.IsManual, time.Now())
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
				id, poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, is_manual, voted_at,
				ROW_NUMBER() OVER (PARTITION BY tg_user_id ORDER BY voted_at DESC) as rn
			FROM votes
			WHERE poll_id = ?
		)
		SELECT id, poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, is_manual, voted_at
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
		err := rows.Scan(&v.ID, &v.PollID, &v.TgUserID, &v.TgUsername, &v.TgFirstName, &v.TgOptionIndex, &v.IsManual, &v.VotedAt)
		if err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		votes = append(votes, &v)
	}

	return votes, rows.Err()
}

// LookupUserIDByUsername returns the user ID for a given username from voting history.
// Returns the most recent user ID associated with the username, true if found, or 0, false if not found.
// Username is normalized to lowercase for lookup.
func (r *VoteRepository) LookupUserIDByUsername(username string) (int64, bool, error) {
	var userID int64
	err := r.db.db.QueryRow(`
		SELECT tg_user_id FROM votes
		WHERE tg_username = ? AND tg_user_id > 0
		ORDER BY voted_at DESC
		LIMIT 1
	`, poll.NormalizeUsername(username)).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lookup user id by username: %w", err)
	}
	return userID, true, nil
}

// UpdateVotesUserID updates votes with oldUserID to use newUserID for a specific poll.
// Used for backfilling votes when nickname is set. Only affects the specified poll.
func (r *VoteRepository) UpdateVotesUserID(pollID int64, oldUserID, newUserID int64) error {
	_, err := r.db.db.Exec(`
		UPDATE votes SET tg_user_id = ? WHERE poll_id = ? AND tg_user_id = ?
	`, newUserID, pollID, oldUserID)
	if err != nil {
		return fmt.Errorf("update votes user id: %w", err)
	}
	return nil
}
