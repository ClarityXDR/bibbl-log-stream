package api

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"
)

// benchEngine builds a memory engine wired with the current severity filter logic
// so benchmarks exercise the same code paths as production pipelines.
func benchEngine() *memoryEngine {
	hub, _ := NewLogHub("")
	pipeFns := []string{"filter:severity=critical|high|med"}
	filters, err := compileKVFilters(pipeFns)
	if err != nil {
		panic(err)
	}
	return &memoryEngine{
		sources:     []*memSource{},
		dests:       []memDest{},
		pipelines:   []memPipe{{ID: "default", Name: "default", Functions: pipeFns, Filters: filters}},
		routes:      []memRoute{{Name: "default", PipelineID: "default", Filter: "true"}},
		hub:         hub,
		filterCache: make(map[string]*regexp.Regexp),
	}
}

// BenchmarkProcessAndAppend measures single-event processing throughput (baseline).

func BenchmarkProcessAndAppend(b *testing.B) {
	eng := benchEngine()

	msg := `{"timestamp":"2025-11-15T14:20:00Z","level":"info","message":"test event","ip":"192.168.1.100","severity":"critical"}`
	sourceID := "bench-source"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		eng.processAndAppend(sourceID, msg)
	}

	b.StopTimer()
	eps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(eps, "events/sec")
}

// BenchmarkProcessAndAppendBatch measures batch processing throughput (optimized).
func BenchmarkProcessAndAppendBatch(b *testing.B) {
	eng := benchEngine()

	msg := `{"timestamp":"2025-11-15T14:20:00Z","level":"info","message":"test event","ip":"192.168.1.100","severity":"critical"}`
	sourceID := "bench-source"
	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		batch := make([]string, size)
		for i := 0; i < size; i++ {
			batch[i] = msg
		}

		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				eng.processAndAppendBatch(sourceID, batch)
			}

			b.StopTimer()
			totalEvents := b.N * size
			eps := float64(totalEvents) / b.Elapsed().Seconds()
			b.ReportMetric(eps, "events/sec")
		})
	}
}

// BenchmarkSyntheticLoad simulates full pipeline with synthetic source.
func BenchmarkSyntheticLoad(b *testing.B) {
	rates := []int{1000, 10000, 50000, 100000}

	for _, rate := range rates {
		b.Run(fmt.Sprintf("rate_%d", rate), func(b *testing.B) {
			eng := benchEngine()

			srcID := "synth-bench"
			var received uint64
			var mu sync.Mutex

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create synthetic source
			src := &memSource{
				ID:      srcID,
				Name:    "Synthetic Bench",
				Type:    "synthetic",
				Enabled: true,
				Config: map[string]interface{}{
					"rate":    rate,
					"size":    200,
					"workers": 4,
				},
			}
			eng.sources = append(eng.sources, src)

			b.ResetTimer()
			start := time.Now()

			// Start source (this triggers synthetic generation)
			if err := eng.StartSource(srcID); err != nil {
				b.Fatalf("Failed to start source: %v", err)
			}

			// Wait for test duration
			<-ctx.Done()

			// Stop source
			_ = eng.StopSource(srcID)

			elapsed := time.Since(start)
			b.StopTimer()

			mu.Lock()
			received = src.produced.Load()
			mu.Unlock()

			actualEPS := float64(received) / elapsed.Seconds()
			b.ReportMetric(actualEPS, "events/sec")
			b.ReportMetric(float64(received), "total_events")
		})
	}
}

// BenchmarkLockFreeRing measures ring buffer performance.
func BenchmarkLockFreeRing(b *testing.B) {
	hub, _ := NewLogHub("")
	sourceID := "ring-bench"
	msg := "benchmark message payload with some reasonable length to simulate real log events"

	b.Run("sequential", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hub.Append(sourceID, msg)
		}
		b.StopTimer()
		ops := float64(b.N) / b.Elapsed().Seconds()
		b.ReportMetric(ops, "ops/sec")
	})

	b.Run("concurrent_4", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var wg sync.WaitGroup
		workers := 4
		perWorker := b.N / workers

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < perWorker; i++ {
					hub.Append(sourceID, msg)
				}
			}()
		}
		wg.Wait()

		b.StopTimer()
		ops := float64(b.N) / b.Elapsed().Seconds()
		b.ReportMetric(ops, "ops/sec")
	})
}
