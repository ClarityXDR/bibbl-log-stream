package api

import (
	"net"
	"testing"
)

func TestExtractNamed(t *testing.T) {
    caps, err := extractNamed("1.2.3.4 hello", `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if caps["ip"] != "1.2.3.4" { t.Fatalf("expected ip capture, got %#v", caps) }
}

func TestExtractNamedNoMatch(t *testing.T) {
    caps, err := extractNamed("no ip here", `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(caps) != 0 { t.Fatalf("expected empty map, got %#v", caps) }
}

func TestIsPrivateIP(t *testing.T) {
    priv := []string{"10.1.2.3", "192.168.0.1", "172.16.5.9", "169.254.10.5"}
    for _, ip := range priv {
        if !isPrivateIP(net.ParseIP(ip)) { t.Fatalf("expected private: %s", ip) }
    }
    pub := []string{"8.8.8.8", "1.1.1.1"}
    for _, ip := range pub {
        if isPrivateIP(net.ParseIP(ip)) { t.Fatalf("expected public: %s", ip) }
    }
}

func TestASNLookupNil(t *testing.T) {
    // calling with nil db should not panic and return zero result
    res, err := asnLookup(nil, net.ParseIP("1.1.1.1"))
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if res.Number != 0 || res.Org != "" { t.Fatalf("expected zero result for nil db: %#v", res) }
}

func TestGeoLookupNil(t *testing.T) {
    res, err := geoipLookup(nil, net.ParseIP("1.1.1.1"))
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if res.City != "" || res.Country != "" { t.Fatalf("expected empty geo result: %#v", res) }
}
