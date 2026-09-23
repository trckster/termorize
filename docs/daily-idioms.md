# Daily idiom selections

Authenticated clients can request `GET /api/daily-idiom?language=en` with any
supported app language. The language is required; missing or unsupported values
return HTTP 400. Guests use the same authenticated endpoint.

The date comes from the user's saved `settings.time_zone`, as with exercise
statistics. Missing or invalid timezones fall back to UTC. The response is not
HTTP-cacheable, so a later request can observe the next local date.

```json
{
  "date": "2026-09-23",
  "language": "en",
  "idiom": {
    "id": "<daily assignment UUID>",
    "word_id": "<word UUID>",
    "word": "piece of cake"
  }
}
```

When the language has no imported idioms, HTTP 200 returns the date and language
with `"idiom": null`. No assignment is stored, so a subsequent request can select
an idiom after an import.

Migration `0024_create_daily_idioms` adds shared assignments unique by date and
language, with no user ownership and no uniqueness on the selected word. Selection
only runs on a request; dates need not be consecutive and are never backfilled.

Only words in the requested language classified as `idiom` are eligible. Each
word's persisted assignments are counted, including zero-use words. Equal counts
allow the entire pool; otherwise every word below the language's highest count is
eligible, including words above the minimum. Selection is random within that pool.

A transaction-scoped advisory lock per language serializes selection across dates
and server processes. After taking the lock, the transaction checks for an existing
assignment and reuses it unchanged. Read-committed isolation lets requests waiting
on the lock see the assignments committed by earlier requests before counting.

This implements Part 2, points 1–2 only. Generated descriptions, translations,
response caching, and the client block remain future work.
