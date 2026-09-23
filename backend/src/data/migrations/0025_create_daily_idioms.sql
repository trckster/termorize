-- This migration was previously named 0024_create_daily_idioms; allow it to run
-- under the new name on databases that already applied the old file.
CREATE TABLE IF NOT EXISTS daily_idioms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    language VARCHAR(10) NOT NULL,
    word_id UUID NOT NULL REFERENCES words(id),
    UNIQUE (date, language)
);

CREATE INDEX IF NOT EXISTS words_language_type_idx ON words (language, type);
CREATE INDEX IF NOT EXISTS daily_idioms_word_id_idx ON daily_idioms (word_id);
