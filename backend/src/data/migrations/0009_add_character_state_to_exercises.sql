ALTER TABLE exercises
ADD COLUMN IF NOT EXISTS character_state JSONB;

CREATE INDEX IF NOT EXISTS index_exercises_pending_characters
ON exercises (scheduled_for)
WHERE type IN ('characters/direct', 'characters/reversed') AND status = 'pending';
