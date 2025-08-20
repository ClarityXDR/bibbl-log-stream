package api

import (
	"bibbl/internal/config"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter429(t *testing.T) {
    cfg := &config.Config{}
    cfg.Server.Host = "127.0.0.1"; cfg.Server.Port = 0
    cfg.Server.RateLimitPerMin = 5
    srv := NewServer(cfg)
    // exceed limit quickly
    var last int
    for i:=0;i<20;i++ {
        req := httptest.NewRequest("GET","/api/v1/sources", nil)
        resp, _ := srv.app.Test(req)
        last = resp.StatusCode
        if last == 429 { return }
    }
    if last != 429 { t.Fatalf("expected a 429 status after bursts, last=%d", last) }
}
