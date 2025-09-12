package api

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Start synthetic load generator (idempotent)
func (s *Server) handleLoadTestStart(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if _, ok := body["rate"]; !ok {
		body["rate"] = 10000
	}
	if _, ok := body["size"]; !ok {
		body["size"] = 300
	}
	if _, ok := body["workers"]; !ok {
		body["workers"] = 4
	}
	name := "Synthetic Load"
	var srcID string
	for _, s2 := range s.pipeline.GetSources() {
		if s2.Type == "synthetic" {
			srcID = s2.ID
			break
		}
	}
	if srcID == "" {
		created, err := s.pipeline.CreateSource(name, "synthetic", body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if m, ok := created.(*memSource); ok {
			srcID = m.ID
		} else if mm, ok := created.(map[string]interface{}); ok {
			if id, _ := mm["ID"].(string); id != "" {
				srcID = id
			}
		}
	} else {
		_ = s.pipeline.UpdateSource(srcID, name, body)
	}
	if err := s.pipeline.StartSource(srcID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "started", "sourceId": srcID})
}

func (s *Server) handleLoadTestStop(w http.ResponseWriter, r *http.Request) {
	var srcID string
	for _, s2 := range s.pipeline.GetSources() {
		if s2.Type == "synthetic" {
			srcID = s2.ID
			break
		}
	}
	if srcID == "" {
		http.Error(w, "no synthetic source", http.StatusNotFound)
		return
	}
	if err := s.pipeline.StopSource(srcID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

var loadTestLastProduced atomic.Uint64
var loadTestLastTS atomic.Int64

func (s *Server) handleLoadTestStatus(w http.ResponseWriter, r *http.Request) {
	var srcID string
	var cfg map[string]interface{}
	var produced uint64
	for _, s2 := range s.pipeline.GetSources() {
		if s2.Type == "synthetic" {
			srcID = s2.ID
			cfg = s2.Config
			break
		}
	}
	if srcID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "no synthetic source configured", "sourceId": "", "produced": 0, "eps": 0.0, "config": nil})
		return
	}
	// The produced counter is now tracked internally; we expose a lightweight snapshot via a synthetic metric API on the engine when available.
	if me, ok := s.pipeline.(*memoryEngine); ok {
		// find source and read atomic counter
		me.mu.RLock()
		for _, ms := range me.sources {
			if ms.ID == srcID {
				produced = ms.produced.Load()
				break
			}
		}
		me.mu.RUnlock()
	}
	prevC := loadTestLastProduced.Load()
	prevTS := loadTestLastTS.Load()
	now := time.Now().UnixNano()
	loadTestLastProduced.Store(produced)
	loadTestLastTS.Store(now)
	eps := 0.0
	if prevTS > 0 && produced >= prevC {
		dt := float64(now-prevTS) / 1e9
		if dt > 0 {
			eps = float64(produced-prevC) / dt
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"sourceId": srcID, "produced": produced, "eps": eps, "config": cfg})
}
