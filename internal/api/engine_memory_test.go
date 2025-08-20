package api

import (
	"strings"
	"testing"
)

// fake geo/ASN lookup for testing
func fakeLookup(ip string) (map[string]interface{}, bool) {
    return map[string]interface{}{"city": "X", "ip": ip}, true
}

func TestProcessAndAppend_IPSourceField(t *testing.T) {
    eng := NewMemoryEngine().(*memoryEngine)
    eng.geo = fakeLookup
    eng.asn = fakeLookup
    eng.hub, _ = NewLogHub("")

    // create pipeline using field extraction
    eng.pipelines = []memPipe{{ID: "p1", Name: "FieldPipe", IPSource: "field:client_ip", Functions: []string{"geoip_enrich"}}}
    eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "client_ip", PipelineID: "p1", Destination: "d1", Final: true}}

    msg := "level=info client_ip=203.0.113.9 action=login"
    eng.processAndAppend("src1", msg)
    tail := eng.hub.Tail("src1", 1)
    if len(tail) != 1 { t.Fatalf("expected 1 message, got %d", len(tail)) }
    if want := "203.0.113.9"; !strings.Contains(tail[0], want) || !strings.Contains(tail[0], "geo") {
        t.Fatalf("expected enriched JSON with ip %s: %s", want, tail[0])
    }
}

func TestProcessAndAppend_IPSourceFallback(t *testing.T) {
    eng := NewMemoryEngine().(*memoryEngine)
    eng.geo = fakeLookup
    eng.hub, _ = NewLogHub("")
    eng.pipelines = []memPipe{{ID: "p1", Name: "DefaultPipe", IPSource: "field:missing", Functions: []string{"geoip_enrich"}}}
    eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "user", PipelineID: "p1", Destination: "d1", Final: true}}
    msg := "user=alice src=deny 198.51.100.7 proto=tcp"
    eng.processAndAppend("s", msg)
    tail := eng.hub.Tail("s", 1)
    if len(tail) != 1 { t.Fatalf("expected 1, got %d", len(tail)) }
    if !strings.Contains(tail[0], "198.51.100.7") { t.Fatalf("expected fallback ip: %s", tail[0]) }
}

func TestFilterCache(t *testing.T) {
    eng := NewMemoryEngine().(*memoryEngine)
    eng.hub, _ = NewLogHub("")
    eng.pipelines = []memPipe{{ID: "p1", Name: "P", IPSource: "first_ipv4"}}
    eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "alpha", PipelineID: "p1", Final: true}}
    // first call compiles
    if re := eng.getFilterRegex("alpha"); re == nil { t.Fatal("expected compiled regex") }
    if _, ok := eng.filterCache["alpha"]; !ok { t.Fatalf("expected cache entry") }
    // second call hits cache
    if re := eng.getFilterRegex("alpha"); re == nil { t.Fatal("expected cached regex") }
}
