package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"bibbl/internal/api"
	"bibbl/internal/config"
	"bibbl/internal/diagnostics"
	"bibbl/internal/metrics"
	"bibbl/internal/platform/logger"
	"bibbl/internal/version"
	bibbltls "bibbl/pkg/tls"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	healthCheck := flag.Bool("health", false, "Perform health check and exit")
	diagMode := flag.Bool("diagnostics", false, "Print diagnostic information and exit")
	diagFormat := flag.String("diag-format", "text", "Diagnostics output format (text|json)")
	diagEnv := flag.Bool("diag-env", false, "Include environment variables in diagnostics")
	cfg := config.Load()

	// CLI overrides
	hostFlag := flag.String("host", "", "Server host to bind (overrides config)")
	portFlag := flag.Int("port", 0, "Server port to bind (overrides config, default 9444)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate (PEM)")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key (PEM)")
	tlsMin := flag.String("tls-min", "", "Minimum TLS version (1.2 or 1.3)")
	// Syslog input flags (configure engine-managed input)
	syslogEnable := flag.Bool("syslog", false, "Enable Syslog listener (engine-managed)")
	syslogHost := flag.String("syslog-host", "", "Syslog bind host (default 0.0.0.0)")
	syslogPort := flag.Int("syslog-port", 0, "Syslog bind port (default 6514)")
	syslogCert := flag.String("syslog-cert", "", "Path to Syslog TLS certificate (PEM)")
	syslogKey := flag.String("syslog-key", "", "Path to Syslog TLS private key (PEM)")
	syslogMin := flag.String("syslog-tls-min", "", "Syslog minimum TLS version (1.2 or 1.3)")
	flag.Parse()
	if *showVersion {
		fmt.Printf("Bibbl Log Stream %s (commit %s, date %s)\n", version.Version, version.Commit, version.Date)
		return
	}
	if *healthCheck {
		os.Exit(performHealthCheck(cfg))
	}
	if *diagMode {
		info := diagnostics.Collect(cfg, *diagEnv)
		if err := diagnostics.Print(info, *diagFormat); err != nil {
			fmt.Fprintf(os.Stderr, "Error printing diagnostics: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *hostFlag != "" {
		cfg.Server.Host = *hostFlag
	}
	if *portFlag > 0 {
		cfg.Server.Port = *portFlag
	}
	if *tlsCert != "" {
		cfg.Server.TLS.CertFile = *tlsCert
	}
	if *tlsKey != "" {
		cfg.Server.TLS.KeyFile = *tlsKey
	}
	if *tlsMin != "" {
		cfg.Server.TLS.MinVersion = *tlsMin
	}

	// Ensure web server has TLS: generate self-signed cert if missing
	if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
		// Base hostnames for certificate
		hosts := []string{"localhost", "127.0.0.1", cfg.Server.Host}

		// Add any additional hosts from environment variable
		if extraHosts := os.Getenv("BIBBL_TLS_EXTRA_HOSTS"); extraHosts != "" {
			for _, host := range strings.Split(extraHosts, ",") {
				host = strings.TrimSpace(host)
				if host != "" {
					hosts = append(hosts, host)
				}
			}
		}

		cpath, kpath, err := bibbltls.EnsurePairExists(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile, hosts, 0)
		if err != nil {
			log.Printf("web tls self-signed generation failed: %v", err)
		} else {
			cfg.Server.TLS.CertFile = cpath
			cfg.Server.TLS.KeyFile = kpath
			if cfg.Server.TLS.MinVersion == "" {
				cfg.Server.TLS.MinVersion = "1.2"
			}
			log.Printf("web tls self-signed cert generated: %s, key: %s", cpath, kpath)
			if cfg.Server.Host != "127.0.0.1" && cfg.Server.Host != "localhost" {
				log.Printf("WARNING: using auto-generated self-signed TLS cert on host %s - not recommended for production", cfg.Server.Host)
			}
		}
	}

	// Syslog overrides
	if *syslogEnable {
		cfg.Inputs.Syslog.Enabled = true
	}
	if *syslogHost != "" {
		cfg.Inputs.Syslog.Host = *syslogHost
	}
	if *syslogPort > 0 {
		cfg.Inputs.Syslog.Port = *syslogPort
	}
	if *syslogCert != "" {
		cfg.Inputs.Syslog.TLS.CertFile = *syslogCert
	}
	if *syslogKey != "" {
		cfg.Inputs.Syslog.TLS.KeyFile = *syslogKey
	}
	if *syslogMin != "" {
		cfg.Inputs.Syslog.TLS.MinVersion = *syslogMin
	}

	if err := ensureSyslogAutoCert(cfg); err != nil {
		log.Fatalf("failed to ensure syslog TLS certificate: %v", err)
	}

	// Validate config early (separate errors and warnings)
	if errs, warns := cfg.Validate(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "config error: %s\n", e)
		}
		os.Exit(2)
	} else if len(warns) > 0 {
		for _, w := range warns {
			fmt.Fprintf(os.Stderr, "config warning: %s\n", w)
		}
	}
	// Initialize structured logger
	logger.Init(logger.Config{Level: cfg.Logging.Level, Format: cfg.Logging.Format})
	logger.Slog().Info("starting bibbl", "version", version.Version, "commit", version.Commit, "date", version.Date)

	// Initialize Prometheus metrics
	metrics.Init()

	srv := api.NewServer(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Note: syslog listener is started by the API engine when the Syslog source is created and started.

	go func() {
		if err := srv.Start(); err != nil {
			logger.Slog().Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Slog().Info("shutdown signal received")
	sdCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.ShutdownWithContext(sdCtx); err != nil {
		logger.Slog().Error("graceful shutdown failed", "err", err)
	}
	logger.Slog().Info("shutdown complete")
}

// performHealthCheck performs a health check against the local server
// Returns 0 on success, 1 on failure
func performHealthCheck(cfg *config.Config) int {
	addr := cfg.HTTPAddr()

	// Create HTTP client with TLS config for self-signed certs
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip cert verification for health checks
			},
		},
	}

	// Try HTTPS first (most common case), then HTTP if that fails
	schemes := []string{"https", "http"}

	for _, scheme := range schemes {
		url := fmt.Sprintf("%s://%s/api/v1/health", scheme, addr)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Health check %s failed: %v", scheme, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Health check %s failed: HTTP %d", scheme, resp.StatusCode)
			continue
		}

		log.Printf("Health check %s passed", scheme)
		return 0
	}

	log.Printf("Health check failed on all schemes")
	return 1
}

