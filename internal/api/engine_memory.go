package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	akamaiinput "bibbl/internal/inputs/akamai"
	syninput "bibbl/internal/inputs/synthetic"
	sysloginput "bibbl/internal/inputs/syslog"
	"bibbl/internal/metrics"
	"bibbl/pkg/filters"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type memoryEngine struct {
	mu        sync.RWMutex
	sources   []*memSource
	dests     []memDest
	pipelines []memPipe
	routes    []memRoute
	seq       int
	hub       *LogHub
	// optional enrichment hook provided by server
	geo func(ip string) (map[string]interface{}, bool)
	asn func(ip string) (map[string]interface{}, bool)
	// cache for compiled route filters (regex)
	filterCache map[string]*regexp.Regexp
	// buffer config state (auto sizing, capacity overrides)
	bufferAuto map[string]bool
	bufferCap  map[string]int
	bufferMin  map[string]int
	bufferMax  map[string]int
	// parsers
	versaParser    *filters.VersaKVPParser
	paloAltoParser *filters.PaloAltoCSVParser
}

type memSource struct {
	ID      string
	Name    string
	Type    string
	Config  map[string]interface{}
	Status  string
	Enabled bool
	// runtime fields (not exported via JSON)
	cancel     context.CancelFunc
	syslogSrv  *sysloginput.Server
	synthGen   *syninput.Generator
	akamaiPoll *akamaiinput.Poller
	produced   atomic.Uint64
}

type memDest struct {
	ID      string
	Name    string
	Type    string
	Status  string
	Config  map[string]interface{}
	Enabled bool
}

type memPipe struct {
	ID          string
	Name        string
	Description string
	Functions   []string
	// IPSource controls how an IP is derived for enrichment when geo/ASN
	// functions are present. Supported formats:
	//   "" or "first_ipv4" (default) -> first IPv4 literal in the raw message
	//   "field:<name>" -> attempts to extract IPv4 from either key=value or JSON field
	IPSource string
}

type memRoute struct {
	ID          string
	Name        string
	Filter      string
	PipelineID  string
	Destination string
	Final       bool
}

func NewMemoryEngine() PipelineEngine {
	return &memoryEngine{
		seq:            1,
		filterCache:    map[string]*regexp.Regexp{},
		versaParser:    filters.NewVersaKVPParser(),
		paloAltoParser: filters.NewPaloAltoCSVParser(),
	}
}

// withHub attaches a LogHub if the concrete type is memoryEngine.
func withHub(p PipelineEngine, hub *LogHub) PipelineEngine {
	if m, ok := p.(*memoryEngine); ok {
		m.hub = hub
	}
	return p
}

// withGeo attaches a geo lookup function if the concrete type is memoryEngine.
func withGeo(p PipelineEngine, fn func(string) (map[string]interface{}, bool)) PipelineEngine {
	if m, ok := p.(*memoryEngine); ok {
		m.geo = fn
	}
	return p
}

// withASN attaches an ASN lookup function.
func withASN(p PipelineEngine, fn func(string) (map[string]interface{}, bool)) PipelineEngine {
	if m, ok := p.(*memoryEngine); ok {
		m.asn = fn
	}
	return p
}

func NewMemoryEngineWithSamples() PipelineEngine {
	m := &memoryEngine{seq: 1, filterCache: map[string]*regexp.Regexp{}}
	m.sources = []*memSource{
		{ID: "syslog-udp", Type: "syslog", Name: "Syslog UDP", Enabled: true, Status: "healthy", Config: map[string]interface{}{"port": 9514}},
		{ID: "http-bulk", Type: "http", Name: "HTTP Bulk", Enabled: false, Status: "disabled", Config: map[string]interface{}{"port": 10080}},
	}
	m.dests = []memDest{
		{ID: "sentinel", Type: "sentinel", Name: "Microsoft Sentinel Data Lake", Status: "connected", Enabled: true, Config: map[string]interface{}{"tableName": "Custom_BibblLogs_CL"}},
		{ID: "s3-archive", Type: "s3", Name: "S3 Archive", Status: "warning", Enabled: false, Config: map[string]interface{}{}},
	}
	m.pipelines = []memPipe{
		{ID: "main", Name: "Main", Description: "Normalize -> filter -> ship", Functions: []string{"Parse CEF", "Drop noisy", "Eval fields"}, IPSource: "first_ipv4"},
		{ID: "passthru", Name: "Passthru", Description: "No-op", Functions: []string{}, IPSource: "first_ipv4"},
	}
	m.routes = []memRoute{
		{ID: "r1", Name: "CEF to Sentinel", Filter: "_raw?.includes('CEF:')", PipelineID: "main", Destination: "sentinel", Final: true},
		{ID: "default", Name: "default", Filter: "true", PipelineID: "passthru", Destination: "devnull", Final: true},
	}
	return m
}

