package config

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// Secret holds a sensitive value. It prints as [redacted], or [not set] when empty, under every fmt verb.
type Secret string

// String implements fmt.Stringer and prints [not set] when the key is missing.
func (s Secret) String() string {
	if s == "" {
		return "[not set]"
	}
	return "[redacted]"
}

// Format implements fmt.Formatter, so every verb, including %d and %x, prints the placeholder.
func (s Secret) Format(f fmt.State, verb rune) {
	io.WriteString(f, s.String())
}

// Config holds every setting the app reads, already parsed into its real type.
type Config struct {
	Port        int
	DBPath      string
	OllamaURL   *url.URL
	OllamaModel string
	GroqAPIKey  Secret // empty means Groq is off
}

// Addr is the listen address for the HTTP server.
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}

// Load reads each setting through lookup, applies defaults to unset values,
// parses each value into its type, and reports every invalid setting together.
// Pass os.LookupEnv in main, and a fake in tests.
func Load(lookup func(key string) (string, bool)) (Config, error) {
	get := func(key, def string) string {
		if v, ok := lookup(key); ok {
			return v
		}
		return def
	}

	var cfg Config
	var errs []error

	rawPort := get("PORT", "8080")
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("PORT %q: must be a number from 1 to 65535", rawPort))
	}
	cfg.Port = port

	cfg.DBPath = get("DB_PATH", "chat.db")
	if cfg.DBPath == "" {
		errs = append(errs, errors.New("DB_PATH: must not be empty"))
	}

	rawURL := get("OLLAMA_BASE_URL", "http://localhost:11434")
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, fmt.Errorf("OLLAMA_BASE_URL %q: must be an http or https URL with a host", rawURL))
	}
	cfg.OllamaURL = u

	cfg.OllamaModel = get("OLLAMA_MODEL", "llama3.2:3b")
	if cfg.OllamaModel == "" {
		errs = append(errs, errors.New("OLLAMA_MODEL: must not be empty"))
	}

	cfg.GroqAPIKey = Secret(get("GROQ_API_KEY", ""))

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
