# Dictionary imports and daily idioms

This document preserves the agreed plan so implementation can continue in a new
conversation without relying on earlier context. Part 1 is implemented; see
`docs/dictionary-imports.md` for source evidence, operations and verification. Part 2 records the agreed feature behavior and schema
direction and remains unimplemented.

## Scope and decisions

- Implement in two parts: dictionary imports/admin management first, daily idioms
  and the client experience second.
- Initially import English, Russian, and Italian idioms from Kaikki's Wiktionary
  extracts. Other supported app languages are German, Spanish, French, Polish,
  Turkish, Portuguese, and Ukrainian; importing those is outside the initial scope.
- Store only the idiom text and language in `words`, classified as `idiom`.
  Do not import complete dictionary JSON, dictionary definitions, examples,
  etymologies, or dictionary translations in this phase.
- Use a `type` enum on `words`, not an `is_idiom` boolean.
- Add a table named `dictionaries` for source metadata. Do not add
  `dictionary_entries`, `dictionaries_data`, or a `words.dictionary_data` JSONB
  column for this feature.
- Imports are started by an administrator through the frontend. They do not run
  automatically on application startup.
- **Do not cache or archive downloaded dictionary files.** Download into temporary
  storage for each import and clean up afterward. Retries may download again.
  Do not commit dictionary downloads to Git or include them in application images.
- Caching generated OpenRouter responses in PostgreSQL is a separate, agreed
  feature. The decision against file caching does not disable response caching.
- Daily choices are shared by calendar date and language, with no user ownership.
  Determine the calendar date using the requester's timezone, not a universal UTC
  cutoff. Users with the same local date and language see the same idiom.
- Reuse `word_descriptions` for generated idiom content. Do not create a separate
  `idiom_explanations` table.

## Current application baseline

Recheck these files when implementation starts, since the repository may evolve.

- `backend/src/data/migrations/0001_create_initial_schema.sql`:
  - `words`: `id UUID`, `word TEXT`, `language VARCHAR(10)`, `created_at TIMESTAMP`.
  - `words` has uniqueness on `(word, language)`.
  - `translations`: `id`, `original_id`, `translation_id`, `source`, `user_id`,
    `created_at`. Both expression IDs reference `words`.
- `backend/src/services/vocabulary.service.go`: `GetOrCreateWord` currently looks
  up words case-insensitively and uses the existing casing normalization. Align
  importer deduplication with app behavior; exact SQL uniqueness alone does not
  reproduce this lookup behavior.
- `backend/src/models/word_description.go` and migrations `0016`, `0018`, `0019`,
  `0021`: `word_descriptions` currently contains `id`, `word_id`,
  `translation_word_id`, `model`, `description`, `created_at`, `approved_at`.
  Both word references point to `words`, not to a `translations` row.
  `translation_word_id` is currently required by the database. Current uniqueness
  is `(word_id, translation_word_id, model)`.
- `backend/src/services/exercise.description.service.go`: existing descriptions
  are exercise clues generated with a translated counterpart as context.
  Approved descriptions can take precedence across models. Preserve that behavior
  for exercise clues when extending this table.
- `frontend/src/views/TranslationPage.vue`: proposed location for the daily block.
  `RootView.vue` currently directs signed-in users to the Translation page.
- Existing admin routing is under `/admin`; use its access-control conventions.

## Part 1 — Dictionary imports and admin management

### 1. Classify imported words

- [x] Add PostgreSQL enum `type` with `unknown` and `idiom`.
- [x] Add `words.type type NOT NULL DEFAULT 'unknown'`.
- [x] Update backend models/types as needed. Existing words and ordinary new saves
  remain `unknown` unless explicitly classified.
- [x] Import complete expressions, such as `бить баклуши`, as a single `words.word`.
- [x] When an imported idiom already exists, reuse the row and set `type = 'idiom'`.
  Preserve IDs, user vocabulary, progress, and existing translations.

