package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const metadataFileName = "metadata.json"

// IndexMetadata tracks indexing state
type IndexMetadata struct {
	LastIndexTime time.Time              `json:"last_index_time"`
	FileModTimes  map[string]time.Time   `json:"file_mod_times"` // file path -> modification time
}

// LoadMetadata loads metadata from disk
func (s *LanceDBStore) LoadMetadata() (*IndexMetadata, error) {
	metadataPath := filepath.Join(s.dbDir, metadataFileName)
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty metadata if file doesn't exist
			return &IndexMetadata{
				LastIndexTime: time.Time{},
				FileModTimes:  make(map[string]time.Time),
			}, nil
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata IndexMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.FileModTimes == nil {
		metadata.FileModTimes = make(map[string]time.Time)
	}

	return &metadata, nil
}

// SaveMetadata saves metadata to disk
func (s *LanceDBStore) SaveMetadata(metadata *IndexMetadata) error {
	metadataPath := filepath.Join(s.dbDir, metadataFileName)
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}
