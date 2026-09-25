CREATE TABLE daily_idiom_deliveries (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    daily_idiom_id UUID NOT NULL REFERENCES daily_idioms(id),
    sent_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, date)
);
