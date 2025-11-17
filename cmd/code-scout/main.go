package main

import (
	"fmt"
	"os"

	"github.com/jlanders/code-scout/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "code-scout",
	Short: "Code Scout - Semantic code search with dual-model embeddings",
	Long: `Code Scout is a CLI tool for semantic code search using dual-model embeddings.
It provides AI coding agents with deep codebase understanding by embedding both
code and documentation into a local vector database.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from file
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check for endpoint flag override
		endpoint, _ := cmd.Flags().GetString("endpoint")
		if endpoint != "" {
			cfg.Endpoint = endpoint
		}

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		globalConfig = cfg
		return nil
	},
}

func main() {
	// Add global flags
	rootCmd.PersistentFlags().String("endpoint", "", "Embedding API endpoint (overrides config file)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
