package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type PipelineFilter struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
	Mode   string   `json:"mode,omitempty"`
}

type Pipeline struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Functions   []string         `json:"functions"`
	Filters     []PipelineFilter `json:"filters"`
}

type PipelineStats struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Filtered  uint64  `json:"filtered"`
	Processed uint64  `json:"processed"`
	DropRate  float64 `json:"dropRate"`
}

func (s *Server) handlePipelinesList(w http.ResponseWriter, r *http.Request) {
	pls := s.pipeline.GetPipelines()
	full := make([]Pipeline, 0, len(pls))
	for _, p := range pls {
		full = append(full, Pipeline{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Functions:   p.Functions,
			Filters:     pipelineFiltersFromFunctions(p.Functions),
		})
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	page, total, effLimit, effOffset := paginate(full, limit, offset)
	if q.Get("stream") == "1" || strings.Contains(r.Header.Get("Accept"), "application/x-ndjson") {
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		for _, item := range page {
			_ = enc.Encode(item)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		return
	}
	count := len(page)
	if link := buildPaginationLinks(r, total, effLimit, effOffset, count); link != "" {
		w.Header().Set("Link", link)
	}
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
	fns := mergeFilterFunctions(p.Functions, p.Filters)
	created, err := s.pipeline.CreatePipeline(p.Name, p.Description, fns)
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
	fns := mergeFilterFunctions(p.Functions, p.Filters)
	if err := s.pipeline.UpdatePipeline(id, p.Name, p.Description, fns); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func pipelineFiltersFromFunctions(fns []string) []PipelineFilter {
	filters := make([]PipelineFilter, 0)
	for _, fn := range fns {
		kv, ok := newKVFilter(fn)
		if !ok {
			continue
		}
		values := make([]string, 0, len(kv.values))
		for v := range kv.values {
			values = append(values, v)
		}
		sort.Strings(values)
		mode := "include"
		if kv.op == filterOpExclude {
			mode = "exclude"
		}
		filters = append(filters, PipelineFilter{Field: kv.field, Values: values, Mode: mode})
	}
	return filters
}

func mergeFilterFunctions(base []string, filters []PipelineFilter) []string {
	if len(filters) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(filters))
	for _, fn := range base {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(fn)), "filter:") {
			out = append(out, fn)
		}
	}
	for _, filter := range filters {
		if expr := filter.toFunction(); expr != "" {
			out = append(out, expr)
		}
	}
	return out
}

func (pf PipelineFilter) toFunction() string {
	field := strings.TrimSpace(pf.Field)
	if field == "" {
		return ""
	}
	vals := make([]string, 0, len(pf.Values))
	for _, v := range pf.Values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			vals = append(vals, trimmed)
		}
	}
	if len(vals) == 0 {
		return ""
	}
	mode := strings.ToLower(pf.Mode)
	delim := "="
	if mode == "exclude" {
		delim = "!="
	}
	return "filter:" + field + delim + strings.Join(vals, "|")
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

func (s *Server) handlePipelineStats(w http.ResponseWriter, r *http.Request) {
	stats := s.pipeline.GetPipelineStats()
	for i := range stats {
		if stats[i].Processed == 0 {
			stats[i].DropRate = 0
			continue
		}
		stats[i].DropRate = float64(stats[i].Filtered) / float64(stats[i].Processed)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
