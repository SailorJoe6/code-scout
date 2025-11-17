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
	markdownChunker *MarkdownChunker
}

// NewSemantic creates a new semantic chunker
func NewSemantic() (*SemanticChunker, error) {
	return &SemanticChunker{
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
	case "go", "python", "javascript", "typescript", "java", "rust", "c", "cpp", "ruby", "php", "scala":
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

// chunkCode handles code files with tree-sitter for all supported languages
func (s *SemanticChunker) chunkCode(filePath, language string) ([]Chunk, error) {
	// Read the source file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect language from file path and content
	lang := parser.DetectLanguage(filePath, sourceCode)
	if lang == parser.LanguageUnknown {
		return nil, fmt.Errorf("could not detect language for file: %s", filePath)
	}

	// Create parser for the detected language
	p, err := parser.NewParser(lang)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser for %s: %w", lang.String(), err)
	}

	// Extract semantic chunks using Tree-sitter
	extractor := parser.NewExtractor(p, sourceCode)
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