// Sources
func (m *memoryEngine) GetSources() []struct {
	ID      string
	Name    string
	Type    string
	Config  map[string]interface{}
	Status  string
	Enabled bool
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]struct {
		ID      string
		Name    string
		Type    string
		Config  map[string]interface{}
		Status  string
		Enabled bool
	}, 0, len(m.sources))
	for _, s := range m.sources {
		res = append(res, struct {
			ID      string
			Name    string
			Type    string
			Config  map[string]interface{}
			Status  string
			Enabled bool
		}{ID: s.ID, Name: s.Name, Type: s.Type, Config: s.Config, Status: s.Status, Enabled: s.Enabled})
	}
	return res
}

func (m *memoryEngine) CreateSource(name, typ string, cfg map[string]interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("src-%d", m.seq)
	m.seq++
	s := &memSource{ID: id, Name: name, Type: typ, Config: cfg, Status: "stopped", Enabled: false}
	m.sources = append(m.sources, s)
	return s, nil
}

func (m *memoryEngine) UpdateSource(id, name string, cfg map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.sources {
		if m.sources[i].ID == id {
			m.sources[i].Name = name
			m.sources[i].Config = cfg
			return nil
		}
	}
	return errors.New("source not found")
}

func (m *memoryEngine) DeleteSource(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.sources {
		if m.sources[i].ID == id {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			return nil
		}
	}
	return errors.New("source not found")
}

