CREATE TYPE part_of_speech AS ENUM (
    'noun', 'verb', 'adjective', 'adverb', 'pronoun', 'preposition',
    'conjunction', 'numeral', 'interjection', 'phrase', 'unknown'
);

ALTER TABLE words ADD COLUMN part_of_speech part_of_speech DEFAULT NULL;
