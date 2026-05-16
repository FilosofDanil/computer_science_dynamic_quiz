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
  internal/
    bot/                     # bot service layer
      bot.go                 # Bot struct, Run(), sendOrEdit helper
      handlers.go            # onStart, onQuit, onCallback* — methods on Bot
      keyboards.go           # inline keyboard builders
      messages.go            # message text builders + HTML helpers
      session.go             # SessionStore interface + InMemorySessionStore
      errors.go              # handleErr helper
    quiz/                    # business-logic layer
      topic.go               # Topic struct
      categories.go          # CategoryMenu, CountInCategory, FilterTopics
      options.go             # BuildOptions (4-choice distractor generator)
    storage/                 # persistence layer
      store.go               # TopicStore interface
      json_store.go          # JSONTopicStore — reads data/topics.json at startup
    config/
      config.go              # LoadDotEnv, MustToken
    apperrors/
      errors.go              # sentinel errors (ErrUnknownCategory, ErrInvalidStage, ErrNoTopics)
  cli/                       # DEPRECATED — terminal quiz interface, not used by the bot
    main.go                  # runCLI(); shares quiz + storage packages with the bot
  data/
    topics.json              # all 263 quiz topics (source of truth)
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
- Calls the quiz layer to filter topics and build answer options.
- Renders messages and inline keyboards, then sends or edits them via `sendOrEdit`.
- Logs errors and sends user-friendly fallback messages on failure.

**Key types:**
```go
type Bot struct {
    store    storage.TopicStore
    sessions SessionStore
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

Responsible for loading and serving topic data.

**Interface:**
```go
type TopicStore interface {
    All() []quiz.Topic
    ByCategory(cat string) []quiz.Topic
}
```

**Current implementation — `JSONTopicStore`:**
- Reads `data/topics.json` once at startup into memory.
- `ByCategory("")` returns all topics.
- The file is never written at runtime; it is the authoritative topic catalogue.
- A different implementation (database, remote API) can be swapped in by satisfying the `TopicStore` interface — no other layer changes.

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
```

---

## Data Flow

```
Telegram ──long-poll──► internal/bot
                              │
                     reads/writes SessionStore (in-memory)
                              │
                         calls internal/quiz
                              │
                    reads via TopicStore interface
                              │
                     internal/storage/JSONTopicStore
                              │
                         data/topics.json
```

---

## Error Policy

| Situation | Behaviour |
|---|---|
| Missing token or unreadable `topics.json` at startup | `slog.Error` + `os.Exit(1)` (fail fast) |
| Unknown category or invalid stage in a callback | Log error with `chatID` + `err`, send user-friendly fallback message |
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
cp .env.example .env       # add your TELEGRAM_BOT_TOKEN
go run ./cmd/bot           # start the bot
```

Optional environment variables:

| Variable | Default | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | (required) | Token from @BotFather |
| `TOPICS_PATH` | `data/topics.json` | Path to the topic catalogue |
| `LOG_LEVEL` | `info` | `debug` or `info` |
| `LOG_FORMAT` | `text` | `text` or `json` |
