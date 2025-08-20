package api

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogHub maintains in-memory per-source ring buffers, subscribers for SSE,
// and capture sessions writing to .log or .json files in a library dir.
type LogHub struct {
    mu          sync.RWMutex
    buffers     map[string]*ring
    subscribers map[string]map[chan string]struct{}
    captures    map[string]*capture
    libraryDir  string
    capSeq      int
    last        map[string]int64 // unix seconds per sourceID
}

type ring struct {
    mu   sync.RWMutex
    max  int
    data []string
}

func newRing(max int) *ring { return &ring{max: max, data: make([]string, 0, max)} }

func (r *ring) add(s string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if len(r.data) == r.max {
        // drop oldest
        copy(r.data, r.data[1:])
        r.data[len(r.data)-1] = s
        return
    }
    r.data = append(r.data, s)
}

func (r *ring) tail(n int) []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    L := len(r.data)
    if n <= 0 || n > L { n = L }
    out := make([]string, n)
    copy(out, r.data[L-n:])
    return out
}

type capture struct {
    id       string
    sourceID string
    format   string // "log" or "json"
    file     *os.File
    writer   *bufio.Writer
}

func NewLogHub(libraryDir string) (*LogHub, error) {
    if libraryDir == "" { libraryDir = "./sandbox/library" }
    if err := os.MkdirAll(libraryDir, 0o755); err != nil {
        return nil, err
    }
    return &LogHub{
        buffers:     map[string]*ring{},
        subscribers: map[string]map[chan string]struct{}{},
        captures:    map[string]*capture{},
        libraryDir:  libraryDir,
        capSeq:      1,
    last:        map[string]int64{},
    }, nil
}

// Append adds a new message to a source buffer and notifies subscribers/captures.
func (h *LogHub) Append(sourceID, msg string) {
    if strings.TrimSpace(msg) == "" { return }
    h.mu.Lock()
    r := h.buffers[sourceID]
    if r == nil { r = newRing(1000); h.buffers[sourceID] = r }
    subs := h.subscribers[sourceID]
    // Copy captures to avoid holding lock while writing
    var caps []*capture
    for _, c := range h.captures {
        if c.sourceID == sourceID { caps = append(caps, c) }
    }
    h.last[sourceID] = time.Now().Unix()
    h.mu.Unlock()

    r.add(msg)

    // Broadcast to subscribers
    for ch := range subs { select { case ch <- msg: default: } }
    // Write to captures
    for _, c := range caps {
        _ = h.writeCapture(c, msg)
    }
}

// LastUnix returns the last-seen unix timestamp for a source, or 0 if none.
func (h *LogHub) LastUnix(sourceID string) int64 {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return h.last[sourceID]
}

func (h *LogHub) writeCapture(c *capture, msg string) error {
    if c == nil || c.writer == nil { return nil }
    // For now, both formats are newline-delimited; JSON assumes msg is JSON already
    if _, err := c.writer.WriteString(msg+"\n"); err != nil { return err }
    return c.writer.Flush()
}

func (h *LogHub) Subscribe(sourceID string) (ch chan string, cancel func()) {
    ch = make(chan string, 256)
    h.mu.Lock()
    if h.subscribers[sourceID] == nil { h.subscribers[sourceID] = map[chan string]struct{}{} }
    h.subscribers[sourceID][ch] = struct{}{}
    h.mu.Unlock()

    cancel = func() {
        h.mu.Lock()
        defer h.mu.Unlock()
        if subs := h.subscribers[sourceID]; subs != nil {
            delete(subs, ch)
            close(ch)
        }
    }
    return
}

func (h *LogHub) Tail(sourceID string, n int) []string {
    h.mu.RLock()
    r := h.buffers[sourceID]
    h.mu.RUnlock()
    if r == nil { return nil }
    return r.tail(n)
}

func (h *LogHub) StartCapture(sourceID, format, name string) (string, string, error) {
    if format != "log" && format != "json" {
        return "", "", errors.New("invalid format")
    }
    h.mu.Lock()
    id := fmt.Sprintf("cap-%d", h.capSeq)
    h.capSeq++
    if name == "" { name = fmt.Sprintf("%s-%s-%d", sourceID, format, time.Now().Unix()) }
    ext := ".log"
    if format == "json" { ext = ".json" }
    path := filepath.Join(h.libraryDir, name+ext)
    f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil { h.mu.Unlock(); return "", "", err }
    c := &capture{id: id, sourceID: sourceID, format: format, file: f, writer: bufio.NewWriter(f)}
    h.captures[id] = c
    h.mu.Unlock()
    return id, path, nil
}

func (h *LogHub) StopCapture(id string) error {
    h.mu.Lock()
    c := h.captures[id]
    if c != nil { delete(h.captures, id) }
    h.mu.Unlock()
    if c != nil {
        _ = c.writer.Flush()
        return c.file.Close()
    }
    return errors.New("capture not found")
}

type LibraryItem struct {
    Name string `json:"name"`
    Size int64  `json:"size"`
    ModTime time.Time `json:"modTime"`
}

func (h *LogHub) ListLibrary() ([]LibraryItem, error) {
    entries, err := os.ReadDir(h.libraryDir)
    if err != nil { return nil, err }
    var out []LibraryItem
    for _, e := range entries {
        if e.IsDir() { continue }
        if !strings.HasSuffix(e.Name(), ".log") && !strings.HasSuffix(e.Name(), ".json") { continue }
        info, err := e.Info(); if err != nil { continue }
        out = append(out, LibraryItem{Name: e.Name(), Size: info.Size(), ModTime: info.ModTime()})
    }
    sort.Slice(out, func(i,j int) bool { return out[i].ModTime.After(out[j].ModTime) })
    return out, nil
}

func (h *LogHub) ReadLibraryFile(name string, maxBytes int64) ([]byte, error) {
    if strings.Contains(name, "..") { return nil, errors.New("invalid name") }
    p := filepath.Join(h.libraryDir, name)
    f, err := os.Open(p)
    if err != nil { return nil, err }
    defer f.Close()
    if maxBytes <= 0 || maxBytes > 10*1024*1024 { maxBytes = 10*1024*1024 }
    st, _ := f.Stat()
    size := st.Size()
    if size > maxBytes {
        // read last maxBytes
        off := size - maxBytes
        if off < 0 { off = 0 }
        if _, err := f.Seek(off, 0); err != nil { return nil, err }
    }
    return io.ReadAll(f)
}

