package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Pipeline struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Functions   []string `json:"functions"`
}

func (s *Server) handlePipelinesList(w http.ResponseWriter, r *http.Request) {
	pls := s.pipeline.GetPipelines()
	full := make([]Pipeline, 0, len(pls))
	for _, p := range pls { full = append(full, Pipeline{ID: p.ID, Name: p.Name, Description: p.Description, Functions: p.Functions}) }
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

func (s *Server) handlePipelineCreate(w http.ResponseWriter, r *http.Request) {
	var p Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.pipeline.CreatePipeline(p.Name, p.Description, p.Functions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handlePipelineUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var p Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.pipeline.UpdatePipeline(id, p.Name, p.Description, p.Functions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePipelineDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := s.pipeline.DeletePipeline(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
