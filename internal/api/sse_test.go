package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"bibbl/internal/config"
)

// Ensure SSE endpoint returns quickly even if client disconnects early.
func TestSSEEarlyDisconnect(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0
	srv := NewServer(cfg)
	if _, err := srv.pipeline.CreateSource("S1", "synthetic", map[string]interface{}{}); err != nil {
		t.Fatalf("create source: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		_ = srv.app.Listener(ln)
		close(done)
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		_ = srv.app.Shutdown()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	url := fmt.Sprintf("http://%s/api/v1/sources/S1/stream?tail=5", ln.Addr().String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatalf("expected request to cancel")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("SSE request exceeded 1s after context cancel")
	}
}
