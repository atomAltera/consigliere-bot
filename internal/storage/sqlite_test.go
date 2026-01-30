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
