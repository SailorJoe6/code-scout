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

## Embedding Models

Code Scout uses custom-configured Ollama models with persistent context window settings to ensure reliable code embedding without silent truncation.

### Why Custom Model Files?

**Problem**: Ollama's default context window (2048 tokens) is too small for code files and silently discards content beyond this limit. Runtime API parameters must be sent with every request or the server reverts to defaults.

**Solution**: We use custom Modelfiles to persistently configure larger context windows that match each model's native capacity. This ensures:
- No silent truncation of code files
- Consistent behavior across all API calls
- Optimal context usage for embedding generation
- Simpler client code (no per-request parameter management)

### Supported Models

Code Scout uses two embedding models optimized for different purposes:

**nomic-embed-text** (8K context)
- General-purpose text and code embedding
- Excellent for mixed content (code + documentation)
- Context window: 8,192 tokens (~500-600 lines of code)
- Best for: Documentation, comments, shorter code files

**nomic-embed-code** (32K context)
- Specialized code embedding model
- Optimized for Python, Java, Ruby, PHP, JavaScript, Go
- Context window: 32,768 tokens (~2,000-2,500 lines of code)
- Best for: Large code files, entire modules, complex classes

### Setting Up Custom Models

The `ollama-models/` directory contains Modelfiles with pre-configured context windows.

**1. Install Ollama** (if not already installed):
```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Start the Ollama service
brew services start ollama
```

**2. Pull the base models**:
```bash
# Pull nomic-embed-text
ollama pull nomic-embed-text

# Pull nomic-embed-code
ollama pull manutic/nomic-embed-code
```

**3. Create custom models from Modelfiles**:
```bash
# Navigate to repo root
cd /path/to/code_scout

# Create custom nomic-embed-text with 8K context
ollama create code-scout-text -f ollama-models/nomic-embed-text.Modelfile

# Create custom nomic-embed-code with 32K context
ollama create code-scout-code -f ollama-models/nomic-embed-code.Modelfile
```

**4. Verify the models**:
```bash
# List your models
ollama list

# Test the custom models
ollama run code-scout-text
ollama run code-scout-code
```

**5. Use in Code Scout**:
```python
import ollama

# Use custom models with persistent context settings
response = ollama.embeddings(
    model='code-scout-code',  # or 'code-scout-text'
    prompt='your code here'
)
# Context window is automatically set to 32K (no need for options parameter)
```

### Model Selection Guidelines

| File Type | Recommended Model | Reason |
|-----------|------------------|---------|
| `.md`, `.txt`, `.rst` | `code-scout-text` | General text content |
| `.py`, `.js`, `.java`, `.go`, `.rb`, `.php` | `code-scout-code` | Code-optimized embeddings |
| Mixed code+docs | `code-scout-text` | Balanced for both |
| Large files (>500 lines) | `code-scout-code` | 32K context handles large files |

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

## Configuration

Code Scout can be configured to use custom embedding API endpoints, making it compatible with OpenAI-compatible services like OpenRouter, remote GPU hosts, or any other compatible API.

### Configuration Files

Configuration can be specified in two ways:

1. **User-level**: `~/.code-scout/config.json` - Global defaults for all projects
2. **Project-level**: `.code-scout.json` - Project-specific settings (overrides user-level)

### Configuration Format

Create a JSON file with the following structure:

```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "code-scout-code",
  "text_model": "code-scout-text"
}
```

**Fields:**
- `endpoint`: The base URL of the OpenAI-compatible embedding API (no trailing slash)
- `code_model`: Model name to use for code embeddings
- `text_model`: Model name to use for documentation embeddings

### Example Configurations

**Default (Ollama Local)**:
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "code-scout-code",
  "text_model": "code-scout-text"
}
```

**OpenRouter**:
```json
{
  "endpoint": "https://openrouter.ai/api",
  "code_model": "nomic-ai/nomic-embed-text",
  "text_model": "nomic-ai/nomic-embed-text"
}
```

**Remote Ollama Server**:
```json
{
  "endpoint": "http://my-gpu-server:11434",
  "code_model": "code-scout-code",
  "text_model": "code-scout-text"
}
```

### CLI Flag Override

You can override the endpoint for a single command using the `--endpoint` flag:

```bash
# Use a different endpoint for this indexing operation
code-scout index --endpoint http://remote-server:11434

# Use a different endpoint for searching
code-scout search "authentication" --endpoint https://api.example.com
```

### Setup Example

```bash
# Create user-level config directory
mkdir -p ~/.code-scout

# Create default configuration
cat > ~/.code-scout/config.json << 'EOF'
{
  "endpoint": "http://localhost:11434",
  "code_model": "code-scout-code",
  "text_model": "code-scout-text"
}
EOF

# Or create project-specific config
cat > .code-scout.json << 'EOF'
{
  "endpoint": "http://my-team-server:11434",
  "code_model": "custom-code-model",
  "text_model": "custom-text-model"
}
EOF
```

## Getting Started

*Coming soon - installation and usage instructions*

## Contributing

*Coming soon - contribution guidelines*

## License

*To be determined*
