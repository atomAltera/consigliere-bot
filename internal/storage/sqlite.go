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
