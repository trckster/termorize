# Word categorization

`words.part_of_speech` is independent of the idiom `type`. NULL means pending or
failed; `unknown` is a completed decision that only admins resolve. Concrete
categories are permanent, enforced by conditional updates and a database trigger.
Shared labels use the word's language, never a user's translation context.

One backend goroutine calls the OpenRouter Decisions API sequentially with
`typesafe/jev-1.13`, the researched concise Choice prompt, and the existing
`OPENROUTER_API_KEY`. Valid answers below 0.90 confidence become unknown; malformed
answers and failures remain NULL. The client uses `answers.category.confidence`.
The model was verified against [OpenRouter's model listing](https://openrouter.ai/typesafe/jev-1.13)
and the [Decisions guide](https://openrouter.ai/blog/insights/what-is-jev/).

New-word transactions publish IDs only after the outer commit. Any future code
enclosing word creation must use `db.WordTransaction`, including outer
transactions around `GetOrCreateWord`. Rolled-back savepoints discard their IDs.
Delivery is nonblocking and bounded to 1,024 outstanding IDs; a transaction retains
at most 2,048 events. Excess work stays NULL for recovery. Recovery scans use
64-ID keyset batches and alternate with queued events. Retries run at most three
times, with 1- and 2-second delays and a 15-second timeout per attempt; other work
can proceed during those delays. A panic restarts the loop and requests a sweep.
Shutdown cancels in-flight calls. A crash after inference but before saving may
repeat an API call.

The scheduler's `0 * * * *` entry invokes `/app/trigger-classification`. This
command requests `POST /api/internal/classification/sweep`; it never connects to
the database or starts a classifier. The endpoint authenticates a purpose-specific
HMAC token derived from `SECRET`, which backend and scheduler must share. It
coalesces overlapping sweeps and returns 202 immediately. Trigger failures exit
nonzero and appear in scheduler logs. `BACKEND_INTERNAL_URL` defaults to
`http://backend:8080`; set it for a different internal hostname or local runs.

## First deployment

1. Build and deploy the backend **and scheduler** from this change. The SQL
   migration leaves all existing words NULL and performs no network work.
2. Once the backend is accepting requests, run the one-time backfill trigger:

   ```sh
   docker compose exec -T scheduler /app/trigger-classification
   ```

   Outside Compose, from `backend/` with `SECRET` configured:

   ```sh
   BACKEND_INTERNAL_URL=http://127.0.0.1:8080 go run ./cmd/cron/trigger-classification
   ```

3. Monitor `classification sweep` / `word classification failed` backend logs,
   scheduler trigger errors, and remaining work:

   ```sql
   SELECT count(*) FILTER (WHERE part_of_speech IS NULL) AS pending,
          count(*) FILTER (WHERE part_of_speech = 'unknown') AS unknown,
          count(*) FILTER (WHERE part_of_speech IS NOT NULL
                           AND part_of_speech <> 'unknown') AS categorized
   FROM words;
   ```

4. Hourly sweeps recover missed events, outages, and restarts. Unknowns are
   completed and do not enter automatic recovery. Review them in Admin →
   Categorization. Different completed labels remain visible as mismatched
   vocabulary records, including separate records saved by different users;
   legitimate differences do not need correction.

This assumes **one active backend classifier**. Revisit routing and coordination
before running backend replicas. Vocabulary labels change on ordinary reloads,
without classification polling. Admin saves refresh both review lists and affect
all uses of the shared word.

## Verification

Normal tests use recorded Jev responses and fake providers. From `backend/`, run
`go test -race ./...` with the test PostgreSQL settings described in `TESTING.md`.
From `frontend/`, run `pnpm test`, `pnpm build`, and `pnpm test:e2e`.

For rollout, run `go run ./cmd/test/classify-canary` from `backend/` with the
OpenRouter key configured. It makes four live calls using public example words,
checks the production parser and confidence threshold, and writes no database
rows. This small canary verifies availability and the request contract, not
production accuracy.
