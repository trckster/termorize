ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS id UUID DEFAULT gen_random_uuid();

UPDATE vocabulary_exercises
SET id = gen_random_uuid()
WHERE id IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'vocabulary_exercises'::regclass
            AND contype = 'p'
    ) THEN
        ALTER TABLE vocabulary_exercises
        ADD CONSTRAINT vocabulary_exercises_pkey PRIMARY KEY (id);
    END IF;
END $$;

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS result VARCHAR(20);

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS result_reason VARCHAR(30);

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS progress_delta INTEGER;

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS knowledge_after INTEGER;

ALTER TABLE vocabulary_exercises
ADD COLUMN IF NOT EXISTS answered_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS index_vocabulary_exercises_exercise_id_position
ON vocabulary_exercises (exercise_id, position);

CREATE INDEX IF NOT EXISTS index_vocabulary_exercises_exercise_id_is_correct
ON vocabulary_exercises (exercise_id, is_correct);
