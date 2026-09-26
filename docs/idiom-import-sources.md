# Per-language idiom downloads

Migration `0029_add_language_idiom_imports` adds seven independent sources to
Admin → Dictionaries. Each has the existing Import / Update action, background
progress, history, and retry behavior. No manual database edits or starter-only
fallback are needed. Existing English, Russian, and Italian sources remain.

Each new job snapshots `target_language`, so only entries in the selected language
are stored even when its upstream dump contains other languages. Imports use the
existing word normalization, case-insensitive lookup, and classification path.
Updates skip existing idioms and preserve Vocabulary entries.

## Measured availability — 2026-09-26

| Language | Unique imported idioms | Source |
| --- | ---: | --- |
| German | 2,736 | German Wiktionary via Kaikki |
| Spanish | 3,327 | English Wiktionary: Spanish idiom category |
| French | 5,187 | French Wiktionary via Kaikki |
| Polish | 610 | Polish Wiktionary via Kaikki |
| Portuguese | 991 | English Wiktionary: Portuguese idiom category |
| Turkish | 3,287 | Turkish Wiktionary via Kaikki |
| Ukrainian | 416 | Ukrainian Wiktionary: Ukrainian phraseology category |
| **Total** | **16,554** | |

These are actual database insert counts from the production `DictionaryWorker`,
using all seven sources in an isolated PostgreSQL database with no starter words
or other pre-existing entries. All seven jobs completed with zero failed records.
The four bulk downloads were cached locally for repeatability; API imports used
live source responses. Counts are unique after the application's normalization
and deduplication, not raw records, category page counts, or estimates.

The number newly added to an existing installation will be lower if entries
already exist. Upstream updates can change future counts. This is measured
coverage of the configured sources, not a claim to contain every idiom in a
language. Classification follows the source's explicit idiom labels/categories;
it is not a linguistic quality review of every entry.

## Sources and classification

The [Kaikki raw-data catalog](https://kaikki.org/dictionary/rawdata.html) supplies:

- [German extract](https://kaikki.org/dictionary/downloads/de/de-extract.jsonl.gz):
  `idiomatic` tags or the exact German `Redewendung (Deutsch)` category.
- [French extract](https://kaikki.org/dictionary/downloads/fr/fr-extract.jsonl.gz):
  `idiomatic` tags or French `Idiotismes … en français` categories.
- [Polish extract](https://kaikki.org/dictionary/downloads/pl/pl-extract.jsonl.gz):
  explicit `idiomatic` tags.
- [Turkish extract](https://kaikki.org/dictionary/downloads/tr/tr-extract.jsonl.gz):
  `idiomatic` tags or `Türkçe deyimler`.

These downloads were last modified on 2026-09-25. Spanish and Portuguese native
extracts did not preserve enough explicit idiom classifications for the requested
coverage. They deliberately use well-populated, language-specific categories in
English Wiktionary instead. Ukrainian has no Kaikki native-edition extract.

The [MediaWiki category API](https://www.mediawiki.org/wiki/API:Categorymembers)
provides these sources:

- [Spanish idioms](https://en.wiktionary.org/wiki/Category:Spanish_idioms)
- [Portuguese idioms](https://en.wiktionary.org/wiki/Category:Portuguese_idioms)
- [Ukrainian phraseology](https://uk.wiktionary.org/wiki/Категорія:Фразеологізми/uk)

The importer follows continuation tokens, accepts main-namespace entries only,
excludes redirects and entries carrying the configured proverb category, and
passes normalized records through the same validator and database importer as
bulk downloads. API failures, malformed responses, and repeated continuation
fail the job without importing an incomplete category download. Responses and
page counts are bounded. Ordinary phrase/POS labels alone are not enough for the
bulk classifier; known proverb/morpheme labels are excluded.

Source links, attribution, and reuse terms are stored alongside each source and
shown by the existing Source and attribution disclosure in admin.

## Reproduce on a disposable database

Run the backend with a new database and let migrations finish. For a count that
excludes the starter words, remove the starter words only in that disposable
database before any imports. Start each of the seven named imports from admin,
wait for completion, then inspect:

```sql
SELECT target_language, status, inserted, classified, skipped, failed
FROM dictionary_import_jobs
WHERE target_language <> ''
ORDER BY created_at;

SELECT language, COUNT(*)
FROM words WHERE type = 'idiom'
GROUP BY language ORDER BY language;
```

Bulk file SHA-256 values used for the measurement:

```text
de 126519c04f2c648c56c0cf577f8f1929a4a93dd498167ecf20904dffda54c69a
fr a180388de7e7497c3ad1b749e82b631263405ce2348586a915fcef19266217ca
pl 057734b402324a4166325b5d2f8381722af1566ff8433eff03e3f9c644ef0cad
tr 908a1652e39a45a20df165756670017a7358d3d450252bd9b6c59d45ae1c11d3
```
