package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	geoip2 "github.com/oschwald/geoip2-golang"
)

// GeoIP status response
type geoIPStatus struct {
    Loaded bool   `json:"loaded"`
    Path   string `json:"path,omitempty"`
    Size   int64  `json:"size,omitempty"`
    Mtime  int64  `json:"mtime,omitempty"`
}

func (s *Server) handleGeoIPStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    s.geoMu.RLock()
    path := s.geoIPPath
    loaded := s.geoIP != nil
    s.geoMu.RUnlock()
    st := geoIPStatus{Loaded: loaded}
    if path != "" {
        if fi, err := os.Stat(path); err == nil {
            st.Path = path
            st.Size = fi.Size()
            st.Mtime = fi.ModTime().Unix()
        }
    }
    _ = json.NewEncoder(w).Encode(st)
}

// POST multipart/form-data with field name "file" to upload a .mmdb database
func (s *Server) handleGeoIPUpload(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 128<<20) // 128MB hard cap
    if err := r.ParseMultipartForm(64 << 20); err != nil { // 64MB form limit
        http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
        return
    }
    f, hdr, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer f.Close()
    if hdr.Size <= 0 {
        http.Error(w, "empty file", http.StatusBadRequest)
        return
    }
    if err := os.MkdirAll("./data", 0o755); err != nil {
        http.Error(w, "failed to create data dir: "+err.Error(), http.StatusInternalServerError)
        return
    }
    dstPath := filepath.Join("./data", "GeoLite2-City.mmdb")
    tmpPath := dstPath + ".tmp"
    out, err := os.Create(tmpPath)
    if err != nil {
        http.Error(w, "failed to create temp file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    if _, err := io.Copy(out, f); err != nil {
        out.Close()
        _ = os.Remove(tmpPath)
        http.Error(w, "failed to write file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    _ = out.Close()
    if err := os.Rename(tmpPath, dstPath); err != nil {
        http.Error(w, "failed to finalize file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    // Open database and swap
    reader, err := geoipOpen(dstPath)
    if err != nil {
        http.Error(w, "failed to open geoip db: "+err.Error(), http.StatusBadRequest)
        return
    }
    s.geoMu.Lock()
    if old, ok := s.geoIP.(*geoip2.Reader); ok && old != nil { _ = old.Close() }
    s.geoIP = reader
    s.geoIPPath = dstPath
    s.geoMu.Unlock()

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(map[string]any{"status":"ok","path":dstPath})
}

// Request body can include either an explicit IP, or sample+pattern with named group "ip"
type enrichPreviewReq struct {
    Sample  string `json:"sample"`
    Pattern string `json:"pattern"`
    IP      string `json:"ip"`
}

func (s *Server) handleEnrichPreview(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    s.geoMu.RLock()
    reader := s.geoIP
    s.geoMu.RUnlock()
    if reader == nil {
        http.Error(w, `{"error":"geoip not loaded"}`, http.StatusPreconditionFailed)
        return
    }
    var req enrichPreviewReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
        return
    }
    ipStr := req.IP
    if ipStr == "" && req.Sample != "" && req.Pattern != "" {
        // Reuse existing regex preview to extract named group
        // Minimal inline extraction to avoid tight coupling
        captures, err := extractNamed(req.Sample, req.Pattern)
        if err == nil {
            if v, ok := captures["ip"]; ok { ipStr = v }
        }
    }
    ip := net.ParseIP(ipStr)
    if ip == nil {
        _ = json.NewEncoder(w).Encode(map[string]any{"enriched": false, "reason": "no ip"})
        return
    }
    res, err := geoipLookup(reader, ip)
    if err != nil {
        _ = json.NewEncoder(w).Encode(map[string]any{"enriched": false, "reason": err.Error()})
        return
    }
    _ = json.NewEncoder(w).Encode(map[string]any{"enriched": true, "ip": ipStr, "geo": res})
}

// Helpers are split for easy unit tests
func extractNamed(sample, pattern string) (map[string]string, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, err
    }
    m := re.FindStringSubmatch(sample)
    if m == nil { return map[string]string{}, nil }
    names := re.SubexpNames()
    out := map[string]string{}
    for i, v := range m {
        if i == 0 { continue }
        key := ""
        if i < len(names) { key = names[i] }
        if key == "" { key = string(rune('0'+i)) }
        out[key] = v
    }
    return out, nil
}

// geoip helpers
func geoipOpen(path string) (*geoip2.Reader, error) { return geoip2.Open(path) }

type geoResult struct {
    City       string  `json:"city,omitempty"`
    Country    string  `json:"country,omitempty"`
    CountryISO string  `json:"countryIso,omitempty"`
    Subdiv     string  `json:"subdivision,omitempty"`
    Lat        float64 `json:"lat,omitempty"`
    Lon        float64 `json:"lon,omitempty"`
    Timezone   string  `json:"timezone,omitempty"`
    Private    bool    `json:"private,omitempty"`
    IPv6       bool    `json:"ipv6,omitempty"`
}

func geoipLookup(reader any, ip net.IP) (geoResult, error) {
    rr, ok := reader.(*geoip2.Reader)
    if !ok || rr == nil { return geoResult{}, nil }
    rec, err := rr.City(ip)
    if err != nil { return geoResult{}, err }
    var sub string
    if len(rec.Subdivisions) > 0 {
        if name := rec.Subdivisions[0].Names["en"]; name != "" { sub = name } else { sub = rec.Subdivisions[0].IsoCode }
    }
    return geoResult{
        City:       rec.City.Names["en"],
        Country:    rec.Country.Names["en"],
        CountryISO: rec.Country.IsoCode,
        Subdiv:     sub,
        Lat:        rec.Location.Latitude,
        Lon:        rec.Location.Longitude,
        Timezone:   rec.Location.TimeZone,
        Private:    isPrivateIP(ip),
        IPv6:       ip.To4() == nil,
    }, nil
}

func isPrivateIP(ip net.IP) bool {
    if ip == nil { return false }
    if ip4 := ip.To4(); ip4 != nil {
        // 10.0.0.0/8
        if ip4[0] == 10 { return true }
        // 172.16.0.0/12
        if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 { return true }
        // 192.168.0.0/16
        if ip4[0] == 192 && ip4[1] == 168 { return true }
        // 169.254.0.0/16 link-local
        if ip4[0] == 169 && ip4[1] == 254 { return true }
    } else {
        // fc00::/7 unique local, fe80::/10 link-local
        if len(ip) >= 1 && (ip[0]&0xfe) == 0xfc { return true }
        if len(ip) >= 2 && ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 { return true }
    }
    return false
}

// ASN support (optional)
type asnResult struct { Number uint `json:"asn"`; Org string `json:"org"` }

func (s *Server) handleASNStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    s.geoMu.RLock()
    path := s.asnPath
    loaded := s.asnDB != nil
    s.geoMu.RUnlock()
    var size int64
    var mtime int64
    if path != "" { if fi, err := os.Stat(path); err == nil { size, mtime = fi.Size(), fi.ModTime().Unix() } }
    _ = json.NewEncoder(w).Encode(map[string]any{"loaded": loaded, "path": path, "size": size, "mtime": mtime})
}

func (s *Server) handleASNUpload(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
    if err := r.ParseMultipartForm(32 << 20); err != nil { http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest); return }
    f, hdr, err := r.FormFile("file"); if err != nil { http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest); return }
    defer f.Close()
    if hdr.Size <= 0 { http.Error(w, "empty file", http.StatusBadRequest); return }
    if err := os.MkdirAll("./data", 0o755); err != nil { http.Error(w, "failed to create data dir: "+err.Error(), http.StatusInternalServerError); return }
    dstPath := filepath.Join("./data", "GeoLite2-ASN.mmdb")
    tmpPath := dstPath+".tmp"
    out, err := os.Create(tmpPath); if err != nil { http.Error(w, "failed to create temp file: "+err.Error(), http.StatusInternalServerError); return }
    if _, err := io.Copy(out, f); err != nil { out.Close(); _ = os.Remove(tmpPath); http.Error(w, "failed to write file: "+err.Error(), http.StatusInternalServerError); return }
    _ = out.Close()
    if err := os.Rename(tmpPath, dstPath); err != nil { http.Error(w, "failed to finalize file: "+err.Error(), http.StatusInternalServerError); return }
    db, err := geoip2.Open(dstPath); if err != nil { http.Error(w, "failed to open asn db: "+err.Error(), http.StatusBadRequest); return }
    s.geoMu.Lock()
    if old, ok := s.asnDB.(*geoip2.Reader); ok && old != nil { _ = old.Close() }
    s.asnDB = db; s.asnPath = dstPath
    s.geoMu.Unlock()
    w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(map[string]any{"status":"ok","path":dstPath})
}

func asnLookup(db any, ip net.IP) (asnResult, error) {
    rr, ok := db.(*geoip2.Reader); if !ok || rr == nil { return asnResult{}, nil }
    rec, err := rr.ASN(ip); if err != nil { return asnResult{}, err }
    return asnResult{ Number: uint(rec.AutonomousSystemNumber), Org: rec.AutonomousSystemOrganization }, nil
}
