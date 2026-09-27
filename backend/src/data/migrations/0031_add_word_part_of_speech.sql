BEGIN;

CREATE TYPE part_of_speech AS ENUM (
    'noun', 'verb', 'adjective', 'adverb', 'pronoun', 'preposition',
    'conjunction', 'numeral', 'interjection', 'phrase', 'unknown'
);

ALTER TABLE words ADD COLUMN part_of_speech part_of_speech DEFAULT NULL;
CREATE INDEX words_pending_classification_idx ON words (id) WHERE part_of_speech IS NULL;

CREATE FUNCTION protect_word_part_of_speech() RETURNS trigger AS $$
BEGIN
    IF OLD.part_of_speech IS NOT NULL AND OLD.part_of_speech <> 'unknown'
       AND NEW.part_of_speech IS DISTINCT FROM OLD.part_of_speech THEN
        RAISE EXCEPTION 'A concrete part of speech is permanent' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER words_permanent_part_of_speech
BEFORE UPDATE OF part_of_speech ON words
FOR EACH ROW EXECUTE FUNCTION protect_word_part_of_speech();

COMMIT;
