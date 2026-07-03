// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"snip/storage"
	"strings"

	"github.com/spf13/cobra"
)

// RunCmd defines the configuration and behavior of the 'snip run' command.
// It requires a unique lookup name to find and directly execute a snippet in a subshell.
var RunCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Execute a saved snippet directly in your terminal shell",
	Long: `Accepts a unique lookup name as an argument, retrieves the corresponding 
command string from storage, and executes it inside a platform-appropriate subshell.`,
	Args: cobra.ExactArgs(1), // Enforces that exactly one argument (the snippet name) is supplied
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			return fmt.Errorf("❌ Error initializing storage configuration: %w", err)
		}
		defer store.Close()

		targetSnippet, err := store.GetByName(name)
		if err == storage.ErrNotFound {
			return fmt.Errorf("❌ Error: No snippet found with the name '%s'", name)
		}
		if err != nil {
			return fmt.Errorf("❌ Error reading snippet: %w", err)
		}

		finalCommand := targetSnippet.Command

		vars := extractVariables(finalCommand)
		if len(vars) > 0 {
			fmt.Printf("📋 Snippet '%s' requires context variable values:\n\n", name)
			replacements := make(map[string]string)
			reader := bufio.NewReader(os.Stdin)

			for _, v := range vars {
				fmt.Printf("➡️ Enter value for [%s]: ", v)
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("❌ Error reading input: %w", err)
				}
				replacements["{{"+v+"}}"] = strings.TrimRight(input, "\r\n")
			}

			finalCommand = substituteAll(finalCommand, replacements)
			fmt.Println()
		}

		execCmd := buildExecCommand(finalCommand)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("❌ Error executing command shortcut: %w", err)
		}

		if err := store.IncrementUsage(name); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Telemetry Warning: %v\n", err)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(RunCmd)
}