package cmd

import (
	"fmt"
	"os"
	"snip/storage"

	"github.com/spf13/cobra"
)

var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show usage metrics and snippet analytics",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		snippets, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error loading metrics: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("📊 SNIP TELEMETRY DASHBOARD")
		fmt.Println("======================================")

		var totalRuns int
		var usedCount int

		for _, s := range snippets {
			totalRuns += s.UsageCount
			if s.UsageCount > 0 {
				usedCount++
			}
		}

		fmt.Printf("📦 Total Saved Snippets : %d\n", len(snippets))
		fmt.Printf("🚀 Total Executions      : %d\n", totalRuns)
		fmt.Printf("🎯 Active Rotation Ratio : %d%%\n\n", (usedCount*100)/max(1, len(snippets)))

		fmt.Println("🔥 Top 5 Most Frequently Used:")
		fmt.Println("--------------------------------------")
		
		// Print out up to top 5 items (Load() is already sorted by usage desc)
		displayLimit := min(5, len(snippets))
		for i := 0; i < displayLimit; i++ {
			s := snippets[i]
			fmt.Printf("  %-15s | Runs: %-3d | %s\n", s.Name, s.UsageCount, s.Description)
		}
	},
}

func init() {
	RootCmd.AddCommand(StatsCmd)
}