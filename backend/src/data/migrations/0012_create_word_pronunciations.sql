CREATE TABLE IF NOT EXISTS "word_pronunciations"
(
    "id"               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "word_id"          UUID      NOT NULL,
    "model"            TEXT      NOT NULL,
    "voice"            TEXT      NOT NULL,
    "audio"            BYTEA     NOT NULL,
    "mime_type"        TEXT      NOT NULL DEFAULT 'audio/mpeg',
    "telegram_file_id" TEXT,
    "created_at"       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "fk_word_pronunciations_word_id" FOREIGN KEY ("word_id") REFERENCES "words" ("id") ON DELETE CASCADE,
    CONSTRAINT "uq_word_pronunciations_word_model_voice" UNIQUE ("word_id", "model", "voice")
);

CREATE INDEX IF NOT EXISTS "index_word_pronunciations_word_id" ON "word_pronunciations" ("word_id");
