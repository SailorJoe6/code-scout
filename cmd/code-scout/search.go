package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jlanders/code-scout/internal/embeddings"
	"github.com/jlanders/code-scout/internal/storage"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	limitFlag  int
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the codebase semantically",
	Long: `Search the indexed codebase using semantic similarity.
Returns relevant code chunks with file paths, line numbers, and relevance scores.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Open existing LanceDB store
		store, err := storage.NewLanceDBStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer store.Close()

		// Load existing table
		if err := store.OpenTable(); err != nil {
			return fmt.Errorf("failed to open table: %w (have you run 'code-scout index' first?)", err)
		}

		// Generate query embedding
		embedClient := embeddings.NewOllamaClient()
		queryEmbedding, err := embedClient.Embed(query)
		if err != nil {
			return fmt.Errorf("failed to generate query embedding: %w", err)
		}

		// Search for similar vectors
		results, err := store.Search(queryEmbedding, limitFlag)
		if err != nil {
			return fmt.Errorf("failed to search: %w", err)
		}

		// Format output
		output := map[string]interface{}{
			"query":         query,
			"mode":          "code",
			"total_results": len(results),
			"returned":      len(results),
			"results":       formatResults(results),
		}

		if jsonOutput {
			jsonOutput, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			// Human-readable output
			fmt.Printf("Found %d results for: %s\n\n", len(results), query)
			for i, result := range formatResults(results) {
				fmt.Printf("%d. %s:%d-%d (score: %.4f)\n", i+1, result.FilePath, result.LineStart, result.LineEnd, result.Score)
				fmt.Printf("   Language: %s\n", result.Language)
				// Show first 100 chars of code
				code := result.Code
				if len(code) > 100 {
					code = code[:100] + "..."
				}
				fmt.Printf("   %s\n\n", code)
			}
		}

		return nil
	},
}

type SearchResult struct {
	ChunkID   string  `json:"chunk_id"`
	FilePath  string  `json:"file_path"`
	LineStart int     `json:"line_start"`
	LineEnd   int     `json:"line_end"`
	Language  string  `json:"language"`
	Code      string  `json:"code"`
	Score     float64 `json:"score"`
}

func formatResults(results []map[string]interface{}) []SearchResult {
	formatted := make([]SearchResult, len(results))
	for i, r := range results {
		formatted[i] = SearchResult{
			ChunkID:   getStringOrDefault(r, "chunk_id", ""),
			FilePath:  getStringOrDefault(r, "file_path", ""),
			LineStart: getIntOrDefault(r, "line_start", 0),
			LineEnd:   getIntOrDefault(r, "line_end", 0),
			Language:  getStringOrDefault(r, "language", ""),
			Code:      getStringOrDefault(r, "code", ""),
			Score:     getFloat64OrDefault(r, "_distance", 0.0),
		}
	}
	return formatted
}

func getStringOrDefault(m map[string]interface{}, key string, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func getIntOrDefault(m map[string]interface{}, key string, defaultVal int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

func getFloat64OrDefault(m map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultVal
}

func init() {
	searchCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	searchCmd.Flags().IntVar(&limitFlag, "limit", 10, "Maximum number of results to return")
	rootCmd.AddCommand(searchCmd)
}
