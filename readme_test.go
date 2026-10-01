package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StphnLwnga/go-ollama-chat/config"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// Every setting the config loader reads must appear in the README and in .env.example.
func TestREADMEDocumentsEverySetting(t *testing.T) {
	var keys []string
	record := func(key string) (string, bool) {
		keys = append(keys, key)
		return "", false // unset, so every default applies
	}
	if _, err := config.Load(record); err != nil {
		t.Fatalf("Load with defaults: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("the loader asked for no settings")
	}

	readme := readFile(t, "README.md")
	example := readFile(t, ".env.example")
	for _, key := range keys {
		if !strings.Contains(readme, "`"+key+"`") {
			t.Errorf("README.md does not document the setting %s", key)
		}
		if !strings.Contains(example, key+"=") {
			t.Errorf(".env.example does not list the setting %s", key)
		}
	}
}

// Every registered route must appear in the README, written as it is in the routes table.
func TestREADMEDocumentsEveryRoute(t *testing.T) {
	readme := readFile(t, "README.md")
	for _, r := range routes() {
		documented := strings.Replace(r.pattern, "{$}", "", 1) // "GET /{$}" is written "GET /"
		if !strings.Contains(readme, "`"+documented+"`") {
			t.Errorf("README.md does not document the route %s", documented)
		}
	}
}

// Every folder that holds Go code must appear in the README's project layout.
func TestREADMEDocumentsEveryPackage(t *testing.T) {
	readme := readFile(t, "README.md")
	seen := map[string]bool{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir // .git, .github and other hidden folders
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			seen[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	for dir := range seen {
		if dir == "." {
			continue // the root package is main.go, listed by name
		}
		if !strings.Contains(readme, filepath.ToSlash(dir)+"/") {
			t.Errorf("README.md project layout does not mention %s/", dir)
		}
	}
}
