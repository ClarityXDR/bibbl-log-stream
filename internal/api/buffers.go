package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type BufferStatus struct {
    SourceID   string `json:"sourceId"`
    Size       int    `json:"size"`
    Capacity   int    `json:"capacity"`
    Dropped    int    `json:"dropped"`
    OldestUnix int64  `json:"oldestUnix"`
    NewestUnix int64  `json:"newestUnix"`
    LastError  string `json:"lastError"`
    Auto       bool   `json:"auto"`
    MinCap     int    `json:"minCap"`
    MaxCap     int    `json:"maxCap"`
}

func (s *Server) handleBuffersList(w http.ResponseWriter, r *http.Request) {
    bufs := s.pipeline.GetBuffers()
    full := make([]BufferStatus, 0, len(bufs))
    for _, b := range bufs {
        status := BufferStatus{SourceID: b.SourceID, Size: b.Size, Capacity: b.Capacity, Dropped: b.Dropped, OldestUnix: b.OldestUnix, NewestUnix: b.NewestUnix, LastError: b.LastError}
        if one, ok := s.pipeline.GetBuffer(b.SourceID); ok { status.Auto = one.Auto; status.MinCap = one.MinCap; status.MaxCap = one.MaxCap }
        full = append(full, status)
    }
    q := r.URL.Query()
    limit, _ := strconv.Atoi(q.Get("limit"))
    offset, _ := strconv.Atoi(q.Get("offset"))
    page, total, effLimit, effOffset := paginate(full, limit, offset)
    if q.Get("stream") == "1" || strings.Contains(r.Header.Get("Accept"), "application/x-ndjson") {
        w.Header().Set("Content-Type", "application/x-ndjson")
        enc := json.NewEncoder(w)
        for _, item := range page { _ = enc.Encode(item); if f, ok := w.(http.Flusher); ok { f.Flush() } }
        return
    }
    count := len(page)
    if link := buildPaginationLinks(r, total, effLimit, effOffset, count); link != "" { w.Header().Set("Link", link) }
    w.Header().Set("Pagination-Total", strconv.Itoa(total))
    w.Header().Set("Pagination-Limit", strconv.Itoa(effLimit))
    w.Header().Set("Pagination-Offset", strconv.Itoa(effOffset))
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{"items": page, "page": map[string]any{"total": total, "offset": effOffset, "limit": effLimit, "count": count}, "total": total, "offset": effOffset, "limit": effLimit})
}

func (s *Server) handleBufferReset(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["sourceId"]
    if err := s.pipeline.ResetBuffer(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// handleBufferGet returns a single buffer status
func (s *Server) handleBufferGet(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["sourceId"]
    if b, ok := s.pipeline.GetBuffer(id); ok {
        resp := BufferStatus{
            SourceID: b.SourceID,
            Size: b.Size,
            Capacity: b.Capacity,
            Dropped: b.Dropped,
            OldestUnix: b.OldestUnix,
            NewestUnix: b.NewestUnix,
            LastError: b.LastError,
            Auto: b.Auto,
            MinCap: b.MinCap,
            MaxCap: b.MaxCap,
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(resp)
        return
    }
    http.Error(w, "not found", http.StatusNotFound)
}

// handleBufferUpdate allows adjusting capacity and auto settings
func (s *Server) handleBufferUpdate(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["sourceId"]
    var req struct {
        Capacity *int  `json:"capacity"`
        Auto     *bool `json:"auto"`
        MinCap   *int  `json:"minCap"`
        MaxCap   *int  `json:"maxCap"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if err := s.pipeline.UpdateBufferConfig(id, req.Capacity, req.Auto, req.MinCap, req.MaxCap); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
