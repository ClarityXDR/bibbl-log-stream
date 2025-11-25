package api

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	azureauth "bibbl/internal/azure/auth"
	"bibbl/internal/config"
	akamaiinput "bibbl/internal/inputs/akamai"
	"bibbl/internal/metrics"
	"bibbl/internal/platform/logger"
	"bibbl/internal/version"
	"bibbl/internal/web"
	tlsutil "bibbl/pkg/tls"
)

// WorkerRegistrar allows subsystems (e.g. inputs) to register background goroutines
// without tight coupling. Set by NewServer.
var WorkerRegistrar func() func()

// PipelineEngine declares the minimal contract the API layer needs.
// Implementations live elsewhere; this keeps api decoupled and tests easy.
type PipelineEngine interface {
	// Sources
	GetSources() []struct {
		ID      string
		Name    string
		Type    string
		Config  map[string]interface{}
		Status  string
		Enabled bool
	}
	CreateSource(name, typ string, cfg map[string]interface{}) (interface{}, error)
	UpdateSource(id, name string, cfg map[string]interface{}) error
	DeleteSource(id string) error
	StartSource(id string) error
	StopSource(id string) error

	// Buffers (per-source)
	GetBuffers() []struct {
		SourceID   string
		Size       int
		Capacity   int
		Dropped    int
		OldestUnix int64
		NewestUnix int64
		LastError  string
	}
	ResetBuffer(sourceID string) error
	// New granular buffer access
	GetBuffer(sourceID string) (struct {
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
	}, bool)
	UpdateBufferConfig(sourceID string, capacity *int, auto *bool, minCap *int, maxCap *int) error

	// Destinations
	GetDestinations() []struct {
		ID      string
		Name    string
		Type    string
		Status  string
		Config  map[string]interface{}
		Enabled bool
	}
	CreateDestination(name, typ string, cfg map[string]interface{}) (interface{}, error)
	UpdateDestination(id, name string, cfg map[string]interface{}) error
	DeleteDestination(id string) error
	PatchDestination(id string, patch map[string]interface{}) error

	// Pipelines
	GetPipelines() []struct {
		ID          string
		Name        string
		Description string
		Functions   []string
	}
	CreatePipeline(name, desc string, fns []string) (interface{}, error)
	UpdatePipeline(id, name, desc string, fns []string) error
	DeletePipeline(id string) error
	GetPipelineStats() []PipelineStats

	// Routes
	GetRoutes() []struct {
		ID          string
		Name        string
		Filter      string
		PipelineID  string
		Destination string
		Final       bool
	}
	CreateRoute(name, filter, pipelineID, destination string, final bool) (interface{}, error)
	UpdateRoute(id, name, filter, pipelineID, destination string, final bool) error
	DeleteRoute(id string) error
}

type Server struct {
	app       *fiber.App
	cfg       *config.Config
	pipeline  PipelineEngine
	hub       *LogHub
	azureAuth *azureauth.Manager
	activeSSE sync.WaitGroup // tracks active SSE connections
	auditMu   sync.Mutex
	auditFile *os.File
	workers   sync.WaitGroup
	tracer    trace.Tracer
	// Enrichment assets (in-memory)
	geoMu     sync.RWMutex
	geoIP     any
	geoIPPath string
	asnDB     any
	asnPath   string
}

// paginate slices a list given limit & offset with sane defaults and bounds.
func paginate[T any](all []T, limit, offset int) ([]T, int, int, int) {
	// Returns page slice, total, effectiveLimit, effectiveOffset
	total := len(all)
	if offset < 0 {
		offset = 0
	}
	maxLimit := 500
	defLimit := 100
	switch {
	case limit <= 0:
		limit = defLimit
	case limit > maxLimit:
		limit = maxLimit
	}
	if offset > total {
		return []T{}, total, limit, offset
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, limit, offset
}

// buildPaginationLinks constructs RFC5988 Link header values for pagination
// rels emitted: first, last, prev, next (only those applicable)
func buildPaginationLinks(r *http.Request, total, limit, offset, count int) string {
	if limit <= 0 {
		return ""
	}
	if total <= 0 {
		return ""
	}
	// base URL without existing paging params
	u := *r.URL
	q := u.Query()
	// helper to set limit/offset and serialize
	makeURL := func(off int) string {
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(off))
		u.RawQuery = q.Encode()
		return u.String()
	}
	var parts []string
	// first
	parts = append(parts, fmt.Sprintf("<%s>; rel=\"first\"", makeURL(0)))
	// last page offset
	lastOffset := 0
	if total > 0 {
		lastOffset = ((total - 1) / limit) * limit
	}
	parts = append(parts, fmt.Sprintf("<%s>; rel=\"last\"", makeURL(lastOffset)))
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		parts = append(parts, fmt.Sprintf("<%s>; rel=\"prev\"", makeURL(prev)))
	}
	if offset+count < total {
		next := offset + count
		parts = append(parts, fmt.Sprintf("<%s>; rel=\"next\"", makeURL(next)))
	}
	return strings.Join(parts, ", ")
}

// simple token bucket (per-process) for coarse rate limiting
type rateLimiter struct {
	mu           sync.Mutex
	cap          int
	tokens       float64
	refillPerSec float64
	last         time.Time
}

