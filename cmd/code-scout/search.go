package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jlanders/code-scout/internal/embeddings"
	"github.com/jlanders/code-scout/internal/storage"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	limitFlag  int
	codeMode   bool
	docsMode   bool
	hybridMode bool
)

type searchMode string

const (
	modeCode   searchMode = "code"
	modeDocs   searchMode = "docs"
	modeHybrid searchMode = "hybrid"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the codebase semantically",
	Long: `Search the indexed codebase using semantic similarity.
Returns relevant code chunks with file paths, line numbers, and relevance scores.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		mode, err := resolveSearchMode()
		if err != nil {
			return err
		}

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

		var (
			results      []SearchResult
			totalMatches int
		)

		switch mode {
		case modeHybrid:
			results, totalMatches, err = runHybridSearch(store, query, limitFlag)
		default:
			results, totalMatches, err = runSingleModeSearch(store, query, limitFlag, mode)
		}
		if err != nil {
			return err
		}

		if len(results) > limitFlag && limitFlag > 0 {
			results = results[:limitFlag]
		}

		rerankApplied := false
		for _, result := range results {
			if result.RerankScore != nil {
				rerankApplied = true
				break
			}
		}

		// Format output
		output := map[string]interface{}{
			"query":         query,
			"mode":          string(mode),
			"total_results": totalMatches,
			"returned":      len(results),
			"results":       results,
		}
		if rerankApplied {
			output["rerank"] = map[string]interface{}{
				"model": globalConfig.RerankModel,
				"top_k": rerankTopK(limitFlag),
			}
		}

		if jsonOutput {
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			if rerankApplied {
				fmt.Printf("Found %d unique %s results (reranked by %s, from %d total) for: %s\n\n",
					len(results), string(mode), globalConfig.RerankModel, totalMatches, query)
			} else {
				fmt.Printf("Found %d unique %s results (from %d total) for: %s\n\n",
					len(results), string(mode), totalMatches, query)
			}
			for i, result := range results {
				if result.RerankScore != nil {
					fmt.Printf("%d. %s:%d-%d (vector: %.4f, rerank: %.4f)\n",
						i+1, result.FilePath, result.LineStart, result.LineEnd, result.Score, *result.RerankScore)
				} else {
					fmt.Printf("%d. %s:%d-%d (score: %.4f)\n",
						i+1, result.FilePath, result.LineStart, result.LineEnd, result.Score)
				}
				fmt.Printf("   Language: %s | Source: %s", result.Language, result.EmbeddingType)
				if result.ChunkType != "" {
					fmt.Printf(" | Chunk: %s", result.ChunkType)
				}
				fmt.Println()
				if result.Heading != "" {
					fmt.Printf("   Heading: %s", result.Heading)
					if result.HeadingLevel != "" {
						fmt.Printf(" (level %s)", result.HeadingLevel)
					}
					if result.ParentHeading != "" {
						fmt.Printf(" | Parents: %s", result.ParentHeading)
					}
					fmt.Println()
				}
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
	ChunkID       string   `json:"chunk_id"`
	FilePath      string   `json:"file_path"`
	LineStart     int      `json:"line_start"`
	LineEnd       int      `json:"line_end"`
	Language      string   `json:"language"`
	Code          string   `json:"code"`
	Score         float64  `json:"score"`
	RerankScore   *float64 `json:"rerank_score,omitempty"`
	EmbeddingType string   `json:"embedding_type"`
	ChunkType     string   `json:"chunk_type,omitempty"`
	Heading       string   `json:"heading,omitempty"`
	HeadingLevel  string   `json:"heading_level,omitempty"`
	ParentHeading string   `json:"parent_heading,omitempty"`
}

func resolveSearchMode() (searchMode, error) {
	selectionCount := 0
	var selected searchMode

	if codeMode {
		selectionCount++
		selected = modeCode
	}
	if docsMode {
		selectionCount++
		selected = modeDocs
	}
	if hybridMode {
		selectionCount++
		selected = modeHybrid
	}

	if selectionCount > 1 {
		return "", fmt.Errorf("flags --code, --docs, and --hybrid are mutually exclusive")
	}
	if selectionCount == 0 {
		return modeHybrid, nil
	}
	return selected, nil
}

func runSingleModeSearch(store *storage.LanceDBStore, query string, limit int, mode searchMode) ([]SearchResult, int, error) {
	if limit <= 0 {
		limit = 10
	}

	searchLimit := limit
	rerankTopK := rerankTopK(limit)
	if rerankTopK > searchLimit {
		searchLimit = rerankTopK
	}

	queryEmbedding, err := embedQueryForMode(query, mode)
	if err != nil {
		return nil, 0, err
	}

	filter := filterForMode(mode)
	rawResults, err := store.Search(queryEmbedding, searchLimit, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search %s embeddings: %w", mode, err)
	}

	deduplicated := deduplicateResults(formatResults(rawResults))
	if rerankTopK > 0 {
		reranked, err := rerankResults(deduplicated, query, rerankTopK)
		if err != nil {
			return nil, 0, err
		}
		deduplicated = reranked
	}
	return deduplicated, len(rawResults), nil
}

func runHybridSearch(store *storage.LanceDBStore, query string, limit int) ([]SearchResult, int, error) {
	if limit <= 0 {
		limit = 10
	}

	searchLimit := limit
	rerankTopK := rerankTopK(limit)
	if rerankTopK > searchLimit {
		searchLimit = rerankTopK
	}

	codeEmbedding, err := embedQueryForMode(query, modeCode)
	if err != nil {
		return nil, 0, err
	}
	docsEmbedding, err := embedQueryForMode(query, modeDocs)
	if err != nil {
		return nil, 0, err
	}

	codeResults, err := store.Search(codeEmbedding, searchLimit, filterForMode(modeCode))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search code embeddings: %w", err)
	}

	docsResults, err := store.Search(docsEmbedding, searchLimit, filterForMode(modeDocs))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search documentation embeddings: %w", err)
	}

	formatted := append(formatResults(codeResults), formatResults(docsResults)...)
	deduplicated := deduplicateResults(formatted)
	if rerankTopK > 0 {
		reranked, err := rerankResults(deduplicated, query, rerankTopK)
		if err != nil {
			return nil, 0, err
		}
		deduplicated = reranked
	}

	return deduplicated, len(codeResults) + len(docsResults), nil
}

func embedQueryForMode(query string, mode searchMode) ([]float64, error) {
	var client embeddings.Client
	switch mode {
	case modeDocs:
		client = newDocsEmbeddingClient()
	default:
		client = newCodeEmbeddingClient()
	}

	embedding, err := client.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate %s query embedding: %w", mode, err)
	}
	return embedding, nil
}

func filterForMode(mode searchMode) string {
	switch mode {
	case modeCode:
		return "embedding_type = 'code'"
	case modeDocs:
		return "embedding_type = 'docs'"
	default:
		return ""
	}
}

func formatResults(results []map[string]interface{}) []SearchResult {
	formatted := make([]SearchResult, len(results))
	for i, r := range results {
		formatted[i] = SearchResult{
			ChunkID:       getStringOrDefault(r, "chunk_id", ""),
			FilePath:      getStringOrDefault(r, "file_path", ""),
			LineStart:     getIntOrDefault(r, "line_start", 0),
			LineEnd:       getIntOrDefault(r, "line_end", 0),
			Language:      getStringOrDefault(r, "language", ""),
			Code:          getStringOrDefault(r, "code", ""),
			Score:         getFloat64OrDefault(r, "_distance", 0.0),
			EmbeddingType: getStringOrDefault(r, "embedding_type", ""),
			ChunkType:     getStringOrDefault(r, "chunk_type", ""),
			Heading:       getStringOrDefault(r, "heading", ""),
			HeadingLevel:  getStringOrDefault(r, "heading_level", ""),
			ParentHeading: getStringOrDefault(r, "parent_heading", ""),
		}
	}
	return formatted
}

func rerankTopK(limit int) int {
	if globalConfig == nil || globalConfig.RerankModel == "" {
		return 0
	}
	topK := globalConfig.RerankTopK
	if topK <= 0 {
		topK = limit
	}
	if topK <= 0 {
		topK = 10
	}
	return topK
}

func rerankResults(results []SearchResult, query string, topK int) ([]SearchResult, error) {
	if len(results) == 0 || topK <= 0 {
		return results, nil
	}
	if topK > len(results) {
		topK = len(results)
	}

	client := newRerankClient()
	if client == nil {
		return results, nil
	}

	// Build context texts for each result
	texts := make([]string, topK)
	for i := 0; i < topK; i++ {
		texts[i] = buildRerankText(results[i])
	}

	// Use cross-encoder to rerank
	rerankResults, err := client.Rerank(query, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to rerank results: %w", err)
	}

	// Create reranked slice with scores
	reranked := make([]SearchResult, topK)
	for _, rr := range rerankResults {
		if rr.Index >= 0 && rr.Index < topK {
			reranked[rr.Index] = results[rr.Index]
			reranked[rr.Index].RerankScore = float64Ptr(rr.Score)
		}
	}

	// Sort by rerank score (highest first)
	sort.SliceStable(reranked, func(i, j int) bool {
		if reranked[i].RerankScore == nil || reranked[j].RerankScore == nil {
			return reranked[i].Score < reranked[j].Score
		}
		if *reranked[i].RerankScore == *reranked[j].RerankScore {
			return reranked[i].Score < reranked[j].Score
		}
		return *reranked[i].RerankScore > *reranked[j].RerankScore
	})

	combined := append(reranked, results[topK:]...)
	return combined, nil
}

func buildRerankText(result SearchResult) string {
	var builder strings.Builder
	if result.FilePath != "" {
		builder.WriteString("File: ")
		builder.WriteString(result.FilePath)
		builder.WriteString("\n")
	}
	if result.Language != "" {
		builder.WriteString("Language: ")
		builder.WriteString(result.Language)
		builder.WriteString("\n")
	}
	if result.ChunkType != "" {
		builder.WriteString("Chunk: ")
		builder.WriteString(result.ChunkType)
		builder.WriteString("\n")
	}
	if result.Heading != "" {
		builder.WriteString("Heading: ")
		builder.WriteString(result.Heading)
		if result.HeadingLevel != "" {
			builder.WriteString(" (level ")
			builder.WriteString(result.HeadingLevel)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}
	if result.ParentHeading != "" {
		builder.WriteString("Parents: ")
		builder.WriteString(result.ParentHeading)
		builder.WriteString("\n")
	}

	// Truncate code to stay under model's 512 token limit (~1200 chars with metadata)
	code := result.Code
	if len(code) > 1200 {
		code = code[:1200] + "..."
	}
	builder.WriteString(code)
	return strings.TrimSpace(builder.String())
}

func float64Ptr(val float64) *float64 {
	return &val
}

// deduplicateResults removes duplicate code chunks, keeping the highest-scoring (lowest distance) entry
func deduplicateResults(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return results
	}

	// Group by code content
	type resultGroup struct {
		bestResult SearchResult
		bestScore  float64
	}

	groups := make(map[string]*resultGroup)

	for _, result := range results {
		if group, exists := groups[result.Code]; exists {
			// Keep the result with the lower distance (better match)
			if result.Score < group.bestScore {
				group.bestResult = result
				group.bestScore = result.Score
			}
		} else {
			// New unique code
			groups[result.Code] = &resultGroup{
				bestResult: result,
				bestScore:  result.Score,
			}
		}
	}

	// Extract deduplicated results
	deduplicated := make([]SearchResult, 0, len(groups))
	for _, group := range groups {
		deduplicated = append(deduplicated, group.bestResult)
	}

	// Sort by score (ascending - lower distance is better)
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score < deduplicated[j].Score
	})

	return deduplicated
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
	searchCmd.Flags().BoolVarP(&codeMode, "code", "c", false, "Search code embeddings only")
	searchCmd.Flags().BoolVarP(&docsMode, "docs", "d", false, "Search documentation embeddings only")
	searchCmd.Flags().BoolVar(&hybridMode, "hybrid", false, "Search both code and documentation embeddings (default)")
	searchCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	searchCmd.Flags().IntVar(&limitFlag, "limit", 10, "Maximum number of results to return")
	rootCmd.AddCommand(searchCmd)
}