func (m *memoryEngine) StartSource(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.sources {
		if m.sources[i].ID == id {
			m.sources[i].Enabled = true
			// If this is a syslog source, start the listener and wire to LogHub
			if m.sources[i].Type == "syslog" {
				// Build address and TLS
				host := "0.0.0.0"
				if v, ok := m.sources[i].Config["host"].(string); ok && v != "" {
					host = v
				}
				port := 6514
				if v, ok := m.sources[i].Config["port"].(int); ok && v > 0 {
					port = v
				} else if vf, ok := m.sources[i].Config["port"].(float64); ok && int(vf) > 0 {
					port = int(vf)
				}
				addr := fmt.Sprintf("%s:%d", host, port)

				var tlsConf *tls.Config
				protocol := "tcp"
				if v, ok := m.sources[i].Config["protocol"].(string); ok && v != "" {
					protocol = v
				}
				if protocol == "tls" {
					certFile, _ := m.sources[i].Config["certFile"].(string)
					keyFile, _ := m.sources[i].Config["keyFile"].(string)
					if certFile == "" || keyFile == "" {
						m.sources[i].Status = "error: missing TLS cert/key"
						return errors.New("missing TLS cert/key for syslog TLS listener")
					}
					cert, err := tls.LoadX509KeyPair(certFile, keyFile)
					if err != nil {
						m.sources[i].Status = "error: tls load failed"
						return fmt.Errorf("load tls cert: %w", err)
					}
					min := tls.VersionTLS12
					if mv, ok := m.sources[i].Config["minVersion"].(string); ok && mv == "1.3" {
						min = tls.VersionTLS13
					}
					tlsConf = &tls.Config{MinVersion: uint16(min), Certificates: []tls.Certificate{cert}}
				}

				if m.hub == nil {
					m.sources[i].Status = "error: hub unavailable"
					return errors.New("log hub not attached")
				}

				// handler to append to hub with batching for high throughput
				srcID := m.sources[i].ID
				batchHandler := syslogBatchHandler{sourceID: srcID, engine: m}
				collector := sysloginput.NewBatchCollector(batchHandler, 1000, 100*time.Millisecond)

				srv := sysloginput.New(addr, tlsConf, collector)
				// Optional allowlist (array of strings)
				if al, ok := m.sources[i].Config["allow"]; ok {
					var items []string
					switch v := al.(type) {
					case []string:
						items = v
					case []interface{}:
						for _, x := range v {
							if s, ok := x.(string); ok {
								items = append(items, s)
							}
						}
					}
					if len(items) > 0 {
						srv.SetAllowList(items)
					}
				}
				ctx, cancel := context.WithCancel(context.Background())
				if err := srv.Start(ctx); err != nil {
					m.sources[i].Status = "error: bind failed"
					cancel()
					return fmt.Errorf("start syslog listener on %s: %w", addr, err)
				}
				if WorkerRegistrar != nil {
					done := WorkerRegistrar()
					go func() { <-ctx.Done(); done() }()
				}
				log.Printf("source %s (%s) listening on %s", m.sources[i].Name, srcID, addr)
				m.sources[i].syslogSrv = srv
				m.sources[i].cancel = cancel
				m.sources[i].Status = "running"
				return nil
			}

			// Synthetic source with batch processing
			if m.sources[i].Type == "synthetic" {
				if m.hub == nil {
					m.sources[i].Status = "error: hub unavailable"
					return errors.New("log hub not attached")
				}
				idx := i
				if m.sources[i].synthGen == nil {
					srcID := m.sources[idx].ID
					m.sources[i].synthGen = syninput.NewBatch(func(batch []string) {
						m.processAndAppendBatch(srcID, batch)
						m.sources[idx].produced.Add(uint64(len(batch)))
					})
				}
				m.sources[i].synthGen.Start(m.sources[i].Config)
				m.sources[i].Status = "running"
				return nil
			}

			// Akamai DataStream 2 source
			if m.sources[i].Type == "akamai_ds2" {
				if m.hub == nil {
					m.sources[i].Status = "error: hub unavailable"
					return errors.New("log hub not attached")
				}
				cfg := m.sources[i].Config
				host, _ := cfg["host"].(string)
				clientToken, _ := cfg["clientToken"].(string)
				clientSecret, _ := cfg["clientSecret"].(string)
				accessToken, _ := cfg["accessToken"].(string)
				if host == "" || clientToken == "" || clientSecret == "" || accessToken == "" {
					m.sources[i].Status = "error: missing creds"
					return errors.New("akamai ds2 missing credentials")
				}
				creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
				cli := akamaiinput.NewClient(creds)
				p := &akamaiinput.Poller{Client: cli}
				if v, ok := cfg["intervalSeconds"].(int); ok && v > 0 {
					p.Interval = time.Duration(v) * time.Second
				} else if vf, ok := cfg["intervalSeconds"].(float64); ok && int(vf) > 0 {
					p.Interval = time.Duration(int(vf)) * time.Second
				}
				p.Streams = akamaiinput.ParseStreamIDs(cfg["streams"])
				srcID := m.sources[i].ID
				_ = p.Start(func(line string) { m.processAndAppend(srcID, line) })
				if WorkerRegistrar != nil {
					done := WorkerRegistrar()
					go func() { <-p.Done(); done() }()
				}
				m.sources[i].akamaiPoll = p
				m.sources[i].Status = "running"
				return nil
			}

			// Other source types: mark running; no demo emission
			m.sources[i].Status = "running"
			return nil
		}
	}
	return errors.New("source not found")
}

func (m *memoryEngine) StopSource(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.sources {
		if m.sources[i].ID == id {
			m.sources[i].Enabled = false
			// Stop any running syslog server
			if m.sources[i].syslogSrv != nil {
				_ = m.sources[i].syslogSrv.Stop()
				m.sources[i].syslogSrv = nil
			}
			if m.sources[i].synthGen != nil {
				m.sources[i].synthGen.Stop()
			}
			if m.sources[i].akamaiPoll != nil {
				m.sources[i].akamaiPoll.Stop()
				m.sources[i].akamaiPoll = nil
			}
			if m.sources[i].cancel != nil {
				m.sources[i].cancel()
				m.sources[i].cancel = nil
			}
			m.sources[i].Status = "stopped"
			return nil
		}
	}
	return errors.New("source not found")
}

// syslogHandler adapts incoming syslog messages to a callback.
type syslogHandler struct{ on func(string) }

func (h syslogHandler) Handle(message string) {
	if h.on != nil {
		h.on(message)
	}
}

// syslogBatchHandler processes batches of messages for high throughput.
type syslogBatchHandler struct {
	sourceID string
	engine   *memoryEngine
}

