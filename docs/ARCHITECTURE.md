# Architecture of the CS Foundations Pyramid Quiz Bot

## Overview

The application is a **Telegram quiz bot** that teaches computer-science fundamentals via multiple-choice questions. It is written in Go and organised into four distinct layers. A deprecated terminal (CLI) interface is preserved for reference but is not part of the active bot flow.

The bot receives updates from Telegram via **long-polling** (`b.Start(ctx)`), not WebHooks.

---

## Folder Layout

```
basics/
  cmd/
    bot/main.go              # primary entrypoint — wires all layers and starts the bot
    migrate/main.go          # one-shot seeder: data/topics.json → PostgreSQL (curated tests)
  internal/
    bot/                     # bot service layer
      bot.go                 # Bot struct, Run(), sendTestMenu, sendOrEdit helper
      handlers.go            # onStart, onQuit, onCallback* (quiz flow) — methods on Bot
      manage.go              # test CRUD handlers + AI generation + validation
      keyboards.go           # inline keyboard builders (test menu, manage, delete confirm)
      messages.go            # message text builders + HTML helpers
      session.go             # SessionStore interface + InMemorySessionStore
      errors.go              # handleErr helper
    ai/                      # AI integration layer
      claude.go              # Anthropic/Claude client — generates tests from a description
    quiz/                    # business-logic layer
      topic.go               # Topic struct (one quiz question)
      categories.go          # CategoryMenu, CountInCategory, FilterTopics (used by CLI/migration)
      options.go             # BuildOptions (4-choice distractor generator)
    storage/                 # persistence layer
      store.go               # TopicStore interface (read-only; used by CLI + migration source)
      json_store.go          # JSONTopicStore — reads data/topics.json (migration source)
      test_store.go          # Test struct + TestStore interface (read/write CRUD)
      pg_store.go            # PGTestStore — PostgreSQL/JSONB implementation
    config/
      config.go              # LoadDotEnv, MustToken, MustDatabaseURL
    apperrors/
      errors.go              # sentinel errors (ErrUnknownCategory, ErrInvalidStage, ErrNoTopics, ErrTestNotFound, ErrInvalidTest)
  cli/                       # DEPRECATED — terminal quiz interface, not used by the bot
    main.go                  # runCLI(); shares quiz + JSONTopicStore with the original design
  data/
    topics.json              # curated quiz topics — seed source for the migration
  docs/
    ARCHITECTURE.md          # this file
```

---

## Layer Descriptions

### 1. Telegram (external)

Telegram's servers deliver user messages and callback events to the bot. The bot library (`github.com/go-telegram/bot`) retrieves them via **long-polling** — the process repeatedly asks Telegram for new updates. There is no inbound HTTP server and no WebHook configuration required.

---

### 2. Bot Service Layer (`internal/bot`)

Receives updates from the Telegram library and drives the user through the quiz flow.

**Responsibilities:**
- Registers command and callback-query handlers with the bot library.
- Reads and writes per-user `Session` state via the `SessionStore`.
- Lists the tests available to a chat, loads a chosen test's questions, and builds answer options.
- Lets users create and edit their own tests by describing them; the bot calls Claude (via `internal/ai`) to generate the questions, then validates and saves them. Users can also delete their own tests.
- Renders messages and inline keyboards, then sends or edits them via `sendOrEdit`.
- Logs errors and sends user-friendly fallback messages on failure.

**Commands:**

| Command | Purpose |
|---|---|
| `/start` | Show the test-selection menu and start a quiz |
| `/mytests` | List the user's own tests with edit / delete buttons |
| `/newtest` | Generate a new test with AI from a short description |
| `/settings` | Show test-management help |
| `/help`, `/quit` | Help text / end the session |

**Key types:**
```go
type Bot struct {
    store    storage.TestStore
    sessions SessionStore
    gen      TestGenerator // optional Claude client; nil = AI disabled
}
```

Handlers are methods on `*Bot`, giving them access to dependencies through the receiver rather than package-level globals.

---

### 3. Quiz / Business Logic Layer (`internal/quiz`)

Stateless functions that implement the quiz rules.

| Function | Purpose |
|---|---|
| `FilterTopics(topics, cat)` | Returns topics matching a category string |
| `CountInCategory(topics, cat)` | Counts topics in a category |
| `BuildOptions(all, correct)` | Produces 4 shuffled answer choices (1 correct + 3 distractors from the same category when possible) |

The `CategoryMenu` slice defines the ordered list of selectable categories shown to the user.

---

### 4. Persistence Layer (`internal/storage`)

Responsible for storing and serving **tests**. A *test* is a named set of quiz
questions persisted as a single JSON document. Curated tests are global
(`owner_chat IS NULL`); user-created tests belong to the chat that made them.

