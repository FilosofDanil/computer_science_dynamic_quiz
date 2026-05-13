# CS Foundations Pyramid — Quiz Game

A terminal-based flashcard quiz that drills you on core computer science concepts across 18 categories — from CPU internals and OS primitives to networking, system design, and LLMs.

---

## Running from the command line

### Option 1 — run the pre-built binary (Windows)

```bat
basics.exe
```

### Option 2 — build & run from source (requires [Go 1.21+](https://go.dev/dl/))

```bash
# clone / enter the repo, then:
go run .
```

### Option 3 — build a binary for your platform

```bash
# current platform
go build -o basics .

# Linux (from any OS)
GOOS=linux GOARCH=amd64 go build -o basics-linux .

# macOS (from any OS)
GOOS=darwin GOARCH=arm64 go build -o basics-mac .

# Windows (from any OS)
GOOS=windows GOARCH=amd64 go build -o basics.exe .
```

Then run the produced binary:

```bash
# Unix / macOS
./basics          # or ./basics-linux / ./basics-mac

# Windows (PowerShell / cmd)
.\basics.exe
```

---

## Requirements

| Tool | Version |
|------|---------|
| Go   | 1.21 +  |

No external runtime is needed once the binary is built.

---

## Running as a Telegram bot

The same quiz runs inside Telegram via inline keyboards — no typing required, works on any device.

### Setup

1. Open Telegram, talk to [@BotFather](https://t.me/BotFather), run `/newbot`, and follow the prompts to get a bot token.
2. Copy `.env.example` to `.env` and paste your token:
   ```
   TELEGRAM_BOT_TOKEN=123456:your-token-here
   ```
3. Start the bot server:
   ```bash
   go run . bot
   # or, if you built a binary:
   .\basics.exe bot
   ```
4. Open your bot in Telegram and send `/start`.

### Bot commands

| Command | Effect |
|---------|--------|
| `/start` | Start (or restart) a quiz session |
| `/quit`  | End your current session |

### How it works

- Pick a category from an inline keyboard grid.
- Choose **In order** or **Shuffle**.
- Answer each question by tapping A / B / C / D buttons.
- After each answer the correct answer, overview, and explanation are shown.
- At the end your score is displayed with a **Play again** button.
- Multiple users can play simultaneously; each chat has its own independent session.

> Sessions are kept in memory. Restarting the bot process resets any in-progress quizzes.
