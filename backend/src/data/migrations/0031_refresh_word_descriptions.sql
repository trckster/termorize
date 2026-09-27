-- Earlier clues had no translation context. The 0020/0021 backfill attached a
-- counterpart without regenerating them, so they may describe another sense.
WITH retired AS (
    DELETE FROM word_descriptions
    WHERE translation_word_id IS NOT NULL
        AND created_at < (
            SELECT applied_at FROM migrations
            WHERE name = '0019_add_description_translation_context'
        )
    RETURNING id, word_id, translation_word_id, model, description, created_at
)
INSERT INTO word_description_backfill_archive
    (id, word_id, translation_word_id, model, description, created_at, reason)
SELECT id, word_id, translation_word_id, model, description, created_at,
    'created_before_translation_context'
FROM retired;

DELETE FROM word_descriptions AS descriptions
USING words AS original, words AS translated
WHERE descriptions.word_id = original.id
    AND descriptions.translation_word_id = translated.id
    AND original.language = 'en'
    AND LOWER(original.word) = 'run down'
    AND translated.language = 'ru'
    AND LOWER(translated.word) = 'наезжать'
    AND LOWER(TRIM(descriptions.description)) =
        'to be in a poor or neglected state, often due to lack of maintenance.';
