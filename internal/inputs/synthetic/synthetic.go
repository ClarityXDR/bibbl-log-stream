package synthetic

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

// Generator produces synthetic events at a target rate for load testing.
type Generator struct {
    Rate      int
    Size      int
    Template  string
    Workers   int
    JitterPct int
    Compress  bool // if true, gzip-compress the rendered event then base64 encode it
    running   atomic.Bool
    produced  atomic.Uint64
    dropped   atomic.Uint64
    stopFn    context.CancelFunc
    callback  func(string)
}

func New(callback func(string)) *Generator { return &Generator{Template: "synthetic event ${seq}", Size: 200, Workers: 1, callback: callback} }

func (g *Generator) Start(cfg map[string]interface{}) {
    if g.running.Load() { g.Stop() }
    if v, ok := getInt(cfg, "rate"); ok { g.Rate = v }
    if v, ok := getInt(cfg, "size"); ok { g.Size = v }
    if v, ok := getInt(cfg, "workers"); ok && v > 0 { g.Workers = v }
    if v, ok := getInt(cfg, "jitterPct"); ok { g.JitterPct = v }
    if v, ok := cfg["template"].(string); ok && v != "" { g.Template = v }
    if v, ok := cfg["compress"].(bool); ok { g.Compress = v }
    ctx, cancel := context.WithCancel(context.Background())
    g.stopFn = cancel
    g.running.Store(true)
    per := g.Rate
    if g.Workers > 0 { per = g.Rate / g.Workers }
    if per < 1 { per = g.Rate }
    for w := 0; w < g.Workers; w++ { go g.worker(ctx, w, per) }
}

func (g *Generator) worker(ctx context.Context, wid int, rate int) {
    if rate <= 0 { return }
    seqBase := uint64(wid) << 32
    start := time.Now()
    produced := 0
    target := withJitter(rate, g.JitterPct)
    for i := 0; ; i++ {
        select { case <-ctx.Done(): return; default: }
        if produced >= target {
            now := time.Now()
            if sleep := start.Add(time.Second).Sub(now); sleep > 0 { time.Sleep(sleep) }
            start = time.Now()
            produced = 0
            target = withJitter(rate, g.JitterPct)
            continue
        }
        seq := seqBase + uint64(i)
    begin := time.Now()
    msg := g.render(seq)
    if g.Compress { msg = compressAndB64(msg) }
    if g.callback != nil { g.callback(msg); g.produced.Add(1); synthMessages.WithLabelValues("produced").Inc(); synthBytes.Add(float64(len(msg))); synthGenSeconds.Observe(time.Since(begin).Seconds()) } else { g.dropped.Add(1); synthMessages.WithLabelValues("dropped").Inc() }
        produced++
    }
}

func withJitter(rate, pct int) int {
    if pct <= 0 { return rate }
    j := rand.Intn(pct*2+1) - pct
    v := rate + (rate * j / 100)
    if v < 1 { v = 1 }
    return v
}

func (g *Generator) render(seq uint64) string {
    t := strings.ReplaceAll(g.Template, "${seq}", utoa(seq))
    if g.Size <= 0 || len(t) >= g.Size { return t }
    pad := g.Size - len(t)
    if pad > 4096 { pad = 4096 }
    return t + strings.Repeat("x", pad)
}

func (g *Generator) Stop() { if g.running.Load() { if g.stopFn != nil { g.stopFn(); g.stopFn = nil }; g.running.Store(false) } }

// Produced returns the total produced events since last Start.
func (g *Generator) Produced() uint64 { return g.produced.Load() }
// Dropped returns the total dropped events since last Start.
func (g *Generator) Dropped() uint64 { return g.dropped.Load() }

func getInt(m map[string]interface{}, k string) (int, bool) {
    if v, ok := m[k]; ok { switch vv := v.(type) { case int: return vv, true; case int64: return int(vv), true; case float64: return int(vv), true } }
    return 0, false
}

func utoa(v uint64) string { if v==0 { return "0" }; var b [20]byte; i:=len(b); for v>0 { i--; b[i]=byte('0'+v%10); v/=10 }; return string(b[i:]) }

func compressAndB64(s string) string {
    var buf bytes.Buffer
    zw := gzip.NewWriter(&buf)
    _, _ = zw.Write([]byte(s))
    _ = zw.Close()
    return base64.StdEncoding.EncodeToString(buf.Bytes())
}
