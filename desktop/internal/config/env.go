// Package config loads runtime configuration from environment variables, with optional support
// for a `.env` file. Precedence is: real environment > .env file > built-in default; command-line
// flags (parsed in main) override all of it. No external dependency — the .env parser is a few
// lines and easy to audit.
package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Env var names. Prefixed to avoid clashing with unrelated environment variables.
const (
	EnvPort          = "LETZPLAY_PORT"
	EnvAdminPassword = "LETZPLAY_ADMIN_PASSWORD"
	EnvGuestPassword = "LETZPLAY_GUEST_PASSWORD"
	EnvOpen          = "LETZPLAY_OPEN"
	EnvHeadless      = "LETZPLAY_HEADLESS"
)

// LoadDotEnv reads the first existing path (e.g. "desktop/.env") and sets any variable that is not
// already present in the real environment, so the actual environment always wins over the file.
// Missing files are not an error — .env is optional.
func LoadDotEnv(paths ...string) {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := parseLine(scanner.Text())
			if !ok {
				continue
			}
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
		return // first existing file wins
	}
}

// String returns the env value for key, or def if unset/empty.
func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// Int returns the env value parsed as an int, or def if unset/invalid.
func Int(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// Bool returns the env value parsed as a bool (1/true/yes/on), or def if unset/invalid.
func Bool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return def
}

// parseLine extracts KEY=VALUE, ignoring blanks, comments, and surrounding quotes.
func parseLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	trimmed = strings.TrimPrefix(trimmed, "export ")
	idx := strings.IndexByte(trimmed, '=')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(trimmed[:idx])
	value = strings.TrimSpace(trimmed[idx+1:])
	value = strings.Trim(value, `"'`)
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
