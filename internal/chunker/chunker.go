package chunker

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// Chunk represents a code chunk with metadata
type Chunk struct {
	ID        string            `json:"chunk_id"`
	FilePath  string            `json:"file_path"`
	LineStart int               `json:"line_start"`
	LineEnd   int               `json:"line_end"`
	Language  string            `json:"language"`
	Code      string            `json:"code"`
	ChunkType string            `json:"chunk_type,omitempty"`     // function, method, struct, interface, etc.
	Name      string            `json:"name,omitempty"`           // Name of the function/type
	Metadata  map[string]string `json:"metadata,omitempty"`       // Additional metadata (imports, package, etc.)
}

// Chunker chunks source code files
type Chunker struct{}

// New creates a new Chunker
func New() *Chunker {
	return &Chunker{}
}

// ChunkFile splits a file into chunks at blank line boundaries
func (c *Chunker) ChunkFile(filePath, language string) ([]Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var chunks []Chunk
	var currentLines []string
	var chunkStartLine int = 1
	lineNum := 1

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if line is blank (empty or only whitespace)
		if strings.TrimSpace(line) == "" {
			// If we have accumulated lines, create a chunk
			if len(currentLines) > 0 {
				chunk := Chunk{
					ID:        uuid.New().String(),
					FilePath:  filePath,
					LineStart: chunkStartLine,
					LineEnd:   lineNum - 1,
					Language:  language,
					Code:      strings.Join(currentLines, "\n"),
				}
				chunks = append(chunks, chunk)
				currentLines = nil
			}
			// Next chunk starts after this blank line
			chunkStartLine = lineNum + 1
		} else {
			// Add non-blank line to current chunk
			currentLines = append(currentLines, line)
		}
		lineNum++
	}

	// Don't forget the last chunk if file doesn't end with blank line
	if len(currentLines) > 0 {
		chunk := Chunk{
			ID:        uuid.New().String(),
			FilePath:  filePath,
			LineStart: chunkStartLine,
			LineEnd:   lineNum - 1,
			Language:  language,
			Code:      strings.Join(currentLines, "\n"),
		}
		chunks = append(chunks, chunk)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return chunks, nil
}
