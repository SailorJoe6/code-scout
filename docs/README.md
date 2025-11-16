# Code Scout Documentation

Welcome to the Code Scout documentation. This directory contains comprehensive documentation for AI agents and developers working with Code Scout.

## Documentation Sections

### Design Documentation

The [design/](design/) folder contains detailed technical documentation about Code Scout's architecture and implementation:

- **[Design Overview](design/README.md)** - Index of all design documents
- **[Architecture](design/architecture.md)** - High-level system architecture and design principles
- **[Data Flow](design/data-flow.md)** - How data flows from source code to search results
- **[Components](design/components.md)** - Component responsibilities and interfaces
- **[Semantic Chunking](design/semantic-chunking.md)** - Tree-sitter AST extraction details
- **[Embedding Strategy](design/embedding-strategy.md)** - Embedding generation and deduplication
- **[Vector Storage](design/vector-storage.md)** - LanceDB schema and query patterns
- **[CLI Interface](design/cli-interface.md)** - Command-line interface and workflows
- **[Extension Points](design/extension-points.md)** - How to extend Code Scout

### Reference Material

The [reference_material/](reference_material/) folder contains external documentation and examples used during development.

## Quick Navigation

**For AI Agents:**
- Start with [Architecture](design/architecture.md) for system overview
- Read [Data Flow](design/data-flow.md) to understand the indexing and search pipeline
- See [Components](design/components.md) for detailed component interactions

**For Developers:**
- See [DEVELOPERS.md](../DEVELOPERS.md) for build setup and requirements
- See [QUICKSTART.md](../QUICKSTART.md) for getting started
- See [Extension Points](design/extension-points.md) for adding new languages or features

**For Users:**
- See [README.md](../README.md) for installation and usage
- See [CLI Interface](design/cli-interface.md) for command examples

## Documentation Philosophy

This documentation is optimized for AI agents to understand the codebase structure, design decisions, and implementation patterns. Each document:

- **Focuses on "why" not just "what"** - Explains reasoning behind design decisions
- **Includes code examples** - Shows actual implementation patterns from the codebase
- **References specific files** - Points to exact file paths and line numbers
- **Explains relationships** - Shows how components interact and depend on each other
- **Provides context** - Includes background on technologies used (Tree-sitter, LanceDB, Ollama)

## Contributing to Documentation

When adding new features:

1. Update relevant design documents with implementation details
2. Add examples showing the new functionality
3. Update architecture diagrams if component interactions change
4. Keep file references (paths and line numbers) up to date

## Related Resources

- **[CLAUDE.md](../CLAUDE.md)** - Instructions for AI agents working in this repo
- **[AGENTS.md](../AGENTS.md)** - Guide for AI agents using Code Scout
- **[beads](https://github.com/jlanders/beads)** - Issue tracking system used in this project
