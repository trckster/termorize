-- Keep legacy descriptions for admin review, but do not reuse them for a
-- specific translation: their generating requests did not identify a meaning.
ALTER TABLE word_descriptions
    ADD COLUMN translation_word_id UUID REFERENCES words (id) ON DELETE CASCADE,
    DROP CONSTRAINT uq_word_descriptions_word_model,
    ADD CONSTRAINT uq_word_descriptions_word_translation_model
        UNIQUE NULLS NOT DISTINCT (word_id, translation_word_id, model);
