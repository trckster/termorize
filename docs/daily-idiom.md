# Daily idioms

The language registry is `backend/src/enums/language.go` (`AllLanguageValues`):
English, Russian, Italian, German, Spanish, French, Polish, Turkish, Portuguese,
and Ukrainian. All ten are covered. Migration 0028 supplies three starter idioms
per language; the multilingual Wiktionary importer extends the same pool and
now accepts every registered language. Selections rotate by usage, preferring a
word different from adjacent days when another eligible word is available.

## Delivery

`settings.telegram.daily_idiom_enabled` defaults to false, including existing
accounts whose JSON omits the field. Users opt in from the existing Telegram
settings section. Bot access must also be enabled.

The runner checks every 15 seconds and sends during the 11:00 minute in the
account's IANA timezone, including DST and fractional offsets. It uses the saved
`main_learning_language`. Web and Telegram share `daily_idioms` assignments by
local date and language. No time preference was added. Invalid legacy timezones
use the existing UTC fallback.

A database advisory lock per user and a unique `(user_id, date)` delivery receipt
prevent ordinary concurrent/restart duplicates. A failed API call can retry during
that minute. There is no late catch-up after an outage. As with the existing
Telegram delivery, the external send and database commit cannot be atomic: a
crash after Telegram accepts a message but before the receipt commits can repeat
it. Telegram's sendMessage API has no application idempotency key.

## Translation and Vocabulary

The card has an explicit, labelled Translate button. It preserves text selection
and fits the existing outline-button style. The source becomes the idiom language;
the target becomes the previous source language unless that already matches the
idiom, in which case the previous target remains.

The action calls `POST /api/daily-idiom/:id/translate`. The server resolves the
persisted selection and calls `TranslateDailyIdiom`, then OpenRouter's configured
model through `TranslateIdiom`. Its separate prompt requests a natural equivalent
idiom, or a natural explanation when no equivalent exists, rather than a literal
translation. Only `idiom_llm` translations satisfy this cache. Google translations
and generic collection LLM translations are excluded. Failure returns 503; there
is no fallback to Google Translate. Normal translation retains its existing route.
The separate description workflow still uses the existing language detector to
validate generated descriptions; this is not used to translate idioms.

The LLM result is persisted as words plus a Translation with both language codes.
The response supplies that translation's ID and stored text. Existing Ctrl+S,
Ctrl+E without edits, and mobile Save use the existing Vocabulary endpoint and
that exact ID. They neither translate again nor replace the displayed result.
User edits in the existing edit dialog remain explicit custom translations.

The initial Telegram message includes the same short description shown on the
frontend, in the idiom’s language. Both use `GetDailyIdiomDescription` and its
shared cache. If description generation fails, no incomplete message or delivery
receipt is created; delivery remains eligible for the existing retry window.

Telegram messages offer Translate, Add to vocabulary, and Dismiss on separate
rows. Translate reveals the LLM result without creating a Vocabulary entry, then
keeps Add and Dismiss available. The Add callback pins the preview's target language
so later preference changes cannot replace the translation the user saw. Dismiss
removes the keyboard while preserving the text and daily subscription; it does
not call the LLM. Both actions use localized labels.

Telegram's `idiom:add:<selection UUID>` callback uses the same service. Its target
comes from the existing saved translator source/target pair with the same rule
as the web action. After translation succeeds it saves Vocabulary, then replaces
the message with the original, translation and localized confirmation. Duplicate
pairs produce the existing already-saved feedback, including after replay;
concurrent web and Telegram saves are serialized. Deleted entries can be restored.
Translation/save errors retain the button and send a localized retry message;
no success is reported before the write completes.

## Verification

Backend tests exercise the real router, webhook and PostgreSQL migrations, with
Google, OpenRouter and Telegram faked. Coverage includes timezone/DST scheduling,
opt-in/off, concurrent delivery and saves, API failure/retry, cache separation,
translation/save failures, duplicate pairs, and all registry languages through
selection, descriptions, translation, delivery and both Vocabulary entry points.

```sh
cd backend
DB_PORT=55432 go test ./... -count=1
DB_PORT=55432 go test -race ./... -count=1
go vet ./...
go build ./...
```

Use a dedicated test PostgreSQL database; the test harness defaults to
`termorize_test`. Set `DB_PORT` to the available test service's port.

```sh
cd frontend
pnpm test
pnpm exec playwright install chromium
pnpm test:e2e
pnpm build
```

Browser tests cover the language rules, automatic translation, Ctrl+S/Ctrl+E,
retry, swap, target changes, ordinary translation, the settings toggle and the
landing. Asset capture is opt-in:

```sh
CAPTURE_IDIOM_PREVIEW=1 pnpm test:e2e
```

## Screenshot provenance

`frontend/public/images/daily-idiom-translation.png` is a Chromium screenshot of
this repository's `/translation` page, taken at 1280 px width in the Emerald dark
theme. The browser test provides synthetic example account/API data (Alex,
“break the ice” → “растопить лёд”); it contains no real user's data. It includes the
current daily idiom card and its action. No image generation or pixel editing was
used. The prior landing used an HTML translation mock, now replaced by this actual
UI capture. Desktop and mobile app/landing renders were inspected.
