package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDaemonPID_NotExist(t *testing.T) {
	// Test with non-existent PID file
	pid, running := getDaemonPID("/nonexistent/path/daemon.pid")
	if running {
		t.Error("Expected daemon to not be running when PID file doesn't exist")
	}
	if pid != 0 {
		t.Errorf("Expected PID 0, got %d", pid)
	}
}

func TestGetDaemonPID_InvalidContent(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	// Write invalid PID content
	if err := os.WriteFile(pidPath, []byte("not-a-number"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, running := getDaemonPID(pidPath)
	if running {
		t.Error("Expected daemon to not be running with invalid PID content")
	}
	if pid != 0 {
		t.Errorf("Expected PID 0, got %d", pid)
	}
}

func TestGetDaemonPID_NonExistentProcess(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	// Write a PID that definitely doesn't exist (999999)
	if err := os.WriteFile(pidPath, []byte("999999"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, running := getDaemonPID(pidPath)
	if running {
		t.Error("Expected daemon to not be running with non-existent process PID")
	}
	// PID should still be parsed
	if pid == 0 {
		t.Error("Expected non-zero PID from file content")
	}
}

func TestShouldSkipFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a scanner (nil is okay for this test since we're just checking file types)
	if shouldSkipFile(dirPath, nil) != true {
		t.Error("Expected directories to be skipped")
	}
}

func TestShouldSkipFile_HiddenFile(t *testing.T) {
	tmpDir := t.TempDir()
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if shouldSkipFile(hiddenFile, nil) != true {
		t.Error("Expected hidden files to be skipped")
	}
}

func TestShouldSkipFile_DeletedFile(t *testing.T) {
	// Test that deleted files don't cause skipping
	// (the file doesn't exist, so os.Stat will fail)
	if shouldSkipFile("/nonexistent/file.go", nil) != false {
		t.Error("Expected non-existent files to not be skipped (might be deleted)")
	}
}
