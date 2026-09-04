CREATE TABLE IF NOT EXISTS "exercise_language_suggestion_counts"
(
    "user_id"     INTEGER     NOT NULL,
    "family"      VARCHAR(20) NOT NULL,
    "language"    VARCHAR(10) NOT NULL,
    "shown_count" SMALLINT    NOT NULL DEFAULT 0,
    "created_at"  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "fk_exercise_language_suggestion_counts_user_id"
        FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE,
    CONSTRAINT "ck_exercise_language_suggestion_counts_family"
        CHECK ("family" IN ('audio', 'description')),
    CONSTRAINT "ck_exercise_language_suggestion_counts_shown_count"
        CHECK ("shown_count" BETWEEN 0 AND 5),
    CONSTRAINT "pk_exercise_language_suggestion_counts"
        PRIMARY KEY ("user_id", "family", "language")
);