func (h syslogBatchHandler) HandleBatch(messages []string) {
	if h.engine != nil {
		h.engine.processAndAppendBatch(h.sourceID, messages)
	}
}

// applyParsers executes parser functions on an event
// Returns the modified event and whether parsing was successful
func (m *memoryEngine) applyParsers(functions []string, event map[string]interface{}) (map[string]interface{}, bool) {
	for _, fn := range functions {
		switch fn {
		case "Parse Versa KVP":
			if m.versaParser != nil {
				if err := m.versaParser.Parse(event); err != nil {
					// Log error but continue processing (lenient mode)
					log.Printf("Versa KVP parser error: %v", err)
				}
			}
		case "Parse Palo Alto CSV":
			if m.paloAltoParser != nil {
				if err := m.paloAltoParser.Parse(event); err != nil {
					log.Printf("Palo Alto CSV parser error: %v", err)
				}
			}
		}
	}
	return event, true
}

// processAndAppend performs a minimal in-memory routing and optional enrichment,
// then appends a rendered string to the hub for preview/UI purposes.
func (m *memoryEngine) processAndAppend(sourceID, msg string) {
	start := time.Now()
	ctx := context.Background()
	tr := otel.Tracer("bibbl/pipeline")
	ctx, span := tr.Start(ctx, "processAndAppend", trace.WithAttributes(
		attribute.String("source.id", sourceID),
		attribute.Int("msg.len", len(msg)),
	))
	_ = ctx
	// pick first matching route by regex, fallback to default (true)
	m.mu.RLock()
	routes := append([]memRoute(nil), m.routes...)
	pipes := append([]memPipe(nil), m.pipelines...)
	m.mu.RUnlock()
	matched := memRoute{}
	found := false
	for _, r := range routes {
		if r.Filter == "" || r.Filter == "true" {
			matched = r
			found = true
			break
		}
		if re := m.getFilterRegex(r.Filter); re != nil && re.MatchString(msg) {
			matched = r
			found = true
			break
		}
	}
	if !found {
		// try default route by name
		for _, r := range routes {
			if r.Name == "default" {
				matched = r
				found = true
				break
			}
		}
	}
	if !found || m.hub == nil {
		m.hub.Append(sourceID, msg)
		metrics.IngestEvents.WithLabelValues(sourceID, "", "").Inc()
		return
	}
	// resolve pipeline
	var pl memPipe
	for _, p := range pipes {
		if p.ID == matched.PipelineID {
			pl = p
			break
		}
	}
	// Create initial payload
	payload := map[string]interface{}{"_raw": msg}

	// Apply parsers first
	if len(pl.Functions) > 0 {
		payload, _ = m.applyParsers(pl.Functions, payload)
	}

	// Detect enrichment requirements
	wantGeo := false
	wantASN := false
	for _, f := range pl.Functions {
		if f == "geoip_enrich" || f == "GeoIP Enrich" {
			wantGeo = true
		}
		if f == "asn_enrich" || f == "ASN Enrich" {
			wantASN = true
		}
	}

	// Apply enrichment if requested
	if (wantGeo && m.geo != nil) || (wantASN && m.asn != nil) {
		// Resolve IP based on pipeline configuration
		ip := ""
		if pl.IPSource == "" || pl.IPSource == "first_ipv4" {
			ip = extractFirstIPv4(msg)
		} else {
			ip = m.extractIPBySource(pl.IPSource, msg)
		}
		if ip != "" {
			payload["ip"] = ip
		}
		if ip != "" && wantGeo && m.geo != nil {
			if geo, ok := m.geo(ip); ok && geo != nil {
				payload["geo"] = geo
			}
		}
		if ip != "" && wantASN && m.asn != nil {
			if asn, ok := m.asn(ip); ok && asn != nil {
				payload["asn"] = asn
			}
		}
	}

	// Render as JSON
	out := msg
	if b, err := json.Marshal(payload); err == nil {
		out = string(b)
	}
	m.hub.Append(sourceID, out)
	metrics.IngestEvents.WithLabelValues(sourceID, matched.Name, matched.Destination).Inc()
	lat := time.Since(start).Seconds()
	metrics.PipelineLatency.WithLabelValues(pl.Name, matched.Name, sourceID).Observe(lat)
	span.SetAttributes(attribute.String("pipeline", pl.Name), attribute.String("route", matched.Name), attribute.Bool("geo", wantGeo), attribute.Bool("asn", wantASN))
	span.End()
}

