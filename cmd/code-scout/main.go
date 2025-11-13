package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "code-scout",
	Short: "Code Scout - Semantic code search with dual-model embeddings",
	Long: `Code Scout is a CLI tool for semantic code search using dual-model embeddings.
It provides AI coding agents with deep codebase understanding by embedding both
code and documentation into a local vector database.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
