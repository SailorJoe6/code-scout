package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlanders/code-scout/internal/embeddings"
)

type fakeEmbeddingClient struct {
	offset float64
}

func (f *fakeEmbeddingClient) Embed(text string) ([]float64, error) {
	vecs, err := f.EmbedMany([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (f *fakeEmbeddingClient) EmbedMany(texts []string) ([][]float64, error) {
	vectors := make([][]float64, len(texts))
	for i, text := range texts {
		vectors[i] = fakeVector(text, f.offset)
	}
	return vectors, nil
}

func fakeVector(text string, offset float64) []float64 {
	vec := make([]float64, 3584)
	var total float64
	for _, r := range text {
		total += float64(int(r))
	}
	vec[0] = float64(len(text)) + offset
	vec[1] = total + offset
	return vec
}

func installFakeEmbeddings(t *testing.T) {
	codeClient := &fakeEmbeddingClient{offset: 1}
	docsClient := &fakeEmbeddingClient{offset: 1000}
	prevCode := newCodeEmbeddingClient
	prevDocs := newDocsEmbeddingClient
	newCodeEmbeddingClient = func() embeddings.Client { return codeClient }
	newDocsEmbeddingClient = func() embeddings.Client { return docsClient }
	t.Cleanup(func() {
		newCodeEmbeddingClient = prevCode
		newDocsEmbeddingClient = prevDocs
	})
}

func TestIndexAndSearchEndToEnd(t *testing.T) {
	installFakeEmbeddings(t)
	workDir := t.TempDir()
	writeTestFile(t, workDir, "main.go", `package main

func Add(a, b int) int {
	return a + b
}
`)
	writeTestFile(t, workDir, "README.md", `# Project Docs

## Architecture Overview

This section explains the architecture.
`)
	writeTestFile(t, workDir, "notes.txt", "some text file")

	runInDir(t, workDir, func() error {
		indexCmd.Flags().Set("workers", "2")
		indexCmd.Flags().Set("batch-size", "2")
		return indexCmd.RunE(indexCmd, []string{})
	})

	docs := runSearchJSON(t, workDir, "architecture overview", modeDocs)
	if docs.Mode != string(modeDocs) {
		t.Fatalf("expected docs mode, got %s", docs.Mode)
	}
	if len(docs.Results) == 0 {
		t.Fatalf("expected documentation results")
	}
	if !containsFile(docs.Results, "README.md", "docs") {
		t.Fatalf("expected README docs result, got %+v", docs.Results)
	}

	code := runSearchJSON(t, workDir, "add", modeCode)
	if code.Mode != string(modeCode) {
		t.Fatalf("expected code mode, got %s", code.Mode)
	}
	if len(code.Results) == 0 {
		t.Fatalf("expected code results")
	}
	if !containsFile(code.Results, "main.go", "code") {
		t.Fatalf("expected main.go code result, got %+v", code.Results)
	}

	hybrid := runSearchJSON(t, workDir, "architecture overview", modeHybrid)
	if hybrid.Mode != string(modeHybrid) {
		t.Fatalf("expected hybrid mode, got %s", hybrid.Mode)
	}
	var foundDocs, foundCode bool
	for _, res := range hybrid.Results {
		if strings.HasSuffix(res.FilePath, "README.md") {
			foundDocs = true
		}
		if strings.HasSuffix(res.FilePath, "main.go") {
			foundCode = true
		}
	}
	if !foundDocs || !foundCode {
		t.Fatalf("hybrid search did not include both docs and code results")
	}
}

func runSearchJSON(t *testing.T, dir, query string, mode searchMode) searchResponse {
	t.Helper()
	prevJSON := jsonOutput
	prevLimit := limitFlag
	prevCode := codeMode
	prevDocs := docsMode
	prevHybrid := hybridMode
	jsonOutput = true
	limitFlag = 5
	codeMode = mode == modeCode
	docsMode = mode == modeDocs
	hybridMode = mode == modeHybrid
	defer func() {
		jsonOutput = prevJSON
		limitFlag = prevLimit
		codeMode = prevCode
		docsMode = prevDocs
		hybridMode = prevHybrid
	}()
	output := captureStdout(t, func() {
		runInDir(t, dir, func() error {
			return searchCmd.RunE(searchCmd, []string{query})
		})
	})
	var resp searchResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse search output: %v\n%s", err, output)
	}
	return resp
}

type searchResponse struct {
	Query   string         `json:"query"`
	Mode    string         `json:"mode"`
	Results []SearchResult `json:"results"`
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-outC
}

func runInDir(t *testing.T, dir string, fn func() error) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(orig)
	}()
	if err := fn(); err != nil {
		t.Fatalf("command error: %v", err)
	}
}

func writeTestFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file %s: %v", name, err)
	}
}

func containsFile(results []SearchResult, suffix, embeddingType string) bool {
	for _, res := range results {
		if strings.HasSuffix(res.FilePath, suffix) && res.EmbeddingType == embeddingType {
			return true
		}
	}
	return false
}
