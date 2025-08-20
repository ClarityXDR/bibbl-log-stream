package synthetic

import "github.com/prometheus/client_golang/prometheus"

var (
    synthMessages = prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "bibbl", Subsystem: "synthetic", Name: "messages_total", Help: "Synthetic generator messages (produced or dropped)."}, []string{"state"})
    synthBytes    = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "bibbl", Subsystem: "synthetic", Name: "bytes_total", Help: "Total bytes (post optional compression) generated."})
    synthGenSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{Namespace: "bibbl", Subsystem: "synthetic", Name: "generate_seconds", Help: "Time to render (and optionally compress) an event.", Buckets: prometheus.DefBuckets})
)

func init(){
    prometheus.MustRegister(synthMessages, synthBytes, synthGenSeconds)
}
