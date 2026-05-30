# Deploying the Bot on Railway with Docker

This guide walks you through hosting the Telegram bot on [Railway](https://railway.app) using the included `Dockerfile`.

---

## Prerequisites

| What | Where |
|------|-------|
| A Telegram bot token | Create one via [@BotFather](https://t.me/BotFather) (`/newbot`) |
| A Railway account | [railway.app](https://railway.app) — free tier is sufficient |
| Git repository | The project must be pushed to GitHub (or GitLab / Bitbucket) |

---

## Step 1 — Create a Telegram bot token

1. Open Telegram and start a chat with **@BotFather**.
2. Send `/newbot` and follow the prompts to choose a name and username.
3. Copy the token BotFather gives you — it looks like `123456789:ABCdef...`.  
   Keep it secret; it goes nowhere except the Railway environment variable.

---

## Step 2 — Push the project to GitHub

If the repo is already on GitHub you can skip this step.

```bash
git add .
git commit -m "add Dockerfile and Makefile"
git remote add origin https://github.com/<your-username>/<your-repo>.git
git push -u origin main
```

---

## Step 3 — Create a new Railway project

1. Log in at [railway.app](https://railway.app).
2. Click **New Project** → **Deploy from GitHub repo**.
3. Authorise Railway to access your GitHub account when prompted.
4. Select the repository (`basics` or whatever you named it).

Railway will detect the `Dockerfile` at the root automatically — no extra configuration files are needed.

---

## Step 4 — Add a PostgreSQL database

The bot stores all tests (quiz sets) in PostgreSQL, so it needs a database.

1. In your Railway project, click **+ New** → **Database** → **Add PostgreSQL**.
2. Railway provisions the database and exposes a `DATABASE_URL` connection string for it.
3. The bot creates its `tests` table automatically on first startup — no manual SQL needed.

---

## Step 5 — Set the environment variables

The bot **will not start** without `TELEGRAM_BOT_TOKEN` and `DATABASE_URL`. Add them before the first deploy:

1. In your Railway project, open the bot service's **Variables** tab (left sidebar).
2. Add:
   - **`TELEGRAM_BOT_TOKEN`** — your token from Step 1.
   - **`DATABASE_URL`** — reference the Postgres plugin's variable. In Railway you can add a variable referencing `${{ Postgres.DATABASE_URL }}` so it stays in sync.
3. Optionally add:
   - `LOG_FORMAT` = `json` *(already the Docker default — cleaner logs in Railway)*
   - `LOG_LEVEL` = `info` or `debug`

> **Never commit the token to the repository.** The `.gitignore` already excludes `.env` but double-check before pushing.

---

## Step 6 — Seed the curated tests (one-shot)

The database starts empty. To load the bundled curated tests from `data/topics.json`,
run the migration command once against the production database:

```bash
# Locally, pointing at the Railway Postgres connection string:
DATABASE_URL="<railway-postgres-url>" go run ./cmd/migrate
```

This groups the curated topics by category and inserts one global test per
category. It is idempotent, so re-running it only adds categories that are
missing. Users can then add their own tests through the bot with `/newtest`.

---

## Step 7 — Deploy

Railway triggers a deployment automatically when you push to the tracked branch (`main` by default), or you can kick one off manually:

1. Go to the **Deployments** tab.
2. Click **Deploy Now** (or just push a commit).
3. Watch the build log — you should see the Go build complete and the binary start.

A successful startup looks like:

```
time=... level=INFO msg="bot started"
```

---

## Step 8 — Verify the bot is alive

1. Open Telegram and find your bot by its username.
2. Send `/start` — you should get the welcome reply immediately.

---

## Redeployments and updates

Every `git push` to the tracked branch triggers a new build and zero-downtime swap automatically.  
To force a redeploy without a code change, use **Deployments → Redeploy** in the Railway dashboard.

---

## Local Docker testing (optional)

Before pushing, you can verify the Docker image works on your machine:

```bash
# Build
make docker-build

# Run (PowerShell)
$env:TELEGRAM_BOT_TOKEN="<your-token>"; make docker-run

# Run (bash / WSL)
TELEGRAM_BOT_TOKEN="<your-token>" make docker-run
```

Press `Ctrl+C` to stop the container.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Startup error: `TELEGRAM_BOT_TOKEN is not set` | Variable not added | Add it in Railway **Variables** tab |
| Startup error: `DATABASE_URL is not set` | Variable not added | Add the PostgreSQL plugin and set `DATABASE_URL` (Step 4–5) |
| Startup error: `failed to open test store` | Database unreachable or wrong URL | Verify the Postgres plugin is running and `DATABASE_URL` is correct |
| Bot starts but the test menu is empty | Curated tests not seeded | Run the migration (Step 6), or create a test with `/newtest` |
| Bot starts but does not respond | Wrong token or bot not activated | Double-check the token; send `/start` to the bot first |
| Image build error on `golang:1.26-alpine` | Go 1.26 image not yet published | Change the `FROM` line in `Dockerfile` to `golang:1.24-alpine` |
