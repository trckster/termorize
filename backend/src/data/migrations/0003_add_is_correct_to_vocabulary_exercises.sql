ALTER TABLE vocabulary_exercises ADD COLUMN IF NOT EXISTS is_correct BOOLEAN NOT NULL DEFAULT false;

UPDATE vocabulary_exercises
SET is_correct = true
WHERE is_correct = false;
