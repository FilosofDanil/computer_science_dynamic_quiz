package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadDotEnv reads key=value pairs from path and sets them via os.Setenv.
// Blank lines and lines starting with # are skipped.
// Already-set environment variables are not overwritten.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

// MustToken returns the TELEGRAM_BOT_TOKEN environment variable or exits
// with a helpful message if it is not set.
func MustToken() string {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: TELEGRAM_BOT_TOKEN is not set.")
		fmt.Fprintln(os.Stderr, "Copy .env.example to .env and paste your bot token from @BotFather.")
		os.Exit(1)
	}
	return token
}

// MustDatabaseURL returns the DATABASE_URL environment variable or exits with
// a helpful message if it is not set. The value is a standard PostgreSQL
// connection string, e.g. postgres://user:pass@host:5432/dbname?sslmode=disable.
func MustDatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL is not set.")
		fmt.Fprintln(os.Stderr, "Copy .env.example to .env and set your PostgreSQL connection string.")
		fmt.Fprintln(os.Stderr, "Example: postgres://user:pass@localhost:5432/quizbot?sslmode=disable")
		os.Exit(1)
	}
	return url
}

// ClaudeAPIKey returns the ANTHROPIC_API_KEY environment variable. It is
// optional: when empty, AI-assisted test generation is disabled and the rest of
// the bot keeps working.
func ClaudeAPIKey() string {
	return os.Getenv("ANTHROPIC_API_KEY")
}

// ClaudeModel returns the ANTHROPIC_MODEL override, or an empty string to let
// the AI client choose its default model.
func ClaudeModel() string {
	return os.Getenv("ANTHROPIC_MODEL")
}
