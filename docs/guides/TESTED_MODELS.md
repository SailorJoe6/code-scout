# Tested Reranker Models

This document tracks reranker model compatibility with TEI and Code Scout.
Update this list whenever you validate new models or TEI versions.

## Verified (Unsandboxed)

| Model | TEI Version | Status | Notes |
|-------|-------------|--------|-------|
| `BAAI/bge-reranker-large` | 1.8.3 | Verified | Starts cleanly; `/rerank` returns scores |
| `BAAI/bge-reranker-v2-m3` | 1.8.3 | Verified | Starts cleanly; `/rerank` returns scores |
| `BAAI/bge-reranker-base` | 1.8.3 | User-reported | User report: starts cleanly and serves `/rerank` |

## Not Yet Verified

| Model | Notes |
|-------|-------|
| `Alibaba-NLP/gte-multilingual-reranker-base` | Pending unsandboxed validation |
| `Alibaba-NLP/gte-reranker-modernbert-base` | Pending unsandboxed validation |
| `cross-encoder/ms-marco-MiniLM-L-6-v2` | Pending unsandboxed validation |
| `jinaai/jina-reranker-v1-turbo-en` | Pending unsandboxed validation |

## Validation Checklist

- Start TEI: `text-embeddings-router --model-id <model> --port 8080`
- Confirm `/health` returns 200
- Confirm `/rerank` returns scores for a simple query
- Record TEI version and any startup warnings
