package main

import (
	"fmt"
	"os"
	"time"

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

		// Initialize storage and load metadata
		store, err := storage.NewLanceDBStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create LanceDB store: %w", err)
		}
		defer store.Close()

		metadata, err := store.LoadMetadata()
		if err != nil {
			return fmt.Errorf("failed to load metadata: %w", err)
		}

		// Scan for code files
		s := scanner.New(cwd)
		allFiles, err := s.ScanCodeFiles()
		if err != nil {
			return fmt.Errorf("failed to scan files: %w", err)
		}

		// Determine which files need indexing
		var filesToIndex []scanner.FileInfo
		var filesToDelete []string
		now := time.Now()

		for _, f := range allFiles {
			lastModTime, exists := metadata.FileModTimes[f.Path]
			if !exists || f.ModTime.After(lastModTime) {
				// File is new or has been modified
				filesToIndex = append(filesToIndex, f)
				if exists {
					// File was previously indexed, mark for deletion
					filesToDelete = append(filesToDelete, f.Path)
				}
			}
		}

		// Check for deleted files (files in metadata but not in scan)
		for filePath := range metadata.FileModTimes {
			found := false
			for _, f := range allFiles {
				if f.Path == filePath {
					found = true
					break
				}
			}
			if !found {
				// File was deleted, mark for deletion
				filesToDelete = append(filesToDelete, filePath)
			}
		}

		// Delete old chunks for changed/deleted files
		if len(filesToDelete) > 0 {
			fmt.Printf("Removing %d changed/deleted file(s) from index...\n", len(filesToDelete))
			if err := store.DeleteChunksByFilePath(filesToDelete); err != nil {
				return fmt.Errorf("failed to delete old chunks: %w", err)
			}
		}

		// If nothing to index, we're done
		if len(filesToIndex) == 0 {
			fmt.Printf("✓ All files up to date. Indexing complete!\n")
			return nil
		}

		// Count files by language
		langCounts := make(map[string]int)
		for _, f := range filesToIndex {
			langCounts[f.Language]++
		}

		fmt.Printf("Indexing %d file(s)", len(filesToIndex))
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

		for _, f := range filesToIndex {
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
		if err := store.StoreChunks(allChunks, allEmbeddings); err != nil {
			return fmt.Errorf("failed to store chunks: %w", err)
		}

		// Update metadata with new file modification times
		metadata.LastIndexTime = now
		for _, f := range filesToIndex {
			metadata.FileModTimes[f.Path] = f.ModTime
		}
		// Remove deleted files from metadata
		for _, filePath := range filesToDelete {
			delete(metadata.FileModTimes, filePath)
		}

		if err := store.SaveMetadata(metadata); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}

		fmt.Println("✓ Indexing complete!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
}
