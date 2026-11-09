ALTER TABLE vocabulary ADD COLUMN share_token TEXT;
CREATE UNIQUE INDEX vocabulary_share_token_idx ON vocabulary (share_token) WHERE share_token IS NOT NULL;
