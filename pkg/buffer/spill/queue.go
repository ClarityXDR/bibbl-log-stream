package spill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Directory   string
	MaxBytes    int64
	SegmentSize int64
}

type Queue struct {
	cfg        Config
	mu         sync.Mutex
	totalBytes int64
}

var fileSeq atomic.Uint64

// NewQueue initializes a spill queue on disk.
func NewQueue(cfg Config) (*Queue, error) {
	if strings.TrimSpace(cfg.Directory) == "" {
		return nil, fmt.Errorf("spill directory required")
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 10 * 1024 * 1024 * 1024
	}
	if cfg.SegmentSize <= 0 {
		cfg.SegmentSize = 1 * 1024 * 1024
	}
	if err := os.MkdirAll(cfg.Directory, 0o750); err != nil {
		return nil, err
	}
	total := int64(0)
	entries, _ := os.ReadDir(cfg.Directory)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return &Queue{cfg: cfg, totalBytes: total}, nil
}

// Append persists the batch to disk.
func (q *Queue) Append(events []map[string]interface{}) error {
	if len(events) == 0 {
		return nil
	}
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal spill batch: %w", err)
	}
	if q.cfg.SegmentSize > 0 && int64(len(data)) > q.cfg.SegmentSize && len(events) > 1 {
		mid := len(events) / 2
		if err := q.Append(events[:mid]); err != nil {
			return err
		}
		return q.Append(events[mid:])
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := os.MkdirAll(q.cfg.Directory, 0o750); err != nil {
		return err
	}
	fname := fmt.Sprintf("spill-%d-%d.json", time.Now().UnixNano(), fileSeq.Add(1))
	full := filepath.Join(q.cfg.Directory, fname)
	if err := os.WriteFile(full, data, 0o640); err != nil {
		return err
	}
	q.totalBytes += int64(len(data))
	return q.enforceLimitLocked()
}

// Replay replays buffered batches until the handler returns an error or the queue is empty.
func (q *Queue) Replay(handler func([]map[string]interface{}) error) error {
	files, err := q.listFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		full := filepath.Join(q.cfg.Directory, f.Name())
		info, err := f.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		var batch []map[string]interface{}
		if err := json.Unmarshal(data, &batch); err != nil {
			return fmt.Errorf("decode spill batch %s: %w", f.Name(), err)
		}
		if err := handler(batch); err != nil {
			return err
		}
		if err := os.Remove(full); err != nil {
			return err
		}
		q.mu.Lock()
		q.totalBytes -= info.Size()
		if q.totalBytes < 0 {
			q.totalBytes = 0
		}
		q.mu.Unlock()
	}
	return nil
}

func (q *Queue) listFiles() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(q.cfg.Directory)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		infoI, err := entries[i].Info()
		if err != nil {
			return false
		}
		infoJ, err := entries[j].Info()
		if err != nil {
			return true
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})
	var files []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "spill-") {
			files = append(files, e)
		}
	}
	return files, nil
}

func (q *Queue) enforceLimitLocked() error {
	if q.cfg.MaxBytes <= 0 {
		return nil
	}
	for q.totalBytes > q.cfg.MaxBytes {
		files, err := q.listFiles()
		if err != nil {
			return err
		}
		if len(files) == 0 {
			q.totalBytes = 0
			return nil
		}
		oldest := filepath.Join(q.cfg.Directory, files[0].Name())
		info, err := os.Stat(oldest)
		if err == nil {
			q.totalBytes -= info.Size()
		}
		if err := os.Remove(oldest); err != nil {
			return err
		}
	}
	return nil
}