func newRateLimiter(maxPerMin int) *rateLimiter {
	if maxPerMin <= 0 {
		return nil
	}
	return &rateLimiter{cap: maxPerMin, tokens: float64(maxPerMin), refillPerSec: float64(maxPerMin) / 60.0, last: time.Now()}
}
func (r *rateLimiter) allow() bool {
	if r == nil {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	dt := now.Sub(r.last).Seconds()
	r.last = now
	r.tokens += dt * r.refillPerSec
	if r.tokens > float64(r.cap) {
		r.tokens = float64(r.cap)
	}
	if r.tokens >= 1 {
		r.tokens -= 1
		return true
	}
	return false
}

func NewServer(cfg *config.Config) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		BodyLimit:    cfg.Server.MaxRequestBytes,
	})

	log := logger.Slog()
	if log == nil {
		logger.Init(logger.Config{Level: cfg.Logging.Level, Format: cfg.Logging.Format})
		log = logger.Slog()
	}
	rl := newRateLimiter(cfg.Server.RateLimitPerMin)

	metrics.Init()

	tr := otel.Tracer("bibbl/server")

	// Panic recovery, tracing, auth & logging middleware
	app.Use(func(c *fiber.Ctx) (err error) {
		start := time.Now()
		metrics.HTTPInFlight.Inc()
		defer metrics.HTTPInFlight.Dec()
		// Request / Trace ID
		rid := c.Get("X-Request-Id")
		if rid == "" {
			rid = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		c.Set("X-Request-Id", rid)
		// Basic auth token check (skip for health/metrics/version)
		path := c.Path()
		if !strings.HasPrefix(path, "/metrics") && !strings.Contains(path, "/health") && !strings.Contains(path, "/version") {
			if cfg.Server.AuthToken != "" {
				auth := c.Get("Authorization")
				if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != cfg.Server.AuthToken {
					return c.Status(401).JSON(fiber.Map{"error": "unauthorized", "requestId": rid})
				}
			}
		}
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic", "err", rec, "requestId", rid)
				_ = c.Status(500).JSON(fiber.Map{"error": "internal server error", "requestId": rid})
			}
			lat := time.Since(start)
			status := c.Response().StatusCode()
			requestSize := int64(len(c.Request().Body()))
			responseSize := int64(len(c.Response().Body()))

			// Record HTTP metrics
			metrics.RecordHTTPRequest(c.Method(), path, status, lat, requestSize, responseSize)

			log.Debug("req", "method", c.Method(), "path", path, "status", status, "latency_ms", lat.Milliseconds(), "requestId", rid)
		}()
		if !rl.allow() {
			return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
		}
		return c.Next()
	})

	// Metrics endpoint using custom registry
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
		Registry:      metrics.Registry(),
	})))

	// Basic health endpoint (composite added later once server fully constructed)
	app.Get("/api/v1/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/api/v1/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": version.Version, "commit": version.Commit, "date": version.Date})
	})
	// Readiness endpoint: reports if core subsystems initialized (currently always true once server constructed)
	app.Get("/ready", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/live", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	// Security headers and isolation policies (configurable via server.security_headers)
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "0")
		c.Set("Referrer-Policy", "no-referrer")
		// CSP configurable for React compatibility: allows inline styles and necessary scripts
		c.Set("Content-Security-Policy", cfg.Server.ContentSecurityPolicy)
		if cfg.Server.SecurityHeaders.HSTS.Enabled && cfg.TLSConfigured() {
			maxAge := cfg.Server.SecurityHeaders.HSTS.MaxAge
			if maxAge <= 0 {
				maxAge = 63072000
			}
			parts := []string{fmt.Sprintf("max-age=%d", maxAge)}
			if cfg.Server.SecurityHeaders.HSTS.IncludeSubdomains {
				parts = append(parts, "includeSubDomains")
			}
			if cfg.Server.SecurityHeaders.HSTS.Preload {
				parts = append(parts, "preload")
			}
			c.Set("Strict-Transport-Security", strings.Join(parts, "; "))
		}
		if pp := strings.TrimSpace(cfg.Server.SecurityHeaders.PermissionsPolicy); pp != "" {
			c.Set("Permissions-Policy", pp)
		}
		if coop := strings.TrimSpace(cfg.Server.SecurityHeaders.COOP); coop != "" {
			c.Set("Cross-Origin-Opener-Policy", coop)
		}
		if coep := strings.TrimSpace(cfg.Server.SecurityHeaders.COEP); coep != "" {
			c.Set("Cross-Origin-Embedder-Policy", coep)
		}
		if corp := strings.TrimSpace(cfg.Server.SecurityHeaders.CORP); corp != "" {
			c.Set("Cross-Origin-Resource-Policy", corp)
		}
		return c.Next()
	})

	// RBAC middleware (after basic auth). Adds roles to locals and enforces role for mutating endpoints
	app.Use(func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			tok := strings.TrimPrefix(auth, "Bearer ")
			if roles, ok := cfg.Server.AuthTokens[tok]; ok {
				c.Locals("roles", roles)
			}
		}
		// Require at least one role for mutating methods on /api (POST/PUT/PATCH/DELETE)
		if strings.HasPrefix(c.Path(), "/api/") && (c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodPatch || c.Method() == fiber.MethodDelete) {
			// Allow regex/enrich previews without roles (considered read-like even if POST)
			if strings.Contains(c.Path(), "/preview/regex") || strings.Contains(c.Path(), "/preview/enrich") {
				return c.Next()
			}
			if c.Locals("roles") == nil {
				return c.Status(403).JSON(fiber.Map{"error": "forbidden - role required"})
			}
		}
		return c.Next()
	})
	// Basic server info for UI
	app.Get("/api/v1/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":        cfg.Server.Host,
			"port":        cfg.Server.Port,
			"http_addr":   cfg.HTTPAddr(),
			"tls_enabled": cfg.TLSConfigured(),
			"tls_min":     cfg.Server.TLS.MinVersion,
		})
	})

	// Register Gorilla Mux API routes and mount into Fiber
	// Note: place BEFORE static UI so APIs are always matched first.
	muxRouter := mux.NewRouter()
	hub, _ := NewLogHub("./sandbox/library")
	srv := &Server{app: app, cfg: cfg, hub: hub, azureAuth: azureauth.NewManager(), tracer: tr}

	// Test-mode flag: when BIBBL_TEST=1 we skip automatic creation of network-listening
	// inputs (e.g., default syslog) to avoid OS firewall prompts during `go test`.
	skipDefaults := os.Getenv("BIBBL_TEST") == "1"

	// Composite health endpoint now that srv exists
	app.Get("/api/v1/healthz", func(c *fiber.Ctx) error {
		sources := len(srv.pipeline.GetSources())
		destinations := len(srv.pipeline.GetDestinations())
		buffers := len(srv.pipeline.GetBuffers())
		// We cannot get WaitGroup count directly; expose 0 placeholder
		workers := 0
		audit := false
		if srv.auditFile != nil {
			if _, err := srv.auditFile.Write([]byte{}); err == nil {
				audit = true
			}
		}
		return c.JSON(fiber.Map{
			"status":        "ok",
			"sources":       sources,
			"destinations":  destinations,
			"buffers":       buffers,
			"workers":       workers,
			"build":         fiber.Map{"version": version.Version, "commit": version.Commit, "date": version.Date},
			"auditWritable": audit,
		})
	})
	WorkerRegistrar = srv.RegisterWorker
	// Build pipeline engine with hub and enrichment hooks
	base := NewMemoryEngine()
	fmt.Printf("DEBUG: Base engine created, pipelines: %d\n", len(base.GetPipelines()))
	// Geo hook
	geoHook := func(ip string) (map[string]interface{}, bool) {
		srv.geoMu.RLock()
		reader := srv.geoIP
		srv.geoMu.RUnlock()
		if reader == nil {
			return nil, false
		}
		res, err := geoipLookup(reader, net.ParseIP(ip))
		if err != nil {
			return nil, false
		}
		m := map[string]interface{}{
			"city":        res.City,
			"country":     res.Country,
			"countryIso":  res.CountryISO,
			"subdivision": res.Subdiv,
			"lat":         res.Lat,
			"lon":         res.Lon,
			"timezone":    res.Timezone,
			"private":     res.Private,
			"ipv6":        res.IPv6,
		}
		return m, true
	}
	// ASN hook
	asnHook := func(ip string) (map[string]interface{}, bool) {
		srv.geoMu.RLock()
		reader := srv.asnDB
		srv.geoMu.RUnlock()
		if reader == nil {
			return nil, false
		}
		res, err := asnLookup(reader, net.ParseIP(ip))
		if err != nil {
			return nil, false
		}
		if res.Number == 0 && res.Org == "" {
			return nil, false
		}
		return map[string]interface{}{"asn": res.Number, "org": res.Org}, true
	}
	eng := withASN(withGeo(withHub(base, hub), geoHook), asnHook)
	srv.pipeline = eng

	// Periodic buffer metrics scrape (best-effort; simple polling)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for _, b := range srv.pipeline.GetBuffers() {
				metrics.BufferSize.WithLabelValues(b.SourceID).Set(float64(b.Size))
				metrics.BufferDroppedCurrent.WithLabelValues(b.SourceID).Set(float64(b.Dropped))
			}
		}
	}()

	// Special-case: Fiber-native SSE streaming endpoint.
	// Using Fiber here avoids buffering issues with the net/http adaptor and ensures
	// we can flush events to the client in real time.
	app.Get("/api/v1/sources/:id/stream", func(c *fiber.Ctx) error {
		srv := srv // capture
		srv.activeSSE.Add(1)
		defer srv.activeSSE.Done()
		id := c.Params("id")
		tail := 200
		if t := c.Query("tail"); t != "" {
			if n, err := strconv.Atoi(t); err == nil && n > 0 && n <= 2000 {
				tail = n
			}
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")

		// Subscribe before writing stream
		ch, cancel := srv.hub.Subscribe(id)
		done := c.Context().Done()

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			// initial comment to open stream
			_, _ = w.WriteString(": connected\n\n")
			_ = w.Flush()
			// send tail first
			for _, line := range srv.hub.Tail(id, tail) {
				_, _ = w.WriteString("data: " + escapeSSE(line) + "\n\n")
				_ = w.Flush()
			}
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case line, ok := <-ch:
					if !ok {
						cancel()
						return
					}
					_, _ = w.WriteString("data: " + escapeSSE(line) + "\n\n")
					_ = w.Flush()
				case <-done:
					cancel()
					return
				case <-ticker.C:
					_, _ = w.WriteString(": ping\n\n")
					_ = w.Flush()
				}
			}
		})
		return nil
	})

	// Seed sensible defaults
	// Only route /api to Gorilla Mux; UI/static served by Fiber below
	app.Use("/api", adaptor.HTTPHandler(muxRouter))
	if !skipDefaults && cfg.Inputs.Syslog.Enabled {
		// Only create default syslog if explicitly enabled via config/flag
		haveSyslog := false
		for _, s := range srv.pipeline.GetSources() {
			if s.Type == "syslog" {
				haveSyslog = true
				break
			}
		}
		if !haveSyslog {
			host := cfg.Inputs.Syslog.Host
			if host == "" {
				host = "0.0.0.0"
			}
			port := cfg.Inputs.Syslog.Port
			if port <= 0 {
				port = 6514
			}
			certFile := cfg.Inputs.Syslog.TLS.CertFile
			keyFile := cfg.Inputs.Syslog.TLS.KeyFile
			minVer := cfg.Inputs.Syslog.TLS.MinVersion
			if certFile == "" || keyFile == "" {
				cf, kf, err := tlsutil.EnsurePairExists("./certs/bibbl.crt", "./certs/bibbl.key", []string{"127.0.0.1", "localhost"}, 0)
				if err == nil {
					certFile, keyFile = cf, kf
				}
				if minVer == "" {
					minVer = "1.2"
				}
			}
			sysCfg := map[string]interface{}{
				"host":       host,
				"port":       port,
				"protocol":   "tls",
				"certFile":   certFile,
				"keyFile":    keyFile,
				"minVersion": minVer,
			}
			name := "Syslog"
			_, _ = srv.pipeline.CreateSource(name, "syslog", sysCfg)
			for _, s := range srv.pipeline.GetSources() {
				if s.Name == name {
					_ = srv.pipeline.StartSource(s.ID)
					break
				}
			}
		}
	}

	// 1b) Ensure a default Akamai DataStream 2 polling source if enabled by config and none exists
	haveAkamai := false
	for _, s := range srv.pipeline.GetSources() {
		if s.Type == "akamai_ds2" {
			haveAkamai = true
			break
		}
	}
	if !haveAkamai {
		aCfg := map[string]interface{}{
			"host":            cfg.Inputs.AkamaiDS2.Host,
			"clientToken":     cfg.Inputs.AkamaiDS2.ClientToken,
			"clientSecret":    cfg.Inputs.AkamaiDS2.ClientSecret,
			"accessToken":     cfg.Inputs.AkamaiDS2.AccessToken,
			"intervalSeconds": cfg.Inputs.AkamaiDS2.IntervalSeconds,
			"streams":         cfg.Inputs.AkamaiDS2.Streams,
			"status":          "needs_config",
		}
		_, _ = srv.pipeline.CreateSource("Akamai DataStream 2", "akamai_ds2", aCfg)
		// Start only if enabled + credentials present
		if cfg.Inputs.AkamaiDS2.Enabled && cfg.Inputs.AkamaiDS2.Host != "" && cfg.Inputs.AkamaiDS2.ClientToken != "" {
			for _, s := range srv.pipeline.GetSources() {
				if s.Type == "akamai_ds2" {
					_ = srv.pipeline.StartSource(s.ID)
					break
				}
			}
		}
	}
	// 2) Passthrough pipeline
	fmt.Printf("DEBUG: Before Passthrough - pipelines: %d\n", len(srv.pipeline.GetPipelines()))
	passthroughPipe, _ := srv.pipeline.CreatePipeline("Passthrough", "No-op pipeline", []string{})
	fmt.Printf("DEBUG: Passthrough created: %+v\n", passthroughPipe)
	var passthruID string
	for _, p := range srv.pipeline.GetPipelines() {
		if p.Name == "Passthrough" {
			passthruID = p.ID
			break
		}
	}
	fmt.Printf("DEBUG: After Passthrough - pipelines: %d\n", len(srv.pipeline.GetPipelines()))

	// 2b) Versa SD-WAN Parser pipeline
	fmt.Printf("DEBUG: Before Versa - pipelines: %d\n", len(srv.pipeline.GetPipelines()))
	versaPipe, err := srv.pipeline.CreatePipeline("Versa SD-WAN Parser", "Parse Versa Networks SD-WAN syslog (KVP format)", []string{"Parse Versa KVP"})
	if err != nil {
		fmt.Printf("DEBUG: Error creating Versa parser pipeline: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Versa pipeline created: %+v\n", versaPipe)
	}
	fmt.Printf("DEBUG: After Versa - pipelines: %d\n", len(srv.pipeline.GetPipelines()))
	_ = versaPipe // Reserved for future route creation

	// 2c) Palo Alto NGFW Parser pipeline
	fmt.Printf("DEBUG: Before Palo Alto - pipelines: %d\n", len(srv.pipeline.GetPipelines()))
	paloAltoPipe, err := srv.pipeline.CreatePipeline("Palo Alto NGFW Parser", "Parse Palo Alto Networks NGFW syslog (CSV format)", []string{"Parse Palo Alto CSV"})
	if err != nil {
		fmt.Printf("DEBUG: Error creating Palo Alto parser pipeline: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Palo Alto pipeline created: %+v\n", paloAltoPipe)
	}
	fmt.Printf("DEBUG: After Palo Alto - pipelines: %d\n", len(srv.pipeline.GetPipelines()))
	_ = paloAltoPipe // Reserved for future route creation

	// 2d) Example severity-based routing pipelines (users can enable these routes)
	// These routes demonstrate how to split traffic by severity to different destinations
	// Versa Critical Route: severity="Critical"
	fmt.Println("DEBUG: Creating example severity routes...")
	versaCritical := `event.severity === "Critical" || event.severity === "critical"`
	versaHigh := `event.severity === "High" || event.severity === "high"`
	versaMedium := `event.severity === "Medium" || event.severity === "medium" || event.severity === "Warning"`

	// Palo Alto severity patterns
	paloAltoCritical := `event.threat_severity === "critical" || event.severity_level_d > 8`
	paloAltoHigh := `event.threat_severity === "high" || (event.severity_level_d >= 6 && event.severity_level_d <= 8)`
	paloAltoMedium := `event.threat_severity === "medium" || (event.severity_level_d >= 3 && event.severity_level_d <= 5)`

	// Store these filter expressions for later use (commented out by default)
	_ = versaCritical
	_ = versaHigh
	_ = versaMedium
	_ = paloAltoCritical
	_ = paloAltoHigh
	_ = paloAltoMedium

	// 3) Sentinel Data Lake destination (idempotent create)
	_, _ = srv.pipeline.CreateDestination("Microsoft Sentinel Data Lake", "sentinel", map[string]interface{}{
		"tableName": "Custom_BibblLogs_CL",
		// performance tuning defaults
		"batchMaxEvents":   500,
		"batchMaxBytes":    512 * 1024, // 512KB per batch
		"flushIntervalSec": 5,
		"concurrency":      2,
		"compression":      "gzip",
	})
	var sentID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "Microsoft Sentinel Data Lake" {
			sentID = d.ID
			break
		}
	}

	// 3b) Azure Data Lake Gen2 destination (standard storage vs Sentinel). We seed a basic config
	// so users can test connectivity quickly without Sentinel onboarding friction.
	_, _ = srv.pipeline.CreateDestination("Azure Data Lake Gen2", "azure_datalake", map[string]interface{}{
		"storageAccount":   "<account>",
		"filesystem":       "logs",
		"directory":        "bibbl/raw/$(yyyy)/$(MM)/$(dd)/",
		"pathTemplate":     "bibbl/raw/$(yyyy)/$(MM)/$(dd)/$(HH)/data-$(mm).jsonl",
		"format":           "jsonl", // future: parquet
		"compression":      "gzip",
		"batchMaxEvents":   1000,
		"batchMaxBytes":    1024 * 1024, // 1MB
		"flushIntervalSec": 5,
		"concurrency":      4,
		"maxOpenFiles":     4,
		"status":           "disconnected",
	})
	var adlsID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "Azure Data Lake Gen2" {
			adlsID = d.ID
			break
		}
	}

	// 3c) Optional normalization pipeline for ADLS (ensures flattened JSON). Only create if absent.
	_, _ = srv.pipeline.CreatePipeline("ADLS Normalize", "Flatten & ensure timestamp for Azure Data Lake", []string{"Flatten Fields", "Ensure Timestamp", "Rename Standard Fields"})
	var adlsPipeID string
	for _, p := range srv.pipeline.GetPipelines() {
		if p.Name == "ADLS Normalize" {
			adlsPipeID = p.ID
			break
		}
	}

	// 3d) Azure Log Analytics Workspace destinations for severity-based routing
	// These can be configured with your Workspace ID and Shared Key via the UI
	// Critical Alerts - Fast flushing, smaller batches
	_, _ = srv.pipeline.CreateDestination("Sentinel Critical Alerts", "azure_loganalytics", map[string]interface{}{
		"workspaceID":      "", // Configure via UI
		"sharedKey":        "", // Configure via UI
		"logType":          "CriticalSecurityAlerts",
		"resourceGroup":    "",
		"batchMaxEvents":   250,
		"batchMaxBytes":    524288, // 512KB
		"flushIntervalSec": 5,      // 5 second flush for critical
		"concurrency":      3,
		"maxRetries":       5,
		"retryDelaySec":    2,
	})
	var criticalAlertDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "Sentinel Critical Alerts" {
			criticalAlertDestID = d.ID
			break
		}
	}

	// High Priority Alerts - Balanced performance
	_, _ = srv.pipeline.CreateDestination("Sentinel High Priority", "azure_loganalytics", map[string]interface{}{
		"workspaceID":      "", // Configure via UI
		"sharedKey":        "", // Configure via UI
		"logType":          "HighPriorityAlerts",
		"resourceGroup":    "",
		"batchMaxEvents":   500,
		"batchMaxBytes":    1048576, // 1MB
		"flushIntervalSec": 10,
		"concurrency":      2,
		"maxRetries":       3,
		"retryDelaySec":    2,
	})
	var highAlertDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "Sentinel High Priority" {
			highAlertDestID = d.ID
			break
		}
	}

	// Medium/Warning Alerts - Cost-optimized batching
	_, _ = srv.pipeline.CreateDestination("Sentinel Medium Alerts", "azure_loganalytics", map[string]interface{}{
		"workspaceID":      "", // Configure via UI
		"sharedKey":        "", // Configure via UI
		"logType":          "MediumPriorityAlerts",
		"resourceGroup":    "",
		"batchMaxEvents":   1000,
		"batchMaxBytes":    2097152, // 2MB
		"flushIntervalSec": 20,
		"concurrency":      2,
		"maxRetries":       3,
		"retryDelaySec":    3,
	})
	var mediumAlertDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "Sentinel Medium Alerts" {
			mediumAlertDestID = d.ID
			break
		}
	}
	_ = criticalAlertDestID // Reserved for future route creation
	_ = highAlertDestID     // Reserved for future route creation
	_ = mediumAlertDestID   // Reserved for future route creation

	// 3e) Full-stream archive destinations (S3 for Versa, ADLS for Palo Alto)
	// These capture ALL events from each parser for compliance/replay scenarios
	fmt.Println("DEBUG: Creating full-stream archive destinations...")

	// S3 destination for Versa SD-WAN full stream
	_, _ = srv.pipeline.CreateDestination("S3 Versa Full Stream", "s3", map[string]interface{}{
		"bucket":           "", // Configure via UI
		"region":           "us-east-1",
		"prefix":           "versa/raw/",
		"pathTemplate":     "versa/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/versa-${mm}-${ss}.jsonl.gz",
		"compression":      "gzip",
		"batchMaxEvents":   5000,
		"batchMaxBytes":    10485760, // 10MB
		"flushIntervalSec": 60,       // 1 minute
		"concurrency":      3,
	})
	var s3VersaDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "S3 Versa Full Stream" {
			s3VersaDestID = d.ID
			break
		}
	}

	// ADLS destination for Palo Alto NGFW full stream
	_, _ = srv.pipeline.CreateDestination("ADLS Palo Alto Full Stream", "azure_datalake", map[string]interface{}{
		"storageAccount":   "", // Configure via UI
		"filesystem":       "security-logs",
		"directory":        "paloalto/raw/",
		"pathTemplate":     "paloalto/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/paloalto-${mm}-${ss}.jsonl.gz",
		"format":           "jsonl",
		"compression":      "gzip",
		"batchMaxEvents":   5000,
		"batchMaxBytes":    10485760, // 10MB
		"flushIntervalSec": 60,       // 1 minute
		"concurrency":      3,
		"maxOpenFiles":     2,
	})
	var adlsPaloAltoDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "ADLS Palo Alto Full Stream" {
			adlsPaloAltoDestID = d.ID
			break
		}
	}

	// S3 destination for Palo Alto (alternative to ADLS)
	_, _ = srv.pipeline.CreateDestination("S3 Palo Alto Full Stream", "s3", map[string]interface{}{
		"bucket":           "", // Configure via UI
		"region":           "us-east-1",
		"prefix":           "paloalto/raw/",
		"pathTemplate":     "paloalto/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/paloalto-${mm}-${ss}.jsonl.gz",
		"compression":      "gzip",
		"batchMaxEvents":   5000,
		"batchMaxBytes":    10485760, // 10MB
		"flushIntervalSec": 60,       // 1 minute
		"concurrency":      3,
	})
	var s3PaloAltoDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "S3 Palo Alto Full Stream" {
			s3PaloAltoDestID = d.ID
			break
		}
	}

	// ADLS destination for Versa (alternative to S3)
	_, _ = srv.pipeline.CreateDestination("ADLS Versa Full Stream", "azure_datalake", map[string]interface{}{
		"storageAccount":   "", // Configure via UI
		"filesystem":       "security-logs",
		"directory":        "versa/raw/",
		"pathTemplate":     "versa/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/versa-${mm}-${ss}.jsonl.gz",
		"format":           "jsonl",
		"compression":      "gzip",
		"batchMaxEvents":   5000,
		"batchMaxBytes":    10485760, // 10MB
		"flushIntervalSec": 60,       // 1 minute
		"concurrency":      3,
		"maxOpenFiles":     2,
	})
	var adlsVersaDestID string
	for _, d := range srv.pipeline.GetDestinations() {
		if d.Name == "ADLS Versa Full Stream" {
			adlsVersaDestID = d.ID
			break
		}
	}

	// 3f) Create example routes for the parser pipelines and full-stream destinations
	// These routes demonstrate the routing patterns - users can enable/modify them
	fmt.Println("DEBUG: Creating example routes for parsers and destinations...")

	// Get pipeline IDs for routes
	var versaPipeID, paloAltoPipeID string
	for _, p := range srv.pipeline.GetPipelines() {
		if p.Name == "Versa SD-WAN Parser" {
			versaPipeID = p.ID
		} else if p.Name == "Palo Alto NGFW Parser" {
			paloAltoPipeID = p.ID
		}
	}

	// Example Route 1: Versa Critical to Sentinel
	if versaPipeID != "" && criticalAlertDestID != "" {
		checkAndCreateRoute := func(name, filter, pipeID, destID string, final bool) {
			exists := false
			for _, r := range srv.pipeline.GetRoutes() {
				if r.Name == name {
					exists = true
					break
				}
			}
			if !exists {
				_, _ = srv.pipeline.CreateRoute(name, filter, pipeID, destID, final)
				fmt.Printf("DEBUG: Created route: %s\n", name)
			}
		}

		checkAndCreateRoute(
			"Example: Versa Critical → Sentinel",
			`event.severity === "Critical" || event.severity === "critical"`,
			versaPipeID,
			criticalAlertDestID,
			false, // Don't stop - allow other routes to process
		)
	}

	// Example Route 2: Palo Alto High to Sentinel
	if paloAltoPipeID != "" && highAlertDestID != "" {
		exists := false
		for _, r := range srv.pipeline.GetRoutes() {
			if r.Name == "Example: Palo Alto High → Sentinel" {
				exists = true
				break
			}
		}
		if !exists {
			_, _ = srv.pipeline.CreateRoute(
				"Example: Palo Alto High → Sentinel",
				`event.threat_severity === "high" || (event.severity_level_d >= 6 && event.severity_level_d <= 8)`,
				paloAltoPipeID,
				highAlertDestID,
				false,
			)
			fmt.Println("DEBUG: Created route: Example: Palo Alto High → Sentinel")
		}
	}

	// Example Route 3: Versa Full Stream to S3
	if versaPipeID != "" && s3VersaDestID != "" {
		exists := false
		for _, r := range srv.pipeline.GetRoutes() {
			if r.Name == "Example: Versa Full → S3" {
				exists = true
				break
			}
		}
		if !exists {
			_, _ = srv.pipeline.CreateRoute(
				"Example: Versa Full → S3",
				"true", // All events
				versaPipeID,
				s3VersaDestID,
				false,
			)
			fmt.Println("DEBUG: Created route: Example: Versa Full → S3")
		}
	}

	// Example Route 4: Palo Alto Full Stream to ADLS
	if paloAltoPipeID != "" && adlsPaloAltoDestID != "" {
		exists := false
		for _, r := range srv.pipeline.GetRoutes() {
			if r.Name == "Example: Palo Alto Full → ADLS" {
				exists = true
				break
			}
		}
		if !exists {
			_, _ = srv.pipeline.CreateRoute(
				"Example: Palo Alto Full → ADLS",
				"true", // All events
				paloAltoPipeID,
				adlsPaloAltoDestID,
				false,
			)
			fmt.Println("DEBUG: Created route: Example: Palo Alto Full → ADLS")
		}
	}

	_ = s3PaloAltoDestID  // Reserved for additional routes
	_ = adlsVersaDestID   // Reserved for additional routes
	_ = mediumAlertDestID // Reserved for additional routes

	// 4) Default route to sentinel via passthrough (create only if absent)
	if passthruID != "" && sentID != "" {
		haveDefault := false
		for _, r := range srv.pipeline.GetRoutes() {
			if r.Name == "default" {
				haveDefault = true
				break
			}
		}
		if !haveDefault {
			_, _ = srv.pipeline.CreateRoute("default", "true", passthruID, sentID, true)
		}
	}
	// 4b) Route all to ADLS Normalize -> ADLS destination (create only if absent and we have IDs)
	if adlsPipeID != "" && adlsID != "" {
		haveAdls := false
		for _, r := range srv.pipeline.GetRoutes() {
			if r.Name == "adls-all" {
				haveAdls = true
				break
			}
		}
		if !haveAdls {
			_, _ = srv.pipeline.CreateRoute("adls-all", "true", adlsPipeID, adlsID, false)
		}
	}
	srv.RegisterRoutes(muxRouter)
	// Only route /api/* to Gorilla Mux; leave / to the SPA/static handler
	app.Use("/api", adaptor.HTTPHandler(muxRouter))

	// Serve index.html with no-cache to avoid stale builds
	serveIndex := func(c *fiber.Ctx) error {
		index, err := web.ReadIndex()
		if err != nil {
			return fiber.ErrNotFound
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Cache-Control", "no-store, max-age=0")
		return c.Send(index)
	}
	app.Get("/", serveIndex)
	app.Get("/index.html", serveIndex)
	app.Get("/sources", serveIndex)
	app.Get("/routes", serveIndex)
	app.Get("/pipelines", serveIndex)
	app.Get("/destinations", serveIndex)
	app.Get("/buffers", serveIndex)
	app.Get("/preview", serveIndex)
	app.Get("/azure", serveIndex)
	app.Get("/loadtest", serveIndex)

	// Static UI (embedded assets)
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       web.Static(),
		PathPrefix: "",
		Browse:     false,
		// Keep default cache headers for hashed assets
	}))

	// Note: Real API routes are handled by Gorilla Mux mounted above.

	// Regex preview endpoint: applies a named-capture regex to a sample string
	// Regex preview handler is provided via Gorilla Mux API under /api/v1

	// SPA fallback to index.html
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == http.MethodGet {
			return serveIndex(c)
		}
		return fiber.ErrNotFound
	})

	// open audit log file (append)
	if f, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		srv.auditFile = f
	}
	return srv
}

