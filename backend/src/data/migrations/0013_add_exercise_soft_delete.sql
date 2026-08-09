ALTER TABLE "exercises"
    ADD COLUMN IF NOT EXISTS "deleted_at" TIMESTAMP;

CREATE INDEX IF NOT EXISTS "index_exercises_active_status_scheduled_for"
    ON "exercises" ("status", "scheduled_for")
    WHERE "deleted_at" IS NULL;
