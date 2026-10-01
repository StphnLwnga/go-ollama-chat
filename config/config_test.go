package config

import (
	"fmt"
	"strings"
	"testing"
)

// env returns a lookup backed by a map. It stands in for the process environment.
func env(vars map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

// TestLoadAppliesDefaults verifies that Load applies the default values
func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr() != ":8080" {
		t.Errorf("Addr() = %q, want %q", cfg.Addr(), ":8080")
	}
	if cfg.DBPath != "chat.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "chat.db")
	}
	if got := cfg.OllamaURL.String(); got != "http://localhost:11434" {
		t.Errorf("OllamaURL = %q, want %q", got, "http://localhost:11434")
	}
	if cfg.OllamaModel != "llama3.2:3b" {
		t.Errorf("OllamaModel = %q, want %q", cfg.OllamaModel, "llama3.2:3b")
	}
	if cfg.GroqAPIKey != "" {
		t.Error("GroqAPIKey is set, want empty")
	}
}

// TestLoadRejectsInvalidSettings verifies that Load rejects invalid settings
func TestLoadRejectsInvalidSettings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		vars    map[string]string
		wantKey string // the setting the error must name
	}{
		{"port not a number", map[string]string{"PORT": "80a0"}, "PORT"},
		{"port too large", map[string]string{"PORT": "70000"}, "PORT"},
		{"port zero", map[string]string{"PORT": "0"}, "PORT"},
		{"db path set but empty", map[string]string{"DB_PATH": ""}, "DB_PATH"},
		{"url without scheme", map[string]string{"OLLAMA_BASE_URL": "localhost:11434"}, "OLLAMA_BASE_URL"},
		{"url with wrong scheme", map[string]string{"OLLAMA_BASE_URL": "ftp://localhost"}, "OLLAMA_BASE_URL"},
		{"model set but empty", map[string]string{"OLLAMA_MODEL": ""}, "OLLAMA_MODEL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(env(tc.vars))
			if err == nil {
				t.Fatal("Load succeeded, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("error %q does not name %s", err, tc.wantKey)
			}
		})
	}
}

// TestLoadReportsEveryInvalidSetting verifies that Load reports every invalid setting
func TestLoadReportsEveryInvalidSetting(t *testing.T) {
	t.Parallel()
	_, err := Load(env(map[string]string{"PORT": "x", "OLLAMA_MODEL": ""}))
	if err == nil {
		t.Fatal("Load succeeded, want an error")
	}
	for _, key := range []string{"PORT", "OLLAMA_MODEL"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not name %s", err, key)
		}
	}
}

// TestEmptySecretPrintsNotSet verifies that Load reports empty secrets as [not set]
func TestEmptySecretPrintsNotSet(t *testing.T) {
	t.Parallel()
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, verb := range []string{"%v", "%#v"} {
		if got := fmt.Sprintf(verb, cfg.GroqAPIKey); got != "[not set]" {
			t.Errorf("%s printed %q, want %q", verb, got, "[not set]")
		}
	}
}

// TestSecretIsRedactedWhenPrinted verifies that Load redacts the secret when printed
func TestSecretIsRedactedWhenPrinted(t *testing.T) {
	t.Parallel()
	cfg, err := Load(env(map[string]string{"GROQ_API_KEY": "gsk-real-key"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		if out := fmt.Sprintf(verb, cfg); strings.Contains(out, "gsk-real-key") {
			t.Errorf("%s printed the key: %s", verb, out)
		}
	}
	if string(cfg.GroqAPIKey) != "gsk-real-key" {
		t.Error("an explicit conversion must still return the key")
	}
}
