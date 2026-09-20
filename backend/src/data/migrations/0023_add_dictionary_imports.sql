CREATE TYPE word_type AS ENUM ('unknown', 'idiom');
ALTER TABLE words ADD COLUMN type word_type NOT NULL DEFAULT 'unknown';
CREATE INDEX words_language_lower_word_idx ON words (language, LOWER(word));

CREATE TABLE dictionaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    edition TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    license TEXT NOT NULL,
    attribution TEXT NOT NULL,
    download_url TEXT NOT NULL
);

INSERT INTO dictionaries (name, edition, url, license, attribution, download_url) VALUES
('English Wiktionary', 'enwiktionary', 'https://en.wiktionary.org/',
 'Wiktionary reuse terms: https://en.wiktionary.org/wiki/Wiktionary:Copyrights',
 'Wiktionary contributors; extracted by Tatu Ylonen and contributors using Wiktextract, via https://kaikki.org/dictionary/rawdata.html',
 'https://kaikki.org/dictionary/raw-wiktextract-data.jsonl.gz'),
('Russian Wiktionary', 'ruwiktionary', 'https://ru.wiktionary.org/',
 'Wiktionary reuse terms: https://ru.wiktionary.org/wiki/Викисловарь:Авторские_права',
 'Wiktionary contributors; extracted by Tatu Ylonen and contributors using Wiktextract, via https://kaikki.org/dictionary/rawdata.html',
 'https://kaikki.org/dictionary/downloads/ru/ru-extract.jsonl.gz'),
('Italian Wiktionary', 'itwiktionary', 'https://it.wiktionary.org/',
 'Wiktionary reuse terms: https://it.wiktionary.org/wiki/Wikizionario:Copyright',
 'Wiktionary contributors; extracted by Tatu Ylonen and contributors using Wiktextract, via https://kaikki.org/dictionary/rawdata.html',
 'https://kaikki.org/dictionary/downloads/it/it-extract.jsonl.gz');

CREATE TABLE dictionary_import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_id UUID NOT NULL REFERENCES dictionaries(id),
    source_name TEXT NOT NULL,
    edition TEXT NOT NULL,
    download_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'downloading', 'importing', 'succeeded', 'failed', 'interrupted')),
    processed BIGINT NOT NULL DEFAULT 0,
    inserted BIGINT NOT NULL DEFAULT 0,
    classified BIGINT NOT NULL DEFAULT 0,
    skipped BIGINT NOT NULL DEFAULT 0,
    failed BIGINT NOT NULL DEFAULT 0,
    downloaded_bytes BIGINT NOT NULL DEFAULT 0,
    total_bytes BIGINT,
    error TEXT NOT NULL DEFAULT '',
    record_errors JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX dictionary_import_jobs_active_source_idx ON dictionary_import_jobs (dictionary_id)
    WHERE status IN ('queued', 'downloading', 'importing');
CREATE INDEX dictionary_import_jobs_history_idx ON dictionary_import_jobs (dictionary_id, created_at DESC);
