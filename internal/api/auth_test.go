package api

import (
	"bibbl/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that mutating endpoint without roles is forbidden, but with role allowed.
func TestRBACEnforcement(t *testing.T) {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"; cfg.Server.Port = 0
    cfg.Server.AuthTokens = map[string][]string{"tok1": {"admin"}}
    srv := NewServer(cfg)
    req := httptest.NewRequest("POST", "/api/v1/sources", strings.NewReader(`{"name":"x","type":"synthetic"}`))
    resp, _ := srv.app.Test(req)
    if resp.StatusCode != 403 { t.Fatalf("expected 403 got %d", resp.StatusCode) }
    req2 := httptest.NewRequest("POST", "/api/v1/sources", strings.NewReader(`{"name":"x","type":"synthetic"}`))
    req2.Header.Set("Authorization", "Bearer tok1")
    resp2, _ := srv.app.Test(req2)
    if resp2.StatusCode == 403 { t.Fatalf("expected allowed with role") }
}
