package chunker

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jlanders/code-scout/internal/parser"
)

// SemanticChunker uses Tree-sitter for code and header-based chunking for docs
type SemanticChunker struct {
	parser          *parser.Parser
	markdownChunker *MarkdownChunker
}

// NewSemantic creates a new semantic chunker
func NewSemantic() (*SemanticChunker, error) {
	p, err := parser.NewGoParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &SemanticChunker{
		parser:          p,
		markdownChunker: NewMarkdownChunker(),
	}, nil
}

// ChunkFile splits a file into semantic chunks based on language type
func (s *SemanticChunker) ChunkFile(filePath, language string) ([]Chunk, error) {
	// Route to appropriate chunker based on language
	var chunks []Chunk
	var err error

	switch language {
	case "markdown", "text", "rst":
		// Documentation files - use markdown chunker
		chunks, err = s.chunkDocumentation(filePath, language)
	case "go", "python":
		// Code files - use tree-sitter
		chunks, err = s.chunkCode(filePath, language)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// chunkDocumentation handles markdown, text, and rst files
func (s *SemanticChunker) chunkDocumentation(filePath, language string) ([]Chunk, error) {
	var chunks []Chunk
	var err error

	if language == "markdown" {
		chunks, err = s.markdownChunker.ChunkMarkdown(filePath)
	} else {
		// For plain text and rst, treat entire file as one chunk
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read file: %w", readErr)
		}

		chunks = []Chunk{{
			ID:        uuid.New().String(),
			FilePath:  filePath,
			LineStart: 1,
			LineEnd:   len(content),
			Language:  language,
			Code:      string(content),
			ChunkType: "document",
			Metadata: map[string]string{
				"filename": filePath,
			},
		}}
	}

	if err != nil {
		return nil, err
	}

	// Set embedding_type to "docs" for all documentation chunks
	for i := range chunks {
		chunks[i].EmbeddingType = "docs"
	}

	return chunks, nil
}

// chunkCode handles Go and Python files with tree-sitter
func (s *SemanticChunker) chunkCode(filePath, language string) ([]Chunk, error) {
	// Only support Go for now
	if language != "go" {
		return nil, fmt.Errorf("code chunker only supports Go files currently, got: %s", language)
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
			ID:            uuid.New().String(),
			FilePath:      filePath,
			LineStart:     pc.StartLine,
			LineEnd:       pc.EndLine,
			Language:      language,
			Code:          pc.Content,
			ChunkType:     string(pc.Type),
			Name:          pc.Name,
			Metadata:      pc.Metadata,
			EmbeddingType: "code", // Code files use code model
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
