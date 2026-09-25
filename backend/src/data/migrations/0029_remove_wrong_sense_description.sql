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
