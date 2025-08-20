package config

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type TLSConfig struct {
	CertFile   string
	KeyFile    string
	MinVersion string // e.g. "1.2", "1.3"
	ClientCAFile string // optional path to CA bundle for client cert auth
	ClientAuth   string // "", "require", "verify" (verify=RequireAnyClientCert, require=RequireAndVerifyClientCert)
}

type Config struct {
	Server struct {
		Host         string
		Port         int
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		TLS          TLSConfig
		MaxRequestBytes int
		RateLimitPerMin int // simple global limit (best-effort)
			AuthToken string // placeholder static bearer token (future pluggable providers)
		AuthTokens map[string][]string // map of bearer token -> roles (e.g. admin,write,read)
	}
	Logging struct {
		Level  string // debug|info|warn|error
		Format string // text|json
	}
	Inputs struct {
		Syslog struct {
			Enabled bool
			Host    string
			Port    int
			TLS     TLSConfig
		}
		AkamaiDS2 struct {
			Enabled         bool
			Host            string
			ClientToken     string
			ClientSecret    string
			AccessToken     string
			IntervalSeconds int
			Streams         interface{} // string (comma separated) or []any
		}
	}
}

func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")
	// Environment variable support. Example: BIBBL_SERVER_PORT=9555
	v.SetEnvPrefix("BIBBL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults (prefer IPv4 loopback by default to avoid IPv6-only binding)
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 9444)
	v.SetDefault("server.readtimeout", "15s")
	v.SetDefault("server.writetimeout", "15s")
	v.SetDefault("server.tls.cert_file", "")
	v.SetDefault("server.tls.key_file", "")
	v.SetDefault("server.tls.min_version", "1.2")
	v.SetDefault("server.max_request_bytes", 10*1024*1024) // 10MB
	v.SetDefault("server.rate_limit_per_min", 600) // 10 rps average
	v.SetDefault("server.auth_token", "")
	v.SetDefault("server.auth_tokens", map[string]any{})

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")

	// Syslog input defaults (TLS on 6514)
	v.SetDefault("inputs.syslog.enabled", false)
	v.SetDefault("inputs.syslog.host", "127.0.0.1")
	v.SetDefault("inputs.syslog.port", 6514)
	v.SetDefault("inputs.syslog.tls.cert_file", "")
	v.SetDefault("inputs.syslog.tls.key_file", "")
	v.SetDefault("inputs.syslog.tls.min_version", "1.2")

	// Akamai DataStream 2 defaults (disabled by default; interval 60s)
	v.SetDefault("inputs.akamai_ds2.enabled", false)
	v.SetDefault("inputs.akamai_ds2.host", "")
	v.SetDefault("inputs.akamai_ds2.client_token", "")
	v.SetDefault("inputs.akamai_ds2.client_secret", "")
	v.SetDefault("inputs.akamai_ds2.access_token", "")
	v.SetDefault("inputs.akamai_ds2.interval_seconds", 60)
	v.SetDefault("inputs.akamai_ds2.streams", "")

	_ = v.ReadInConfig()

	cfg := &Config{}
	cfg.Server.Host = v.GetString("server.host")
	cfg.Server.Port = v.GetInt("server.port")
	if cfg.Server.Port == 0 { cfg.Server.Port = 9444 }
	cfg.Server.ReadTimeout = v.GetDuration("server.readtimeout")
	cfg.Server.WriteTimeout = v.GetDuration("server.writetimeout")
	cfg.Server.TLS.CertFile = v.GetString("server.tls.cert_file")
	cfg.Server.TLS.KeyFile = v.GetString("server.tls.key_file")
	cfg.Server.TLS.MinVersion = v.GetString("server.tls.min_version")
	cfg.Server.TLS.ClientCAFile = v.GetString("server.tls.client_ca_file")
	cfg.Server.TLS.ClientAuth = v.GetString("server.tls.client_auth")
	cfg.Server.MaxRequestBytes = v.GetInt("server.max_request_bytes")
	cfg.Server.RateLimitPerMin = v.GetInt("server.rate_limit_per_min")
	cfg.Server.AuthToken = v.GetString("server.auth_token")
	// Flexible parse of auth_tokens: values can be string (comma separated) or array
	ats := map[string][]string{}
	raw := v.Get("server.auth_tokens")
	switch m := raw.(type) {
	case map[string]any:
		for k, v := range m {
			switch vv := v.(type) {
			case string:
				if strings.TrimSpace(vv) == "" { continue }
				parts := strings.Split(vv, ",")
				for i := range parts { parts[i] = strings.TrimSpace(parts[i]) }
				ats[k] = parts
			case []any:
				var parts []string
				for _, x := range vv { if s, ok := x.(string); ok { parts = append(parts, strings.TrimSpace(s)) } }
				if len(parts) > 0 { ats[k] = parts }
			}
		}
	case map[string]string:
		for k, v := range m { if v != "" { ats[k] = []string{v} } }
	}
	cfg.Server.AuthTokens = ats
	cfg.Logging.Level = v.GetString("logging.level")
	cfg.Logging.Format = v.GetString("logging.format")

	// Inputs.Syslog
	cfg.Inputs.Syslog.Enabled = v.GetBool("inputs.syslog.enabled")
	cfg.Inputs.Syslog.Host = v.GetString("inputs.syslog.host")
	cfg.Inputs.Syslog.Port = v.GetInt("inputs.syslog.port")
	cfg.Inputs.Syslog.TLS.CertFile = v.GetString("inputs.syslog.tls.cert_file")
	cfg.Inputs.Syslog.TLS.KeyFile = v.GetString("inputs.syslog.tls.key_file")
	cfg.Inputs.Syslog.TLS.MinVersion = v.GetString("inputs.syslog.tls.min_version")

	// Inputs.AkamaiDS2
	cfg.Inputs.AkamaiDS2.Enabled = v.GetBool("inputs.akamai_ds2.enabled")
	cfg.Inputs.AkamaiDS2.Host = v.GetString("inputs.akamai_ds2.host")
	cfg.Inputs.AkamaiDS2.ClientToken = v.GetString("inputs.akamai_ds2.client_token")
	cfg.Inputs.AkamaiDS2.ClientSecret = v.GetString("inputs.akamai_ds2.client_secret")
	cfg.Inputs.AkamaiDS2.AccessToken = v.GetString("inputs.akamai_ds2.access_token")
	cfg.Inputs.AkamaiDS2.IntervalSeconds = v.GetInt("inputs.akamai_ds2.interval_seconds")
	cfg.Inputs.AkamaiDS2.Streams = v.Get("inputs.akamai_ds2.streams")
	return cfg
}

