-- 0028 classified these exact phrases without recording row provenance or the
-- previous type. Clear their seed-only classification, retaining words and every
-- vocabulary, translation, delivery, description and pronunciation relation.
-- A processed import may have confirmed even a skipped existing seed, or committed
-- batches before failure. Preserve ambiguous matches for any overlapping source;
-- the original three editions (and unknown editions) can contain any language.
-- Run under the importer/word-writer locks to avoid racing an active import.
SELECT pg_advisory_xact_lock(814760923);
SELECT pg_advisory_xact_lock(814760924);
WITH starters(language, word) AS (VALUES
    ('en', 'break the ice'), ('en', 'piece of cake'), ('en', 'under the weather'),
    ('ru', 'бить баклуши'), ('ru', 'спустя рукава'), ('ru', 'водить за нос'),
    ('it', 'rompere il ghiaccio'), ('it', 'essere al settimo cielo'), ('it', 'prendere due piccioni con una fava'),
    ('de', 'das Eis brechen'), ('de', 'Tomaten auf den Augen haben'), ('de', 'ins kalte Wasser springen'),
    ('es', 'romper el hielo'), ('es', 'estar en las nubes'), ('es', 'tirar la toalla'),
    ('fr', 'briser la glace'), ('fr', 'donner sa langue au chat'), ('fr', 'avoir le cafard'),
    ('pl', 'przełamać lody'), ('pl', 'bujać w obłokach'), ('pl', 'mieć muchy w nosie'),
    ('tr', 'ağzı kulaklarına varmak'), ('tr', 'etekleri zil çalmak'), ('tr', 'ipe un sermek'),
    ('pt', 'quebrar o gelo'), ('pt', 'estar nas nuvens'), ('pt', 'dar com a língua nos dentes'),
    ('uk', 'бити байдики'), ('uk', 'водити за ніс'), ('uk', 'пекти раків')
)
UPDATE words w SET type = 'unknown'
FROM starters s
WHERE w.language = s.language AND LOWER(w.word) = LOWER(s.word)
  AND w.type = 'idiom'
  AND NOT EXISTS (
    SELECT 1 FROM dictionary_import_jobs j
    WHERE j.processed > 0
      AND CASE j.edition
        WHEN 'dewiktionary' THEN 'de'
        WHEN 'enwiktionary-es' THEN 'es'
        WHEN 'frwiktionary' THEN 'fr'
        WHEN 'plwiktionary' THEN 'pl'
        WHEN 'trwiktionary' THEN 'tr'
        WHEN 'enwiktionary-pt' THEN 'pt'
        WHEN 'ukwiktionary' THEN 'uk'
        ELSE s.language
      END = s.language
  );
