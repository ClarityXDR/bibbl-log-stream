package syslog

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
)

// Handler is a callback invoked per line/message.
type Handler interface {
    Handle(message string)
}

type Server struct {
    addr    string
    tlsConf *tls.Config
    handler Handler

    ln   net.Listener
    wg   sync.WaitGroup
    stop chan struct{}
    allowList []netip.Prefix // empty => allow all
}

func New(addr string, tlsConf *tls.Config, h Handler) *Server {
    return &Server{addr: addr, tlsConf: tlsConf, handler: h, stop: make(chan struct{})}
}

func (s *Server) Start(ctx context.Context) error {
    var err error
    if s.tlsConf != nil {
        s.ln, err = tls.Listen("tcp", s.addr, s.tlsConf)
    } else {
        s.ln, err = net.Listen("tcp", s.addr)
    }
    if err != nil {
        return err
    }
    log.Printf("syslog listener started on %s (TLS=%v)", s.addr, s.tlsConf != nil)

    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        for {
            conn, err := s.ln.Accept()
            if err != nil {
                select {
                case <-s.stop:
                    return
                default:
                }
                continue
            }
            s.wg.Add(1)
            go s.handleConn(conn)
        }
    }()

    // Bind to context cancellation
    go func() {
        <-ctx.Done()
        _ = s.Stop()
    }()
    return nil
}

func (s *Server) handleConn(c net.Conn) {
    defer s.wg.Done()
    defer c.Close()
    // Enforce allow-list if configured
    if len(s.allowList) > 0 {
        ra := c.RemoteAddr()
        var ipStr string
        if ta, ok := ra.(*net.TCPAddr); ok && ta.IP != nil {
            ipStr = ta.IP.String()
        } else {
            host, _, _ := net.SplitHostPort(ra.String())
            ipStr = host
        }
        if ipStr != "" {
            if ip, err := netip.ParseAddr(ipStr); err == nil {
                allowed := false
                for _, pfx := range s.allowList {
                    if pfx.Contains(ip) { allowed = true; break }
                }
                if !allowed {
                    log.Printf("syslog: drop connection from %s (not allowed)", ipStr)
                    return
                }
            }
        }
    }
    r := bufio.NewReader(c)
    for {
        line, err := r.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                return
            }
            return
        }
        msg := strings.TrimRight(line, "\r\n")
        if s.handler != nil && msg != "" {
            s.handler.Handle(msg)
        }
    }
}

func (s *Server) Stop() error {
    select {
    case <-s.stop:
        // already stopped
    default:
        close(s.stop)
    }
    if s.ln != nil {
        _ = s.ln.Close()
    }
    s.wg.Wait()
    return nil
}

// SetAllowList permits setting a list of allowed IP prefixes (CIDRs or single IPs converted to /32 or /128).
func (s *Server) SetAllowList(prefixes []string) {
    s.allowList = s.allowList[:0]
    for _, p := range prefixes {
        p = strings.TrimSpace(p)
        if p == "" { continue }
        // Try CIDR
        if pfx, err := netip.ParsePrefix(p); err == nil {
            s.allowList = append(s.allowList, pfx)
            continue
        }
        // Try single IP => convert to /32 or /128
        if ip, err := netip.ParseAddr(p); err == nil {
            bits := 32
            if ip.Is6() { bits = 128 }
            s.allowList = append(s.allowList, netip.PrefixFrom(ip, bits))
        }
    }
}
