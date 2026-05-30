# Connecting Claude (Anthropic) for AI Test Generation

The bot can generate quiz tests from a plain-language description using Anthropic's
Claude API. For example, a user sends `/newtest` and then types:

> Create me a test with 10 topics about Africa

Claude returns a structured set of questions, which the bot validates and saves as
one of the user's tests. This guide explains how to get an API key and connect it.

> AI generation is **optional**. Without an API key the bot still runs and serves
> existing tests; only `/newtest` and editing are disabled (the bot tells the user
> so). Quizzes, `/start`, `/mytests`, and delete all keep working.

---

## 1. Create an Anthropic account

1. Go to the [Anthropic Console](https://console.anthropic.com/) and sign up (or log in).
2. Verify your email and complete any required account setup.

## 2. Add billing / credits

The Messages API is a paid product (usage-based, billed per token).

1. In the Console, open **Settings → Billing** (also called **Plans & Billing**).
2. Add a payment method and purchase some prepaid credits (a few dollars is plenty
   for testing — generating one 10-question test costs a fraction of a cent on the
   Haiku model).
3. Optional: set a **monthly spend limit** under billing so costs stay bounded.

## 3. Create an API key

1. In the Console, open **Settings → API Keys**.
2. Click **Create Key**, give it a name (e.g. `quiz-bot`), and copy the value.
   - It looks like `sk-ant-api03-...`.
   - You can only see it once — store it safely (a password manager is ideal).
3. Treat it like a password. Never commit it to git or share it.

## 4. Connect the key to this application

The bot reads the key from the `ANTHROPIC_API_KEY` environment variable.

### Local development

1. Copy the example env file if you haven't already:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` and set your key:
   ```env
   ANTHROPIC_API_KEY=sk-ant-api03-your-key-here
   ```
3. (Optional) Pick a specific model:
   ```env
   ANTHROPIC_MODEL=claude-haiku-4-5-20251001
   ```
4. Run the bot:
   ```bash
   go run ./cmd/bot
   ```
   On startup you should see `level=INFO msg="AI test generation enabled"`.
   If the key is missing you'll instead see a warning that generation is disabled.

> `.env` is already in `.gitignore`. Double-check it is never committed.

### Production (Railway)

1. Open your bot service → **Variables** tab.
2. Add a new variable:
   - **Name:** `ANTHROPIC_API_KEY`
   - **Value:** your key
3. (Optional) Add `ANTHROPIC_MODEL` to override the default model.
4. Redeploy. See `DEPLOY.md` for the full deployment walkthrough.

---

## 5. Choosing a model

Set `ANTHROPIC_MODEL` to control which Claude model is used. If unset, the bot uses
a fast, inexpensive Haiku model that is well-suited to returning strict JSON.

| Use case | Suggested model | Notes |
|---|---|---|
| Default — quick test generation | `claude-haiku-4-5-20251001` | Fast, cheap, reliable JSON |
| Higher-quality questions/explanations | a Sonnet model (e.g. `claude-sonnet-4-6`) | Slower and pricier, deeper reasoning |

Model names and availability change over time, and use exact IDs (the `-latest`
suffix is not a valid model ID). Check the
[Anthropic models documentation](https://docs.anthropic.com/en/docs/about-claude/models)
for the current list, and set `ANTHROPIC_MODEL` to any model your account can access.

---

## 6. How the bot uses the key

- The client lives in [`internal/ai/claude.go`](internal/ai/claude.go). It calls the
  Messages endpoint `https://api.anthropic.com/v1/messages` over HTTPS with the
  headers `x-api-key`, `anthropic-version`, and `content-type`.
- A system prompt instructs Claude to return **only** a JSON object of the form
  `{ "title": ..., "questions": [ { "Name", "Overview", "Question", "Explanation", "Layer" } ] }`.
- The bot strips any stray markdown fences, then validates the JSON (non-empty title,
  at least one question, required fields) before saving it as a user-owned test.
- The user's free-text description is sent as the message content, so a request like
  "Create a test with 10 questions about Africa" yields a 10-question test.

---

## 7. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `/newtest` says AI generation isn't configured | `ANTHROPIC_API_KEY` not set | Set the variable and restart/redeploy |
| Startup log: `ANTHROPIC_API_KEY not set; ... disabled` | Same as above | Set the key |
| "AI generation failed" after sending a description | Invalid key, no credits, or rate limit | Verify the key, check billing/credits, retry |
| "The AI returned an unexpected result" | Model returned malformed JSON | Rephrase the description; consider a stronger model via `ANTHROPIC_MODEL` |
| Authentication errors in logs (HTTP 401) | Wrong or revoked key | Create a new key in the Console and update the variable |
