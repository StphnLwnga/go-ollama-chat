// Package db persists chat messages to a local SQLite file so a conversation
// survives page refreshes and server restarts. All database access lives here
// (the same boundary idea as the ai/ package owning Ollama).
package db

import (
	"database/sql"
	"fmt"

	"github.com/StphnLwnga/go-ollama-chat/ai"

	_ "modernc.org/sqlite" // registers the "sqlite" driver via its init() (side-effect import)
)

// conn is the shared connection pool, set by Open and used by Save/Load.
var conn *sql.DB

// Open opens (or creates) the SQLite file at path and ensures the schema exists.
func Open(path string) error {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("db: open: %w", err)
	}

	// Idempotent: only creates the table the first time.
	_, err = d.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			role    TEXT NOT NULL,
			content TEXT NOT NULL,
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("db: create table: %w", err)
	}

	conn = d
	return nil
}

// Save appends one message (a single turn) to the conversation.
func Save(role, content string) error {
	_, err := conn.Exec(`INSERT INTO messages (role, content) VALUES (?, ?)`, role, content)
	if err != nil {
		return fmt.Errorf("db: save: %w", err)
	}
	return nil
}

// Load returns every saved message, oldest first.
func Load() ([]ai.Message, error) {
	rows, err := conn.Query(`SELECT role, content FROM messages ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("db: load: %w", err)
	}
	defer rows.Close()

	var msgs []ai.Message
	for rows.Next() {
		var m ai.Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, fmt.Errorf("db: scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
