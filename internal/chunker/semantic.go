package chunker

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jlanders/code-scout/internal/parser"
)

// SemanticChunker uses Tree-sitter to create semantic chunks
type SemanticChunker struct {
	parser *parser.Parser
}

// NewSemantic creates a new semantic chunker for Go files
func NewSemantic() (*SemanticChunker, error) {
	p, err := parser.NewGoParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &SemanticChunker{
		parser: p,
	}, nil
}

// ChunkFile splits a Go file into semantic chunks (functions, methods, types)
func (s *SemanticChunker) ChunkFile(filePath, language string) ([]Chunk, error) {
	// Only support Go for now
	if language != "go" {
		return nil, fmt.Errorf("semantic chunker only supports Go files, got: %s", language)
	}

	// Read the source file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract semantic chunks using Tree-sitter
	extractor := parser.NewExtractor(s.parser, sourceCode)
	parserChunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to extract chunks: %w", err)
	}

	// Convert parser chunks to chunker chunks
	chunks := make([]Chunk, 0, len(parserChunks))
	for _, pc := range parserChunks {
		chunk := Chunk{
			ID:        uuid.New().String(),
			FilePath:  filePath,
			LineStart: pc.StartLine,
			LineEnd:   pc.EndLine,
			Language:  language,
			Code:      pc.Content,
			ChunkType: string(pc.Type),
			Name:      pc.Name,
			Metadata:  pc.Metadata,
		}

		// Add receiver for methods
		if pc.Receiver != "" {
			if chunk.Metadata == nil {
				chunk.Metadata = make(map[string]string)
			}
			chunk.Metadata["receiver"] = pc.Receiver
		}

		// Add signature for functions/methods
		if pc.Signature != "" {
			if chunk.Metadata == nil {
				chunk.Metadata = make(map[string]string)
			}
			chunk.Metadata["signature"] = pc.Signature
		}

		// Add doc comment if present
		if pc.DocComment != "" {
			if chunk.Metadata == nil {
				chunk.Metadata = make(map[string]string)
			}
			chunk.Metadata["doc_comment"] = pc.DocComment
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
