package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
    HTTPLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace:"bibbl", Subsystem:"http", Name:"request_seconds", Help:"HTTP request latency.", Buckets: prometheus.DefBuckets}, []string{"method","path","status"})
    HTTPInFlight = prometheus.NewGauge(prometheus.GaugeOpts{Namespace:"bibbl", Subsystem:"http", Name:"inflight", Help:"In-flight HTTP requests."})
    HTTPRequests = prometheus.NewCounter(prometheus.CounterOpts{Namespace:"bibbl", Subsystem:"http", Name:"requests_total", Help:"Total HTTP requests."})
    BufferSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace:"bibbl", Subsystem:"buffer", Name:"size", Help:"Current buffer size per source."}, []string{"source"})
    BufferDropped = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace:"bibbl", Subsystem:"buffer", Name:"dropped", Help:"Dropped messages per source."}, []string{"source"})
    PipelineLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace:"bibbl", Subsystem:"pipeline", Name:"process_seconds", Help:"Pipeline processing latency from receipt to hub append.", Buckets: prometheus.DefBuckets}, []string{"pipeline","route","source"})
    IngestEvents = prometheus.NewCounterVec(prometheus.CounterOpts{Namespace:"bibbl", Subsystem:"ingest", Name:"events_total", Help:"Events ingested (post routing) per source/route/destination (destination optional)."}, []string{"source","route","destination"})
)

var regOnce sync.Once
func Init() { regOnce.Do(func(){ prometheus.MustRegister(HTTPLatency, HTTPInFlight, HTTPRequests, BufferSize, BufferDropped, PipelineLatency, IngestEvents) }) }
