ALTER TABLE exercises
ADD COLUMN IF NOT EXISTS match_state JSONB;

CREATE INDEX IF NOT EXISTS index_exercises_pending_match
ON exercises (scheduled_for)
WHERE type = 'match/pairs' AND status = 'pending';