func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) TLSConfigured() bool {
	return c.Server.TLS.CertFile != "" && c.Server.TLS.KeyFile != ""
}

// SyslogAddr returns host:port for the Syslog listener
func (c *Config) SyslogAddr() string {
	host := c.Inputs.Syslog.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.Inputs.Syslog.Port
	if port <= 0 {
		port = 6514
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// Validate performs static validation and returns a slice of error messages (empty if valid).
func (c *Config) Validate() (errors []string, warnings []string) {
	if c.Server.Port <= 0 || c.Server.Port > 65535 { errors = append(errors, "server.port must be 1-65535") }
	if c.Server.MaxRequestBytes <= 0 || c.Server.MaxRequestBytes > 100*1024*1024 { errors = append(errors, "server.max_request_bytes out of range (1 .. 104857600)") }
	switch c.Server.TLS.MinVersion {
	case "", "1.2", "1.3":
	default: errors = append(errors, "server.tls.min_version must be 1.2 or 1.3")
	}
	if c.Server.TLS.ClientAuth != "" && c.Server.TLS.ClientAuth != "require" && c.Server.TLS.ClientAuth != "verify" { errors = append(errors, "server.tls.client_auth must be empty, 'require' or 'verify'") }
	switch strings.ToLower(c.Logging.Level) {
	case "debug","info","warn","error":
	default: errors = append(errors, "logging.level must be debug|info|warn|error")
	}
	if c.Logging.Format != "text" && c.Logging.Format != "json" { errors = append(errors, "logging.format must be text|json") }
	if c.Inputs.Syslog.Port < 0 || c.Inputs.Syslog.Port > 65535 { errors = append(errors, "inputs.syslog.port invalid") }
	if t := c.Server.TLS.ClientAuth; t != "" && c.Server.TLS.ClientCAFile == "" { errors = append(errors, "server.tls.client_ca_file required when client_auth set") }
	// warnings (do not block startup)
	if c.Server.AuthToken == "" { warnings = append(warnings, "server.auth_token empty - API unprotected") }
	if len(c.Server.AuthTokens) == 0 { warnings = append(warnings, "server.auth_tokens empty - RBAC disabled") }
	return
}

// TLSClientAuthType converts config to tls.ClientAuthType
func (c *Config) TLSClientAuthType() tls.ClientAuthType {
	switch c.Server.TLS.ClientAuth {
	case "require":
		return tls.RequireAndVerifyClientCert
	case "verify":
		return tls.RequireAnyClientCert
	default:
		return tls.NoClientCert
	}
}
