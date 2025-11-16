package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"bibbl/internal/config"
	"bibbl/internal/version"
)

// SystemInfo contains diagnostic information about the system.
type SystemInfo struct {
	Version     VersionInfo     `json:"version"`
	Runtime     RuntimeInfo     `json:"runtime"`
	Environment EnvironmentInfo `json:"environment"`
	Config      ConfigSummary   `json:"config"`
	Timestamp   string          `json:"timestamp"`
}

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

type RuntimeInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemStats     struct {
		Alloc      uint64 `json:"alloc_bytes"`
		TotalAlloc uint64 `json:"total_alloc_bytes"`
		Sys        uint64 `json:"sys_bytes"`
		NumGC      uint32 `json:"num_gc"`
	} `json:"mem_stats"`
}

type EnvironmentInfo struct {
	Hostname string            `json:"hostname"`
	WorkDir  string            `json:"work_dir"`
	EnvVars  map[string]string `json:"env_vars,omitempty"`
}

type ConfigSummary struct {
	ServerHost       string `json:"server_host"`
	ServerPort       int    `json:"server_port"`
	TLSEnabled       bool   `json:"tls_enabled"`
	LogLevel         string `json:"log_level"`
	SyslogEnabled    bool   `json:"syslog_enabled"`
	SyslogPort       int    `json:"syslog_port"`
	SyslogTLSEnabled bool   `json:"syslog_tls_enabled"`
	AuthTokenSet     bool   `json:"auth_token_set"`
	SecurityHeaders  bool   `json:"security_headers_enabled"`
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	AkamaiDS2Enabled bool   `json:"akamai_ds2_enabled"`
}

// Collect gathers diagnostic information.
func Collect(cfg *config.Config, includeEnv bool) SystemInfo {
	info := SystemInfo{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Version information
	info.Version = VersionInfo{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.Date,
		GoVersion: runtime.Version(),
	}

	// Runtime information
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info.Runtime = RuntimeInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
	info.Runtime.MemStats.Alloc = m.Alloc
	info.Runtime.MemStats.TotalAlloc = m.TotalAlloc
	info.Runtime.MemStats.Sys = m.Sys
	info.Runtime.MemStats.NumGC = m.NumGC

	// Environment information
	hostname, _ := os.Hostname()
	workdir, _ := os.Getwd()

	info.Environment = EnvironmentInfo{
		Hostname: hostname,
		WorkDir:  workdir,
	}

	if includeEnv {
		info.Environment.EnvVars = collectSafeEnvVars()
	}

	// Config summary (redacted)
	if cfg != nil {
		info.Config = ConfigSummary{
			ServerHost:       cfg.Server.Host,
			ServerPort:       cfg.Server.Port,
			TLSEnabled:       cfg.TLSConfigured(),
			LogLevel:         cfg.Logging.Level,
			SyslogEnabled:    cfg.Inputs.Syslog.Enabled,
			SyslogPort:       cfg.Inputs.Syslog.Port,
			SyslogTLSEnabled: cfg.Inputs.Syslog.TLS.CertFile != "",
			AuthTokenSet:     cfg.Server.AuthToken != "",
			SecurityHeaders:  cfg.Server.SecurityHeaders.HSTS.Enabled,
			RateLimitEnabled: cfg.Server.RateLimitPerMin > 0,
			AkamaiDS2Enabled: cfg.Inputs.AkamaiDS2.Enabled,
		}
	}

	return info
}

// collectSafeEnvVars returns environment variables that don't contain secrets.
func collectSafeEnvVars() map[string]string {
	safeVars := make(map[string]string)

	// Allow-list of safe environment variables
	safeKeys := []string{
		"HOME",
		"HOSTNAME",
		"PATH",
		"USER",
		"SHELL",
		"LANG",
		"TZ",
		"GOMAXPROCS",
		"GOGC",
		"GOMEMLIMIT",
		"GODEBUG",
	}

	for _, key := range safeKeys {
		if val := os.Getenv(key); val != "" {
			safeVars[key] = val
		}
	}

	return safeVars
}

// Print outputs the diagnostic information in the specified format.
func Print(info SystemInfo, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)

	case "text":
		fmt.Printf("Bibbl Log Stream Diagnostics\n")
		fmt.Printf("=============================\n\n")

		fmt.Printf("Version Information:\n")
		fmt.Printf("  Version:    %s\n", info.Version.Version)
		fmt.Printf("  Commit:     %s\n", info.Version.Commit)
		fmt.Printf("  Build Date: %s\n", info.Version.BuildDate)
		fmt.Printf("  Go Version: %s\n\n", info.Version.GoVersion)

		fmt.Printf("Runtime Information:\n")
		fmt.Printf("  OS:          %s\n", info.Runtime.OS)
		fmt.Printf("  Arch:        %s\n", info.Runtime.Arch)
		fmt.Printf("  CPUs:        %d\n", info.Runtime.NumCPU)
		fmt.Printf("  Goroutines:  %d\n", info.Runtime.NumGoroutine)
		fmt.Printf("  Memory:\n")
		fmt.Printf("    Allocated: %d MB\n", info.Runtime.MemStats.Alloc/1024/1024)
		fmt.Printf("    System:    %d MB\n", info.Runtime.MemStats.Sys/1024/1024)
		fmt.Printf("    GC Cycles: %d\n\n", info.Runtime.MemStats.NumGC)

		fmt.Printf("Environment:\n")
		fmt.Printf("  Hostname:   %s\n", info.Environment.Hostname)
		fmt.Printf("  Work Dir:   %s\n", info.Environment.WorkDir)
		if len(info.Environment.EnvVars) > 0 {
			fmt.Printf("  Env Vars:\n")
			for k, v := range info.Environment.EnvVars {
				fmt.Printf("    %s=%s\n", k, v)
			}
		}
		fmt.Printf("\n")

		fmt.Printf("Configuration Summary:\n")
		fmt.Printf("  Server:       %s:%d\n", info.Config.ServerHost, info.Config.ServerPort)
		fmt.Printf("  TLS:          %v\n", info.Config.TLSEnabled)
		fmt.Printf("  Log Level:    %s\n", info.Config.LogLevel)
		fmt.Printf("  Syslog:       %v (port %d, TLS: %v)\n",
			info.Config.SyslogEnabled, info.Config.SyslogPort, info.Config.SyslogTLSEnabled)
		fmt.Printf("  Auth Token:   %v\n", info.Config.AuthTokenSet)
		fmt.Printf("  Security:     %v\n", info.Config.SecurityHeaders)
		fmt.Printf("  Rate Limit:   %v\n", info.Config.RateLimitEnabled)
		fmt.Printf("  Akamai DS2:   %v\n\n", info.Config.AkamaiDS2Enabled)

		fmt.Printf("Timestamp: %s\n", info.Timestamp)

		return nil

	default:
		return fmt.Errorf("unsupported format: %s (use 'json' or 'text')", format)
	}
}
