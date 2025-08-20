package api

import (
    "context"
    "bibbl/internal/config"
    "net/http/httptest"
    "testing"
    "time"
)

// Ensure SSE endpoint returns quickly even if client disconnects early.
func TestSSEEarlyDisconnect(t *testing.T) {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"; cfg.Server.Port = 0
    srv := NewServer(cfg)
    // create a synthetic source and write some log lines
    src, _ := srv.pipeline.CreateSource("S1","synthetic", map[string]interface{}{})
    _ = src
    // open SSE stream
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    req := httptest.NewRequest("GET", "/api/v1/sources/S1/stream?tail=5", nil).WithContext(ctx)
    start := time.Now()
    _, _ = srv.app.Test(req, -1)
    if time.Since(start) > time.Second { t.Fatalf("SSE request exceeded 1s after context cancel") }
}
