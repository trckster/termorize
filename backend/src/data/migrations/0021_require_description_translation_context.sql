-- Keep unresolved or superseded legacy clues recoverable outside the active cache.
CREATE TABLE IF NOT EXISTS word_description_backfill_archive (
    LIKE word_descriptions INCLUDING DEFAULTS,
    archived_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reason TEXT NOT NULL,
    PRIMARY KEY (id)
);

-- Search both directions; the earliest saved translation determines the context.
WITH counterparts AS (
    SELECT original_id AS word_id, translation_id AS translation_word_id, created_at, id FROM translations
    UNION ALL
    SELECT translation_id AS word_id, original_id AS translation_word_id, created_at, id FROM translations
), first_counterparts AS (
    SELECT DISTINCT ON (word_id) word_id, translation_word_id
    FROM counterparts
    ORDER BY word_id, created_at ASC, id ASC
)
UPDATE word_descriptions AS descriptions
SET translation_word_id = first_counterparts.translation_word_id
FROM first_counterparts
WHERE descriptions.word_id = first_counterparts.word_id
    AND descriptions.translation_word_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM word_descriptions AS contextual
        WHERE contextual.word_id = descriptions.word_id
            AND contextual.translation_word_id = first_counterparts.translation_word_id
            AND contextual.model = descriptions.model
    );

WITH archived AS (
    DELETE FROM word_descriptions
    WHERE translation_word_id IS NULL
    RETURNING *
)
INSERT INTO word_description_backfill_archive
    (id, word_id, translation_word_id, model, description, created_at, approved_at, reason)
SELECT id, word_id, translation_word_id, model, description, created_at, approved_at,
    CASE WHEN EXISTS (
        SELECT 1 FROM translations
        WHERE original_id = archived.word_id OR translation_id = archived.word_id
    ) THEN 'existing_context' ELSE 'no_translation' END
FROM archived;

ALTER TABLE word_descriptions ALTER COLUMN translation_word_id SET NOT NULL;