var reIPv4 = regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`)

func extractFirstIPv4(s string) string {
	if m := reIPv4.FindString(s); m != "" {
		return m
	}
	return ""
}

// processAndAppendBatch is a high-throughput batch processor that amortizes
// lock acquisition and regex compilation across multiple events.
func (m *memoryEngine) processAndAppendBatch(sourceID string, messages []string) {
	if len(messages) == 0 {
		return
	}

	start := time.Now()
	ctx := context.Background()
	tr := otel.Tracer("bibbl/pipeline")
	ctx, span := tr.Start(ctx, "processAndAppendBatch", trace.WithAttributes(
		attribute.String("source.id", sourceID),
		attribute.Int("batch.size", len(messages)),
	))
	defer span.End()

	// Acquire route/pipeline snapshot once for entire batch
	m.mu.RLock()
	routes := append([]memRoute(nil), m.routes...)
	pipes := append([]memPipe(nil), m.pipelines...)
	m.mu.RUnlock()

	// Pre-compile all filter regexes to avoid repeated compilation
	filterRegexes := make(map[string]*regexp.Regexp)
	for _, r := range routes {
		if r.Filter != "" && r.Filter != "true" {
			if re := m.getFilterRegex(r.Filter); re != nil {
				filterRegexes[r.Filter] = re
			}
		}
	}

	// Find default route once
	var defaultRoute *memRoute
	for i := range routes {
		if routes[i].Name == "default" {
			defaultRoute = &routes[i]
			break
		}
	}

	// Process each message with minimal overhead
	for _, msg := range messages {
		if strings.TrimSpace(msg) == "" {
			continue
		}

		// Fast route matching with cached regexes
		matched := memRoute{}
		found := false
		for _, r := range routes {
			if r.Filter == "" || r.Filter == "true" {
				matched = r
				found = true
				break
			}
			if re, ok := filterRegexes[r.Filter]; ok && re.MatchString(msg) {
				matched = r
				found = true
				break
			}
		}
		if !found && defaultRoute != nil {
			matched = *defaultRoute
			found = true
		}

		if !found || m.hub == nil {
			if m.hub != nil {
				m.hub.Append(sourceID, msg)
			}
			metrics.IngestEvents.WithLabelValues(sourceID, "", "").Inc()
			continue
		}

		// Resolve pipeline
		var pl memPipe
		for _, p := range pipes {
			if p.ID == matched.PipelineID {
				pl = p
				break
			}
		}

		// Create initial payload
		payload := map[string]interface{}{"_raw": msg}

		// Apply parsers first
		if len(pl.Functions) > 0 {
			payload, _ = m.applyParsers(pl.Functions, payload)
		}

		// Check enrichment requirements
		wantGeo := false
		wantASN := false
		for _, f := range pl.Functions {
			if f == "geoip_enrich" || f == "GeoIP Enrich" {
				wantGeo = true
			}
			if f == "asn_enrich" || f == "ASN Enrich" {
				wantASN = true
			}
		}

		// Apply enrichment if requested
		if (wantGeo && m.geo != nil) || (wantASN && m.asn != nil) {
			ip := ""
			if pl.IPSource == "" || pl.IPSource == "first_ipv4" {
				ip = extractFirstIPv4(msg)
			} else {
				ip = m.extractIPBySource(pl.IPSource, msg)
			}
			if ip != "" {
				payload["ip"] = ip
			}
			if ip != "" && wantGeo && m.geo != nil {
				if geo, ok := m.geo(ip); ok && geo != nil {
					payload["geo"] = geo
				}
			}
			if ip != "" && wantASN && m.asn != nil {
				if asn, ok := m.asn(ip); ok && asn != nil {
					payload["asn"] = asn
				}
			}
		}

		// Render as JSON
		out := msg
		if b, err := json.Marshal(payload); err == nil {
			out = string(b)
		}

		m.hub.Append(sourceID, out)
		metrics.IngestEvents.WithLabelValues(sourceID, matched.Name, matched.Destination).Inc()
	}

	// Record batch metrics
	batchLatency := time.Since(start).Seconds()
	for _, r := range routes {
		// Find any pipeline used by this route
		for _, p := range pipes {
			if p.ID == r.PipelineID {
				metrics.PipelineLatency.WithLabelValues(p.Name, r.Name, sourceID).Observe(batchLatency / float64(len(messages)))
				break
			}
		}
	}

	span.SetAttributes(
		attribute.Int("processed", len(messages)),
		attribute.Float64("batch_latency_sec", batchLatency),
		attribute.Float64("per_event_latency_ms", batchLatency*1000/float64(len(messages))),
	)
}

// extractIPBySource supports IPSource formats; currently only "field:<name>".
// It attempts to find an IPv4 value for the named field either in key=value
// form (field=1.2.3.4) or JSON ("field":"1.2.3.4"). Falls back to first IPv4.
func (m *memoryEngine) extractIPBySource(src, raw string) string {
	if !strings.HasPrefix(src, "field:") {
		return extractFirstIPv4(raw)
	}
	field := strings.TrimSpace(strings.TrimPrefix(src, "field:"))
	if field == "" {
		return extractFirstIPv4(raw)
	}
	// key=value pattern
	keyEq := regexp.MustCompile(regexp.QuoteMeta(field) + `=(` + `\d+\.\d+\.\d+\.\d+` + `)`)
	if mm := keyEq.FindStringSubmatch(raw); len(mm) == 2 {
		return mm[1]
	}
	// JSON pattern "field":"ip" OR "field" : "ip"
	jsonPat := regexp.MustCompile(`"` + regexp.QuoteMeta(field) + `"\s*:\s*"(` + `\d+\.\d+\.\d+\.\d+` + `)"`)
	if mm := jsonPat.FindStringSubmatch(raw); len(mm) == 2 {
		return mm[1]
	}
	return extractFirstIPv4(raw)
}

// getFilterRegex returns a cached compiled regex for a route filter, compiling
// it on first use. If compilation fails, nil is cached is not stored (to allow
// later changes) and nil returned.
func (m *memoryEngine) getFilterRegex(pattern string) *regexp.Regexp {
	if pattern == "" || pattern == "true" {
		return nil
	}
	m.mu.RLock()
	if m.filterCache != nil {
		if re, ok := m.filterCache[pattern]; ok {
			m.mu.RUnlock()
			return re
		}
	}
	m.mu.RUnlock()
	// compile outside lock
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	m.mu.Lock()
	if m.filterCache == nil {
		m.filterCache = map[string]*regexp.Regexp{}
	}
	m.filterCache[pattern] = re
	m.mu.Unlock()
	return re
}

// Buffers
func (m *memoryEngine) GetBuffers() []struct {
	SourceID   string
	Size       int
	Capacity   int
	Dropped    int
	OldestUnix int64
	NewestUnix int64
	LastError  string
} {
	// simple deterministic samples keyed to sources
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]struct {
		SourceID   string
		Size       int
		Capacity   int
		Dropped    int
		OldestUnix int64
		NewestUnix int64
		LastError  string
	}, 0, len(m.sources))
	for i, s := range m.sources {
		out = append(out, struct {
			SourceID   string
			Size       int
			Capacity   int
			Dropped    int
			OldestUnix int64
			NewestUnix int64
			LastError  string
		}{
			SourceID:   s.ID,
			Size:       (i + 1) * 10,
			Capacity:   1000,
			Dropped:    0,
			OldestUnix: 0,
			NewestUnix: 0,
			LastError:  "",
		})
	}
	return out
}

func (m *memoryEngine) ResetBuffer(sourceID string) error {
	// no-op for memory engine
	return nil
}

// In-memory simulated buffer config state
// (could be persisted later)
var (
	defaultMinCap = 100
	defaultMaxCap = 5000
)

// GetBuffer returns a richer view including auto sizing flags.
func (m *memoryEngine) GetBuffer(sourceID string) (struct {
	SourceID   string
	Size       int
	Capacity   int
	Dropped    int
	OldestUnix int64
	NewestUnix int64
	LastError  string
	Auto       bool
	MinCap     int
	MaxCap     int
}, bool) {
	// Derive base stats from GetBuffers for simplicity
	base := struct {
		SourceID   string
		Size       int
		Capacity   int
		Dropped    int
		OldestUnix int64
		NewestUnix int64
		LastError  string
		Auto       bool
		MinCap     int
		MaxCap     int
	}{}
	m.mu.RLock()
	defer m.mu.RUnlock()
	// find source index
	idx := -1
	for i, s := range m.sources {
		if s.ID == sourceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return base, false
	}
	// fabricate stats consistent with GetBuffers
	size := (idx + 1) * 10
	cap := 1000
	base.SourceID = sourceID
	base.Size = size
	base.Capacity = cap
	base.Dropped = 0
	base.OldestUnix = 0
	base.NewestUnix = 0
	base.LastError = ""
	if m.bufferAuto == nil {
		m.bufferAuto = map[string]bool{}
	}
	if m.bufferCap == nil {
		m.bufferCap = map[string]int{}
	}
	if m.bufferMin == nil {
		m.bufferMin = map[string]int{}
	}
	if m.bufferMax == nil {
		m.bufferMax = map[string]int{}
	}
	base.Auto = m.bufferAuto[sourceID]
	if v, ok := m.bufferCap[sourceID]; ok {
		base.Capacity = v
	}
	if v, ok := m.bufferMin[sourceID]; ok {
		base.MinCap = v
	} else {
		base.MinCap = defaultMinCap
	}
	if v, ok := m.bufferMax[sourceID]; ok {
		base.MaxCap = v
	} else {
		base.MaxCap = defaultMaxCap
	}
	return base, true
}

// UpdateBufferConfig adjusts capacity and auto behavior.
func (m *memoryEngine) UpdateBufferConfig(sourceID string, capacity *int, auto *bool, minCap *int, maxCap *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// verify source exists
	found := false
	for _, s := range m.sources {
		if s.ID == sourceID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("source not found")
	}
	if m.bufferAuto == nil {
		m.bufferAuto = map[string]bool{}
	}
	if m.bufferCap == nil {
		m.bufferCap = map[string]int{}
	}
	if m.bufferMin == nil {
		m.bufferMin = map[string]int{}
	}
	if m.bufferMax == nil {
		m.bufferMax = map[string]int{}
	}
	if capacity != nil && *capacity > 0 {
		m.bufferCap[sourceID] = *capacity
	}
	if auto != nil {
		m.bufferAuto[sourceID] = *auto
	}
	if minCap != nil && *minCap > 0 {
		m.bufferMin[sourceID] = *minCap
	}
	if maxCap != nil && *maxCap > 0 {
		m.bufferMax[sourceID] = *maxCap
	}
	// Simple auto-mode heuristic simulation: if auto enabled and size near capacity, bump capacity up to maxCap (or +25%)
	if m.bufferAuto[sourceID] {
		curCap := m.bufferCap[sourceID]
		if curCap == 0 {
			curCap = 1000
		}
		minv := m.bufferMin[sourceID]
		if minv == 0 {
			minv = defaultMinCap
		}
		maxv := m.bufferMax[sourceID]
		if maxv == 0 {
			maxv = defaultMaxCap
		}
		// simulate current size proportional to capacity usage
		size := curCap / 2
		if size > int(float64(curCap)*0.8) && curCap < maxv { // grow
			next := curCap + curCap/4
			if next > maxv {
				next = maxv
			}
			m.bufferCap[sourceID] = next
		} else if size < int(float64(curCap)*0.2) && curCap > minv { // shrink
			next := curCap - curCap/4
			if next < minv {
				next = minv
			}
			m.bufferCap[sourceID] = next
		}
	}
	return nil
}

// Destinations
func (m *memoryEngine) GetDestinations() []struct {
	ID      string
	Name    string
	Type    string
	Status  string
	Config  map[string]interface{}
	Enabled bool
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]struct {
		ID      string
		Name    string
		Type    string
		Status  string
		Config  map[string]interface{}
		Enabled bool
	}, 0, len(m.dests))
	for _, d := range m.dests {
		res = append(res, struct {
			ID      string
			Name    string
			Type    string
			Status  string
			Config  map[string]interface{}
			Enabled bool
		}{ID: d.ID, Name: d.Name, Type: d.Type, Status: d.Status, Config: d.Config, Enabled: d.Enabled})
	}
	return res
}

func (m *memoryEngine) CreateDestination(name, typ string, cfg map[string]interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("dst-%d", m.seq)
	m.seq++
	// Default to "disconnected" until a real health-check succeeds.
	// This avoids the UI always showing "connected" for new destinations.
	status := "disconnected"
	// If caller explicitly provides a status, honor it.
	if s, ok := cfg["status"].(string); ok && s != "" {
		status = s
	}
	d := memDest{ID: id, Name: name, Type: typ, Status: status, Enabled: true, Config: cfg}
	m.dests = append(m.dests, d)
	return d, nil
}

func (m *memoryEngine) UpdateDestination(id, name string, cfg map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.dests {
		if m.dests[i].ID == id {
			m.dests[i].Name = name
			m.dests[i].Config = cfg
			return nil
		}
	}
	return errors.New("destination not found")
}

func (m *memoryEngine) DeleteDestination(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.dests {
		if m.dests[i].ID == id {
			m.dests = append(m.dests[:i], m.dests[i+1:]...)
			return nil
		}
	}
	return errors.New("destination not found")
}

func (m *memoryEngine) PatchDestination(id string, patch map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.dests {
		if m.dests[i].ID == id {
			if v, ok := patch["name"].(string); ok {
				m.dests[i].Name = v
			}
			if v, ok := patch["status"].(string); ok {
				m.dests[i].Status = v
			}
			if v, ok := patch["config"].(map[string]interface{}); ok {
				m.dests[i].Config = v
			}
			if v, ok := patch["enabled"].(bool); ok {
				m.dests[i].Enabled = v
			}
			return nil
		}
	}
	return errors.New("destination not found")
}

// Pipelines
func (m *memoryEngine) GetPipelines() []struct {
	ID          string
	Name        string
	Description string
	Functions   []string
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]struct {
		ID          string
		Name        string
		Description string
		Functions   []string
	}, 0, len(m.pipelines))
	for _, p := range m.pipelines {
		res = append(res, struct {
			ID          string
			Name        string
			Description string
			Functions   []string
		}{ID: p.ID, Name: p.Name, Description: p.Description, Functions: p.Functions})
	}
	return res
}

func (m *memoryEngine) CreatePipeline(name, desc string, fns []string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("pipe-%d", m.seq)
	m.seq++
	p := memPipe{ID: id, Name: name, Description: desc, Functions: fns}
	m.pipelines = append(m.pipelines, p)
	return p, nil
}

func (m *memoryEngine) UpdatePipeline(id, name, desc string, fns []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.pipelines {
		if m.pipelines[i].ID == id {
			m.pipelines[i].Name = name
			m.pipelines[i].Description = desc
			m.pipelines[i].Functions = fns
			return nil
		}
	}
	return errors.New("pipeline not found")
}

func (m *memoryEngine) DeletePipeline(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.pipelines {
		if m.pipelines[i].ID == id {
			m.pipelines = append(m.pipelines[:i], m.pipelines[i+1:]...)
			return nil
		}
	}
	return errors.New("pipeline not found")
}

// Routes
func (m *memoryEngine) GetRoutes() []struct {
	ID          string
	Name        string
	Filter      string
	PipelineID  string
	Destination string
	Final       bool
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]struct {
		ID          string
		Name        string
		Filter      string
		PipelineID  string
		Destination string
		Final       bool
	}, 0, len(m.routes))
	for _, r := range m.routes {
		res = append(res, struct {
			ID          string
			Name        string
			Filter      string
			PipelineID  string
			Destination string
			Final       bool
		}{ID: r.ID, Name: r.Name, Filter: r.Filter, PipelineID: r.PipelineID, Destination: r.Destination, Final: r.Final})
	}
	return res
}

func (m *memoryEngine) CreateRoute(name, filter, pipelineID, destination string, final bool) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("route-%d", m.seq)
	m.seq++
	r := memRoute{ID: id, Name: name, Filter: filter, PipelineID: pipelineID, Destination: destination, Final: final}
	m.routes = append(m.routes, r)
	return r, nil
}

func (m *memoryEngine) UpdateRoute(id, name, filter, pipelineID, destination string, final bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.routes {
		if m.routes[i].ID == id {
			m.routes[i].Name = name
			m.routes[i].Filter = filter
			m.routes[i].PipelineID = pipelineID
			m.routes[i].Destination = destination
			m.routes[i].Final = final
			return nil
		}
	}
	return errors.New("route not found")
}

func (m *memoryEngine) DeleteRoute(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.routes {
		if m.routes[i].ID == id {
			m.routes = append(m.routes[:i], m.routes[i+1:]...)
			return nil
		}
	}
	return errors.New("route not found")
}