func (s *Server) Start() error {
	addr := s.cfg.HTTPAddr()
	if s.cfg.TLSConfigured() {
		// Ensure type matches tls.Config.MinVersion (uint16)
		var min uint16 = tls.VersionTLS12
		switch s.cfg.Server.TLS.MinVersion {
		case "1.3":
			min = tls.VersionTLS13
		case "1.2", "":
			min = tls.VersionTLS12
		}
		cert, err := tls.LoadX509KeyPair(s.cfg.Server.TLS.CertFile, s.cfg.Server.TLS.KeyFile)
		if err != nil {
			return err
		}
		cipherSuites := []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		}
		tlsCfg := &tls.Config{MinVersion: min, Certificates: []tls.Certificate{cert}, PreferServerCipherSuites: true, CipherSuites: cipherSuites}
		// Optional client cert auth
		if s.cfg.Server.TLS.ClientCAFile != "" {
			pemData, err := os.ReadFile(s.cfg.Server.TLS.ClientCAFile)
			if err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(pemData) {
					tlsCfg.ClientCAs = pool
					tlsCfg.ClientAuth = s.cfg.TLSClientAuthType()
				}
			} else {
				log.Printf("failed to load client CA file: %v", err)
			}
		}
		// Force IPv4 listener to avoid accidental IPv6-only binding on Windows
		network := "tcp4"
		log.Printf("starting HTTPS server on %s (min TLS %s)", addr, s.cfg.Server.TLS.MinVersion)
		ln, err := tls.Listen(network, addr, tlsCfg)
		if err != nil {
			return err
		}
		return s.app.Listener(ln)
	}

	log.Printf("starting HTTP server on %s", addr)
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error { return s.ShutdownWithContext(context.Background()) }

