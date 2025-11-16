package chunker

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// Matches markdown headers: # Header, ## Header, ### Header
	headerRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

// MarkdownChunker chunks markdown files by headers
type MarkdownChunker struct{}

// NewMarkdownChunker creates a new MarkdownChunker
func NewMarkdownChunker() *MarkdownChunker {
	return &MarkdownChunker{}
}

// ChunkMarkdown splits a markdown file into sections based on headers (H1-H3)
func (mc *MarkdownChunker) ChunkMarkdown(filePath string) ([]Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var chunks []Chunk
	var currentLines []string
	var chunkStartLine int = 1
	var currentHeading string
	var currentLevel int
	var parentHeadings []string // Stack of parent headings for context
	lineNum := 1

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line is a header
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			headerLevel := len(matches[1]) // Count the #'s
			headerText := strings.TrimSpace(matches[2])

			// If we have accumulated content, create a chunk for it
			if len(currentLines) > 0 {
				chunk := mc.createChunk(filePath, chunkStartLine, lineNum-1, currentLines, currentHeading, currentLevel, parentHeadings)
				chunks = append(chunks, chunk)
				currentLines = nil
			}

			// Update parent heading stack based on level
			// If we're at level 1, clear the stack
			// If we're deeper, maintain parent context
			if headerLevel == 1 {
				parentHeadings = nil
			} else if headerLevel > currentLevel {
				// Going deeper - add current heading as parent
				if currentHeading != "" {
					parentHeadings = append(parentHeadings, currentHeading)
				}
			} else if headerLevel <= currentLevel {
				// Same level or going up - pop parent headings to match level
				targetParents := headerLevel - 2
				if targetParents < 0 {
					targetParents = 0
				}
				if len(parentHeadings) > targetParents {
					parentHeadings = parentHeadings[:targetParents]
				}
			}

			// Start new section
			currentHeading = headerText
			currentLevel = headerLevel
			chunkStartLine = lineNum
			currentLines = append(currentLines, line)
		} else {
			// Add line to current section
			currentLines = append(currentLines, line)
		}

		lineNum++
	}

	// Create chunk for remaining content
	if len(currentLines) > 0 {
		chunk := mc.createChunk(filePath, chunkStartLine, lineNum-1, currentLines, currentHeading, currentLevel, parentHeadings)
		chunks = append(chunks, chunk)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// If we only have one chunk with no heading, mark it as a document
	if len(chunks) == 1 && chunks[0].Name == "" {
		chunks[0].ChunkType = "document"
		if chunks[0].Metadata == nil {
			chunks[0].Metadata = make(map[string]string)
		}
		chunks[0].Metadata["heading"] = filepath.Base(filePath)
	}

	return chunks, nil
}

// createChunk creates a chunk with appropriate metadata
func (mc *MarkdownChunker) createChunk(filePath string, startLine, endLine int, lines []string, heading string, level int, parents []string) Chunk {
	metadata := make(map[string]string)

	if heading != "" {
		metadata["heading"] = heading
		metadata["heading_level"] = fmt.Sprintf("%d", level)
	}

	if len(parents) > 0 {
		metadata["parent_heading"] = strings.Join(parents, " > ")
	}

	chunkType := "section"
	if heading == "" {
		chunkType = "content"
	}

	return Chunk{
		ID:        uuid.New().String(),
		FilePath:  filePath,
		LineStart: startLine,
		LineEnd:   endLine,
		Language:  "markdown",
		Code:      strings.Join(lines, "\n"),
		ChunkType: chunkType,
		Name:      heading,
		Metadata:  metadata,
	}
}
