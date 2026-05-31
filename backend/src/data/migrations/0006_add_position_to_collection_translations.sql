ALTER TABLE "collection_translations" ADD COLUMN IF NOT EXISTS "position" INTEGER NOT NULL DEFAULT 0;

-- Backfill position from the previous created_at ordering (oldest first), so the existing
-- display order is preserved as the initial manual order.
WITH ordered AS (
    SELECT "collection_id",
           "translation_id",
           ROW_NUMBER() OVER (
               PARTITION BY "collection_id"
               ORDER BY "created_at" ASC, "translation_id" ASC
           ) - 1 AS rn
    FROM "collection_translations"
)
UPDATE "collection_translations" ct
SET "position" = ordered.rn
FROM ordered
WHERE ct."collection_id" = ordered."collection_id"
  AND ct."translation_id" = ordered."translation_id";

ALTER TABLE "collection_translations" DROP COLUMN IF EXISTS "created_at";

CREATE INDEX IF NOT EXISTS "index_collection_translations_collection_id_position"
    ON "collection_translations" ("collection_id", "position");