func (s *Server) ShutdownWithContext(ctx context.Context) error {
	// Wait for SSE (with timeout)
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	done := make(chan struct{})
	go func() { s.activeSSE.Wait(); s.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-c.Done():
	}
	// close app
	_ = s.app.ShutdownWithContext(c)
	if s.auditFile != nil {
		_ = s.auditFile.Close()
	}
	cancel()
	return nil
}

// RegisterWorker increments the background worker WaitGroup and returns a done func to call when the worker exits.
func (s *Server) RegisterWorker() func() {
	s.workers.Add(1)
	once := sync.Once{}
	return func() { once.Do(func() { s.workers.Done() }) }
}

// audit logs a JSON line (best-effort; swallow errors)
func (s *Server) audit(event string, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	meta["event"] = event
	meta["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	// ensure requestId propagated if present in context
	if _, ok := meta["requestId"]; !ok {
		if rid, ok := meta["request_id"]; ok {
			meta["requestId"] = rid
		}
	}
	data, _ := json.Marshal(meta)
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	if s.auditFile != nil {
		_, _ = s.auditFile.Write(append(data, '\n'))
	}
}

func (s *Server) RegisterRoutes(router *mux.Router) {
	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Admin (log level change) endpoint
	v1.HandleFunc("/admin/loglevel", s.handleLogLevel).Methods("PATCH")

	// Sources
	v1.HandleFunc("/sources", s.handleSourcesList).Methods("GET")
	v1.HandleFunc("/sources", s.handleSourceCreate).Methods("POST")
	v1.HandleFunc("/sources/{id}", s.handleSourceUpdate).Methods("PUT")
	v1.HandleFunc("/sources/{id}", s.handleSourceDelete).Methods("DELETE")
	v1.HandleFunc("/sources/{id}/start", s.handleSourceStart).Methods("POST")
	v1.HandleFunc("/sources/{id}/stop", s.handleSourceStop).Methods("POST")
	// Akamai DataStream 2 specific endpoints
	v1.HandleFunc("/sources/{id}/akamai/streams", s.handleAkamaiStreamsList).Methods("GET")
	v1.HandleFunc("/sources/{id}/akamai/streams/{streamId}/activate", s.handleAkamaiStreamActivate).Methods("POST")
	v1.HandleFunc("/sources/{id}/akamai/streams/{streamId}/deactivate", s.handleAkamaiStreamDeactivate).Methods("POST")
	v1.HandleFunc("/sources/{id}/akamai/datasets/{dataset}/fields", s.handleAkamaiDatasetFields).Methods("GET")
	// Generic raw proxy (query: path, method)
	v1.HandleFunc("/sources/{id}/akamai/raw", s.handleAkamaiRaw).Methods("GET", "POST", "PUT", "DELETE")
	// Source streaming and capture
	v1.HandleFunc("/sources/{id}/stream", s.handleSourceStream).Methods("GET")
	v1.HandleFunc("/sources/{id}/capture/start", s.handleCaptureStart).Methods("POST")
	v1.HandleFunc("/sources/{id}/capture/stop/{capId}", s.handleCaptureStop).Methods("POST")

	// Destinations
	v1.HandleFunc("/destinations", s.handleDestinationsList).Methods("GET")
	v1.HandleFunc("/destinations", s.handleDestinationCreate).Methods("POST")
	v1.HandleFunc("/destinations/{id}", s.handleDestinationUpdate).Methods("PUT")
	v1.HandleFunc("/destinations/{id}", s.handleDestinationDelete).Methods("DELETE")
	v1.HandleFunc("/destinations/{id}", s.handleDestinationPatch).Methods("PATCH")

	// Pipelines
	v1.HandleFunc("/pipelines", s.handlePipelinesList).Methods("GET")
	v1.HandleFunc("/pipelines/stats", s.handlePipelineStats).Methods("GET")
	v1.HandleFunc("/pipelines", s.handlePipelineCreate).Methods("POST")
	v1.HandleFunc("/pipelines/{id}", s.handlePipelineUpdate).Methods("PUT")
	v1.HandleFunc("/pipelines/{id}", s.handlePipelineDelete).Methods("DELETE")

	// Routes
	// Buffers
	v1.HandleFunc("/buffers", s.handleBuffersList).Methods("GET")
	v1.HandleFunc("/buffers/{sourceId}/reset", s.handleBufferReset).Methods("POST")
	v1.HandleFunc("/buffers/{sourceId}", s.handleBufferGet).Methods("GET")
	v1.HandleFunc("/buffers/{sourceId}", s.handleBufferUpdate).Methods("PATCH")
	v1.HandleFunc("/routes", s.handleRoutesList).Methods("GET")
	v1.HandleFunc("/routes", s.handleRouteCreate).Methods("POST")
	v1.HandleFunc("/routes/{id}", s.handleRouteUpdate).Methods("PUT")
	v1.HandleFunc("/routes/{id}", s.handleRouteDelete).Methods("DELETE")

	// Tools & Preview
	v1.HandleFunc("/preview/regex", s.handleRegexPreview).Methods("POST")
	v1.HandleFunc("/preview/enrich", s.handleEnrichPreview).Methods("POST")

	// Load test
	v1.HandleFunc("/loadtest/start", s.handleLoadTestStart).Methods("POST")
	v1.HandleFunc("/loadtest/stop", s.handleLoadTestStop).Methods("POST")
	v1.HandleFunc("/loadtest/status", s.handleLoadTestStatus).Methods("GET")

	// Enrichment assets
	v1.HandleFunc("/enrich/geoip/status", s.handleGeoIPStatus).Methods("GET")
	v1.HandleFunc("/enrich/geoip/upload", s.handleGeoIPUpload).Methods("POST")
	v1.HandleFunc("/enrich/asn/status", s.handleASNStatus).Methods("GET")
	v1.HandleFunc("/enrich/asn/upload", s.handleASNUpload).Methods("POST")
	// Library for preview/testing
	v1.HandleFunc("/library", s.handleLibraryList).Methods("GET")
	v1.HandleFunc("/library/{name}", s.handleLibraryRead).Methods("GET")

	// Syslog TLS certificates export (for Versa SD-WAN integration)
	v1.HandleFunc("/syslog/certs", s.handleSyslogCertsList).Methods("GET")
	v1.HandleFunc("/syslog/certs/download", s.handleSyslogCertDownload).Methods("GET")
	v1.HandleFunc("/syslog/certs/bundle", s.handleSyslogCertBundle).Methods("GET")

	// Azure helpers: auth + provisioning (stubs for now)
	azure := v1.PathPrefix("/azure").Subrouter()
	azure.HandleFunc("/login/start", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TenantID string `json:"tenantId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		code, url, msg, err := s.azureAuth.StartDeviceLogin(r.Context(), body.TenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"userCode": code, "verificationUrl": url, "message": msg})
	}).Methods("POST")
	azure.HandleFunc("/login/status", func(w http.ResponseWriter, r *http.Request) {
		authing, authed, msg, code, url := s.azureAuth.Status()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticating": authing, "authenticated": authed, "message": msg, "userCode": code, "verificationUrl": url})
	}).Methods("GET")
	azure.HandleFunc("/provision/sentinel", func(w http.ResponseWriter, r *http.Request) {
		if !s.azureAuth.IsAuthenticated() {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending", "message": "Sentinel provisioning stub"})
	}).Methods("POST")
	azure.HandleFunc("/provision/datalake", func(w http.ResponseWriter, r *http.Request) {
		if !s.azureAuth.IsAuthenticated() {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending", "message": "Data Lake provisioning stub"})
	}).Methods("POST")
}

// structuredError writes a standardized error JSON
func structuredError(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	requestId := r.Header.Get("X-Request-Id")
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg, "code": code, "requestId": requestId})
}

// handleLogLevel adjusts global log level (RBAC placeholder: requires auth token present)
func (s *Server) handleLogLevel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Level string `json:"level"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch body.Level {
	case "debug", "info", "warn", "error":
		logger.SetLevel(body.Level)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"level": body.Level})
		s.audit("loglevel_change", map[string]any{"level": body.Level})
	default:
		structuredError(w, r, http.StatusBadRequest, "invalid_level", "level must be debug|info|warn|error")
	}
}

