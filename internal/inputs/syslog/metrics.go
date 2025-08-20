package syslog

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
    messagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Namespace: "bibbl",
        Subsystem: "syslog",
        Name:      "messages_total",
        Help:      "Total syslog messages received.",
    }, []string{"listener"})
)

func init() {
    prometheus.MustRegister(messagesTotal)
}

type LoggingHandler struct{ listener string }

func NewLoggingHandler(listener string) *LoggingHandler { return &LoggingHandler{listener: listener} }

func (h *LoggingHandler) Handle(message string) {
    messagesTotal.WithLabelValues(h.listener).Inc()
    // Keep PoC simple: log a trimmed preview
    if len(message) > 512 {
        message = message[:512]
    }
    log.Printf("syslog: %s", message)
}
