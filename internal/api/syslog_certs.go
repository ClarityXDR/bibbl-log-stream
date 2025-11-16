package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// SyslogCertInfo describes a syslog TLS certificate file
type SyslogCertInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	ForVersa    bool   `json:"forVersa"`
	Size        int64  `json:"size"`
	Exists      bool   `json:"exists"`
}

// handleSyslogCertsList returns metadata about available syslog TLS certificates
func (s *Server) handleSyslogCertsList(w http.ResponseWriter, r *http.Request) {
	certDir := "./certs/syslog"
	if envDir := os.Getenv("BIBBL_SYSLOG_CERT_DIR"); envDir != "" {
		certDir = envDir
	}

	files := []SyslogCertInfo{
		{
			Name:        "bibbl-syslog.crt",
			Path:        filepath.Join(certDir, "bibbl-syslog.crt"),
			Description: "Server certificate (PEM format) - Bibbl's public certificate for TLS syslog",
			ForVersa:    true,
			Exists:      false,
		},
		{
			Name:        "bibbl-syslog-ca.pem",
			Path:        filepath.Join(certDir, "bibbl-syslog-ca.pem"),
			Description: "CA bundle (PEM format) - For Versa Director remote collector ca-cert-path",
			ForVersa:    true,
			Exists:      false,
		},
		{
			Name:        "bibbl-syslog.key",
			Path:        filepath.Join(certDir, "bibbl-syslog.key"),
			Description: "Private key (PEM format) - KEEP PRIVATE, do not upload to Versa",
			ForVersa:    false,
			Exists:      false,
		},
	}

	// Check which files exist and get sizes
	for i := range files {
		info, err := os.Stat(files[i].Path)
		if err == nil {
			files[i].Exists = true
			files[i].Size = info.Size()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"certificates": files,
		"versaGuide":   getVersaIntegrationGuide(),
	})
}

