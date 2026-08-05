CREATE TABLE IF NOT EXISTS "openrouter_usages"
(
    "id"                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id"           INTEGER        NOT NULL,
    "generation_id"     TEXT,
    "model"             TEXT           NOT NULL,
    "cost"              NUMERIC(20, 10) NOT NULL CHECK ("cost" >= 0),
    "prompt_tokens"     INTEGER        NOT NULL DEFAULT 0 CHECK ("prompt_tokens" >= 0),
    "completion_tokens" INTEGER        NOT NULL DEFAULT 0 CHECK ("completion_tokens" >= 0),
    "total_tokens"      INTEGER        NOT NULL DEFAULT 0 CHECK ("total_tokens" >= 0),
    "created_at"        TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "fk_openrouter_usages_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE,
    CONSTRAINT "uq_openrouter_usages_generation_id" UNIQUE ("generation_id")
);

CREATE INDEX IF NOT EXISTS "index_openrouter_usages_user_id" ON "openrouter_usages" ("user_id");
CREATE INDEX IF NOT EXISTS "index_openrouter_usages_created_at" ON "openrouter_usages" ("created_at");
