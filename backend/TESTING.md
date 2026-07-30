# Testing Guide

Status: required repository policy.

Last researched: 2026-07-30.

This guide defines how Termorize backend tests must be designed, written, and reviewed.
It treats HTTP endpoint testing and Telegram interaction testing as two separate ideas.
They share fixtures and database isolation, but their contracts and failure modes differ.

## Goals

- Prove behavior through public boundaries instead of implementation details.
- Keep tests deterministic, network-free, readable, and safe to rerun.
- Exercise the real Gin router, middleware, validation, services, and persistence.
- Exercise Telegram updates and outbound Bot API calls as a protocol conversation.
- Make regressions explain themselves through narrow names and useful assertions.
- Use coverage as a discovery aid, never as a substitute for contract thinking.

## Research basis

The rules below combine official documentation with patterns in mature Go repositories.

- Go's [`net/http/httptest`](https://pkg.go.dev/net/http/httptest) provides incoming
  server requests, response recorders, and loopback HTTP servers.
- The recorder documentation says to inspect the response returned by `Result` for
  final headers and warns against deep-equality on the whole `http.Response`.
- Gin's official [testing guide](https://gin-gonic.com/en/docs/testing/) recommends
  `httptest`, `gin.TestMode`, table-driven cases, and router-level middleware tests.
- Go's [table-driven test guidance](https://go.dev/wiki/TableDrivenTests) recommends
  a complete input and expected output per case, especially when copy/paste appears.