// handleSyslogCertDownload downloads a single certificate file
func (s *Server) handleSyslogCertDownload(w http.ResponseWriter, r *http.Request) {
	certName := r.URL.Query().Get("name")
	if certName == "" {
		http.Error(w, "missing 'name' query parameter", http.StatusBadRequest)
		return
	}

	// Validate allowed filenames to prevent path traversal
	allowed := map[string]bool{
		"bibbl-syslog.crt":    true,
		"bibbl-syslog-ca.pem": true,
		"bibbl-syslog.key":    true,
	}
	if !allowed[certName] {
		http.Error(w, "invalid certificate name", http.StatusBadRequest)
		return
	}

	certDir := "./certs/syslog"
	if envDir := os.Getenv("BIBBL_SYSLOG_CERT_DIR"); envDir != "" {
		certDir = envDir
	}

	certPath := filepath.Join(certDir, certName)
	data, err := os.ReadFile(certPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("certificate not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", certName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

// handleSyslogCertBundle downloads all Versa-required certificates as a ZIP file
func (s *Server) handleSyslogCertBundle(w http.ResponseWriter, r *http.Request) {
	certDir := "./certs/syslog"
	if envDir := os.Getenv("BIBBL_SYSLOG_CERT_DIR"); envDir != "" {
		certDir = envDir
	}

	// Files to include in bundle (excluding private key for security)
	filesToBundle := []string{
		"bibbl-syslog.crt",
		"bibbl-syslog-ca.pem",
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="bibbl-versa-certs.zip"`)

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Add README
	readme, err := zipWriter.Create("README.txt")
	if err == nil {
		_, _ = readme.Write([]byte(getVersaBundleReadme()))
	}

	// Add certificate files
	for _, filename := range filesToBundle {
		certPath := filepath.Join(certDir, filename)
		data, err := os.ReadFile(certPath)
		if err != nil {
			continue // Skip missing files
		}

		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			continue
		}
		_, _ = fileWriter.Write(data)
	}
}

func getVersaBundleReadme() string {
	return `BIBBL SYSLOG TLS CERTIFICATES FOR VERSA SD-WAN
================================================

This bundle contains the TLS certificates needed to configure Versa SD-WAN Director
to send logs to Bibbl Log Stream over secure syslog (TLS).

FILES INCLUDED:
---------------
1. bibbl-syslog.crt       - Server certificate (PEM format)
2. bibbl-syslog-ca.pem    - CA bundle (PEM format)

VERSA DIRECTOR CONFIGURATION:
-----------------------------
In Versa Director GUI:

1. Navigate to: Analytics > Administration > Configurations > Log Collector Exporter
2. Select the "Remote Collector" tab
3. Click "+ Add" to create a new remote collector
4. Configure the following fields:

   Name:                 bibbl-tls-collector
   Description:          Bibbl Log Stream via TLS
   Destination Address:  <Bibbl server IP or hostname>
   Destination Port:     6514 (or your configured syslog TLS port)
   Type:                 TCP
   Transport:            TLS (Enable TLS checkbox)
   Template:             <Select your syslog template>

5. TLS Configuration:
   - Upload bibbl-syslog-ca.pem to the "CA Certificate" field
   - Leave "Client Certificate" and "Private Key" empty (one-way TLS)
   - Note: Versa requires certificates in PEM format
   - Note: Versa does NOT support certificate chains

6. Click "Save Changes"

7. Create exporter rules to route logs to this collector:
   - Navigate to "Exporter Rules" tab
   - Add rule matching your log types
   - Set "Remote Collector Profile" to reference your new collector

SECURITY NOTES:
---------------
- The private key (bibbl-syslog.key) is NOT included in this bundle for security
- These certificates use one-way TLS (server authentication only)
- For mutual TLS, Versa would need to generate its own client certificate
- Keep your private keys secure and never upload them to network appliances

VERSA DOCUMENTATION:
--------------------
For detailed configuration steps, refer to:
https://docs.versa-networks.com/Management_and_Orchestration/Versa_Analytics/Configuration/Configure_Log_Collectors_and_Log_Exporter_Rules

Specifically the section: "Configure a Remote Collector" with TLS enabled.

TROUBLESHOOTING:
----------------
- Verify Bibbl is listening on the configured port: netstat -tuln | grep 6514
- Check Versa Analytics logs: /var/log/syslog on the Analytics node
- Test connectivity: openssl s_client -connect <bibbl-ip>:6514
- Ensure firewall allows TCP port 6514 between Versa and Bibbl

Generated by Bibbl Log Stream
https://github.com/ClarityXDR/bibbl-log-stream
`
}

func getVersaIntegrationGuide() map[string]any {
	return map[string]any{
		"title": "Versa SD-WAN Integration Guide",
		"steps": []string{
			"Download the certificate bundle using the /bundle endpoint",
			"In Versa Director, navigate to Analytics > Administration > Configurations > Log Collector Exporter",
			"Select the Remote Collector tab and click + Add",
			"Set Destination Address to your Bibbl server IP",
			"Set Destination Port to 6514 (or your Bibbl syslog TLS port)",
			"Set Type to TCP and enable TLS transport",
			"Upload bibbl-syslog-ca.pem as the CA Certificate",
			"Leave Client Certificate and Private Key empty (one-way TLS)",
			"Create exporter rules to route your desired log types to this collector",
		},
		"required_files": []string{
			"bibbl-syslog-ca.pem (CA certificate for Versa to trust Bibbl's server)",
		},
		"optional_files":     []string{},
		"documentation":      "https://docs.versa-networks.com/Management_and_Orchestration/Versa_Analytics/Configuration/Configure_Log_Collectors_and_Log_Exporter_Rules",
		"tls_mode":           "One-way TLS (server authentication only)",
		"certificate_format": "PEM (Privacy Enhanced Mail)",
		"notes": []string{
			"Versa requires certificates in PEM format",
			"Versa does NOT support certificate chains",
			"For mutual TLS, Versa would generate its own client cert/key",
			"The private key (bibbl-syslog.key) should never be uploaded to Versa",
		},
	}
}
