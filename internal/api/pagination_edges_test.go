package api

import (
	"bibbl/internal/config"
	"net/http/httptest"
	"strconv"
	"testing"
)

// Test offset beyond total returns empty items and proper headers.
func TestPaginationOffsetBeyondTotal(t *testing.T) {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"; cfg.Server.Port = 0
    srv := NewServer(cfg)
    req := httptest.NewRequest("GET", "/api/v1/sources?limit=10&offset=9999", nil)
    resp, err := srv.app.Test(req)
    if err != nil { t.Fatalf("err: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("status %d", resp.StatusCode) }
    if resp.Header.Get("Pagination-Offset") != "9999" { t.Fatalf("expected offset header 9999 got %s", resp.Header.Get("Pagination-Offset")) }
    total := resp.Header.Get("Pagination-Total")
    if total == "" { t.Fatalf("expected total header") }
    if l := resp.Header.Get("Pagination-Limit"); l == "" { t.Fatalf("expected limit header") }
    if _, err := strconv.Atoi(total); err != nil { t.Fatalf("invalid total header") }
}
