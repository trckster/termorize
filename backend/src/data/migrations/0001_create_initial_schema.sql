CREATE TABLE IF NOT EXISTS "users"
(
    "id"          SERIAL PRIMARY KEY,
    "username"    VARCHAR(255),
    "telegram_id" BIGINT UNIQUE,
    "name"        VARCHAR(255),
    "photo_url"   VARCHAR(255),
    "created_at"  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at"  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "words"
(
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "word"       TEXT        NOT NULL,
    "language"   VARCHAR(10) NOT NULL,
    "created_at" TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    UNIQUE ("word", "language")
);

CREATE INDEX IF NOT EXISTS "index_words_word_language" ON "words" ("word", "language");

CREATE TABLE IF NOT EXISTS "translations"
(
    "id"         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "word_1_id"  UUID        NOT NULL,
    "word_2_id"  UUID        NOT NULL,
    "source"     VARCHAR(50) NOT NULL,
    "user_id"    INTEGER,
    "created_at" TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_translations_word_1_id" FOREIGN KEY ("word_1_id") REFERENCES "words" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_translations_word_2_id" FOREIGN KEY ("word_2_id") REFERENCES "words" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_translations_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "index_translations_word_1_id" ON "translations" ("word_1_id");
CREATE INDEX IF NOT EXISTS "index_translations_word_2_id" ON "translations" ("word_2_id");
CREATE INDEX IF NOT EXISTS "index_translations_user_id" ON "translations" ("user_id");

CREATE TABLE IF NOT EXISTS "vocabulary"
(
    "id"             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"        INTEGER NOT NULL,
    "translation_id" UUID    NOT NULL,
    "progress"       JSONB            DEFAULT '[]',
    "created_at"     TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    "mastered_at"    TIMESTAMP,
    CONSTRAINT "fk_vocabulary_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_vocabulary_translation_id" FOREIGN KEY ("translation_id") REFERENCES "translations" ("id") ON DELETE CASCADE,
    UNIQUE ("user_id", "translation_id")
);

CREATE INDEX IF NOT EXISTS "index_vocabulary_user_id" ON "vocabulary" ("user_id");
CREATE INDEX IF NOT EXISTS "index_vocabulary_translation_id" ON "vocabulary" ("translation_id");
CREATE INDEX IF NOT EXISTS "index_user_translation" ON "vocabulary" ("user_id", "translation_id");
