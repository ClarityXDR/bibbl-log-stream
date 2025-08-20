package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Destination struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Status string                 `json:"status"`
	Config map[string]interface{} `json:"config"`
	Enabled bool                  `json:"enabled"`
}

func (s *Server) handleDestinationsList(w http.ResponseWriter, r *http.Request) {
	dests := s.pipeline.GetDestinations()
	full := make([]Destination, 0, len(dests))
	for _, d := range dests {
		full = append(full, Destination{ID: d.ID, Name: d.Name, Type: d.Type, Status: d.Status, Config: d.Config, Enabled: d.Enabled})
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

func (s *Server) handleDestinationCreate(w http.ResponseWriter, r *http.Request) {
	var dest Destination
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.pipeline.CreateDestination(dest.Name, dest.Type, dest.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleDestinationUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var dest Destination
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.pipeline.UpdateDestination(id, dest.Name, dest.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDestinationDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := s.pipeline.DeleteDestination(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDestinationPatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.pipeline.PatchDestination(id, patch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
