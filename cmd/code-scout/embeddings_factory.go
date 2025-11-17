package main

import "github.com/jlanders/code-scout/internal/embeddings"

var (
	newCodeEmbeddingClient = func() embeddings.Client {
		return embeddings.NewOllamaClient()
	}
	newDocsEmbeddingClient = func() embeddings.Client {
		return embeddings.NewOllamaClientWithModel(embeddings.DefaultTextModel)
	}
)
