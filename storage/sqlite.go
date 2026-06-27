// Package storage handles data persistence for the snip application
// using a local SQLite flat-file database.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver" // Pure Go SQLite driver registration
)

// Snippet defines the properties of a saved shell shortcut.
type Snippet struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Storage encapsulates the database connection pool state context.
type Storage struct {
	db *sql.DB
}

// NewStorage initializes the cross-platform configuration directory,
// establishes a database connection pool, and ensures the schema exists.
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

	query := `
	CREATE TABLE IF NOT EXISTS snippets (
		name TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		description TEXT
	);`
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize application schema: %w", err)
	}

	return &Storage{db: db}, nil
}

// Load retrieves all saved snippets from the SQLite database.
func (s *Storage) Load() ([]Snippet, error) {
	query := `SELECT name, command, description FROM snippets ORDER BY name ASC;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var snippets []Snippet
	for rows.Next() {
		var snip Snippet
		var desc sql.NullString // Handles potential NULL values cleanly

		if err := rows.Scan(&snip.Name, &snip.Command, &desc); err != nil {
			return nil, fmt.Errorf("failed to parse row entry: %w", err)
		}

		snip.Description = desc.String
		snippets = append(snippets, snip)
	}

	return snippets, nil
}

// Save upserts a collection of snippets into the database. To match the original 
// architecture's bulk write logic without altering the cmd package loop signatures, 
// this clears out the target table and rewrites the provided slice atomically inside a transaction.
func (s *Storage) Save(snippets []Snippet) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin write transaction: %w", err)
	}
	defer tx.Rollback() // Safely rolls back if an execution step fails

	if _, err := tx.Exec(`DELETE FROM snippets;`); err != nil {
		return fmt.Errorf("failed to purge stale data: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO snippets (name, command, description) VALUES (?, ?, ?);`)
	if err != nil {
		return fmt.Errorf("failed to compile prepared statement: %w", err)
	}
	defer stmt.Close()

	for _, snip := range snippets {
		if _, err := stmt.Exec(snip.Name, snip.Command, snip.Description); err != nil {
			return fmt.Errorf("failed to commit row execution context: %w", err)
		}
	}

	return tx.Commit()
}

// Delete removes a single snippet by its unique lookup name.
func (s *Storage) Delete(name string) error {
	_, err := s.db.Exec(`DELETE FROM snippets WHERE name = ?;`, name)
	return err
}

// Upsert adds a snippet or updates its fields if the name already exists.
func (s *Storage) Upsert(snip Snippet) error {
	query := `
	INSERT INTO snippets (name, command, description) 
	VALUES (?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		command = excluded.command,
		description = excluded.description;`
	_, err := s.db.Exec(query, snip.Name, snip.Command, snip.Description)
	return err
}

// Close closes the underlying database pool connection.
func (s *Storage) Close() error {
	return s.db.Close()
}