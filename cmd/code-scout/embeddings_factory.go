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
			return embeddings.NewClientWithConfig(globalConfig.Endpoint, globalConfig.APIKey, globalConfig.CodeModel)
		}
		return embeddings.NewClient()
	}
	newDocsEmbeddingClient = func() embeddings.Client {
		if globalConfig != nil {
			return embeddings.NewClientWithConfig(globalConfig.Endpoint, globalConfig.APIKey, globalConfig.TextModel)
		}
		return embeddings.NewClientWithModel(embeddings.DefaultTextModel)
	}
	newRerankClient = func() *embeddings.RerankClient {
		if globalConfig != nil && globalConfig.RerankModel != "" {
			// Use rerank_endpoint if specified, otherwise fall back to main endpoint
			endpoint := globalConfig.RerankEndpoint
			if endpoint == "" {
				endpoint = globalConfig.Endpoint
			}
			return embeddings.NewRerankClient(endpoint, globalConfig.APIKey, globalConfig.RerankModel)
		}
		return nil
	}
)
