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

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
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
		tg_invitation_message_id INTEGER,
		tg_cancel_message_id INTEGER,
		tg_done_message_id INTEGER,
		event_date DATE NOT NULL,
		options TEXT NOT NULL DEFAULT '0,1,2,3,4',
		is_active INTEGER NOT NULL DEFAULT 1,
		is_pinned INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		poll_id INTEGER NOT NULL REFERENCES polls(id),
		tg_user_id INTEGER NOT NULL,
		tg_username TEXT,
		tg_first_name TEXT NOT NULL,
		tg_option_index INTEGER NOT NULL,
		is_manual INTEGER NOT NULL DEFAULT 0,
		voted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON votes(poll_id);
	CREATE INDEX IF NOT EXISTS idx_votes_user_latest ON votes(poll_id, tg_user_id, voted_at DESC);
	CREATE INDEX IF NOT EXISTS idx_votes_tg_username ON votes(tg_username);

	CREATE TABLE IF NOT EXISTS nicknames (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tg_user_id INTEGER,
		tg_username TEXT,
		game_nick TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_nicknames_tg_user_id ON nicknames(tg_user_id);
	CREATE INDEX IF NOT EXISTS idx_nicknames_tg_username ON nicknames(tg_username);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_nicknames_game_nick_unique ON nicknames(game_nick);
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
