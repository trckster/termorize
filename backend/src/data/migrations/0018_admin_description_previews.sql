ALTER TABLE word_descriptions ADD COLUMN approved_at TIMESTAMP;

CREATE TABLE word_description_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    description_id UUID NOT NULL REFERENCES word_descriptions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    description TEXT NOT NULL,
    original_description TEXT NOT NULL,
    original_model TEXT NOT NULL,
    original_created_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX index_word_description_previews_created_at ON word_description_previews(created_at);