**Interface:**
```go
type TestStore interface {
    ListAvailable(chatID int64) ([]Test, error) // global + owned by chatID
    ListOwned(chatID int64) ([]Test, error)
    Get(id int64) (Test, error)
    Create(t Test) (int64, error)
    Update(t Test) error            // owner-scoped
    Delete(id, ownerChat int64) error
}
```

**Implementation — `PGTestStore` (PostgreSQL):**
- Connects with `pgxpool` and runs `CREATE TABLE IF NOT EXISTS` on startup, so a fresh database is self-bootstrapping.
- Each test is one row; its questions live in a `JSONB` `data` column (`{ "questions": [ ... ] }`).
- Writes are ownership-scoped in SQL: `Update`/`Delete` only affect rows whose `owner_chat` matches the caller, so users cannot modify each other's (or curated) tests.

**Schema:**
```sql
CREATE TABLE tests (
    id          BIGSERIAL PRIMARY KEY,
    owner_chat  BIGINT,                       -- NULL = curated/global
    title       TEXT        NOT NULL,
    data        JSONB       NOT NULL,         -- { "questions": [ {Name, Overview, Question, Explanation, Layer}, ... ] }
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX tests_owner_idx ON tests (owner_chat);
```

**Legacy `TopicStore` / `JSONTopicStore`:** retained as the read-only source for
the one-shot migration (`cmd/migrate`) and the deprecated CLI. The bot itself no
longer reads `data/topics.json` at runtime.

---

### 5. Session Store (`internal/bot/session.go`)

Tracks each user's in-progress quiz state between Telegram messages.

**Interface:**
```go
type SessionStore interface {
    Get(chatID int64) *Session
    Reset(chatID int64) *Session
    Delete(chatID int64)
}
```

**Current implementation — `InMemorySessionStore`:**
- A `map[int64]*Session` protected by a `sync.Mutex`.
- Lives for the process lifetime; sessions are lost on restart.
- A Redis or database-backed implementation can replace it by satisfying the interface.

**Session state machine:**

```
stageCategory → stageOrder → stageQuiz ↔ stageReveal → stageDone
     ↑                                                      |
     └──────────────────────────── "Play again" ────────────┘

Test management (entered via /newtest or the ✏️ edit button):
stageAwaitNewTest  ─ user describes test ─► Claude generates ─► Create ─► stageCategory
stageAwaitEditTest ─ user describes test ─► Claude generates ─► Update ─► stageCategory
```

---

## Data Flow

```
Telegram ──long-poll──► internal/bot
                              │
                     reads/writes SessionStore (in-memory)
                              │
                  reads/writes via TestStore interface
                              │
                     internal/storage/PGTestStore
                              │
                       PostgreSQL (tests, JSONB)

Seeding (one-shot):
  data/topics.json ──► cmd/migrate ──group by category──► PostgreSQL (curated tests)
```

---

## Error Policy

| Situation | Behaviour |
|---|---|
| Missing token, missing `DATABASE_URL`, or unreachable database at startup | `slog.Error` + `os.Exit(1)` (fail fast) |
| Invalid user-submitted test JSON | Validation message back to the user; session stays in the await stage so they can resend |
| Unknown test/category or invalid stage in a callback | Log error with `chatID` + `err`, send user-friendly fallback message |
| Telegram `EditMessage` failure | Log at `slog.LevelDebug`, fall back to sending a new message |
| `SendMessage` failure | Log at `slog.LevelError` |
| Any handler error | `handleErr(ctx, b, chatID, err, userMsg)` in `internal/bot/errors.go` handles logging + fallback uniformly |

Logging uses `log/slog` (stdlib). The level and format are controlled by environment variables:
- `LOG_LEVEL=debug` — enables debug output (edit failures, etc.)
- `LOG_FORMAT=json` — switches to structured JSON logs (useful in production)

---

## Deprecated CLI

`cli/main.go` is a standalone `package main` containing the original terminal quiz interface. It imports `internal/quiz` and `internal/storage` so it shares the same topic data and business logic as the bot. It is **not** compiled or used by the bot flow. Build it with:

```
go run ./cli
```

---

## Running the Bot

```bash
cp .env.example .env       # set TELEGRAM_BOT_TOKEN and DATABASE_URL
go run ./cmd/migrate       # seed PostgreSQL with curated tests (run once)
go run ./cmd/bot           # start the bot
```

The bot creates the `tests` table automatically on startup; the migration step
only seeds the curated content from `data/topics.json`.

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | (required) | Token from @BotFather |
| `DATABASE_URL` | (required) | PostgreSQL connection string |
| `ANTHROPIC_API_KEY` | (optional) | Enables AI test generation (`/newtest`); see INSTRUCTION.md |
| `ANTHROPIC_MODEL` | (optional) | Override the Claude model (defaults to a Haiku model) |
| `TOPICS_PATH` | `data/topics.json` | Seed source used only by `cmd/migrate` |
| `LOG_LEVEL` | `info` | `debug` or `info` |
| `LOG_FORMAT` | `text` | `text` or `json` |
