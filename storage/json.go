// Package storage handles the persistence layer for the snip utility,
// managing the reading and writing of snippets to the local filesystem.
package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Snippet represents an individual saved command, including its lookup name,
// the executable text, and context description.
type Snippet struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Storage manages the file operations for saving and retrieving snippets.
// It tracks the absolute path to the data storage file.
type Storage struct {
	FilePath string
}

// NewStorage initializes a new Storage instance. It dynamically resolves 
// the user's standard OS configuration directory, ensures a dedicated 'snip' 
// directory exists, and sets up the path to the snippets JSON file.
func NewStorage() (*Storage, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	snipDir := filepath.Join(configDir, "snip")
	err = os.MkdirAll(snipDir, 0755)
	if err != nil {
		return nil, err
	}

	return &Storage{
		FilePath: filepath.Join(snipDir, "snippets.json"),
	}, nil
}

// Load reads, parses, and returns the collection of Snippets from disk.
// If the target JSON file does not exist yet, it returns an empty slice without error.
func (s *Storage) Load() ([]Snippet, error) {
	if _, err := os.Stat(s.FilePath); errors.Is(err, os.ErrNotExist) {
		return []Snippet{}, nil
	}

	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, err
	}

	var snippets []Snippet
	err = json.Unmarshal(data, &snippets)
	if err != nil {
		return nil, err
	}

	return snippets, nil
}

// Save marshals the provided slice of Snippets into human-readable JSON
// and performs an atomic write to overwrite the existing file on disk.
func (s *Storage) Save(snippets []Snippet) error {
	data, err := json.MarshalIndent(snippets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.FilePath, data, 0644)
}