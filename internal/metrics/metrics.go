package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	// HTTP Metrics
	HTTPLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "bibbl",
		Subsystem: "http",
		Name:      "request_seconds",
		Help:      "HTTP request latency.",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"method", "path", "status"})

	HTTPInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "http",
		Name:      "inflight",
		Help:      "In-flight HTTP requests.",
	})

	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total HTTP requests.",
	}, []string{"method", "path", "status"})

	HTTPRequestSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "bibbl",
		Subsystem: "http",
		Name:      "request_size_bytes",
		Help:      "HTTP request size in bytes.",
		Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})

	HTTPResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "bibbl",
		Subsystem: "http",
		Name:      "response_size_bytes",
		Help:      "HTTP response size in bytes.",
		Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})

	// Buffer Metrics
	BufferSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "buffer",
		Name:      "size",
		Help:      "Current buffer size per source.",
	}, []string{"source"})

	BufferDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "buffer",
		Name:      "dropped_total",
		Help:      "Total dropped messages per source.",
	}, []string{"source", "reason"})

	BufferDroppedCurrent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "buffer",
		Name:      "dropped_current",
		Help:      "Current dropped message count per source.",
	}, []string{"source"})

	BufferCapacity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "buffer",
		Name:      "capacity",
		Help:      "Buffer capacity per source.",
	}, []string{"source"})

	// Pipeline Metrics
	PipelineLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "bibbl",
		Subsystem: "pipeline",
		Name:      "process_seconds",
		Help:      "Pipeline processing latency from receipt to hub append.",
		Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"pipeline", "route", "source"})

	PipelineThroughput = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "pipeline",
		Name:      "events_processed_total",
		Help:      "Total events processed by pipeline.",
	}, []string{"pipeline", "route", "source", "status"})

	PipelineFiltered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "pipeline",
		Name:      "filtered_events_total",
		Help:      "Total events dropped by pipeline filters.",
	}, []string{"pipeline", "route", "source"})

	PipelineErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "pipeline",
		Name:      "errors_total",
		Help:      "Total pipeline processing errors.",
	}, []string{"pipeline", "route", "source", "error_type"})

	// Ingestion Metrics
	IngestEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "ingest",
		Name:      "events_total",
		Help:      "Events ingested (post routing) per source/route/destination.",
	}, []string{"source", "route", "destination"})

	IngestBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "ingest",
		Name:      "bytes_total",
		Help:      "Bytes ingested per source/route/destination.",
	}, []string{"source", "route", "destination"})

	IngestLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "bibbl",
		Subsystem: "ingest",
		Name:      "latency_seconds",
		Help:      "Ingestion latency to external systems.",
		Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"destination"})

	// System Metrics
	SystemInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "system",
		Name:      "info",
		Help:      "System information.",
	}, []string{"version", "commit", "build_date", "go_version"})

	SystemUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "system",
		Name:      "uptime_seconds",
		Help:      "System uptime in seconds.",
	})

	ConfigReloads = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "system",
		Name:      "config_reloads_total",
		Help:      "Total configuration reloads.",
	})

	// Authentication Metrics
	AuthAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "auth",
		Name:      "attempts_total",
		Help:      "Authentication attempts.",
	}, []string{"provider", "status"})

	AuthSessions = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "auth",
		Name:      "active_sessions",
		Help:      "Number of active authenticated sessions.",
	})

	// Cost Metrics (for Azure monitoring)
	AzureCost = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "bibbl",
		Subsystem: "azure",
		Name:      "estimated_cost_usd",
		Help:      "Estimated Azure costs.",
	}, []string{"service", "resource_group"})

	AzureIngestionEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bibbl",
		Subsystem: "azure",
		Name:      "ingestion_events_total",
		Help:      "Events sent to Azure services.",
	}, []string{"service", "table", "status"})
)

var (
	registry  *prometheus.Registry
	regOnce   sync.Once
	startTime time.Time
)

// Init initializes the metrics registry with safe registration
func Init() {
	regOnce.Do(func() {
		startTime = time.Now()

		// Create a new registry to avoid conflicts
		registry = prometheus.NewRegistry()

		// Add Go runtime metrics
		registry.MustRegister(collectors.NewGoCollector())
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

		// Register all custom metrics
		registry.MustRegister(
			HTTPLatency, HTTPInFlight, HTTPRequests, HTTPRequestSize, HTTPResponseSize,
			BufferSize, BufferDropped, BufferDroppedCurrent, BufferCapacity,
			PipelineLatency, PipelineThroughput, PipelineFiltered, PipelineErrors,
			IngestEvents, IngestBytes, IngestLatency,
			SystemInfo, SystemUptime, ConfigReloads,
			AuthAttempts, AuthSessions,
			AzureCost, AzureIngestionEvents,
		)

		// Set system uptime updater
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				SystemUptime.Set(time.Since(startTime).Seconds())
			}
		}()
	})
}

// Registry returns the custom Prometheus registry
func Registry() *prometheus.Registry {
	return registry
}

// RecordHTTPRequest records an HTTP request with all relevant metrics
func RecordHTTPRequest(method, path string, status int, duration time.Duration, requestSize, responseSize int64) {
	statusStr := strconv.Itoa(status)

	HTTPRequests.WithLabelValues(method, path, statusStr).Inc()
	HTTPLatency.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
	HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(provider, status string) {
	AuthAttempts.WithLabelValues(provider, status).Inc()
}

// UpdateActiveSessions updates the count of active sessions
func UpdateActiveSessions(count int) {
	AuthSessions.Set(float64(count))
}

// RecordPipelineEvent records pipeline processing metrics
func RecordPipelineEvent(pipeline, route, source, status string, duration time.Duration) {
	PipelineThroughput.WithLabelValues(pipeline, route, source, status).Inc()
	if status == "success" {
		PipelineLatency.WithLabelValues(pipeline, route, source).Observe(duration.Seconds())
	}
}

// RecordPipelineError records a pipeline error
func RecordPipelineError(pipeline, route, source, errorType string) {
	PipelineErrors.WithLabelValues(pipeline, route, source, errorType).Inc()
}
