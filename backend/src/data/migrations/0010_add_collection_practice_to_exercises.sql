ALTER TABLE "exercises"
    ADD COLUMN IF NOT EXISTS "practice_collection_id" UUID,
    ADD COLUMN IF NOT EXISTS "practice_collection_title" VARCHAR(255);

ALTER TABLE "exercises"
    ADD CONSTRAINT "fk_exercises_practice_collection_id"
    FOREIGN KEY ("practice_collection_id") REFERENCES "collections" ("id") ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS "index_exercises_practice_collection_id"
    ON "exercises" ("practice_collection_id");
