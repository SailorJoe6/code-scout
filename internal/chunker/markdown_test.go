package chunker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownChunker_ChunkMarkdown(t *testing.T) {
	// Create temp file with markdown content
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")

	content := `# Main Title

This is the introduction.

## Section 1

Content for section 1.

### Subsection 1.1

Detailed content here.

### Subsection 1.2

More detailed content.

## Section 2

Content for section 2.

# Another Top Level

New top level section.
`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Chunk the file
	chunker := NewMarkdownChunker()
	chunks, err := chunker.ChunkMarkdown(mdFile)
	if err != nil {
		t.Fatalf("ChunkMarkdown failed: %v", err)
	}

	// Should have 6 chunks (1 H1, 1 intro, 2 H2, 2 H3, 1 H2, 1 H1)
	// Actually: # Main + intro, ## Section 1, ### Subsection 1.1, ### Subsection 1.2, ## Section 2, # Another Top Level
	if len(chunks) < 5 {
		t.Errorf("Expected at least 5 chunks, got %d", len(chunks))
		for i, c := range chunks {
			t.Logf("Chunk %d: %s (lines %d-%d)", i, c.Name, c.LineStart, c.LineEnd)
		}
	}

	// Verify first chunk has heading metadata
	firstChunk := chunks[0]
	if firstChunk.Metadata["heading"] != "Main Title" {
		t.Errorf("Expected heading 'Main Title', got '%s'", firstChunk.Metadata["heading"])
	}
	if firstChunk.Metadata["heading_level"] != "1" {
		t.Errorf("Expected heading_level '1', got '%s'", firstChunk.Metadata["heading_level"])
	}

	// Find a subsection and verify it has parent heading
	for _, chunk := range chunks {
		if chunk.Name == "Subsection 1.1" {
			if chunk.Metadata["heading_level"] != "3" {
				t.Errorf("Subsection 1.1: expected level 3, got %s", chunk.Metadata["heading_level"])
			}
			// Should have Section 1 as parent
			if _, ok := chunk.Metadata["parent_heading"]; !ok {
				t.Errorf("Subsection 1.1 should have parent_heading metadata")
			}
			break
		}
	}
}

func TestMarkdownChunker_NoHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "plain.md")

	content := `This is just plain text.
No headers at all.
Just some content.
`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	chunker := NewMarkdownChunker()
	chunks, err := chunker.ChunkMarkdown(mdFile)
	if err != nil {
		t.Fatalf("ChunkMarkdown failed: %v", err)
	}

	// Should create one chunk for the whole document
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for headerless document, got %d", len(chunks))
	}

	if chunks[0].ChunkType != "document" {
		t.Errorf("Expected chunk_type 'document', got '%s'", chunks[0].ChunkType)
	}
}

func TestMarkdownChunker_ComplexNesting(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "nested.md")

	content := `# Chapter 1

Introduction to chapter 1.

## Part A

Part A content.

### Detail 1

Detail 1 content.

### Detail 2

Detail 2 content.

## Part B

Part B content.

# Chapter 2

New chapter.
`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	chunker := NewMarkdownChunker()
	chunks, err := chunker.ChunkMarkdown(mdFile)
	if err != nil {
		t.Fatalf("ChunkMarkdown failed: %v", err)
	}

	// Verify chunks are created
	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}

	// Verify all chunks have required fields
	for i, chunk := range chunks {
		if chunk.ID == "" {
			t.Errorf("Chunk %d: missing ID", i)
		}
		if chunk.FilePath != mdFile {
			t.Errorf("Chunk %d: wrong file path", i)
		}
		if chunk.Language != "markdown" {
			t.Errorf("Chunk %d: expected language 'markdown', got '%s'", i, chunk.Language)
		}
		if chunk.Code == "" {
			t.Errorf("Chunk %d: empty code content", i)
		}
	}
}
