package cmd

import (
	"testing"

	"github.com/Jarryd-W-Hoffman/snip/storage"
)

func TestStatsCmd_Empty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := StatsCmd.RunE(StatsCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatsCmd_WithSnippets(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s, err := storage.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	s.Upsert(storage.Snippet{Name: "a", Command: "echo a"})
	s.Upsert(storage.Snippet{Name: "b", Command: "echo b"})
	s.Close()

	err = StatsCmd.RunE(StatsCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
