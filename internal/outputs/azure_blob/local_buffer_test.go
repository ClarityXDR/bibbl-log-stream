package azure_blob

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBuffer(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "buffer.dat")

	// Create local buffer
	lb, err := NewLocalBuffer(bufferPath, 10*1024*1024) // 10MB
	if err != nil {
		t.Fatalf("NewLocalBuffer() failed: %v", err)
	}
	defer lb.Close()

	// Test write
	testData := []byte("test log event 1\n")
	err = lb.Write(testData)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Test size
	if lb.Size() == 0 {
		t.Error("Size() should be > 0 after write")
	}

	// Write more data
	testData2 := []byte("test log event 2\n")
	err = lb.Write(testData2)
	if err != nil {
		t.Fatalf("Write() second write failed: %v", err)
	}

	// Close and reopen to test persistence
	lb.Close()

	lb, err = NewLocalBuffer(bufferPath, 10*1024*1024)
	if err != nil {
		t.Fatalf("NewLocalBuffer() reopen failed: %v", err)
	}
	defer lb.Close()

	// Test read
	data, err := lb.Read()
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Read() = %s, want %s", string(data), string(testData))
	}

	// Read second chunk
	data, err = lb.Read()
	if err != nil {
		t.Fatalf("Read() second read failed: %v", err)
	}
	if string(data) != string(testData2) {
		t.Errorf("Read() = %s, want %s", string(data), string(testData2))
	}

	// Should be EOF now
	_, err = lb.Read()
	if err != io.EOF {
		t.Errorf("Read() should return EOF, got %v", err)
	}

	// Buffer should be empty now
	if lb.Size() != 0 {
		t.Errorf("Size() should be 0 after reading all data, got %d", lb.Size())
	}
}

func TestLocalBufferMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "buffer.dat")

	// Create buffer with small max size
	maxSize := int64(100)
	lb, err := NewLocalBuffer(bufferPath, maxSize)
	if err != nil {
		t.Fatalf("NewLocalBuffer() failed: %v", err)
	}
	defer lb.Close()

	// Write data that fits
	smallData := make([]byte, 30)
	err = lb.Write(smallData)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Write data that exceeds max size
	largeData := make([]byte, 200)
	err = lb.Write(largeData)
	if err == nil {
		t.Error("Write() should fail when exceeding max size")
	}
}

func TestLocalBufferEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "buffer.dat")

	// Create buffer
	lb, err := NewLocalBuffer(bufferPath, 10*1024*1024)
	if err != nil {
		t.Fatalf("NewLocalBuffer() failed: %v", err)
	}
	defer lb.Close()

	// Try to read from empty buffer
	_, err = lb.Read()
	if err != io.EOF {
		t.Errorf("Read() on empty buffer should return EOF, got %v", err)
	}
}

func TestLocalBufferMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "buffer.dat")

	lb, err := NewLocalBuffer(bufferPath, 10*1024*1024)
	if err != nil {
		t.Fatalf("NewLocalBuffer() failed: %v", err)
	}
	defer lb.Close()

	// Write multiple chunks
	chunks := []string{
		"chunk 1",
		"chunk 2",
		"chunk 3",
		"chunk 4",
		"chunk 5",
	}

	for _, chunk := range chunks {
		err = lb.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write() failed: %v", err)
		}
	}

	// Read all chunks
	for i, expected := range chunks {
		data, err := lb.Read()
		if err != nil {
			t.Fatalf("Read() chunk %d failed: %v", i, err)
		}
		if string(data) != expected {
			t.Errorf("Read() chunk %d = %s, want %s", i, string(data), expected)
		}
	}

	// Should be EOF
	_, err = lb.Read()
	if err != io.EOF {
		t.Errorf("Read() should return EOF after all chunks, got %v", err)
	}
}

func TestLocalBufferCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use nested path that doesn't exist
	bufferPath := filepath.Join(tmpDir, "nested", "path", "buffer.dat")

	lb, err := NewLocalBuffer(bufferPath, 10*1024*1024)
	if err != nil {
		t.Fatalf("NewLocalBuffer() failed: %v", err)
	}
	defer lb.Close()

	// Check that directory was created
	dir := filepath.Dir(bufferPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("NewLocalBuffer() should create directory")
	}
}
