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
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		targetSnippet, err := store.GetByName(name)
		if err == storage.ErrNotFound {
			fmt.Fprintf(os.Stderr, "❌ Error: No snippet found with the name '%s'.\n", name)
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading snippet: %v\n", err)
			os.Exit(1)
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
					fmt.Fprintf(os.Stderr, "❌ Error reading input: %v\n", err)
					os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "❌ Error executing command shortcut: %v\n", err)
			os.Exit(1)
		}

		if err := store.IncrementUsage(name); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Telemetry Warning: %v\n", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(RunCmd)
}