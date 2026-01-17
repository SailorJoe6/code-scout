package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jlanders/code-scout/internal/scanner"
	"github.com/spf13/cobra"
)

const (
	pidFile               = ".code-scout/daemon.pid"
	logFile               = ".code-scout/daemon.log"
	defaultDebounceDelay  = 5 * time.Second
	defaultIndexingDelay  = 2 * time.Second // Wait before first index
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background indexing daemon",
	Long: `Start, stop, and monitor the background embedding daemon that automatically
re-indexes the codebase on file changes.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background indexing daemon",
	Long: `Start the background daemon that watches for file changes and automatically
re-indexes the codebase.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		pidPath := filepath.Join(cwd, pidFile)
		logPath := filepath.Join(cwd, logFile)

		// Check if daemon is already running
		if pid, running := getDaemonPID(pidPath); running {
			return fmt.Errorf("daemon is already running (PID: %d)", pid)
		}

		// Ensure .code-scout directory exists
		if err := os.MkdirAll(filepath.Join(cwd, ".code-scout"), 0755); err != nil {
			return fmt.Errorf("failed to create .code-scout directory: %w", err)
		}

		// Start daemon process
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Build command to run daemon in background
		daemonArgs := []string{"daemon", "run"}

		// Forward endpoint flag if set
		endpoint, _ := cmd.Flags().GetString("endpoint")
		if endpoint != "" {
			daemonArgs = append(daemonArgs, "--endpoint", endpoint)
		}

		// Create log file
		logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logf.Close()

		// Start the daemon process
		daemonCmd := exec.Command(execPath, daemonArgs...)
		daemonCmd.Dir = cwd
		daemonCmd.Stdout = logf
		daemonCmd.Stderr = logf

		// Detach from parent process
		daemonCmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

		if err := daemonCmd.Start(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Write PID file
		pid := daemonCmd.Process.Pid
		if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
			// Try to kill the daemon since we can't track it
			daemonCmd.Process.Kill()
			return fmt.Errorf("failed to write PID file: %w", err)
		}

		fmt.Printf("✓ Daemon started (PID: %d)\n", pid)
		fmt.Printf("  Log file: %s\n", logPath)

		return nil
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background indexing daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		pidPath := filepath.Join(cwd, pidFile)

		pid, running := getDaemonPID(pidPath)
		if !running {
			return fmt.Errorf("daemon is not running")
		}

		// Send SIGTERM to daemon
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find daemon process: %w", err)
		}

		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		// Wait for graceful shutdown (up to 5 seconds)
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			if _, running := getDaemonPID(pidPath); !running {
				break
			}
		}

		// Check if still running
		if _, running := getDaemonPID(pidPath); running {
			// Force kill
			process.Kill()
			fmt.Println("⚠ Daemon did not stop gracefully, forced shutdown")
		}

		// Remove PID file
		os.Remove(pidPath)

		fmt.Println("✓ Daemon stopped")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		pidPath := filepath.Join(cwd, pidFile)
		logPath := filepath.Join(cwd, logFile)

		pid, running := getDaemonPID(pidPath)

		if !running {
			fmt.Println("Status: Not running")
			return nil
		}

		fmt.Printf("Status: Running (PID: %d)\n", pid)
		fmt.Printf("Log file: %s\n", logPath)

		// Show last index time if available
		if logData, err := os.ReadFile(logPath); err == nil {
			lines := strings.Split(string(logData), "\n")
			// Find last meaningful log entry
			for i := len(lines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if line != "" {
					fmt.Printf("Last activity: %s\n", line)
					break
				}
			}
		}

		return nil
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show daemon logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		logPath := filepath.Join(cwd, logFile)

		// Read and display log file
		logData, err := os.ReadFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("log file does not exist (daemon may not have been started)")
			}
			return fmt.Errorf("failed to read log file: %w", err)
		}

		fmt.Print(string(logData))
		return nil
	},
}

