package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"
)

// handleRegexPreview applies a named-capture regex pattern to a sample string and returns captures.
func (s *Server) handleRegexPreview(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    var req struct {
        Sample  string `json:"sample"`
        Pattern string `json:"pattern"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
        return
    }
    if len(req.Pattern) > 20_000 || len(req.Sample) > 200_000 {
        http.Error(w, `{"error":"input too large"}`, http.StatusBadRequest)
        return
    }
    // Compile regex (trust user input size already bounded). Add a time guard for execution.
    re, err := regexp.Compile(req.Pattern)
    if err != nil {
        http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
        return
    }
    timedOut := false
    done := make(chan struct{})
    var m []string
    go func(){
        m = re.FindStringSubmatch(req.Sample)
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(200 * time.Millisecond):
        timedOut = true
    }
    if timedOut {
        http.Error(w, `{"error":"regex execution timeout"}`, http.StatusBadRequest)
        return
    }
    if m == nil {
        _ = json.NewEncoder(w).Encode(map[string]any{"matched": false, "captures": map[string]string{}})
        return
    }
    names := re.SubexpNames()
    captures := map[string]string{}
    for i, v := range m {
        if i == 0 {
            continue
        }
        name := ""
        if i < len(names) {
            name = names[i]
        }
        key := name
        if key == "" {
            key = string(rune('0' + i))
        }
        captures[key] = v
    }
    _ = json.NewEncoder(w).Encode(map[string]any{"matched": true, "captures": captures})
}
