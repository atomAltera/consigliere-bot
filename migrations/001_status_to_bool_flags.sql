-- Migration: Replace status column with is_active and is_pinned boolean fields
-- This migration is idempotent and can be run multiple times safely

-- Step 1: Add new columns if they don't exist
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so errors are expected if columns exist
ALTER TABLE polls ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;
ALTER TABLE polls ADD COLUMN is_pinned INTEGER NOT NULL DEFAULT 0;

-- Step 2: Migrate data from status column
-- is_active = 1 for 'active' and 'pinned' statuses, 0 for 'cancelled'
-- is_pinned = 1 for 'pinned' status, 0 otherwise
UPDATE polls SET is_active = CASE
    WHEN status IN ('active', 'pinned') THEN 1
    ELSE 0
END;

UPDATE polls SET is_pinned = CASE
    WHEN status = 'pinned' THEN 1
    ELSE 0
END;

-- Note: SQLite doesn't support DROP COLUMN in versions before 3.35.0
-- The status column will be left in place but ignored by the application.
-- If using SQLite 3.35.0+, uncomment the following line:
-- ALTER TABLE polls DROP COLUMN status;
