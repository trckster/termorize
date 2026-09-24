-- A small starter rotation keeps every supported language available before a
-- dictionary import. Imported idioms extend this same pool. Tests compare the
-- actual pool languages with enums.AllLanguageValues(), the application registry.
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
), existing AS (
    UPDATE words w SET type = 'idiom'
    FROM starters s WHERE w.language = s.language AND LOWER(w.word) = LOWER(s.word)
    RETURNING w.id
)
INSERT INTO words (word, language, type)
SELECT s.word, s.language, 'idiom' FROM starters s
WHERE NOT EXISTS (
    SELECT 1 FROM words w WHERE w.language = s.language AND LOWER(w.word) = LOWER(s.word)
);
