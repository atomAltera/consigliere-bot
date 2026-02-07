package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"nuclight.org/consigliere/internal/poll"
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
// Username is normalized to lowercase before storing.
// Gender should be "male", "female", or empty string for not set.
func (r *NicknameRepository) Create(tgUserID *int64, tgUsername *string, gameNick string, gender string) (bool, error) {
	// Normalize username for storage
	var normalizedUsername *string
	if tgUsername != nil {
		normalized := poll.NormalizeUsername(*tgUsername)
		normalizedUsername = &normalized
	}

	// Normalize gender for storage
	var genderVal *string
	if gender != "" {
		genderVal = &gender
	}

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
		INSERT INTO nicknames (tg_user_id, tg_username, game_nick, gender, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, tgUserID, normalizedUsername, gameNick, genderVal, time.Now())
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
// Username is normalized to lowercase for lookup.
func (r *NicknameRepository) FindByTgUsername(username string) (gameNick string, tgUserID *int64, err error) {
	var userID sql.NullInt64
	err = r.db.db.QueryRow(`
		SELECT game_nick, tg_user_id
		FROM nicknames
		WHERE tg_username = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, poll.NormalizeUsername(username)).Scan(&gameNick, &userID)
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
// Username is normalized to lowercase for lookup.
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

	// Fall back to username (FindByTgUsername normalizes internally)
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
// Username is normalized to lowercase for lookup.
func (r *NicknameRepository) UpdateUserIDByUsername(username string, userID int64) error {
	_, err := r.db.db.Exec(`
		UPDATE nicknames
		SET tg_user_id = ?
		WHERE tg_username = ? AND tg_user_id IS NULL
	`, userID, poll.NormalizeUsername(username))
	if err != nil {
		return fmt.Errorf("update user id by username: %w", err)
	}
	return nil
}

// UpdateUserData updates nickname records with user data when we learn new information.
// Updates tg_user_id where tg_username matches (and user_id is missing),
// and updates tg_username where tg_user_id matches (and username is missing).
// This ensures data consistency when we learn user info from voting or other sources.
func (r *NicknameRepository) UpdateUserData(userID int64, username string) error {
	if userID <= 0 && username == "" {
		return nil // Nothing to update
	}

	normalizedUsername := poll.NormalizeUsername(username)

	// Update user ID where username matches (and user_id is missing)
	if userID > 0 && normalizedUsername != "" {
		_, err := r.db.db.Exec(`
			UPDATE nicknames
			SET tg_user_id = ?
			WHERE tg_username = ? AND tg_user_id IS NULL
		`, userID, normalizedUsername)
		if err != nil {
			return fmt.Errorf("update user id by username: %w", err)
		}
	}

	// Update username where user_id matches (and username is missing)
	if userID > 0 && normalizedUsername != "" {
		_, err := r.db.db.Exec(`
			UPDATE nicknames
			SET tg_username = ?
			WHERE tg_user_id = ? AND (tg_username IS NULL OR tg_username = '')
		`, normalizedUsername, userID)
		if err != nil {
			return fmt.Errorf("update username by user id: %w", err)
		}
	}

	return nil
}

// GetAllGameNicksForUser returns all game nicks associated with a user (by ID or username).
// Username is normalized to lowercase for lookup.
func (r *NicknameRepository) GetAllGameNicksForUser(userID int64, username string) ([]string, error) {
	rows, err := r.db.db.Query(`
		SELECT DISTINCT game_nick FROM nicknames
		WHERE tg_user_id = ? OR tg_username = ?
	`, userID, poll.NormalizeUsername(username))
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

// GetDisplayNicksBatch returns game nicknames for multiple users in a single query.
// Returns a map from user ID or username to NicknameInfo (nick + gender).
// For users with both ID and username, the result is keyed by user ID.
// Usernames are normalized to lowercase for lookup and in the returned map.
func (r *NicknameRepository) GetDisplayNicksBatch(keys []poll.NicknameLookupKey) (map[int64]poll.NicknameInfo, map[string]poll.NicknameInfo, error) {
	if len(keys) == 0 {
		return make(map[int64]poll.NicknameInfo), make(map[string]poll.NicknameInfo), nil
	}

	// Collect unique user IDs and usernames (normalized)
	userIDs := make([]int64, 0)
	usernames := make([]string, 0)
	seenIDs := make(map[int64]bool)
	seenUsernames := make(map[string]bool)

	for _, k := range keys {
		if k.UserID > 0 && !seenIDs[k.UserID] {
			userIDs = append(userIDs, k.UserID)
			seenIDs[k.UserID] = true
		}
		if k.Username != "" {
			normalized := poll.NormalizeUsername(k.Username)
			if !seenUsernames[normalized] {
				usernames = append(usernames, normalized)
				seenUsernames[normalized] = true
			}
		}
	}

	byUserID := make(map[int64]poll.NicknameInfo)
	byUsername := make(map[string]poll.NicknameInfo)

	// Query by user IDs if any
	if len(userIDs) > 0 {
		// Build placeholders
		placeholders := make([]string, len(userIDs))
		args := make([]any, len(userIDs))
		for i, id := range userIDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT tg_user_id, game_nick, gender
			FROM nicknames
			WHERE tg_user_id IN (%s)
			ORDER BY created_at DESC
		`, strings.Join(placeholders, ","))

		rows, err := r.db.db.Query(query, args...)
		if err != nil {
			return nil, nil, fmt.Errorf("query nicknames by user ids: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var userID int64
			var nick string
			var gender sql.NullString
			if err := rows.Scan(&userID, &nick, &gender); err != nil {
				return nil, nil, fmt.Errorf("scan nickname by user id: %w", err)
			}
			// Only store first (most recent) nick per user
			if _, exists := byUserID[userID]; !exists {
				byUserID[userID] = poll.NicknameInfo{Nick: nick, Gender: gender.String}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
	}

	// Query by usernames if any
	if len(usernames) > 0 {
		placeholders := make([]string, len(usernames))
		args := make([]any, len(usernames))
		for i, u := range usernames {
			placeholders[i] = "?"
			args[i] = u
		}

		query := fmt.Sprintf(`
			SELECT tg_username, game_nick, gender
			FROM nicknames
			WHERE tg_username IN (%s)
			ORDER BY created_at DESC
		`, strings.Join(placeholders, ","))

		rows, err := r.db.db.Query(query, args...)
		if err != nil {
			return nil, nil, fmt.Errorf("query nicknames by usernames: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var username string
			var nick string
			var gender sql.NullString
			if err := rows.Scan(&username, &nick, &gender); err != nil {
				return nil, nil, fmt.Errorf("scan nickname by username: %w", err)
			}
			// Only store first (most recent) nick per username
			if _, exists := byUsername[username]; !exists {
				byUsername[username] = poll.NicknameInfo{Nick: nick, Gender: gender.String}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
	}

	return byUserID, byUsername, nil
}
