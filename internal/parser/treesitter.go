package parser

import (
	"context"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Parser wraps Tree-sitter functionality for parsing source code
type Parser struct {
	parser *sitter.Parser
}

// NewGoParser creates a new parser configured for Go source code
func NewGoParser() (*Parser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_go.Language())

	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}

	return &Parser{parser: parser}, nil
}

// Parse parses Go source code and returns the syntax tree
func (p *Parser) Parse(ctx context.Context, sourceCode []byte) (*sitter.Tree, error) {
	tree := p.parser.Parse(sourceCode, nil)
	if tree == nil {
		return nil, nil
	}
	return tree, nil
}

// GetRootNode returns the root node of a parsed tree
func (p *Parser) GetRootNode(tree *sitter.Tree) *sitter.Node {
	return tree.RootNode()
}
