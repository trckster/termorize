ALTER TABLE word_descriptions
    DROP COLUMN approved_at,
    DROP CONSTRAINT uq_word_descriptions_word_translation_model;
