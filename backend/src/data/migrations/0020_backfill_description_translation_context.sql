-- Infer context only when all saved translations agree on one counterpart.
-- UNION deduplicates identical pairs across users, sources, and directions.
WITH counterparts AS (
    SELECT original_id AS word_id, translation_id AS translation_word_id FROM translations
    UNION
    SELECT translation_id AS word_id, original_id AS translation_word_id FROM translations
), unambiguous AS (
    SELECT word_id, (array_agg(translation_word_id))[1] AS translation_word_id
    FROM counterparts
    GROUP BY word_id
    HAVING count(*) = 1
)
UPDATE word_descriptions AS descriptions
SET translation_word_id = unambiguous.translation_word_id
FROM unambiguous
WHERE descriptions.word_id = unambiguous.word_id
    AND descriptions.translation_word_id IS NULL
    -- Preserve an already contextualized entry rather than overwrite it or
    -- violate the unique constraint when upgrading an existing installation.
    AND NOT EXISTS (
        SELECT 1 FROM word_descriptions AS contextual
        WHERE contextual.word_id = descriptions.word_id
            AND contextual.translation_word_id = unambiguous.translation_word_id
            AND contextual.model = descriptions.model
    );
