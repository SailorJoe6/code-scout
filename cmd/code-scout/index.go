package main

import (
	"fmt"
	"os"

	"github.com/jlanders/code-scout/internal/chunker"
	"github.com/jlanders/code-scout/internal/embeddings"
	"github.com/jlanders/code-scout/internal/scanner"
	"github.com/jlanders/code-scout/internal/storage"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index the current directory for semantic search",
	Long: `Scan the current directory for code files, chunk them, generate embeddings,
and store them in a local LanceDB vector database (.code-scout/).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Indexing codebase...")

		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Scan for code files
		s := scanner.New(cwd)
		files, err := s.ScanCodeFiles()
		if err != nil {
			return fmt.Errorf("failed to scan files: %w", err)
		}

		// Count files by language
		langCounts := make(map[string]int)
		for _, f := range files {
			langCounts[f.Language]++
		}

		fmt.Printf("Found %d code files", len(files))
		if len(langCounts) > 0 {
			fmt.Print(" (")
			first := true
			for lang, count := range langCounts {
				if !first {
					fmt.Print(", ")
				}
				fmt.Printf("%d %s", count, lang)
				first = false
			}
			fmt.Print(")")
		}
		fmt.Println()

		// Chunk files
		ch := chunker.New()
		var allChunks []chunker.Chunk

		for _, f := range files {
			chunks, err := ch.ChunkFile(f.Path, f.Language)
			if err != nil {
				return fmt.Errorf("failed to chunk file %s: %w", f.Path, err)
			}
			allChunks = append(allChunks, chunks...)
			fmt.Printf("  - %s: %d chunks\n", f.Path, len(chunks))
		}

		fmt.Printf("Total chunks: %d\n", len(allChunks))

		// Generate embeddings
		fmt.Println("Generating embeddings...")
		embedClient := embeddings.NewOllamaClient()

		// Collect all chunk texts for embedding
		chunkTexts := make([]string, len(allChunks))
		for i, chunk := range allChunks {
			chunkTexts[i] = chunk.Code
		}

		// Generate embeddings (collect them as we go)
		allEmbeddings := make([][]float64, len(chunkTexts))
		for i, text := range chunkTexts {
			embedding, err := embedClient.Embed(text)
			if err != nil {
				return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
			}
			allEmbeddings[i] = embedding
			fmt.Printf("  Generated embedding %d/%d (dim: %d)\n", i+1, len(chunkTexts), len(embedding))
		}

		fmt.Println("Embeddings generated successfully!")

		// Store chunks and embeddings in LanceDB
		fmt.Println("Storing in vector database...")
		store, err := storage.NewLanceDBStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create LanceDB store: %w", err)
		}
		defer store.Close()

		if err := store.StoreChunks(allChunks, allEmbeddings); err != nil {
			return fmt.Errorf("failed to store chunks: %w", err)
		}

		fmt.Println("✓ Indexing complete!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
}
