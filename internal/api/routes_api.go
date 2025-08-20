package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Route struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Filter      string `json:"filter"`
	PipelineID  string `json:"pipelineId"`
	Destination string `json:"destination"`
	Final       bool   `json:"final"`
}

func (s *Server) handleRoutesList(w http.ResponseWriter, r *http.Request) {
	rs := s.pipeline.GetRoutes()
	full := make([]Route, 0, len(rs))
	for _, r0 := range rs { full = append(full, Route{ID: r0.ID, Name: r0.Name, Filter: r0.Filter, PipelineID: r0.PipelineID, Destination: r0.Destination, Final: r0.Final}) }
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

func (s *Server) handleRouteCreate(w http.ResponseWriter, r *http.Request) {
	var route Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.pipeline.CreateRoute(route.Name, route.Filter, route.PipelineID, route.Destination, route.Final)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleRouteUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var route Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.pipeline.UpdateRoute(id, route.Name, route.Filter, route.PipelineID, route.Destination, route.Final); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRouteDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := s.pipeline.DeleteRoute(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
