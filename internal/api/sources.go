package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Source struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config"`
	Status  string                 `json:"status"`
	Enabled bool                   `json:"enabled"`
	LastUnix int64                 `json:"lastUnix"`
	Flow    bool                   `json:"flow"`
}

func (s *Server) handleSourcesList(w http.ResponseWriter, r *http.Request) {
	sources := s.pipeline.GetSources()
	// Build full slice first (cheap for now; future: incremental)
	full := make([]Source, 0, len(sources))
	for _, src := range sources {
		var last int64
		if s.hub != nil { last = s.hub.LastUnix(src.ID) }
		flow := last > 0 && time.Now().Unix()-last <= 30 && src.Enabled
		full = append(full, Source{ID: src.ID, Name: src.Name, Type: src.Type, Config: src.Config, Status: src.Status, Enabled: src.Enabled, LastUnix: last, Flow: flow})
	}
	// Parse pagination params
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	page, total, effLimit, effOffset := paginate(full, limit, offset)
	// Streaming NDJSON mode if stream=1 or Accept includes ndjson
	if q.Get("stream") == "1" || strings.Contains(r.Header.Get("Accept"), "application/x-ndjson") {
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		for _, item := range page {
			_ = enc.Encode(item)
			if f, ok := w.(http.Flusher); ok { f.Flush() }
		}
		return
	}
	// Pagination headers
	count := len(page)
	if link := buildPaginationLinks(r, total, effLimit, effOffset, count); link != "" { w.Header().Set("Link", link) }
	w.Header().Set("Pagination-Total", strconv.Itoa(total))
	w.Header().Set("Pagination-Limit", strconv.Itoa(effLimit))
	w.Header().Set("Pagination-Offset", strconv.Itoa(effOffset))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": page,
		"page": map[string]any{"total": total, "offset": effOffset, "limit": effLimit, "count": count},
		// legacy top-level fields for backward compat (to be removed in future)
		"total": total, "offset": effOffset, "limit": effLimit,
	})
}

func (s *Server) handleSourceCreate(w http.ResponseWriter, r *http.Request) {
	var source Source
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil { structuredError(w, r, http.StatusBadRequest, "decode_error", err.Error()); return }
	
	createdSource, err := s.pipeline.CreateSource(source.Name, source.Type, source.Config)
	if err != nil { structuredError(w, r, http.StatusInternalServerError, "create_failed", err.Error()); return }
	// audit
	if m, ok := createdSource.(*Source); ok { s.audit("source_create", map[string]any{"id": m.ID, "name": m.Name, "type": m.Type}) } else { s.audit("source_create", map[string]any{"name": source.Name, "type": source.Type}) }
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdSource)
}

func (s *Server) handleSourceUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var source Source
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil { structuredError(w, r, http.StatusBadRequest, "decode_error", err.Error()); return }
	
	if err := s.pipeline.UpdateSource(id, source.Name, source.Config); err != nil { structuredError(w, r, http.StatusInternalServerError, "update_failed", err.Error()); return }
	s.audit("source_update", map[string]any{"id": id, "name": source.Name})
	
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSourceDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := s.pipeline.DeleteSource(id); err != nil { structuredError(w, r, http.StatusInternalServerError, "delete_failed", err.Error()); return }
	s.audit("source_delete", map[string]any{"id": id})
	
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSourceStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := s.pipeline.StartSource(id); err != nil { structuredError(w, r, http.StatusInternalServerError, "start_failed", err.Error()); return }
	s.audit("source_start", map[string]any{"id": id})
	
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSourceStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := s.pipeline.StopSource(id); err != nil { structuredError(w, r, http.StatusInternalServerError, "stop_failed", err.Error()); return }
	s.audit("source_stop", map[string]any{"id": id})
	
	w.WriteHeader(http.StatusOK)
}
