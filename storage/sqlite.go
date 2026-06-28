// Package storage handles relational data persistence for the snip application
// using a local SQLite flat-file database.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver" // Pure Go SQLite driver registration
)

// Snippet defines the properties of a saved shell shortcut.
type Snippet struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// Storage encapsulates the database connection pool state context.
type Storage struct {
	db *sql.DB
}

// NewStorage initializes the cross-platform configuration directory,
// establishes a database connection pool, and ensures relational schemas exist.
func NewStorage() (*Storage, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate user config directory: %w", err)
	}

	snipDir := filepath.Join(configDir, "snip")
	if err := os.MkdirAll(snipDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory layout: %w", err)
	}

	dbPath := filepath.Join(snipDir, "snippets.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS snippets (
		name TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS snippet_tags (
		snippet_name TEXT,
		tag_id INTEGER,
		PRIMARY KEY (snippet_name, tag_id),
		FOREIGN KEY (snippet_name) REFERENCES snippets(name) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize relational schema: %w", err)
	}

	return &Storage{db: db}, nil
}

// Load retrieves all saved snippets alongside their compiled tags sorted alphabetically.
func (s *Storage) Load() ([]Snippet, error) {
	query := `
		SELECT s.name, s.command, s.description, GROUP_CONCAT(t.name) as tags
		FROM snippets s
		LEFT JOIN snippet_tags st ON s.name = st.snippet_name
		LEFT JOIN tags t ON st.tag_id = t.id
		GROUP BY s.name
		ORDER BY s.name ASC;`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var snippets []Snippet
	for rows.Next() {
		var snip Snippet
		var desc sql.NullString
		var rawTags sql.NullString

		if err := rows.Scan(&snip.Name, &snip.Command, &desc, &rawTags); err != nil {
			return nil, fmt.Errorf("failed to parse row entry: %w", err)
		}

		snip.Description = desc.String
		if rawTags.Valid && rawTags.String != "" {
			snip.Tags = strings.Split(rawTags.String, ",")
		} else {
			snip.Tags = []string{}
		}

		snippets = append(snippets, snip)
	}

	return snippets, nil
}

// Delete removes a single snippet by its unique lookup name. Foreign key cascades
// handle clearing entries out of the snippet_tags junction table automatically.
func (s *Storage) Delete(name string) error {
	_, err := s.db.Exec(`DELETE FROM snippets WHERE name = ?;`, name)
	return err
}

// Upsert adds a snippet or updates its fields, handling multi-tag associations cleanly inside a transaction.
func (s *Storage) Upsert(snip Snippet) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	querySnippet := `
	INSERT INTO snippets (name, command, description) 
	VALUES (?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		command = excluded.command,
		description = excluded.description;`
	if _, err := tx.Exec(querySnippet, snip.Name, snip.Command, snip.Description); err != nil {
		return fmt.Errorf("failed to upsert base snippet: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM snippet_tags WHERE snippet_name = ?;`, snip.Name); err != nil {
		return fmt.Errorf("failed to clear existing relationships: %w", err)
	}

	for _, tagName := range snip.Tags {
		tagName = strings.TrimSpace(strings.ToLower(tagName))
		if tagName == "" {
			continue
		}

		if _, err := tx.Exec(`INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING;`, tagName); err != nil {
			return fmt.Errorf("failed to ensure tag existence: %w", err)
		}

		var tagID int
		if err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?;`, tagName).Scan(&tagID); err != nil {
			return fmt.Errorf("failed to retrieve tag identity context: %w", err)
		}

		if _, err := tx.Exec(`INSERT INTO snippet_tags (snippet_name, tag_id) VALUES (?, ?);`, snip.Name, tagID); err != nil {
			return fmt.Errorf("failed to link tag junction association: %w", err)
		}
	}

	return tx.Commit()
}

// Close closes the underlying database pool connection.
func (s *Storage) Close() error {
	return s.db.Close()
}