package cmd

import (
	"testing"
)

func TestRunCmd_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := RunCmd.RunE(RunCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing snippet")
	}
}

func TestRunCmd_NoArgs(t *testing.T) {
	err := RunCmd.Args(RunCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
}