// handleSourceStream streams recent logs then live logs via SSE.
func (s *Server) handleSourceStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	// allow optional ?tail=N
	tail := 200
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 && n <= 2000 {
			tail = n
		}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Disable proxy buffering where applicable
	w.Header().Set("X-Accel-Buffering", "no")
	// Send an initial comment to open the stream for some proxies
	_, _ = w.Write([]byte(": connected\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// send tail first
	for _, line := range s.hub.Tail(id, tail) {
		_, _ = w.Write([]byte("data: " + escapeSSE(line) + "\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	ch, cancel := s.hub.Subscribe(id)
	defer cancel()
	ctx := r.Context()
	// Periodic ping to keep the connection alive
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case line := <-ch:
			_, _ = w.Write([]byte("data: " + escapeSSE(line) + "\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func escapeSSE(s string) string {
	// Basic escaping: replace newlines to keep event framing
	return strings.ReplaceAll(s, "\n", "\\n")
}

// Capture start/stop
func (s *Server) handleCaptureStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var body struct {
		Format string `json:"format"`
		Name   string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Format == "" {
		body.Format = "log"
	}
	capId, path, err := s.hub.StartCapture(id, body.Format, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"captureId": capId, "path": path})
}

func (s *Server) handleCaptureStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	capId := vars["capId"]
	if err := s.hub.StopCapture(capId); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Library handlers
func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	items, err := s.hub.ListLibrary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Server) handleLibraryRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	data, err := s.hub.ReadLibraryFile(name, 5*1024*1024)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

// Akamai DataStream 2 handlers (operate only on akamai_ds2 sources)
func (s *Server) akamaiSourceConfig(id string) (map[string]interface{}, bool) {
	for _, src := range s.pipeline.GetSources() {
		if src.ID == id && src.Type == "akamai_ds2" {
			return src.Config, true
		}
	}
	return nil, false
}

func (s *Server) handleAkamaiStreamsList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	cfg, ok := s.akamaiSourceConfig(id)
	if !ok {
		http.Error(w, "source not found or not akamai_ds2", http.StatusNotFound)
		return
	}
	host, _ := cfg["host"].(string)
	clientToken, _ := cfg["clientToken"].(string)
	clientSecret, _ := cfg["clientSecret"].(string)
	accessToken, _ := cfg["accessToken"].(string)
	if host == "" || clientToken == "" || clientSecret == "" || accessToken == "" {
		http.Error(w, "missing credentials", http.StatusBadRequest)
		return
	}
	creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
	cli := akamaiinput.NewClient(creds)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	streams, err := cli.ListStreams(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Streams interface{} `json:"streams"`
	}{Streams: streams})
}

func (s *Server) handleAkamaiStreamActivate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	sid := vars["streamId"]
	cfg, ok := s.akamaiSourceConfig(id)
	if !ok {
		http.Error(w, "source not found or not akamai_ds2", http.StatusNotFound)
		return
	}
	var streamID int
	fmt.Sscanf(sid, "%d", &streamID)
	if streamID <= 0 {
		http.Error(w, "invalid streamId", http.StatusBadRequest)
		return
	}
	host, _ := cfg["host"].(string)
	clientToken, _ := cfg["clientToken"].(string)
	clientSecret, _ := cfg["clientSecret"].(string)
	accessToken, _ := cfg["accessToken"].(string)
	creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
	cli := akamaiinput.NewClient(creds)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := cli.ActivateStream(ctx, streamID); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAkamaiStreamDeactivate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	sid := vars["streamId"]
	cfg, ok := s.akamaiSourceConfig(id)
	if !ok {
		http.Error(w, "source not found or not akamai_ds2", http.StatusNotFound)
		return
	}
	var streamID int
	fmt.Sscanf(sid, "%d", &streamID)
	if streamID <= 0 {
		http.Error(w, "invalid streamId", http.StatusBadRequest)
		return
	}
	host, _ := cfg["host"].(string)
	clientToken, _ := cfg["clientToken"].(string)
	clientSecret, _ := cfg["clientSecret"].(string)
	accessToken, _ := cfg["accessToken"].(string)
	creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
	cli := akamaiinput.NewClient(creds)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := cli.DeactivateStream(ctx, streamID); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Dataset field listing (exposes Akamai dataset field definitions)
func (s *Server) handleAkamaiDatasetFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	dataset := vars["dataset"]
	cfg, ok := s.akamaiSourceConfig(id)
	if !ok {
		http.Error(w, "source not found or not akamai_ds2", http.StatusNotFound)
		return
	}
	host, _ := cfg["host"].(string)
	clientToken, _ := cfg["clientToken"].(string)
	clientSecret, _ := cfg["clientSecret"].(string)
	accessToken, _ := cfg["accessToken"].(string)
	if host == "" || clientToken == "" || clientSecret == "" || accessToken == "" {
		http.Error(w, "missing credentials", http.StatusBadRequest)
		return
	}
	creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
	cli := akamaiinput.NewClient(creds)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	fields, err := cli.GetDatasetFields(ctx, dataset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"dataset": dataset, "fields": fields})
}

