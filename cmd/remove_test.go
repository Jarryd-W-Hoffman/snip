package cmd

import (
	"testing"

	"github.com/Jarryd-W-Hoffman/snip/storage"
)

func TestRemoveCmd_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := RemoveCmd.RunE(RemoveCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing snippet")
	}
}

func TestRemoveCmd_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s, err := storage.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	s.Upsert(storage.Snippet{Name: "test", Command: "echo hi"})
	s.Close()

	err = RemoveCmd.RunE(RemoveCmd, []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveCmd_NoArgs(t *testing.T) {
	err := RemoveCmd.Args(RemoveCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
}