### 2. Register dictionaries

- [x] Add `dictionaries` with the following source metadata:

| Column | Meaning |
| --- | --- |
| `id` | Unique dictionary identifier |
| `name` | Display name, such as English Wiktionary |
| `edition` | Source edition, such as `enwiktionary`, `ruwiktionary`, `itwiktionary` |
| `url` | Dictionary homepage |
| `license` | Applicable reuse license/terms |
| `attribution` | Credit text for use of the source |

- [x] Make the source download location available to the importer through trusted
  configuration or source metadata. Its exact storage location is not settled.
- [x] Keep dictionary edition distinct from entry language: an English Wiktionary
  extract contains entries in many languages, with definitions generally in English.
- [x] Record the source and outcome with each import job. Source metadata must not
  imply that complete dictionary content is stored in PostgreSQL.

### 3. Download and extract idioms

Starting sources (verify availability and schema before implementing adapters):

- [Kaikki raw downloads](https://kaikki.org/dictionary/rawdata.html)
- [English-edition JSONL gzip](https://kaikki.org/dictionary/raw-wiktextract-data.jsonl.gz)
- [Russian-edition JSONL gzip](https://kaikki.org/dictionary/downloads/ru/ru-extract.jsonl.gz)
- [Italian-edition JSONL gzip](https://kaikki.org/dictionary/downloads/it/it-extract.jsonl.gz)
- [Wiktextract documentation and edition schemas](https://github.com/tatuylonen/wiktextract)

- [x] Download the selected source into a temporary file for each job. Clean up on
  success and failure, and clean up abandoned temporary files after interrupted jobs.
- [x] Stream decompression and JSONL parsing instead of loading a full dump into RAM.
- [x] Use edition-specific extraction rules: Kaikki uses JSONL, but editions do not
  have identical schemas. Do not assume every optional field exists.
- [x] Restrict imported entry languages to the initial supported set (`en`, `ru`,
  `it`). Combining evidence from different editions is useful for coverage; do not
  treat each edition's entire multilingual contents as idioms in its own language.
- [x] Identify idioms using explicit source classification, including applicable
  categories and sense tags. Spaces in a phrase or `pos = 'phrase'` alone do not
  establish that it is an idiom. An idiom can also be classified as a verb.
- [x] Validate text and language, deduplicate across entries and editions, and insert
  in bounded batches. Repeating an import must not create duplicate words.
- [x] Skip unrelated entries and report malformed records. Do not create user
  vocabulary, daily selections, or translated words during import.
- [x] Do not mark an expression as an idiom merely because it is listed as a
  translation of one: dictionary translations may be single words or paraphrases.

Exact extraction rules need validation on real samples. Broader locutions,
collocations, and proverbs must not silently be treated as proven idioms merely to
increase counts. Earlier estimates of 15,000–25,000 idioms / a few MB of `words`
storage were preliminary, not measured import totals or acceptance requirements.

### 4. Persistent import jobs and admin UI

- [x] Starting an import creates a persistent background job and returns promptly.
  The job must continue when the administrator closes the browser.
- [x] Persist job state, source, progress, timestamps, results, and errors. Exact
  job-table columns and worker mechanism remain implementation choices.
- [x] Show meaningful stages/progress and counts: processed, inserted, existing
  words newly classified, skipped, and failed records. Do not invent a percentage
  when the total is unknown.
- [x] Prevent duplicate simultaneous imports of the same source; keep deduplication
  safe when different sources contain the same expression.
- [x] Handle worker interruptions with a recoverable/retryable job state rather
  than leaving jobs permanently marked as running. Retries may reprocess earlier
  batches and must be idempotent.
- [x] Add admin-only dictionary management: list configured sources, start
  Import/Update, inspect current and previous job results, and retry failures.
- [x] Enforce permissions in backend endpoints as well as the frontend.
- [x] Ordinary application startup and client visits never trigger imports.

### 5. Part 1 verification

- [x] Test real-format small fixtures for each source edition and missing fields.
- [x] Test duplicate records, overlapping sources, repeated imports, and promotion
  of existing words from `unknown` to `idiom`.
- [x] Test admin permissions, duplicate job starts, worker recovery, failure
  reporting, and temporary-file cleanup.
- [x] Verify existing vocabulary and translation behavior remains intact.

## Part 2 — Daily idioms and client experience

### 1. Shared daily selections

- [ ] Add `daily_idioms`:

| Column | Meaning |
| --- | --- |
| `id UUID` | Primary key |
| `date DATE` | Calendar date without a timezone |
| `language VARCHAR(10)` | Language of the selected idiom |
| `word_id UUID` | Foreign key to the selected idiom in `words` |

- [ ] Require uniqueness on `(date, language)`.
- [ ] **Do not make `word_id` unique.** Idioms recur according to usage balancing.
- [ ] Add indexes supporting eligible-word lookup and usage counting, initially
  `words(language, type)` and `daily_idioms(word_id)` unless existing indexes cover
  the chosen queries.
- [ ] No user reference belongs in this table.

### 2. Daily endpoint, dates, and selection

- [ ] Determine the requester's current local date using their timezone. Store only
  that date. Reuse the app's timezone conventions; define a fallback if unavailable.
- [ ] Return an existing selection for `(date, language)` unchanged.
- [ ] If absent, select and persist an idiom only in response to the request.
  There is no scheduled job, no backfilling, and no requirement for consecutive dates.
- [ ] Before selecting, filter `words` to the requested language and `type = 'idiom'`.
- [ ] Count each eligible idiom's appearances in `daily_idioms`, including zero.
- [ ] If all eligible counts are equal, select randomly from all eligible idioms.
- [ ] Otherwise, select randomly from idioms whose count is below that language's
  highest count. This is the user's exact rule; do not silently replace it with
  selection only from the minimum count.
- [ ] Calculate counts independently per language. English may be at usage count 2
  while Italian is at 5; one language never affects another's candidates.
- [ ] Persist the selection in a transaction with per-language serialization/locking.
  Recheck for an existing daily row after taking the lock. This protects both
  concurrent requests for the same date and different local dates in one language.
- [ ] If no idioms exist for the language, return an explicit empty result.
  Using every idiom once is not an exhaustion error: balanced reuse continues.

With a stable pool, the rule allows every idiom once before any is used twice,
then twice before any is used three times. With new imports, usage counts can be
uneven: the exact below-maximum rule still applies, including zero-use newcomers.
Counts reflect persisted daily assignments, not page views or individual users.

### 3. Reuse `word_descriptions` for generated content

- [ ] Add `purpose`: `exercise_clue` by default, or `idiom_explanation`.
- [ ] Add `language` for the language of the explanation. Backfill existing clues
  using their existing generation behavior, checking that against the code first.
- [ ] Keep the existing `description` column for the explanation text.
- [ ] Allow `translation_word_id` to be null for idiom content. Preserve a required
  counterpart for `exercise_clue` rows through an appropriate database constraint.
  Idiom content is keyed by the original word and output language; generating a
  translation does not require inserting another `words` row.
- [ ] Add nullable `translated_text` and `translation_kind` (`idiom` / `literal`).
  They are null for explanations in the original language and populated for target-
  language results. Preserve `model`, `created_at`, and `approved_at`.
- [ ] Replace/adjust uniqueness to keep separate rules for each purpose:
  - Exercise clues retain uniqueness by word, translated counterpart, and model.
  - Idiom content is unique by original word, explanation language, and model.
  Purpose-specific partial unique indexes are a possible implementation.
- [ ] Update all generation, cache, approval, and admin queries to filter by purpose.
  Exercise generation must never use an idiom explanation as an exercise clue.
- [ ] Preserve the existing approved-clue precedence and word-pair context behavior.
  Define approval handling for idiom content explicitly rather than accidentally
  inheriting a query that ignores output language.

This extends the existing table; do not add a separate explanation cache table.
The cache is shared across users, dates, and later appearances of the same idiom.

### 4. OpenRouter explanation and translation

- [ ] When the idiom is requested, reuse its original-language explanation if
  cached; otherwise ask OpenRouter to generate and persist it.
- [ ] Explanation requirements: very short (a few words), in the requested
  language, lowercase, no final period, multiple meanings separated by `;`.
- [ ] Generate target-language content only after the user chooses Translate.
- [ ] Ask for an idiomatic equivalent if one exists. Otherwise request a literal,
  word-for-word translation and mark it `literal` for display.
- [ ] Also request a short explanation of the **original idiom's meaning** in the
  target language. The explanation must not describe the literal image by mistake.
- [ ] Reuse cached target-language results by original idiom, language, and model.
  Do not call OpenRouter again for every user or every daily recurrence.
- [ ] Use structured responses and validate the expected fields/format before
  caching. Reuse existing OpenRouter integration and configuration where suitable.
- [ ] Coordinate concurrent cache misses so many requests do not produce duplicate
  generations. Avoid holding the daily-selection lock during the network call.
- [ ] Failed generation must remain retryable without replacing the day's idiom
  or caching an error as content.

User-provided example of intended display (illustrative, not dictionary data):

```text
piece of cake
---
something that might be easily done

[Translate]

After Translate, append:

раз плюнуть
---
что-то очень простое, легко выполняемое
```

### 5. Client block

- [ ] Place a compact Idiom of the day block below the existing Translation form.
- [ ] Default idiom language to the user's main learning language. Selection is
  shared per language; it is not personalized to their prior vocabulary.
- [ ] Initially show the original idiom and its same-language explanation.
- [ ] Offer a secondary button or text action for Translate, not a primary CTA.
- [ ] On activation, reveal the translated expression and explanation beneath the
  original. Label literal fallback text as Literal translation.
- [ ] Provide loading, retryable error, and no-imported-idioms states.
- [ ] Refresh for a changed local date or selected idiom language, including when
  returning to a tab after midnight.
- [ ] Follow existing responsive layouts, themes, accessibility, and localization.

The translation target's default/selection control has not been explicitly chosen.
Resolve it using existing app settings and make the target clear to the user. The
earlier suggestion to make Translate merely fill the existing translation form is
superseded: the action now generates and reveals the idiom translation inline.
Saving the displayed pair to personal vocabulary is not an agreed requirement.

### 6. Part 2 verification

- [ ] Test local-midnight boundaries and different timezone requests: same date +
  language gives the same row; different local dates can coexist.
- [ ] Test date gaps, stable existing selections, and concurrent first requests.
- [ ] Test zero usage, equal usage, below-maximum eligibility, successive cycles,
  new imports into an existing pool, and independent counts per language.
- [ ] Test original and translated explanation caching, purpose isolation,
  concurrent misses, response validation, and retry behavior.
- [ ] Regression-test exercise descriptions, required translation context, and
  approval precedence after the schema/query changes.
- [ ] Verify client loading, translation reveal, literal labels, errors, mobile
  layout, and local-date refresh.

## Implementation boundaries still to resolve

These are implementation choices, not reasons to redesign the agreed feature:

- Exact dictionary classification mappings and edition/language import coverage.
- Import-job schema, worker execution/recovery mechanism, and endpoint names.
- Download URL configuration and dictionary-management form details.
- Timezone fallback and target-language selection UI/default.
- Exact migration/index definitions and how existing admin description tooling
  presents the new purpose without mixing it with exercise clues.

Part 1 migrations, importer, worker, admin UI and verification are implemented.
Part 2 daily selections, OpenRouter content and client block remain future work.