// Generic raw Akamai request proxy for Workbench (restricted to datastream-config paths)
func (s *Server) handleAkamaiRaw(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	cfg, ok := s.akamaiSourceConfig(id)
	if !ok {
		http.Error(w, "source not found or not akamai_ds2", http.StatusNotFound)
		return
	}
	host, _ := cfg["host"].(string)
	clientToken, _ := cfg["clientToken"].(string)
	clientSecret, _ := cfg["clientSecret"].(string)
	accessToken, _ := cfg["accessToken"].(string)
	if host == "" || clientToken == "" || clientSecret == "" || accessToken == "" {
		http.Error(w, "missing credentials", http.StatusBadRequest)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(path, "/datastream-") {
		http.Error(w, "path must start with /datastream-", http.StatusBadRequest)
		return
	}
	method := r.URL.Query().Get("method")
	if method == "" {
		method = r.Method
	}
	body := io.Reader(nil)
	if r.Body != nil {
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		if len(b) > 0 {
			body = strings.NewReader(string(b))
		}
	}
	creds := akamaiinput.Credentials{Host: host, ClientToken: clientToken, ClientSecret: clientSecret, AccessToken: accessToken}
	cli := akamaiinput.NewClient(creds)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	resp, err := cli.DoProxy(ctx, method, path, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		if len(v) > 0 {
			w.Header().Set(k, v[0])
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
