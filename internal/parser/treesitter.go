package parser

import (
	"context"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_scala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
)

// Parser wraps Tree-sitter functionality for parsing source code
type Parser struct {
	parser   *sitter.Parser
	language Language
}

// NewParser creates a new parser configured for the specified language
func NewParser(lang Language) (*Parser, error) {
	parser := sitter.NewParser()

	var tsLang *sitter.Language
	switch lang {
	case LanguageGo:
		tsLang = sitter.NewLanguage(tree_sitter_go.Language())
	case LanguagePython:
		tsLang = sitter.NewLanguage(tree_sitter_python.Language())
	case LanguageJavaScript:
		tsLang = sitter.NewLanguage(tree_sitter_javascript.Language())
	case LanguageTypeScript:
		// TypeScript uses JavaScript parser with TSX support
		tsLang = sitter.NewLanguage(tree_sitter_javascript.Language())
	case LanguageJava:
		tsLang = sitter.NewLanguage(tree_sitter_java.Language())
	case LanguageRust:
		tsLang = sitter.NewLanguage(tree_sitter_rust.Language())
	case LanguageC:
		tsLang = sitter.NewLanguage(tree_sitter_c.Language())
	case LanguageCPP:
		tsLang = sitter.NewLanguage(tree_sitter_cpp.Language())
	case LanguageRuby:
		tsLang = sitter.NewLanguage(tree_sitter_ruby.Language())
	case LanguagePHP:
		tsLang = sitter.NewLanguage(tree_sitter_php.LanguagePHP())
	case LanguageScala:
		tsLang = sitter.NewLanguage(tree_sitter_scala.Language())
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang.String())
	}

	if err := parser.SetLanguage(tsLang); err != nil {
		return nil, fmt.Errorf("failed to set language %s: %w", lang.String(), err)
	}

	return &Parser{
		parser:   parser,
		language: lang,
	}, nil
}

// NewGoParser creates a new parser configured for Go source code
// Deprecated: Use NewParser(LanguageGo) instead
func NewGoParser() (*Parser, error) {
	return NewParser(LanguageGo)
}

// Language returns the language this parser is configured for
func (p *Parser) Language() Language {
	return p.language
}

// Parse parses source code and returns the syntax tree
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
