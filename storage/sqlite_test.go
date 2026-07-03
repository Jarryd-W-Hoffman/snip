package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
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
		t.Fatalf("failed to create schema: %v", err)
	}

	if _, err := db.Exec(`ALTER TABLE snippets ADD COLUMN usage_count INTEGER DEFAULT 0;`); err != nil {
		t.Fatalf("failed to add usage_count: %v", err)
	}
	if _, err := db.Exec(`ALTER TABLE snippets ADD COLUMN last_used_at DATETIME;`); err != nil {
		t.Fatalf("failed to add last_used_at: %v", err)
	}

	return &Storage{db: db}
}

func snippet(name, command, desc string, tags ...string) Snippet {
	return Snippet{
		Name:        name,
		Command:     command,
		Description: desc,
		Tags:        tags,
	}
}

func TestUpsertAndLoad(t *testing.T) {
	s := newTestStorage(t)

	err := s.Upsert(snippet("hello", "echo hi", "a greeting"))
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	snippets, err := s.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}
	got := snippets[0]
	if got.Name != "hello" || got.Command != "echo hi" || got.Description != "a greeting" {
		t.Errorf("unexpected snippet: %+v", got)
	}
	if len(got.Tags) != 0 {
		t.Errorf("expected no tags, got %v", got.Tags)
	}
	if got.UsageCount != 0 {
		t.Errorf("expected usage_count 0, got %d", got.UsageCount)
	}
}

func TestUpsertWithTags(t *testing.T) {
	s := newTestStorage(t)

	err := s.Upsert(snippet("ls", "ls -la", "list files", "files", "utils"))
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	snippets, err := s.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}
	got := snippets[0]
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", got.Tags)
	}
	if got.Tags[0] != "files" || got.Tags[1] != "utils" {
		t.Errorf("unexpected tags: %v", got.Tags)
	}
}

func TestUpsertUpdate(t *testing.T) {
	s := newTestStorage(t)

	s.Upsert(snippet("x", "old", "old desc", "tag1"))
	s.Upsert(snippet("x", "new", "new desc", "tag2"))

	snippets, _ := s.Load()
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet after upsert, got %d", len(snippets))
	}
	got := snippets[0]
	if got.Command != "new" || got.Description != "new desc" {
		t.Errorf("expected updated fields, got %+v", got)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "tag2" {
		t.Errorf("expected tags replaced with [tag2], got %v", got.Tags)
	}
}

func TestLoadOrderByUsage(t *testing.T) {
	s := newTestStorage(t)

	s.Upsert(snippet("a", "a", ""))
	s.Upsert(snippet("b", "b", ""))
	s.Upsert(snippet("c", "c", ""))

	s.IncrementUsage("c")
	s.IncrementUsage("c")
	s.IncrementUsage("a")

	snippets, _ := s.Load()
	if len(snippets) != 3 {
		t.Fatalf("expected 3 snippets, got %d", len(snippets))
	}

	names := make([]string, 3)
	for i, sn := range snippets {
		names[i] = sn.Name
	}

	expected := []string{"c", "a", "b"}
	for i, n := range expected {
		if names[i] != n {
			t.Errorf("position %d: expected %s, got %s", i, n, names[i])
		}
	}

	if snippets[0].UsageCount != 2 {
		t.Errorf("expected c usage=2, got %d", snippets[0].UsageCount)
	}
	if snippets[1].UsageCount != 1 {
		t.Errorf("expected a usage=1, got %d", snippets[1].UsageCount)
	}
}

func TestGetByName(t *testing.T) {
	s := newTestStorage(t)
	s.Upsert(snippet("foo", "echo foo", "the foo cmd", "bar", "baz"))

	got, err := s.GetByName("foo")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got.Name != "foo" || got.Command != "echo foo" || got.Description != "the foo cmd" {
		t.Errorf("unexpected snippet: %+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "bar" || got.Tags[1] != "baz" {
		t.Errorf("unexpected tags: %v", got.Tags)
	}
	if got.UsageCount != 0 {
		t.Errorf("expected usage_count 0, got %d", got.UsageCount)
	}
}

func TestGetByNameNotFound(t *testing.T) {
	s := newTestStorage(t)

	_, err := s.GetByName("nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByNameWithTags(t *testing.T) {
	s := newTestStorage(t)
	s.Upsert(snippet("t", "true", "", "tag1", "tag2"))

	got, err := s.GetByName("t")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", got.Tags)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStorage(t)
	s.Upsert(snippet("del", "del", ""))

	err := s.Delete("del")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = s.GetByName("del")
	if err != ErrNotFound {
		t.Fatal("expected snippet to be gone after delete")
	}

	snippets, _ := s.Load()
	if len(snippets) != 0 {
		t.Fatalf("expected 0 snippets after delete, got %d", len(snippets))
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestStorage(t)

	err := s.Delete("nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteCascadeTags(t *testing.T) {
	s := newTestStorage(t)
	s.Upsert(snippet("cascade", "true", "", "mytag"))

	s.Delete("cascade")

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM snippet_tags`).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 snippet_tags rows after cascade, got %d", count)
	}
}

func TestIncrementUsage(t *testing.T) {
	s := newTestStorage(t)
	s.Upsert(snippet("inc", "inc", ""))

	err := s.IncrementUsage("inc")
	if err != nil {
		t.Fatalf("IncrementUsage failed: %v", err)
	}

	got, _ := s.GetByName("inc")
	if got.UsageCount != 1 {
		t.Errorf("expected usage_count 1, got %d", got.UsageCount)
	}

	s.IncrementUsage("inc")
	got, _ = s.GetByName("inc")
	if got.UsageCount != 2 {
		t.Errorf("expected usage_count 2, got %d", got.UsageCount)
	}
}

func TestIncrementUsageNotFound(t *testing.T) {
	s := newTestStorage(t)

	err := s.IncrementUsage("nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStorageClose(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	s := &Storage{db: db}

	if err := s.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	_ = s.Close()
}

func TestNewStorageFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Override both so the test works even when XDG_CONFIG_HOME is set (e.g. CI).
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	s, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer s.Close()

	fi, err := os.Stat(filepath.Join(tmpDir, ".config", "snip", "snippets.db"))
	if err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
	if fi.Size() == 0 {
		t.Error("expected non-empty db file")
	}
}
