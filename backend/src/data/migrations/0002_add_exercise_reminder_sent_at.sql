ALTER TABLE exercises
    ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP;