# Dictionary imports

Part 1 adds an admin-only Dictionaries tab under `/admin`. Choose **Import** for a
configured source; **Update** downloads it again, and **Retry import** starts a new
attempt after a failure or interruption. Every attempt keeps its own history.
Closing the page does not stop a job. Only administrators can access the three API
routes: `GET /api/admin/dictionaries` and `GET` / `POST
/api/admin/dictionaries/:id/imports`. History uses `?page=1`, with 20 jobs per page.

The migration registers English, Russian, and Italian Wiktionary editions with
homepages, reuse terms, attribution, and download URLs. An edition's language is
not the language of every entry. Importers accept only entry languages `en`, `ru`,
and `it`. The source URLs are trusted database configuration, not request input;
change `dictionaries.download_url` through a reviewed migration if Kaikki moves a
source. Each job snapshots the name, edition and download URL it used.

## Classification evidence

Availability and field shapes were checked on 2026-09-21 against the current
[Kaikki raw downloads](https://kaikki.org/dictionary/rawdata.html) and
[Wiktextract edition models](https://github.com/tatuylonen/wiktextract/tree/master/src/wiktextract/extractor).
Downloads inspected in memory were discarded; no dictionary dump is committed.

| Edition | Explicit evidence accepted |
| --- | --- |
| English | Entry or sense `idiomatic` tag; exact `English idioms`, `Russian idioms`, or `Italian idioms` category matching `lang_code` |
| Russian | Entry or sense `idiomatic` tag; exact `Фразеологизмы/<lang_code>` category |
| Italian | Entry or sense `idiomatic` tag or raw `idiomatico` tag |

No inference is made from spaces, `pos=phrase`, glosses, translations, or related
terms. Hard redirects (`pos=hard-redirect`) contain a title and redirect target
instead of lexical word/language fields and count as skipped records. Proverb
entries and bound morphemes are excluded. Broad Russian
`Фразеология` categories and Italian `Locuzioni` categories are not evidence of an
idiom. An English-edition idiom can have `pos=verb`.

A complete streamed inspection of the Italian extract (801,663 records) found
`itsy bitsy` (`lang_code=en`) with a sense-level raw `idiomatico` tag. Familiar
Italian idioms such as `in bocca al lupo` and `rompere il ghiaccio` only carried
broad phrase categories in that edition. Consequently Italian-edition imports
may add very few entries. Italian-language idioms can be imported from explicit
classifications in the English and Russian editions. No import count is promised.

The small JSONL fixtures in `backend/src/integrations/kaikki/testdata` preserve the
classification fields sampled from those extracts and omit definitions/examples.
English `rain cats and dogs`, `a-`, and the `grain of salt` redirect, all Russian
samples, and all Italian samples
were observed directly. The other two English fixture rows are small synthetic
cross-language cases using the verified schema. Source attribution: Wiktionary
contributors, extracted by Tatu Ylonen and contributors using Wiktextract, via
Kaikki. Consult the source-specific reuse terms recorded in `dictionaries`.

## Worker and storage

The application starts a queue worker, but never creates imports on startup or
ordinary visits. Queued jobs were explicitly requested by an administrator.
Workers poll every three seconds. A PostgreSQL session advisory lock permits one
worker to process a job at a time across application instances. A separate lock
serializes short word-writing transactions, shared with vocabulary saves, so
case-insensitive deduplication matches existing application behavior. Network
translation happens outside that lock.

Every job downloads its gzip to a private temporary directory under the OS temp
directory, keyed by database host, port and name. Files use job UUIDs. They are
removed on success and failure; a worker taking the lock also removes abandoned
UUID-named downloads from its local directory. Temporary directories must be
writable and have space for the selected source (the English gzip was about
2.7 GiB at inspection). Downloads have a six-hour timeout and a 10 GiB size limit.
No downloads are cached, archived, or included in application images.

The worker streams decompression and reads JSONL with an 8 MiB per-record limit.
Oversized and malformed records are counted and skipped; the first ten diagnostic
messages are kept with line numbers. Inserts/promotions and progress commit
together every 500 records. Memory does not grow with the number of entries.
Only text, language and classification go into `words`; IDs and existing related
rows are preserved. No vocabulary, translations, daily selections, definitions,
or generated descriptions are created by an import.

Processed records equal inserted + newly classified + skipped + failed records.
Already classified entries, duplicates, unsupported languages, and unrelated
entries count as skipped. Download progress records bytes and an optional total;
parsing has no invented completion percentage. A completed job can have failed
records, which the UI labels separately. Fatal download, gzip, filesystem or
database errors fail the job; counters show committed work only.

The worker holds its SQL connection for the job and uses that same connection for
all writes. Losing the session prevents it from updating a replacement worker's
state. On the next successful lock acquisition, unfinished downloading/importing
jobs become `interrupted`, retaining their committed counts. They can be retried
explicitly; retries download and scan again, safely reusing prior words. Pending
queued jobs remain eligible after restart.

## Verification

Run from `backend`, using the local test PostgreSQL service and testkit defaults:

```sh
go test ./... -count=1
go test -race ./...
go vet ./...
```

Tests cover edition fixtures, missing fields, malformed/oversized records,
cross-edition overlap, repeat imports, vocabulary preservation, concurrent starts,
worker exclusion, recovery, cancellation, download errors, cleanup, atomic batch
rollback, history pagination and admin authorization. A blocked Google fake
verifies unrelated word saves remain available during translation.

From `frontend`, run `pnpm build` and `pnpm test`. Browser verification should
cover desktop and mobile with both English and Russian user preferences (admin
copy always stays English), starting and retrying jobs,
reload during a download, unknown totals, record errors, refresh after failure,
admin routing and horizontal overflow. Browser checks can use API fixtures;
production dictionaries need not be downloaded to verify the UI.