- The [`testing`](https://pkg.go.dev/testing) package provides subtests, `Helper`,
  `Cleanup`, `Setenv`, `TempDir`, and controlled parallel execution.
- `Setenv` and process-global state cannot safely be used by parallel tests.
- Go's [fuzzing guide](https://go.dev/doc/security/fuzz/) recommends fast,
  deterministic, state-independent fuzz targets for parser and validation boundaries.
- Go's [race detector guide](https://go.dev/doc/articles/race_detector) establishes
  `go test -race` as the standard runtime race check.
- Go's [coverage guide](https://go.dev/doc/build-cover) documents unit and larger
  integration-test coverage; coverage identifies unexecuted code, not correct behavior.
- Prometheus's
  [`web/api/v1/api_test.go`](https://github.com/prometheus/prometheus/blob/main/web/api/v1/api_test.go)
  uses large named case tables, runs supported HTTP methods as subtests, injects fakes,
  checks warnings and errors, and compares semantic JSON.
- Prometheus also uses both `httptest.NewRecorder` for direct handlers and
  `httptest.NewServer` where a real HTTP client/server exchange matters.
- Kubernetes tests, for example
  [`create_test.go`](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create_test.go),
  use small tables with descriptive subtest names for input boundary behavior.
- The maintained Go Telegram library
  [`go-telegram/bot`](https://github.com/go-telegram/bot) tests webhook decoding with
  recorder/request pairs and tests Bot API calls against an `httptest.Server`.
- Its
  [`webhook_handler_test.go`](https://github.com/go-telegram/bot/blob/main/webhook_handler_test.go)
  covers success, body-read failure, malformed JSON, and canceled context.
- Its
  [`bot_test.go`](https://github.com/go-telegram/bot/blob/main/bot_test.go)
  routes fake Bot API methods by URL and records calls without contacting Telegram.
- Telegram's official [Bot API](https://core.telegram.org/bots/api) defines webhook
  update shapes, the secret-token header, callback query fields, and Bot API methods.
- Telegram requires `answerCallbackQuery` after a button press so the client's progress
  indicator stops, even when the bot has no notification text to show.
- Telegram retries webhook delivery after a non-2xx response, so mutating update
  handlers must be designed and tested with replay in mind.

## Test taxonomy

Use the smallest boundary that proves the intended contract.

### HTTP endpoint integration tests

- Keep endpoint tests in `tests/`.
- Send requests through `testkit.Router().ServeHTTP`.
- Do not call controller functions directly for an endpoint contract test.
- The real router proves route registration, middleware order, path parameters,
  authentication, validators, controller mapping, service behavior, and serialization.
- Use the real PostgreSQL schema and migrations for persistence-sensitive behavior.
- Fake only systems outside this repository: Telegram, Google, OpenRouter, OAuth servers.

### Telegram interaction tests

- Keep webhook-to-side-effect tests in `tests/telegram_webhook_test.go`.
- Keep pure callback parsing and rendering tests in `src/integrations/telegram`.
- Deliver realistic Telegram `Update` JSON through `POST /api/telegram/webhook`.
- Use `testkit.MockTelegramAPI` for every path that may call the Bot API.
- Assert the inbound webhook response separately from outbound Telegram requests.
- Assert database changes separately from visible chat changes.

## Repository harness

The shared package-level setup is `tests/setup_test.go`.

- `testkit.Main` sets Gin test mode and installs a no-op logger.
- It supplies test-only defaults before loading configuration.
- It creates `termorize_test` unless `TEST_DB_NAME` overrides the name.
- It connects to PostgreSQL, runs real migrations, installs external fakes, and builds
  the router once.
- `testkit.Truncate(t)` removes data from every application table before each test.
- The migrations table is preserved so schema setup remains valid.
- `RESTART IDENTITY CASCADE` resets sequences and dependent rows.
- `testkit.CreateUser` creates valid persisted users with overridable fields.
- `testkit.AuthCookie` issues a real application JWT.
- `testkit.AuthedRequest` exercises the real authentication middleware.
- `testkit.MockGoogleTranslate` and `testkit.MockOpenRouter` replace outbound clients.
- `testkit.MockTelegramAPI` replaces Telegram's base URL with a loopback server.
- Every fake restoration is registered with `t.Cleanup`.

Do not bypass these helpers without a concrete protocol requirement.
When a helper is missing a needed capability, extend `src/testkit` narrowly.
Mark reusable helpers with `t.Helper()` so failures point to the calling test.

## Database setup and isolation

The integration suite expects a reachable PostgreSQL admin database.
Default test settings are host `127.0.0.1`, port `5432`, user `root`, password `password`.
The repository Compose file exposes PostgreSQL on host port `5433`.
Set `DB_PORT=5433` when using that service.

Example from the parent repository directory:

```sh
DB_USER=root DB_PASSWORD=password docker compose up -d postgres
cd backend
DB_PORT=5433 go test ./tests
```

Rules:

- Never point tests at a development or production database.
- The database name must be recognizably test-only.
- Call `testkit.Truncate(t)` at the beginning of every database-using test.
- Seed only the minimum records needed to explain the scenario.
- Use services or HTTP setup when setup behavior is part of the contract.
- Use direct `db.DB` inserts when setup is incidental and HTTP setup would obscure intent.
- Assert persistence with a fresh query after the request.
- For deletes, distinguish hard deletion from GORM soft deletion explicitly.
- For transactions, assert all intended writes and important non-writes.
- For ownership, create at least two users and prove cross-user isolation.
- Do not depend on row ordering unless the API promises it.
- When ordering is promised, seed deliberately out-of-order values and assert the contract.
- Avoid `t.Parallel()` in `tests/` while tests share one database and mutable global fakes.
- Do not use time sleeps to wait for database state.
- Freeze, inject, or compare time within a tolerance when time affects behavior.

## HTTP endpoint test standard

One endpoint suite must begin with a contract inventory.

### Request dimensions

- Correct HTTP method and registered path.
- Authentication absent, valid, stale, and wrong-user where applicable.
- Required headers, cookies, query parameters, and path parameters.
- Empty body, malformed JSON, missing fields, invalid enum values, and valid JSON.
- Boundary values for lengths, counts, pagination, time zones, and schedules.
- Resource absent, soft-deleted, owned by another user, and accessible.
- Dependency success, dependency failure, and cached/existing-data paths.

Do not mechanically test irrelevant combinations.
Choose cases from actual branches, invariants, and security boundaries.

### Response dimensions

- Assert the exact status code first with `require`.
- Decode JSON into a small typed response matching the public contract.
- Assert `Content-Type` when representation negotiation or serialization matters.
- Assert stable fields exactly.
- Assert generated IDs are non-zero and then use them for persistence checks.
- Compare JSON semantically with decoded values or `JSONEq`, not raw formatting.
- Assert error shape and stable public message or validation tag.
- Do not assert logs as the primary behavior.
- Do not deep-compare a complete `http.Response`.
- Do not assert timestamps byte-for-byte unless time is controlled.
- Check that secrets, internal IDs, and private fields are absent when relevant.

### Side-effect dimensions

- Assert the row was created, updated, soft-deleted, or left unchanged.
- Assert the row belongs to the authenticated user.
- Assert dependent rows and join rows have the expected cardinality.
- Assert a failed request performs no partial write.
- Assert repeated requests according to the endpoint's idempotency contract.
- Assert one user's action cannot read or mutate another user's records.
- Query from the database again; do not trust the object used during setup.

## Definition of a fully tested endpoint

For this repository, “fully tested” means all applicable items below are demonstrated:

- Route and method reach the real handler.
- Authentication or public access is proven.
- Valid input returns the exact success status and response schema.
- Invalid path, query, and body inputs map to stable 4xx responses.
- Not-found behavior is proven.
- Ownership or authorization behavior is proven without leaking hidden resources.
- Important domain conflict behavior is proven.
- Persistence and related side effects are queried directly.
- Failure paths leave protected state unchanged.
- External dependencies are faked and their inputs are asserted.
- The test never reaches the public network.

Full does not mean every possible byte sequence.
It means every meaningful contract partition and risk is represented.

## Telegram test standard

Telegram tests are protocol tests, not ordinary controller tests.
They have two directions and often three observable layers:

1. Telegram sends an authenticated webhook update.
2. Termorize mutates domain state.
3. Termorize calls one or more Telegram Bot API methods.

Each layer must be asserted independently.

### Inbound webhook

- Use the exact route `/api/telegram/webhook`.
- Send `Content-Type: application/json`.
- Send `X-Telegram-Bot-Api-Secret-Token` using `telegram.BuildWebhookSecret()`.
- Test a missing secret and an incorrect secret as 401.
- Test malformed JSON as 400.
- Build realistic nested `from`, `chat`, `message`, and `callback_query` objects.
- Keep Telegram IDs distinct from database user IDs.
- Give callback queries stable IDs so acknowledgements can be matched.
- Include `message.text` when callback behavior edits existing text.
- Include the source message ID and chat ID and assert both outbound.
- Unknown but well-formed update types should be safe no-ops.

### Outbound Bot API

- Start `testkit.MockTelegramAPI(t)` before delivering the update.
- Assert the exact method name, such as `answerCallbackQuery` or `editMessageText`.
- Decode the recorded body into a typed local struct or a small map.
- Assert `callback_query_id`, `chat_id`, `message_id`, text, parse mode, and markup.
- Assert call counts, especially that one button press receives one acknowledgement.
- Assert forbidden extra methods; an edit flow should not unexpectedly call `sendMessage`.
- For keyboards, assert visible labels and `callback_data`, not just row counts.
- For consumed actions, assert the obsolete button is removed.
- For Markdown, assert dynamic text is escaped in a package-level rendering test.
- Fake error responses when testing blocked users, expired callbacks, or API failures.
- Never assert only “some Telegram request happened.”

### Callback acknowledgement

- Every callback with a non-empty ID must normally call `answerCallbackQuery`.
- Assert acknowledgement even when the callback action becomes a domain no-op.
- Match the outbound `callback_query_id` to the inbound callback ID.
- Exercise deferred acknowledgement separately for match-tap flows.
- For expired callback-query errors, assert tolerant behavior without hiding other errors.

### State and replay

- Assert the database transition caused by the callback.
- Assert no transition for invalid payload, missing user, or foreign resource.
- Deliver the same update twice for callbacks that create, delete, or complete data.
- The second delivery must not duplicate rows or corrupt progress.
- If an action is intentionally non-idempotent, identify the deduplication boundary.
- Prefer domain operations that naturally converge, such as create-if-absent.
- Test callbacks after the underlying object is already completed or deleted.
- Assert that replay still acknowledges the callback where the Bot API permits it.

### Callback data

- Treat `callback_data` as an external compact protocol.
- Test handler type, action, argument count, malformed UUID, and unknown action.
- Test both standard and compact UUID encodings when supported.
- Keep generated callback data within Telegram's documented size limit.
- Never put secrets or authorization decisions solely in callback data.
- Resolve the Telegram sender to a Termorize user and scope every mutation to that user.

## Definition of a fully tested Telegram callback

- A realistic signed webhook reaches the callback router.
- The callback is acknowledged with the matching ID.
- Valid callback data selects the expected action.
- Sender, chat, and message identifiers are propagated correctly.
- The exact domain mutation is verified in PostgreSQL.
- The visible message text and keyboard transition are verified.
- No unintended Bot API method is called.
- Invalid or stale domain state is safe.
- Replay behavior is explicitly verified.
- No real Telegram request is possible.

## Assertion style

- Use `require` when later assertions depend on success.
- Use `assert` to report multiple independent contract mismatches in one run.
- Put response bodies in fatal status messages for quick diagnosis.
- Prefer exact equality for stable codes, IDs, counts, enum values, and messages.
- Prefer `ElementsMatch` only when ordering is explicitly irrelevant.
- Avoid broad `Contains` when a typed exact field can be asserted.
- Give every negative assertion a reason.
- Never accept either of two statuses unless the contract genuinely permits both.
- A test name should read as behavior, not as an implementation function name alone.

## Commands

Format changed Go files:

```sh
gofmt -w path/to/file_test.go
```

Run package unit tests:

```sh
go test ./src/...
```

Run integration tests with the Compose PostgreSQL port:

```sh
DB_PORT=5433 go test ./tests -count=1
```

Run one endpoint or callback while developing:

```sh
DB_PORT=5433 go test ./tests -run 'TestStartCollectionPractice' -count=1 -v
DB_PORT=5433 go test ./tests -run 'TestTelegramWebhookVocabularyAddCallbackIsReplaySafe' -count=1 -v
```

Run the complete suite:

```sh
DB_PORT=5433 go test ./... -count=1
```

Run race detection in CI or before merging concurrency-sensitive work:

```sh
DB_PORT=5433 go test -race ./...
```

Collect coverage across application packages:

```sh
DB_PORT=5433 go test -coverpkg=./src/... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## CI policy

- Start a dedicated PostgreSQL service with a health check.
- Use a unique test database and test-only credentials.
- Run formatting or a clean-tree formatting check.
- Run `go vet ./...`.
- Run `go test ./... -count=1`.
- Run `go test -race ./...` in a separate job if runtime is material.
- Upload coverage for review, but do not chase a percentage by asserting trivia.
- Cache downloaded modules, never database state.
- Fail on any attempted connection to unapproved public services.
- Keep test logs free of credentials, JWTs, bot tokens, and webhook secrets.

## Implemented reference examples

The endpoint example is `POST /api/collections/:id/practice`.
Its suite now proves:

- authentication is required;
- malformed collection IDs return 400;
- absent collections return 404;
- inaccessible private collections are hidden with 404;
- an accessible collection with no vocabulary intersection returns 422;
- a valid round returns its collection identity and eligible vocabulary IDs;
- deleted and missing vocabulary rows are excluded;
- mastered vocabulary remains eligible for explicit collection practice.

The Telegram example is callback data `vocabulary:add:<translation UUID>`.
Its end-to-end webhook test proves:

- the webhook secret reaches the real callback router;
- `answerCallbackQuery` uses the incoming callback ID;
- one vocabulary row is created for the Telegram user;
- `editMessageText` targets the source chat and message;
- the success suffix is localized;
- the consumed inline keyboard is removed;
- no new message is sent;
- replay creates no duplicate while still following the callback protocol.

## Review checklist

- Does the test use the narrowest sufficient boundary?
- Does an endpoint test traverse the real router?
- Does a Telegram test cover both inbound and outbound protocol directions?
- Is all external network traffic replaced with a fake?
- Is database state reset before the test?
- Are fixtures minimal and named by their role?
- Are status, response schema, and persistence all asserted?
- Are authentication and ownership boundaries covered?
- Are malformed and boundary inputs covered?
- Are side effects and important non-effects covered?
- Is replay or idempotency covered for mutations?
- Are global replacements restored with cleanup?
- Could this test run in a random order repeatedly?
- Would its failure message identify the broken contract quickly?
- Has the changed code been formatted?
- Do the targeted tests and the complete suite pass?
