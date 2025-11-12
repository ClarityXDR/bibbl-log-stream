package azure_blob

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// LocalBuffer provides local disk buffering for failover scenarios
type LocalBuffer struct {
	path      string
	maxSize   int64
	mu        sync.Mutex
	file      *os.File
	size      int64
	readFile  *os.File
	readPos   int64
}

// NewLocalBuffer creates a new local buffer
func NewLocalBuffer(path string, maxSize int64) (*LocalBuffer, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create buffer directory: %w", err)
	}

	// Open or create buffer file
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open buffer file: %w", err)
	}

	// Get current size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat buffer file: %w", err)
	}

	return &LocalBuffer{
		path:    path,
		maxSize: maxSize,
		file:    file,
		size:    info.Size(),
	}, nil
}

// Write writes data to the local buffer
func (lb *LocalBuffer) Write(data []byte) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Check if buffer is full
	if lb.maxSize > 0 && lb.size+int64(len(data)) > lb.maxSize {
		return fmt.Errorf("local buffer full (size: %d, max: %d)", lb.size, lb.maxSize)
	}

	// Write data with length prefix for recovery
	header := fmt.Sprintf("%d\n", len(data))
	if _, err := lb.file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := lb.file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	if _, err := lb.file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	lb.size += int64(len(header)) + int64(len(data)) + 1

	// Sync to disk
	if err := lb.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync buffer: %w", err)
	}

	return nil
}

// Read reads the next chunk of data from the buffer
func (lb *LocalBuffer) Read() ([]byte, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Open read file if not already open
	if lb.readFile == nil {
		file, err := os.Open(lb.path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("failed to open buffer for reading: %w", err)
		}
		lb.readFile = file
		lb.readPos = 0
	}

	// Read length header
	var length int
	_, err := fmt.Fscanf(lb.readFile, "%d\n", &length)
	if err == io.EOF {
		// End of file, close and reset
		lb.readFile.Close()
		lb.readFile = nil
		lb.readPos = 0

		// Truncate the file since we've read everything
		if err := lb.file.Truncate(0); err != nil {
			return nil, fmt.Errorf("failed to truncate buffer: %w", err)
		}
		if _, err := lb.file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to seek buffer: %w", err)
		}
		lb.size = 0

		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	// Read data
	data := make([]byte, length)
	n, err := io.ReadFull(lb.readFile, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Read trailing newline
	if _, err := lb.readFile.Read(make([]byte, 1)); err != nil {
		return nil, fmt.Errorf("failed to read newline: %w", err)
	}

	lb.readPos += int64(len(fmt.Sprintf("%d\n", length))) + int64(n) + 1

	return data, nil
}

// Size returns the current buffer size
func (lb *LocalBuffer) Size() int64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.size
}

// Close closes the local buffer
func (lb *LocalBuffer) Close() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var errs []error

	if lb.file != nil {
		if err := lb.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close write file: %w", err))
		}
	}

	if lb.readFile != nil {
		if err := lb.readFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close read file: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing local buffer: %v", errs)
	}

	return nil
}
