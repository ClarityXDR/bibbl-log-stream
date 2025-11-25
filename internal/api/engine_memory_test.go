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
	pipeFns := []string{"geoip_enrich"}
	eng.pipelines = []memPipe{{ID: "p1", Name: "FieldPipe", IPSource: "field:client_ip", Functions: pipeFns, Filters: mustCompileFilters(t, pipeFns)}}
	eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "client_ip", PipelineID: "p1", Destination: "d1", Final: true}}

	msg := "level=info client_ip=203.0.113.9 action=login"
	eng.processAndAppend("src1", msg)
	tail := eng.hub.Tail("src1", 1)
	if len(tail) != 1 {
		t.Fatalf("expected 1 message, got %d", len(tail))
	}
	if want := "203.0.113.9"; !strings.Contains(tail[0], want) || !strings.Contains(tail[0], "geo") {
		t.Fatalf("expected enriched JSON with ip %s: %s", want, tail[0])
	}
}

func TestProcessAndAppend_IPSourceFallback(t *testing.T) {
	eng := NewMemoryEngine().(*memoryEngine)
	eng.geo = fakeLookup
	eng.hub, _ = NewLogHub("")
	pipeFns := []string{"geoip_enrich"}
	eng.pipelines = []memPipe{{ID: "p1", Name: "DefaultPipe", IPSource: "field:missing", Functions: pipeFns, Filters: mustCompileFilters(t, pipeFns)}}
	eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "user", PipelineID: "p1", Destination: "d1", Final: true}}
	msg := "user=alice src=deny 198.51.100.7 proto=tcp"
	eng.processAndAppend("s", msg)
	tail := eng.hub.Tail("s", 1)
	if len(tail) != 1 {
		t.Fatalf("expected 1, got %d", len(tail))
	}
	if !strings.Contains(tail[0], "198.51.100.7") {
		t.Fatalf("expected fallback ip: %s", tail[0])
	}
}

func TestFilterCache(t *testing.T) {
	eng := NewMemoryEngine().(*memoryEngine)
	eng.hub, _ = NewLogHub("")
	eng.pipelines = []memPipe{{ID: "p1", Name: "P", IPSource: "first_ipv4", Filters: mustCompileFilters(t, nil)}}
	eng.routes = []memRoute{{ID: "r1", Name: "r1", Filter: "alpha", PipelineID: "p1", Final: true}}
	// first call compiles
	if re := eng.getFilterRegex("alpha"); re == nil {
		t.Fatal("expected compiled regex")
	}
	if _, ok := eng.filterCache["alpha"]; !ok {
		t.Fatalf("expected cache entry")
	}
	// second call hits cache
	if re := eng.getFilterRegex("alpha"); re == nil {
		t.Fatal("expected cached regex")
	}
}

func TestNewKVFilterInclude(t *testing.T) {
	f, ok := newKVFilter("filter:severity=critical|high")
	if !ok {
		t.Fatalf("expected filter parsed")
	}
	if f.fieldLower != "severity" {
		t.Fatalf("expected field severity, got %s", f.fieldLower)
	}
	payload := map[string]interface{}{"severity": "CRITICAL"}
	if !f.allows(payload) {
		t.Fatalf("expected critical severity to pass include filter")
	}
	payload["severity"] = "info"
	if f.allows(payload) {
		t.Fatalf("expected info severity to be filtered")
	}
}

func TestNewKVFilterExclude(t *testing.T) {
	f, ok := newKVFilter("filter:env!=dev|test")
	if !ok {
		t.Fatalf("expected filter parsed")
	}
	payload := map[string]interface{}{"env": "prod"}
	if !f.allows(payload) {
		t.Fatalf("prod should not be excluded")
	}
	payload["env"] = "dev"
	if f.allows(payload) {
		t.Fatalf("dev should be excluded by filter")
	}
}

func TestKVFilterRawFallback(t *testing.T) {
	f, ok := newKVFilter("filter:severity=high")
	if !ok {
		t.Fatalf("expected filter parsed")
	}
	payload := map[string]interface{}{"_raw": "ts=1 severity=high msg=ok"}
	if !f.allows(payload) {
		t.Fatalf("expected raw fallback to match")
	}
	payload["_raw"] = "ts=1 severity=low"
	if f.allows(payload) {
		t.Fatalf("expected low severity to be filtered")
	}
}

func mustCompileFilters(t *testing.T, fns []string) []kvFilter {
	t.Helper()
	filters, err := compileKVFilters(fns)
	if err != nil {
		t.Fatalf("compile filters: %v", err)
	}
	return filters
}
