package api

import (
	"bibbl/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthTokenEnforcement(t *testing.T) {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"; cfg.Server.Port = 0
    cfg.Server.AuthToken = "secret"
    srv := NewServer(cfg)
    // request without token
    req := httptest.NewRequest("GET", "/api/v1/sources", nil)
    resp, _ := srv.app.Test(req)
    if resp.StatusCode != 401 { t.Fatalf("expected 401 got %d", resp.StatusCode) }
    // with token
    req2 := httptest.NewRequest("GET", "/api/v1/sources", nil)
    req2.Header.Set("Authorization", "Bearer secret")
    resp2, _ := srv.app.Test(req2)
    if resp2.StatusCode == 401 { t.Fatalf("expected authorized with token") }
    // mutating without role but has auth token (no roles defined) should 403
    req3 := httptest.NewRequest("POST", "/api/v1/sources", strings.NewReader(`{"name":"x","type":"synthetic"}`))
    req3.Header.Set("Authorization", "Bearer secret")
    resp3, _ := srv.app.Test(req3)
    if resp3.StatusCode != 403 { t.Fatalf("expected 403 (no roles) got %d", resp3.StatusCode) }
}
