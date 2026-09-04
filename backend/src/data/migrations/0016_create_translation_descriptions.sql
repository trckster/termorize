CREATE TABLE IF NOT EXISTS "translation_descriptions"
(
    "id"             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "translation_id" UUID      NOT NULL,
    "model"          TEXT      NOT NULL,
    "description"    TEXT      NOT NULL,
    "created_at"     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "fk_translation_descriptions_translation_id" FOREIGN KEY ("translation_id") REFERENCES "translations" ("id") ON DELETE CASCADE,
    CONSTRAINT "uq_translation_descriptions_translation_model" UNIQUE ("translation_id", "model")
);

CREATE INDEX IF NOT EXISTS "index_translation_descriptions_translation_id" ON "translation_descriptions" ("translation_id");
