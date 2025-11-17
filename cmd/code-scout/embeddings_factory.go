package main

import (
	"github.com/jlanders/code-scout/internal/config"
	"github.com/jlanders/code-scout/internal/embeddings"
)

var (
	// globalConfig holds the loaded configuration
	globalConfig *config.Config

	newCodeEmbeddingClient = func() embeddings.Client {
		if globalConfig != nil {
			return embeddings.NewOllamaClientWithEndpoint(globalConfig.Endpoint, globalConfig.CodeModel)
		}
		return embeddings.NewOllamaClient()
	}
	newDocsEmbeddingClient = func() embeddings.Client {
		if globalConfig != nil {
			return embeddings.NewOllamaClientWithEndpoint(globalConfig.Endpoint, globalConfig.TextModel)
		}
		return embeddings.NewOllamaClientWithModel(embeddings.DefaultTextModel)
	}
)
