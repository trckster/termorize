CREATE TABLE IF NOT EXISTS "word_descriptions"
(
    "id"          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "word_id"     UUID      NOT NULL,
    "model"       TEXT      NOT NULL,
    "description" TEXT      NOT NULL,
    "created_at"  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "fk_word_descriptions_word_id" FOREIGN KEY ("word_id") REFERENCES "words" ("id") ON DELETE CASCADE,
    CONSTRAINT "uq_word_descriptions_word_model" UNIQUE ("word_id", "model")
);

CREATE INDEX IF NOT EXISTS "index_word_descriptions_word_id" ON "word_descriptions" ("word_id");
