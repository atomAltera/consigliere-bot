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

// LookupUsernameByUserID returns the username for a user ID from voting history.
// Returns the most recent username associated with the user ID.
func (r *VoteRepository) LookupUsernameByUserID(userID int64) (string, bool, error) {
	var username string
	err := r.db.db.QueryRow(`
		SELECT tg_username FROM votes
		WHERE tg_user_id = ? AND tg_username != ''
		ORDER BY voted_at DESC
		LIMIT 1
	`, userID).Scan(&username)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lookup username by user id: %w", err)
	}
	return username, true, nil
}

// ConsolidateSyntheticVotes updates synthetic votes to use real user data.
// For a given poll, finds votes with synthetic IDs (derived from username or game nicks)
// and updates them to use the real user ID and username.
func (r *VoteRepository) ConsolidateSyntheticVotes(pollID int64, realUserID int64, username string, gameNicks []string) error {
	normalizedUsername := poll.NormalizeUsername(username)

	// Update synthetic votes by username
	if normalizedUsername != "" {
		syntheticID := poll.ManualUserID(normalizedUsername)
		_, err := r.db.db.Exec(`
			UPDATE votes
			SET tg_user_id = ?
			WHERE poll_id = ? AND tg_user_id = ?
		`, realUserID, pollID, syntheticID)
		if err != nil {
			return fmt.Errorf("consolidate votes by username: %w", err)
		}
	}

	// Update synthetic votes by game nicks
	for _, nick := range gameNicks {
		syntheticID := poll.ManualUserID(nick)
		_, err := r.db.db.Exec(`
			UPDATE votes
			SET tg_user_id = ?, tg_username = COALESCE(NULLIF(tg_username, ''), ?)
			WHERE poll_id = ? AND tg_user_id = ?
		`, realUserID, normalizedUsername, pollID, syntheticID)
		if err != nil {
			return fmt.Errorf("consolidate votes by game nick %q: %w", nick, err)
		}
	}

	return nil
}

// UpdateVotesUserID updates votes with oldUserID to use newUserID for a specific poll.
// Also updates tg_username if provided (non-empty) and the vote has no username.
// Used for backfilling votes when nickname is set. Only affects the specified poll.
func (r *VoteRepository) UpdateVotesUserID(pollID int64, oldUserID, newUserID int64, tgUsername string) error {
	var err error
	if tgUsername != "" {
		// Normalize username for consistency with other storage methods
		normalizedUsername := poll.NormalizeUsername(tgUsername)
		// Update user ID and set username (only if username is currently empty)
		_, err = r.db.db.Exec(`
			UPDATE votes
			SET tg_user_id = ?, tg_username = COALESCE(NULLIF(tg_username, ''), ?)
			WHERE poll_id = ? AND tg_user_id = ?
		`, newUserID, normalizedUsername, pollID, oldUserID)
	} else {
		_, err = r.db.db.Exec(`
			UPDATE votes SET tg_user_id = ? WHERE poll_id = ? AND tg_user_id = ?
		`, newUserID, pollID, oldUserID)
	}
	if err != nil {
		return fmt.Errorf("update votes user id: %w", err)
	}
	return nil
}
