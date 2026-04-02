ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_vocabulary_deleted_at ON vocabulary(deleted_at);
