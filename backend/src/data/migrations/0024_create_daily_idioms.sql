CREATE TABLE daily_idioms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    language VARCHAR(10) NOT NULL,
    word_id UUID NOT NULL REFERENCES words(id),
    UNIQUE (date, language)
);

CREATE INDEX words_language_type_idx ON words (language, type);
CREATE INDEX daily_idioms_word_id_idx ON daily_idioms (word_id);
