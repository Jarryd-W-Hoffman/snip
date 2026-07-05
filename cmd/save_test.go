package cmd

import (
	"testing"

	"github.com/Jarryd-W-Hoffman/snip/storage"
)

func TestSaveCmd_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCmd.Flags().Set("command", "echo hello"); err != nil {
		t.Fatal(err)
	}
	if err := SaveCmd.Flags().Set("desc", "test description"); err != nil {
		t.Fatal(err)
	}
	if err := SaveCmd.Flags().Set("tag", "test"); err != nil {
		t.Fatal(err)
	}

	err := SaveCmd.RunE(SaveCmd, []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, err := storage.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	sn, err := s.GetByName("test")
	if err != nil {
		t.Fatalf("expected snippet to exist: %v", err)
	}
	if sn.Command != "echo hello" {
		t.Errorf("expected command 'echo hello', got %q", sn.Command)
	}
	if sn.Description != "test description" {
		t.Errorf("expected desc 'test description', got %q", sn.Description)
	}
}

func TestSaveCmd_MissingName(t *testing.T) {
	err := SaveCmd.Args(SaveCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestSaveCmd_EmptyName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCmd.Flags().Set("command", "echo hi"); err != nil {
		t.Fatal(err)
	}

	err := SaveCmd.RunE(SaveCmd, []string{"  "})
	if err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestSaveCmd_TrimsName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCmd.Flags().Set("command", "echo hi"); err != nil {
		t.Fatal(err)
	}

	err := SaveCmd.RunE(SaveCmd, []string{"  my-snippet  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, err := storage.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	sn, err := s.GetByName("my-snippet")
	if err != nil {
		t.Fatalf("expected trimmed name to be saved: %v", err)
	}
	if sn.Name != "my-snippet" {
		t.Errorf("expected name 'my-snippet', got %q", sn.Name)
	}
}