func ensureSyslogAutoCert(cfg *config.Config) error {
	if !cfg.Inputs.Syslog.Enabled {
		return nil
	}
	auto := cfg.Inputs.Syslog.TLS.AutoCert
	if !auto.Enabled {
		return nil
	}
	outDir := auto.OutputDir
	if outDir == "" {
		outDir = "./certs/syslog"
	}
	certPath := cfg.Inputs.Syslog.TLS.CertFile
	keyPath := cfg.Inputs.Syslog.TLS.KeyFile
	if certPath == "" {
		certPath = filepath.Join(outDir, "bibbl-syslog.crt")
	}
	if keyPath == "" {
		keyPath = filepath.Join(outDir, "bibbl-syslog.key")
	}
	validFor := time.Duration(auto.ValidDays) * 24 * time.Hour
	renewBefore := time.Duration(auto.RenewBeforeDays) * 24 * time.Hour
	hosts := collectSyslogHosts(cfg, auto.Hosts)
	certPath, keyPath, rotated, err := bibbltls.EnsurePairFresh(certPath, keyPath, hosts, validFor, renewBefore, auto.CommonName)
	if err != nil {
		return err
	}
	cfg.Inputs.Syslog.TLS.CertFile = certPath
	cfg.Inputs.Syslog.TLS.KeyFile = keyPath
	if cfg.Inputs.Syslog.TLS.MinVersion == "" {
		cfg.Inputs.Syslog.TLS.MinVersion = "1.2"
	}
	bundlePath := filepath.Join(outDir, "bibbl-syslog-ca.pem")
	if err := exportSyslogBundle(certPath, bundlePath); err != nil {
		log.Printf("warning: failed to write syslog CA bundle: %v", err)
	}
	if rotated {
		log.Printf("syslog TLS certificate generated/renewed: cert=%s key=%s", certPath, keyPath)
	}
	log.Printf("Syslog TLS certificate available for Versa SD-WAN at cert=%s bundle=%s (share the .crt/.pem, keep the .key private)", certPath, bundlePath)
	return nil
}

func collectSyslogHosts(cfg *config.Config, base []string) []string {
	hosts := append([]string{}, base...)
	if h := cfg.Inputs.Syslog.Host; h != "" {
		hosts = append(hosts, h)
	}
	if h := cfg.Server.Host; h != "" {
		hosts = append(hosts, h)
	}
	hosts = append(hosts, "localhost", "127.0.0.1")
	for _, envKey := range []string{"BIBBL_SYSLOG_TLS_EXTRA_HOSTS", "BIBBL_TLS_EXTRA_HOSTS"} {
		if raw := os.Getenv(envKey); raw != "" {
			hosts = append(hosts, splitHostList(raw)...)
		}
	}
	return hosts
}

func splitHostList(raw string) []string {
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func exportSyslogBundle(certPath, bundlePath string) error {
	if certPath == "" || bundlePath == "" {
		return nil
	}
	data, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(bundlePath, data, 0o644)
}
