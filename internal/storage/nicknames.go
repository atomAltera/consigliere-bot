package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type NicknameRepository struct {
	db *DB
}

func NewNicknameRepository(db *DB) *NicknameRepository {
	return &NicknameRepository{db: db}
}

// isUniqueConstraintError checks if the error is a SQLite unique constraint violation.
func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// Create inserts a new nickname record if game_nick is not already taken.
// Returns true if inserted, false if game_nick is already used by another player.
func (r *NicknameRepository) Create(tgUserID *int64, tgUsername *string, gameNick string) (bool, error) {
	// Check if game_nick is already taken (globally unique)
	var exists bool
	err := r.db.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM nicknames WHERE game_nick = ?)
	`, gameNick).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check duplicate: %w", err)
	}
	if exists {
		return false, nil
	}

	_, err = r.db.db.Exec(`
		INSERT INTO nicknames (tg_user_id, tg_username, game_nick, created_at)
		VALUES (?, ?, ?, ?)
	`, tgUserID, tgUsername, gameNick, time.Now())
	if err != nil {
		// Handle unique constraint violation (race condition)
		if isUniqueConstraintError(err) {
			return false, nil
		}
		return false, fmt.Errorf("insert nickname: %w", err)
	}
	return true, nil
}

// FindByGameNick returns the telegram identity for a game nickname.
// Returns the most recently added match if multiple exist.
func (r *NicknameRepository) FindByGameNick(gameNick string) (tgUserID *int64, tgUsername *string, err error) {
	var userID sql.NullInt64
	var username sql.NullString
	err = r.db.db.QueryRow(`
		SELECT tg_user_id, tg_username
		FROM nicknames
		WHERE game_nick = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, gameNick).Scan(&userID, &username)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query by game nick: %w", err)
	}
	if userID.Valid {
		tgUserID = &userID.Int64
	}
	if username.Valid {
		tgUsername = &username.String
	}
	return tgUserID, tgUsername, nil
}

// FindByTgUsername returns the most recent game nickname and user ID for a telegram username.
func (r *NicknameRepository) FindByTgUsername(username string) (gameNick string, tgUserID *int64, err error) {
	var userID sql.NullInt64
	err = r.db.db.QueryRow(`
		SELECT game_nick, tg_user_id
		FROM nicknames
		WHERE tg_username = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, username).Scan(&gameNick, &userID)
	if err == sql.ErrNoRows {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("query by username: %w", err)
	}
	if userID.Valid {
		tgUserID = &userID.Int64
	}
	return gameNick, tgUserID, nil
}

// FindByTgUserID returns the most recent game nickname for a telegram user ID.
func (r *NicknameRepository) FindByTgUserID(userID int64) (gameNick string, err error) {
	err = r.db.db.QueryRow(`
		SELECT game_nick
		FROM nicknames
		WHERE tg_user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&gameNick)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query by user id: %w", err)
	}
	return gameNick, nil
}

// GetDisplayNick returns the game nickname for display, given a user ID or username.
// Checks by user ID first, then by username.
func (r *NicknameRepository) GetDisplayNick(userID int64, username string) (string, error) {
	// Try by user ID first (more reliable)
	if userID > 0 {
		nick, err := r.FindByTgUserID(userID)
		if err != nil {
			return "", err
		}
		if nick != "" {
			return nick, nil
		}
	}

	// Fall back to username
	if username != "" {
		nick, _, err := r.FindByTgUsername(username)
		if err != nil {
			return "", err
		}
		if nick != "" {
			return nick, nil
		}
	}

	return "", nil
}

// UpdateUserIDByUsername sets tg_user_id for all records matching tg_username where tg_user_id is NULL.
func (r *NicknameRepository) UpdateUserIDByUsername(username string, userID int64) error {
	_, err := r.db.db.Exec(`
		UPDATE nicknames
		SET tg_user_id = ?
		WHERE tg_username = ? AND tg_user_id IS NULL
	`, userID, username)
	if err != nil {
		return fmt.Errorf("update user id by username: %w", err)
	}
	return nil
}

// GetAllGameNicksForUser returns all game nicks associated with a user (by ID or username).
func (r *NicknameRepository) GetAllGameNicksForUser(userID int64, username string) ([]string, error) {
	rows, err := r.db.db.Query(`
		SELECT DISTINCT game_nick FROM nicknames
		WHERE tg_user_id = ? OR tg_username = ?
	`, userID, username)
	if err != nil {
		return nil, fmt.Errorf("query game nicks: %w", err)
	}
	defer rows.Close()

	var nicks []string
	for rows.Next() {
		var nick string
		if err := rows.Scan(&nick); err != nil {
			return nil, fmt.Errorf("scan game nick: %w", err)
		}
		nicks = append(nicks, nick)
	}
	return nicks, rows.Err()
}
