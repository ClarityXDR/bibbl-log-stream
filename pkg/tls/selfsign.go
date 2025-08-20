package tls

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// EnsurePairExists generates a self-signed certificate and key at the given paths if either is missing.
// It returns the absolute paths actually written/resolved.
func EnsurePairExists(certPath, keyPath string, hosts []string, validFor time.Duration) (string, string, error) {
    // Default locations if not provided
    if certPath == "" {
        certPath = "./certs/bibbl.crt"
    }
    if keyPath == "" {
        keyPath = "./certs/bibbl.key"
    }

    // If both exist, return as-is
    if fileExists(certPath) && fileExists(keyPath) {
        return certPath, keyPath, nil
    }

    if validFor <= 0 {
        validFor = 365 * 24 * time.Hour
    }

    // Create parent directories for both cert and key
    _ = os.MkdirAll(filepath.Dir(certPath), 0o755)
    _ = os.MkdirAll(filepath.Dir(keyPath), 0o700)

    // Generate key
    priv, err := rsa.GenerateKey(crand.Reader, 2048)
    if err != nil {
        return "", "", err
    }

    // Build certificate template
    serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
    serialNumber, err := crand.Int(crand.Reader, serialNumberLimit)
    if err != nil {
        return "", "", err
    }
    tmpl := x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{CommonName: "Bibbl Self-Signed"},
        NotBefore: time.Now().Add(-1 * time.Hour),
        NotAfter:  time.Now().Add(validFor),
        KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        BasicConstraintsValid: true,
        IsCA:                  true,
    }
    for _, h := range hosts {
        if ip := net.ParseIP(h); ip != nil {
            tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
        } else {
            tmpl.DNSNames = append(tmpl.DNSNames, h)
        }
    }

    derBytes, err := x509.CreateCertificate(crand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
    if err != nil {
        return "", "", err
    }

    if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), 0o644); err != nil {
        return "", "", err
    }
    if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}), 0o600); err != nil {
        return "", "", err
    }
    return certPath, keyPath, nil
}

func fileExists(p string) bool {
    if p == "" { return false }
    _, err := os.Stat(p)
    return err == nil
}
