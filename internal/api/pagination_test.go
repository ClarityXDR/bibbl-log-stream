package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"bibbl/internal/config"
)

// helper to create minimal server for tests
func newTestServer() *Server {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"
    cfg.Server.Port = 0
    // Signal to server to skip default network-listening inputs (avoids firewall prompts on Windows CI)
    tEnv := "BIBBL_TEST"
    if _, ok := os.LookupEnv(tEnv); !ok { os.Setenv(tEnv, "1") }
    return NewServer(cfg)
}

func TestPaginationLinksAndClamping(t *testing.T) {
    srv := newTestServer()
    // seed several sources
    for i := 0; i < 7; i++ {
        _, _ = srv.pipeline.CreateSource("S"+strconv.Itoa(i), "synthetic", map[string]interface{}{})
    }
    req := httptest.NewRequest("GET", "/api/v1/sources?limit=2&offset=0", nil)
    resp, err := srv.app.Test(req)
    if err != nil { t.Fatalf("request failed: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("unexpected status: %d", resp.StatusCode) }
    link := resp.Header.Get("Link")
    if link == "" { t.Fatalf("expected Link header") }
    if !strings.Contains(link, "rel=\"next\"") { t.Fatalf("expected next in Link: %s", link) }
    if !strings.Contains(link, "rel=\"first\"") || !strings.Contains(link, "rel=\"last\"") { t.Fatalf("expected first & last rels: %s", link) }
    // oversized limit should produce Pagination-Limit header
    req2 := httptest.NewRequest("GET", "/api/v1/sources?limit=9999", nil)
    resp2, _ := srv.app.Test(req2)
    if resp2.Header.Get("Pagination-Limit") == "" { t.Fatalf("expected pagination headers") }
}

func TestNDJSONStreaming(t *testing.T) {
    srv := newTestServer()
    // seed some destinations
    for i := 0; i < 3; i++ { _, _ = srv.pipeline.CreateDestination("D"+strconv.Itoa(i), "sentinel", map[string]interface{}{}) }
    req := httptest.NewRequest("GET", "/api/v1/destinations?limit=2", nil)
    req.Header.Set("Accept", "application/x-ndjson")
    resp, err := srv.app.Test(req)
    if err != nil { t.Fatalf("err: %v", err) }
    if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-ndjson") { t.Fatalf("expected ndjson content-type: %s", ct) }
    // read body and ensure multiple lines of JSON objects
    var lines []string
    bodyBytes, _ := io.ReadAll(resp.Body)
    for _, l := range strings.Split(string(bodyBytes), "\n") { if strings.TrimSpace(l) != "" { lines = append(lines, l) } }
    if len(lines) == 0 { t.Fatalf("expected at least one JSON line") }
    // parse first line to ensure valid JSON
    var obj map[string]interface{}
    if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil { t.Fatalf("invalid json line: %v", err) }
}
