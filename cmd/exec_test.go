package cmd

import (
	"runtime"
	"testing"
)

func TestExtractVariables_None(t *testing.T) {
	vars := extractVariables("echo hello world")
	if vars != nil {
		t.Fatalf("expected nil, got %v", vars)
	}
}

func TestExtractVariables_Single(t *testing.T) {
	vars := extractVariables("echo {{name}}")
	if len(vars) != 1 || vars[0] != "name" {
		t.Fatalf("expected [name], got %v", vars)
	}
}

func TestExtractVariables_Multiple(t *testing.T) {
	vars := extractVariables("echo {{greeting}} {{target}}")
	if len(vars) != 2 {
		t.Fatalf("expected 2 vars, got %v", vars)
	}
	if vars[0] != "greeting" || vars[1] != "target" {
		t.Errorf("unexpected vars: %v", vars)
	}
}

func TestExtractVariables_Duplicates(t *testing.T) {
	vars := extractVariables("echo {{name}} {{name}} {{greeting}}")
	if len(vars) != 2 {
		t.Fatalf("expected 2 unique vars, got %v", vars)
	}
}

func TestExtractVariables_EmptyBraces(t *testing.T) {
	vars := extractVariables("echo {{}}")
	if len(vars) != 0 {
		t.Fatalf("expected 0 vars from empty braces, got %v", vars)
	}
}

func TestSubstituteAll_NoMatch(t *testing.T) {
	got := substituteAll("echo hello", map[string]string{"{{x}}": "y"})
	if got != "echo hello" {
		t.Fatalf("expected unchanged, got %q", got)
	}
}

func TestSubstituteAll_Single(t *testing.T) {
	got := substituteAll("echo {{name}}", map[string]string{"{{name}}": "world"})
	if got != "echo world" {
		t.Fatalf("expected 'echo world', got %q", got)
	}
}

func TestSubstituteAll_Multiple(t *testing.T) {
	got := substituteAll("{{a}} and {{b}}", map[string]string{
		"{{a}}": "x",
		"{{b}}": "y",
	})
	if got != "x and y" {
		t.Fatalf("expected 'x and y', got %q", got)
	}
}

func TestSubstituteAll_DuplicatePlaceholders(t *testing.T) {
	got := substituteAll("{{x}} {{x}} {{x}}", map[string]string{"{{x}}": "hi"})
	if got != "hi hi hi" {
		t.Fatalf("expected 'hi hi hi', got %q", got)
	}
}

func TestSubstituteAll_ValueContainsBrace(t *testing.T) {
	got := substituteAll("{{x}}", map[string]string{"{{x}}": "{{y}}"})
	if got != "{{y}}" {
		t.Fatalf("expected '{{y}}', got %q", got)
	}
}

func TestBuildExecCommand_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not applicable on windows")
	}
	cmd := buildExecCommand("echo hi")
	if cmd.Path != "/usr/bin/bash" && cmd.Path != "/bin/bash" {
		t.Errorf("expected bash path, got %q", cmd.Path)
	}
	if len(cmd.Args) != 3 {
		t.Fatalf("expected 3 args, got %v", cmd.Args)
	}
	if cmd.Args[0] != "bash" || cmd.Args[1] != "-c" || cmd.Args[2] != "echo hi" {
		t.Errorf("unexpected args: %v", cmd.Args)
	}
}

func TestBuildExecCommand_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	cmd := buildExecCommand("dir")
	if len(cmd.Args) != 3 {
		t.Fatalf("expected 3 args, got %v", cmd.Args)
	}
	if cmd.Args[0] != "cmd" || cmd.Args[1] != "/c" || cmd.Args[2] != "dir" {
		t.Errorf("unexpected args: %v", cmd.Args)
	}
}
