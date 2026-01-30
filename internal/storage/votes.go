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
		INSERT INTO votes (poll_id, tg_user_id, tg_username, tg_first_name, tg_option_index, is_manual, voted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, v.PollID, v.TgUserID, v.TgUsername, v.TgFirstName, v.TgOptionIndex, v.IsManual, time.Now())
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
