ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS "collections"
(
    "id"           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "title"        VARCHAR(255) NOT NULL,
    "owner_id"     INTEGER,
    "is_admin"     BOOLEAN     NOT NULL DEFAULT false,
    "is_published" BOOLEAN     NOT NULL DEFAULT true,
    "invite_token" VARCHAR(64) UNIQUE,
    "created_at"   TIMESTAMP            DEFAULT CURRENT_TIMESTAMP,
    "updated_at"   TIMESTAMP            DEFAULT CURRENT_TIMESTAMP,
    "deleted_at"   TIMESTAMP,

    CONSTRAINT "fk_collections_owner_id" FOREIGN KEY ("owner_id") REFERENCES "users" ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "index_collections_owner_id" ON "collections" ("owner_id");
CREATE INDEX IF NOT EXISTS "index_collections_is_admin" ON "collections" ("is_admin");
CREATE INDEX IF NOT EXISTS "index_collections_is_published" ON "collections" ("is_published");
CREATE INDEX IF NOT EXISTS "index_collections_deleted_at" ON "collections" ("deleted_at");
CREATE INDEX IF NOT EXISTS "index_collections_invite_token" ON "collections" ("invite_token");

CREATE TABLE IF NOT EXISTS "collection_translations"
(
    "collection_id"  UUID NOT NULL,
    "translation_id" UUID NOT NULL,
    "created_at"     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("collection_id", "translation_id"),
    CONSTRAINT "fk_collection_translations_collection_id" FOREIGN KEY ("collection_id") REFERENCES "collections" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_collection_translations_translation_id" FOREIGN KEY ("translation_id") REFERENCES "translations" ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "index_collection_translations_collection_id" ON "collection_translations" ("collection_id");
CREATE INDEX IF NOT EXISTS "index_collection_translations_translation_id" ON "collection_translations" ("translation_id");

CREATE TABLE IF NOT EXISTS "collection_members"
(
    "collection_id" UUID    NOT NULL,
    "user_id"       INTEGER NOT NULL,
    "created_at"    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("collection_id", "user_id"),
    CONSTRAINT "fk_collection_members_collection_id" FOREIGN KEY ("collection_id") REFERENCES "collections" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_collection_members_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "index_collection_members_user_id" ON "collection_members" ("user_id");
