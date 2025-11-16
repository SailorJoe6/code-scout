package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jlanders/code-scout/internal/chunker"
	"github.com/jlanders/code-scout/internal/embeddings"
	"github.com/jlanders/code-scout/internal/scanner"
	"github.com/jlanders/code-scout/internal/storage"
	"github.com/spf13/cobra"
)

var (
	workers int
)

// computeContentHash generates a SHA256 hash of the content
func computeContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

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

		// Chunk files that need indexing using semantic chunker
		semanticChunker, err := chunker.NewSemantic()
		if err != nil {
			return fmt.Errorf("failed to create semantic chunker: %w", err)
		}

		var allChunks []chunker.Chunk
		for _, f := range filesToIndex {
			chunks, err := semanticChunker.ChunkFile(f.Path, f.Language)
			if err != nil {
				return fmt.Errorf("failed to chunk file %s: %w", f.Path, err)
			}
			allChunks = append(allChunks, chunks...)
			fmt.Printf("  - %s: %d chunks\n", f.Path, len(chunks))
		}

		fmt.Printf("Total chunks: %d\n", len(allChunks))

		// Separate chunks by embedding type
		var codeChunks, docsChunks []chunker.Chunk
		var codeIndices, docsIndices []int

		for i, chunk := range allChunks {
			if chunk.EmbeddingType == "code" {
				codeChunks = append(codeChunks, chunk)
				codeIndices = append(codeIndices, i)
			} else if chunk.EmbeddingType == "docs" {
				docsChunks = append(docsChunks, chunk)
				docsIndices = append(docsIndices, i)
			}
		}

		fmt.Printf("Code chunks: %d, Docs chunks: %d\n", len(codeChunks), len(docsChunks))

		// Initialize all embeddings array
		allEmbeddings := make([][]float64, len(allChunks))

		// TWO-PASS EMBEDDING GENERATION

		// PASS 1: Code chunks with code-scout-code model
		if len(codeChunks) > 0 {
			fmt.Println("\nPass 1: Generating code embeddings...")
			codeClient := embeddings.NewOllamaClient() // Uses DefaultCodeModel

			codeEmbeddings, err := generateEmbeddingsWithDedup(codeClient, codeChunks, workers)
			if err != nil {
				return fmt.Errorf("failed to generate code embeddings: %w", err)
			}

			// Map code embeddings back to allEmbeddings
			for i, embedding := range codeEmbeddings {
				allEmbeddings[codeIndices[i]] = embedding
			}
		}

		// PASS 2: Docs chunks with code-scout-text model
		if len(docsChunks) > 0 {
			fmt.Println("\nPass 2: Generating documentation embeddings...")
			textClient := embeddings.NewOllamaClientWithModel(embeddings.DefaultTextModel)

			docsEmbeddings, err := generateEmbeddingsWithDedup(textClient, docsChunks, workers)
			if err != nil {
				return fmt.Errorf("failed to generate docs embeddings: %w", err)
			}

			// Pad docs embeddings to match code embedding dimensions (3584)
			// nomic-embed-text produces 768-dim vectors, pad with zeros
			const targetDim = 3584
			for i, embedding := range docsEmbeddings {
				if len(embedding) < targetDim {
					padded := make([]float64, targetDim)
					copy(padded, embedding)
					docsEmbeddings[i] = padded
				}
				allEmbeddings[docsIndices[i]] = docsEmbeddings[i]
			}
		}

		fmt.Println("\nAll embeddings generated successfully!")

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

// generateEmbeddingsWithDedup generates embeddings for chunks with content deduplication
func generateEmbeddingsWithDedup(client *embeddings.OllamaClient, chunks []chunker.Chunk, numWorkers int) ([][]float64, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	// Set default workers
	if numWorkers <= 0 {
		numWorkers = 10
	}

	// Compute content hashes for deduplication
	chunkHashes := make([]string, len(chunks))
	hashToFirstIndex := make(map[string]int)

	for i, chunk := range chunks {
		hash := computeContentHash(chunk.Code)
		chunkHashes[i] = hash

		if _, exists := hashToFirstIndex[hash]; !exists {
			hashToFirstIndex[hash] = i
		}
	}

	uniqueCount := len(hashToFirstIndex)
	duplicateCount := len(chunks) - uniqueCount

	if duplicateCount > 0 {
		fmt.Printf("Found %d duplicate chunks (will skip %d embeddings)\n", duplicateCount, duplicateCount)
	}

	fmt.Printf("Using %d concurrent workers\n", numWorkers)

	// Generate embeddings for unique chunks only
	allEmbeddings := make([][]float64, len(chunks))

	type job struct {
		index int
		text  string
	}

	type result struct {
		index     int
		embedding []float64
		err       error
	}

	// Create jobs for unique chunks only
	jobs := make(chan job, uniqueCount)
	results := make(chan result, uniqueCount)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				embedding, err := client.Embed(j.text)
				results <- result{
					index:     j.index,
					embedding: embedding,
					err:       err,
				}
			}
		}()
	}

	// Send jobs for unique chunks
	for _, firstIdx := range hashToFirstIndex {
		jobs <- job{
			index: firstIdx,
			text:  chunks[firstIdx].Code,
		}
	}
	close(jobs)

	// Close results when workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	completed := 0
	for r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", r.err)
		}
		allEmbeddings[r.index] = r.embedding
		completed++
		if completed == 1 || completed%50 == 0 || completed == uniqueCount {
			fmt.Printf("  Generated %d/%d unique embeddings (dim: %d)\n", completed, uniqueCount, len(r.embedding))
		}
	}

	// Copy embeddings to duplicate chunks
	if duplicateCount > 0 {
		fmt.Printf("Copying embeddings to %d duplicate chunks...\n", duplicateCount)
		for i, hash := range chunkHashes {
			if allEmbeddings[i] == nil {
				firstIdx := hashToFirstIndex[hash]
				allEmbeddings[i] = allEmbeddings[firstIdx]
			}
		}
	}

	return allEmbeddings, nil
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().IntVarP(&workers, "workers", "w", 10, "Number of concurrent workers for embedding generation (default: 10)")
}
