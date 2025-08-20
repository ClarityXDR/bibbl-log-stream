package api

import (
	"bibbl/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnrichPreviewNilGeo(t *testing.T) {
    cfg := &config.Config{}; cfg.Server.Host="127.0.0.1"; cfg.Server.Port=0
    // define an auth token to avoid RBAC 403 and let handler reach 412 path
    cfg.Server.AuthToken = "tok"
    srv := NewServer(cfg)
    body := `{"ip":"1.1.1.1"}`
    req := httptest.NewRequest("POST", "/api/v1/preview/enrich", strings.NewReader(body))
    req.Header.Set("Authorization", "Bearer tok")
    resp, _ := srv.app.Test(req)
    if resp.StatusCode != 412 { t.Fatalf("expected 412 precondition failed got %d", resp.StatusCode) }
}