// daemonRunCmd is the actual daemon loop (not exposed to users directly)
var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Hidden: true,
	Short:  "Run the daemon (internal use only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Set up logging
		logPath := filepath.Join(cwd, logFile)
		logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer logf.Close()

		logger := log.New(logf, "", log.LstdFlags)
		logger.Println("Daemon starting...")

		// Run the daemon
		if err := runDaemon(cwd, logger); err != nil {
			logger.Printf("Daemon error: %v", err)
			return err
		}

		logger.Println("Daemon stopped")
		return nil
	},
}

// runDaemon implements the actual daemon logic
func runDaemon(rootDir string, logger *log.Logger) error {
	// Create file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Get list of directories to watch (respecting ignore patterns)
	s := scanner.New(rootDir)
	dirs, err := getWatchableDirs(rootDir, s)
	if err != nil {
		return fmt.Errorf("failed to get watchable directories: %w", err)
	}

	// Add all directories to watcher
	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			logger.Printf("Warning: failed to watch directory %s: %v", dir, err)
		}
	}

	logger.Printf("Watching %d directories for changes", len(dirs))

	// Set up debouncing
	var debounceTimer *time.Timer
	pendingIndex := false

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initial indexing after startup delay
	time.AfterFunc(defaultIndexingDelay, func() {
		logger.Println("Running initial index...")
		if err := runIndexing(rootDir, logger); err != nil {
			logger.Printf("Initial indexing failed: %v", err)
		} else {
			logger.Println("Initial indexing complete")
		}
	})

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			logger.Println("Shutting down...")
			return nil

		case <-sigChan:
			logger.Println("Received shutdown signal")
			cancel()
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only care about write, create, remove, and rename events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// Check if file should be indexed (not ignored)
			if shouldSkipFile(event.Name, s) {
				continue
			}

			logger.Printf("File change detected: %s (%s)", event.Name, event.Op)

			// Reset debounce timer
			pendingIndex = true
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(defaultDebounceDelay, func() {
				if !pendingIndex {
					return
				}

				logger.Println("Debounce timeout reached, starting indexing...")
				if err := runIndexing(rootDir, logger); err != nil {
					logger.Printf("Indexing failed: %v", err)
				} else {
					logger.Println("Indexing complete")
				}
				pendingIndex = false
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Printf("Watcher error: %v", err)
		}
	}
}

// getWatchableDirs returns all directories that should be watched
func getWatchableDirs(rootDir string, s *scanner.Scanner) ([]string, error) {
	var dirs []string
	dirs = append(dirs, rootDir)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}

		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories (except .code-scout is already ignored by scanner)
		if strings.HasPrefix(info.Name(), ".") && path != rootDir {
			return filepath.SkipDir
		}

		// Check if directory should be ignored
		// Note: We can't access s.shouldIgnore directly (it's unexported)
		// Instead, we rely on the scanner's ignore patterns when scanning files
		// For now, just skip hidden directories

		if path != rootDir {
			dirs = append(dirs, path)
		}

		return nil
	})

	return dirs, err
}

// shouldSkipFile checks if a file event should be skipped
func shouldSkipFile(path string, s *scanner.Scanner) bool {
	// Check if file should be ignored
	info, err := os.Stat(path)
	if err != nil {
		// File might have been deleted, don't skip
		return false
	}

	// Skip directories
	if info.IsDir() {
		return true
	}

	// Skip hidden files
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	// TODO: Use scanner's ignore logic (need to make it public or add a helper method)
	// For now, just use basic filtering
	return false
}

// runIndexing triggers the indexing process
func runIndexing(rootDir string, logger *log.Logger) error {
	// Use default workers and batch size for daemon
	const defaultWorkers = 10
	const defaultBatchSize = 8

	// Call the shared indexing logic with verbose=false (logs go to daemon log)
	if err := performIndexing(rootDir, defaultWorkers, defaultBatchSize, false); err != nil {
		return err
	}

	return nil
}

// getDaemonPID reads the PID file and checks if the process is running
// Returns (pid, running) where pid is the stored PID and running indicates
// if the process is actually alive.
func getDaemonPID(pidPath string) (int, bool) {
	pidData, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return 0, false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return pid, false // PID found but process lookup failed
	}

	// Send signal 0 to check if process is alive (doesn't actually kill it)
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return pid, false // PID found but process not running
	}

	return pid, true
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonRunCmd)
}
