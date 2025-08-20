package akamai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Stream minimal representation for DataStream 2 stream list.
type Stream struct {
    ID          int    `json:"streamId"`
    Name        string `json:"streamName"`
    Activated   bool   `json:"activated"`
    DatasetType string `json:"datasetType"`
    Status      string `json:"status"`
}

type Client struct {
    creds      Credentials
    httpClient *http.Client
    baseURL    string
}

func NewClient(creds Credentials) *Client {
    // Normalize host: strip scheme, trim spaces/slashes so signing canon host is correct.
    hostRaw := strings.TrimSpace(creds.Host)
    hostRaw = strings.TrimSuffix(hostRaw, "/")
    hostRaw = strings.TrimPrefix(strings.TrimPrefix(hostRaw, "https://"), "http://")
    // If user pasted a full URL with path, keep only host portion.
    if strings.Contains(hostRaw, "/") {
        parts := strings.Split(hostRaw, "/")
        hostRaw = parts[0]
    }
    creds.Host = hostRaw // ensure signing uses the clean host value (no scheme)
    base := "https://" + hostRaw
    return &Client{creds: creds, httpClient: &http.Client{Timeout: 30 * time.Second}, baseURL: base}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
    if err != nil { return nil, err }
    if req.Header.Get("Content-Type") == "" { req.Header.Set("Content-Type", "application/json") }
    var payload []byte
    if body != nil { payload, _ = io.ReadAll(body); req.Body = io.NopCloser(strings.NewReader(string(payload))) }
    hash := bodySHA256Base64(payload)
    if err := sign(req, c.creds, hash); err != nil { return nil, err }
    return c.httpClient.Do(req)
}

// DoProxy exposes a constrained raw request capability for the API workbench.
// Only intended for internal use to explore /datastream-* endpoints.
func (c *Client) DoProxy(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
    return c.do(ctx, method, path, body)
}

// ListStreams returns configured streams (first page only for now).
func (c *Client) ListStreams(ctx context.Context) ([]Stream, error) {
    resp, err := c.do(ctx, http.MethodGet, "/datastream-config/v2/log/streams", nil)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { b,_:=io.ReadAll(resp.Body); return nil, fmt.Errorf("akamai list streams: %s", strings.TrimSpace(string(b))) }
    var raw struct { Streams []Stream `json:"streams"` }
    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return nil, err }
    return raw.Streams, nil
}

// GetDatasetFields fetches dataset field definitions for a given dataset (e.g. "COMMON")
// Returns raw JSON (decoded) to avoid needing to chase changing schema. Caller can type assert.
func (c *Client) GetDatasetFields(ctx context.Context, dataset string) (interface{}, error) {
    if dataset == "" { return nil, errors.New("dataset required") }
    path := fmt.Sprintf("/datastream-config/v2/log/datasets/%s/fields", url.PathEscape(dataset))
    resp, err := c.do(ctx, http.MethodGet, path, nil)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { b,_:=io.ReadAll(resp.Body); return nil, fmt.Errorf("dataset fields: %s", strings.TrimSpace(string(b))) }
    var any interface{}
    if err := json.NewDecoder(resp.Body).Decode(&any); err != nil { return nil, err }
    return any, nil
}

func (c *Client) ActivateStream(ctx context.Context, id int) error {
    resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/datastream-config/v2/log/streams/%d/activate", id), nil)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { b,_:=io.ReadAll(resp.Body); return fmt.Errorf("activate stream %d: %s", id, strings.TrimSpace(string(b))) }
    return nil
}

func (c *Client) DeactivateStream(ctx context.Context, id int) error {
    resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/datastream-config/v2/log/streams/%d/deactivate", id), nil)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { b,_:=io.ReadAll(resp.Body); return fmt.Errorf("deactivate stream %d: %s", id, strings.TrimSpace(string(b))) }
    return nil
}

// Poller periodically lists streams and emits pseudo-log lines summarizing status.
type Poller struct {
    Client   *Client
    Interval time.Duration
    Streams  []int // filter; empty = all
    cancel context.CancelFunc
    ctx context.Context
}

func (p *Poller) Start(cb func(string)) error {
    if p.Client == nil { return errors.New("nil client") }
    if p.Interval <= 0 { p.Interval = 60 * time.Second }
    ctx, cancel := context.WithCancel(context.Background())
    p.cancel = cancel
    p.ctx = ctx
    go func(){
        ticker := time.NewTicker(p.Interval)
        defer ticker.Stop()
        for {
            if err := p.runOnce(ctx, cb); err != nil { cb("akamai poll error: "+err.Error()) }
            select { case <-ticker.C: continue; case <-ctx.Done(): return }
        }
    }()
    return nil
}

func (p *Poller) runOnce(ctx context.Context, cb func(string)) error {
    streams, err := p.Client.ListStreams(ctx)
    if err != nil { return err }
    filter := map[int]struct{}{}
    for _, id := range p.Streams { filter[id] = struct{}{} }
    now := time.Now().UTC().Format(time.RFC3339)
    for _, s := range streams {
        if len(filter) > 0 { if _, ok := filter[s.ID]; !ok { continue } }
        line := fmt.Sprintf("%s akamai_ds2 stream=%d name=\"%s\" status=%s activated=%v dataset=%s", now, s.ID, escapeSpaces(s.Name), s.Status, s.Activated, s.DatasetType)
        cb(line)
    }
    return nil
}

func (p *Poller) Stop() { if p.cancel != nil { p.cancel(); p.cancel = nil } }

// Done returns a channel closed when the poller stops.
func (p *Poller) Done() <-chan struct{} {
    if p.ctx == nil { ch := make(chan struct{}); close(ch); return ch }
    return p.ctx.Done()
}

func escapeSpaces(s string) string { return strings.ReplaceAll(s, " ", "_") }

// ParseStreamIDs parses config value to a list of stream IDs.
func ParseStreamIDs(v interface{}) []int {
    var out []int
    switch vv := v.(type) {
    case string:
        for _, part := range strings.Split(vv, ",") { part = strings.TrimSpace(part); if part=="" {continue}; var id int; fmt.Sscanf(part, "%d", &id); if id>0 { out = append(out, id) } }
    case []interface{}:
        for _, x := range vv { switch t := x.(type) { case float64: if int(t)>0 { out = append(out, int(t)) }; case int: if t>0 { out = append(out, t) } } }
    }
    return out
}

// BuildCaptureEndpoint suggests an ingestion endpoint path for Akamai push destinations.
func BuildCaptureEndpoint(externalHost, sourceID string) string {
    if externalHost == "" { return "/ingest/akamai/" + url.PathEscape(sourceID) }
    return fmt.Sprintf("https://%s/ingest/akamai/%s", externalHost, url.PathEscape(sourceID))
}
