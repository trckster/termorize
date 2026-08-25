ALTER TABLE "users"
    ADD COLUMN IF NOT EXISTS "guest_expires_at" TIMESTAMP,
    ADD COLUMN IF NOT EXISTS "deleted_at" TIMESTAMP;

CREATE INDEX IF NOT EXISTS "index_users_guest_expires_at"
    ON "users" ("guest_expires_at")
    WHERE "guest_expires_at" IS NOT NULL AND "deleted_at" IS NULL;

CREATE INDEX IF NOT EXISTS "index_users_deleted_at"
    ON "users" ("deleted_at");
