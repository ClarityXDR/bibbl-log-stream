package tls

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsurePairExists generates a self-signed certificate and key at the given paths if either file is missing.
// It preserves existing files otherwise.
func EnsurePairExists(certPath, keyPath string, hosts []string, validFor time.Duration) (string, string, error) {
	cert, key, _, err := ensurePair(certPath, keyPath, hosts, validFor, "Bibbl Self-Signed", false, 0)
	return cert, key, err
}

// EnsurePairFresh generates or renews a self-signed certificate and key, ensuring SAN coverage for hosts
// and reissuing if the certificate will expire within renewBefore. Returns whether a new certificate was written.
func EnsurePairFresh(certPath, keyPath string, hosts []string, validFor, renewBefore time.Duration, commonName string) (string, string, bool, error) {
	return ensurePair(certPath, keyPath, hosts, validFor, commonName, true, renewBefore)
}

func ensurePair(certPath, keyPath string, hosts []string, validFor time.Duration, commonName string, allowRenew bool, renewBefore time.Duration) (string, string, bool, error) {
	if certPath == "" {
		certPath = "./certs/bibbl.crt"
	}
	if keyPath == "" {
		keyPath = "./certs/bibbl.key"
	}
	if validFor <= 0 {
		validFor = 365 * 24 * time.Hour
	}
	if renewBefore < 0 {
		renewBefore = 0
	}
	if commonName == "" {
		commonName = "Bibbl Self-Signed"
	}
	hosts = normalizeHosts(hosts)

	needReissue := false
	if !(fileExists(certPath) && fileExists(keyPath)) {
		needReissue = true
	} else if allowRenew {
		cert, err := readCertificate(certPath)
		if err != nil {
			needReissue = true
		} else if renewBefore > 0 && time.Until(cert.NotAfter) <= renewBefore {
			needReissue = true
		} else if len(hosts) > 0 && !certHasHosts(cert, hosts) {
			needReissue = true
		}
	}
	if !needReissue {
		return certPath, keyPath, false, nil
	}
	if err := generateSelfSigned(certPath, keyPath, hosts, validFor, commonName); err != nil {
		return "", "", false, err
	}
	return certPath, keyPath, true, nil
}

func generateSelfSigned(certPath, keyPath string, hosts []string, validFor time.Duration, commonName string) error {
	if validFor <= 0 {
		validFor = 365 * 24 * time.Hour
	}
	_ = os.MkdirAll(filepath.Dir(certPath), 0o755)
	_ = os.MkdirAll(filepath.Dir(keyPath), 0o700)
	priv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		return err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := crand.Int(crand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}
	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(validFor),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else if h != "" {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	derBytes, err := x509.CreateCertificate(crand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}), 0o600); err != nil {
		return err
	}
	return nil
}

func readCertificate(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(data)
	if blk == nil {
		return nil, errors.New("failed to decode PEM")
	}
	return x509.ParseCertificate(blk.Bytes)
}

func certHasHosts(cert *x509.Certificate, hosts []string) bool {
	if len(hosts) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, dns := range cert.DNSNames {
		set[strings.ToLower(dns)] = struct{}{}
	}
	for _, ip := range cert.IPAddresses {
		set[ip.String()] = struct{}{}
	}
	for _, h := range hosts {
		if h == "" {
			continue
		}
		key := h
		if ip := net.ParseIP(h); ip != nil {
			key = ip.String()
		} else {
			key = strings.ToLower(h)
		}
		if _, ok := set[key]; !ok {
			return false
		}
	}
	return true
}

func normalizeHosts(hosts []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		key := strings.ToLower(h)
		if ip := net.ParseIP(h); ip != nil {
			key = ip.String()
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, h)
	}
	return out
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
