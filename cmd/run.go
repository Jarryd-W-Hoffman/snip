// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
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

		snippets, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading existing records from database: %v\n", err)
			os.Exit(1)
		}

		var targetSnippet *storage.Snippet
		for _, s := range snippets {
			if s.Name == name {
				targetSnippet = &s
				break
			}
		}

		if targetSnippet == nil {
			fmt.Fprintf(os.Stderr, "❌ Error: No snippet found with the name '%s'.\n", name)
			os.Exit(1)
		}

		finalCommand := targetSnippet.Command

		re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
		matches := re.FindAllStringSubmatch(finalCommand, -1)

		if len(matches) > 0 {
			fmt.Printf("📋 Snippet '%s' requires context variable values:\n\n", name)
			seen := make(map[string]bool)
			replacements := make(map[string]string)

			for _, match := range matches {
				varName := match[1]
				if seen[varName] {
					continue
				}
				seen[varName] = true

				fmt.Printf("➡️ Enter value for [%s]: ", varName)
				var input string
				fmt.Scanln(&input)
				replacements["{{"+varName+"}}"] = input
			}

			for placeholder, val := range replacements {
				finalCommand = strings.ReplaceAll(finalCommand, placeholder, val)
			}
			fmt.Println()
		}

		var execCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			execCmd = exec.Command("cmd", "/c", finalCommand)
		} else {
			execCmd = exec.Command("bash", "-c", finalCommand)
		}

		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		fmt.Printf("🚀 Running snippet '%s' -> %s\n\n", name, finalCommand)
		if err := execCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Error executing command loop: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(RunCmd)
}