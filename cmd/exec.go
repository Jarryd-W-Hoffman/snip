package cmd

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var varRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// extractVariables returns deduplicated variable names found in {{var}} placeholders.
func extractVariables(command string) []string {
	matches := varRe.FindAllStringSubmatch(command, -1)
	if len(matches) == 0 {
		return nil
	}
	var vars []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if !seen[match[1]] {
			seen[match[1]] = true
			vars = append(vars, match[1])
		}
	}
	return vars
}

// substituteAll replaces all {{var}} placeholders with their mapped values.
func substituteAll(command string, replacements map[string]string) string {
	result := command
	for placeholder, val := range replacements {
		result = strings.ReplaceAll(result, placeholder, val)
	}
	return result
}

// buildExecCommand creates an exec.Cmd for running the command in a platform-appropriate subshell.
func buildExecCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command)
	}
	return exec.Command("bash", "-c", command)
}
