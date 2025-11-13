# Code Scout

Code Scout is a specialized vector database solution designed to give AI coding agents full awareness of any codebase. By intelligently vectorizing and storing code alongside its documentation, Code Scout enables AI assistants like Claude and Codex to understand and work with your entire project context.

## Overview

Code Scout uses open source embedding models to create semantic representations of your codebase, storing them in a local vector database. Unlike traditional code search tools, Code Scout is specifically adapted to maintain the relationship between code and its markdown documentation, providing AI agents with comprehensive context for better assistance.

## Key Features

- **Open Source Embeddings**: Leverages open source embedding models for code vectorization
- **Code + Documentation**: Specially designed to store code alongside markdown documentation
- **Local Vector Database**: All data stays local - no cloud dependencies
- **AI Agent Optimized**: Built specifically for AI coding assistants like Claude and Codex
- **Full Codebase Awareness**: Enables AI agents to understand the complete context of your project

## Use Cases

- Provide AI coding agents with instant access to your entire codebase
- Enable intelligent code navigation and search
- Help AI assistants understand project architecture and patterns
- Maintain context between code implementation and documentation
- Support code review and refactoring with full project awareness

## Project Status

This is a greenfield project currently in early development.

## Documentation

For AI agents working on this project:
- `AGENTS.md` - Contains workflow instructions for AI agents, including issue tracking with bd (beads)
- `CLAUDE.md` - Symlink to `AGENTS.md` for Claude-specific references

**Note on Symlink**: `CLAUDE.md` is a symbolic link pointing to `AGENTS.md`. This works natively on Unix-like systems (Linux, macOS). On Windows, Developer Mode may need to be enabled for proper symlink support, otherwise the file may appear as a text file containing the target path.

## Getting Started

*Coming soon - installation and usage instructions*

## Contributing

*Coming soon - contribution guidelines*

## License

*To be determined*
