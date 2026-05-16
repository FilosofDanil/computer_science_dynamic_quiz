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
