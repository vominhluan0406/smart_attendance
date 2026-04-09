package wal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a single attendance log entry stored in the WAL.
type Entry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	BranchID  string    `json:"branch_id"`
	WorkDate  string    `json:"work_date"`
	Method    string    `json:"method"`
	IP        string    `json:"ip"`
	Lat       *float64  `json:"lat,omitempty"`
	Lng       *float64  `json:"lng,omitempty"`
	Synced    bool      `json:"synced"`
	Payload   string    `json:"payload"` // Full LogTimeInput as JSON
}

// Writer appends attendance entries to a local JSONL file.
// If DB is down, entries are preserved here for later processing.
type Writer struct {
	mu      sync.Mutex
	dir     string
	maxSize int64 // max file size before rotation (bytes)
}

// NewWriter creates a WAL writer that stores entries in the given directory.
func NewWriter(dir string) (*Writer, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create WAL dir: %w", err)
	}
	return &Writer{
		dir:     dir,
		maxSize: 50 * 1024 * 1024, // 50MB per file
	}, nil
}

// Append writes an entry to the current WAL file (thread-safe).
func (w *Writer) Append(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	filename := w.currentFile()

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open WAL file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal WAL entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write WAL entry: %w", err)
	}

	return nil
}

// MarkSynced updates an entry's synced status in the WAL file.
// For simplicity, we write a separate .synced file tracking synced IDs.
func (w *Writer) MarkSynced(entryID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	syncFile := filepath.Join(w.dir, "synced.log")
	f, err := os.OpenFile(syncFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", entryID)
	return err
}

// ReadPending returns all WAL entries that haven't been synced to DB yet.
func (w *Writer) ReadPending() ([]Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Load synced IDs
	syncedIDs := w.loadSyncedIDs()

	var pending []Entry
	files, err := filepath.Glob(filepath.Join(w.dir, "wal-*.jsonl"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		entries, err := w.readFile(file)
		if err != nil {
			log.Printf("[wal] ERROR reading %s: %v", file, err)
			continue
		}
		for _, e := range entries {
			if !syncedIDs[e.ID] {
				pending = append(pending, e)
			}
		}
	}

	return pending, nil
}

// PendingCount returns how many entries are waiting to be synced.
func (w *Writer) PendingCount() int {
	entries, err := w.ReadPending()
	if err != nil {
		return -1
	}
	return len(entries)
}

// Cleanup removes WAL files older than the given duration where all entries are synced.
func (w *Writer) Cleanup(maxAge time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	syncedIDs := w.loadSyncedIDs()
	files, err := filepath.Glob(filepath.Join(w.dir, "wal-*.jsonl"))
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}

		entries, err := w.readFile(file)
		if err != nil {
			continue
		}

		allSynced := true
		for _, e := range entries {
			if !syncedIDs[e.ID] {
				allSynced = false
				break
			}
		}

		if allSynced {
			os.Remove(file)
			log.Printf("[wal] cleaned up old WAL file: %s", filepath.Base(file))
		}
	}

	return nil
}

// --- internal helpers ---

func (w *Writer) currentFile() string {
	date := time.Now().Format("2006-01-02")
	base := filepath.Join(w.dir, fmt.Sprintf("wal-%s.jsonl", date))

	// Check rotation
	info, err := os.Stat(base)
	if err == nil && info.Size() > w.maxSize {
		return filepath.Join(w.dir, fmt.Sprintf("wal-%s-%d.jsonl", date, time.Now().Unix()))
	}
	return base
}

func (w *Writer) readFile(filename string) ([]Entry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			log.Printf("[wal] skip malformed line in %s: %v", filepath.Base(filename), err)
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (w *Writer) loadSyncedIDs() map[string]bool {
	ids := make(map[string]bool)
	data, err := os.ReadFile(filepath.Join(w.dir, "synced.log"))
	if err != nil {
		return ids
	}
	for _, line := range splitLines(data) {
		if len(line) > 0 {
			ids[string(line)] = true
		}
	}
	return ids
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
