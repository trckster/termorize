# Word categorization

`words.part_of_speech` is independent of the idiom `type`. NULL means pending or
failed; `unknown` is a completed decision that only admins resolve. Concrete
categories are permanent, enforced by conditional updates and a database trigger.
Shared labels use the word's language, never a user's translation context.

One backend goroutine handles new-word events, and a separate hourly command
recovers pending words. Each calls the OpenRouter Decisions API sequentially with
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
can proceed during those delays. In the backend, a panic restarts the loop and
requests a sweep.
Shutdown cancels in-flight calls. A crash after inference but before saving may
repeat an API call.

The scheduler's `0 * * * *` entry launches `/app/classify-words` as a separate
process. It connects directly to PostgreSQL and Jev using the scheduler's existing
environment, scans NULL words in keyset batches, drains bounded retries, and exits.
It uses the same worker and classification rules as the backend. Failed scans,
exhausted word retries, panics, and cancellation exit nonzero and appear in
scheduler logs; unfinished words remain NULL for the next run. The command needs
the migrated schema, `DB_*` settings, and `OPENROUTER_API_KEY`; `ENV` and `SENTRY_DSN`
configure logging/monitoring. It can run while the backend is down.

Backend events and hourly/manual runs can encounter the same pending word and
make duplicate API calls. Conditional saves retain the first persisted result,
including unknown, and protect admin choices. This replaces the research TODO's
HTTP-triggered recovery design with a standalone hourly process.

## First deployment

1. Build and deploy the backend **and scheduler** from this change. The SQL
   migration leaves all existing words NULL and performs no network work.
2. Once the backend has applied the migration, run the one-time backfill:

   ```sh
   docker compose exec -T scheduler /app/classify-words
   ```

   Outside Compose, from `backend/` with `DB_*` and `OPENROUTER_API_KEY` configured:

   ```sh
   go run ./cmd/cron/classify-words
   ```

3. The command waits for the backfill to finish. Monitor `classification sweep` /
   `word classification failed` logs in the backend and scheduler, command failures,
   and remaining work:

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

This assumes one backend event worker plus the scheduled recovery process.
Revisit coordination before adding backend replicas to limit duplicate API calls.
Vocabulary labels change on ordinary reloads, without classification polling.
Admin saves refresh both review lists and affect all uses of the shared word.

## Verification

Normal tests use recorded Jev responses and fake providers. From `backend/`, run
`go test -race ./...` with the test PostgreSQL settings described in `TESTING.md`.
From `frontend/`, run `pnpm test`, `pnpm build`, and `pnpm test:e2e`.

For rollout, run `go run ./cmd/test/classify-canary` from `backend/` with the
OpenRouter key configured. It makes four live calls using public example words,
checks the production parser and confidence threshold, and writes no database
rows. This small canary verifies availability and the request contract, not
production accuracy.
