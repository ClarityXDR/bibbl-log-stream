package config

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type TLSConfig struct {
	CertFile     string
	KeyFile      string
	MinVersion   string   // e.g. "1.2", "1.3"
	CipherSuites []string // optional: explicit TLS cipher suite names
	ClientCAFile string   // optional path to CA bundle for client cert auth
	ClientAuth   string   // "", "require", "verify" (verify=RequireAnyClientCert, require=RequireAndVerifyClientCert)
	AutoCert     AutoCertConfig
}

type AutoCertConfig struct {
	Enabled         bool
	Hosts           []string
	ValidDays       int
	RenewBeforeDays int
	OutputDir       string
	CommonName      string
}

type Config struct {
	Server struct {
		Host                  string
		Port                  int
		ReadTimeout           time.Duration
		WriteTimeout          time.Duration
		TLS                   TLSConfig
		MaxRequestBytes       int
		RateLimitPerMin       int                 // simple global limit (best-effort)
		ContentSecurityPolicy string              // Configurable CSP header value
		AuthToken             string              // placeholder static bearer token (future pluggable providers)
		AuthTokens            map[string][]string // map of bearer token -> roles (e.g. admin,write,read)
		SecurityHeaders       struct {
			HSTS struct {
				Enabled           bool
				MaxAge            int
				IncludeSubdomains bool
				Preload           bool
			}
			PermissionsPolicy string
			COOP              string
			COEP              string
			CORP              string
		}
	}
	Logging struct {
		Level  string // debug|info|warn|error
		Format string // text|json
	}
	Inputs struct {
		Syslog struct {
			Enabled        bool
			Host           string
			Port           int
			TLS            TLSConfig
			AllowList      []string      // IP/CIDR allow-list (empty = allow all)
			IdleTimeout    time.Duration // connection idle timeout (0 = no timeout)
			ReadBufferSize int           // per-connection read buffer (bytes)
			MaxConnections int           // concurrent connection limit (0 = unlimited)
			VerboseLogging bool          // log each connection/disconnection for troubleshooting
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
	v.SetDefault("server.rate_limit_per_min", 600)         // 10 rps average
	// More permissive CSP for React: allow inline styles and necessary script sources
	v.SetDefault("server.content_security_policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' data:; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'")
	v.SetDefault("server.auth_token", "")
	v.SetDefault("server.auth_tokens", map[string]any{})
	v.SetDefault("server.security_headers.hsts.enabled", true)
	v.SetDefault("server.security_headers.hsts.max_age", 63072000)
	v.SetDefault("server.security_headers.hsts.include_subdomains", true)
	v.SetDefault("server.security_headers.hsts.preload", false)
	v.SetDefault("server.security_headers.permissions_policy", "geolocation=(), microphone=(), camera=()")
	v.SetDefault("server.security_headers.coop", "same-origin")
	v.SetDefault("server.security_headers.coep", "require-corp")
	v.SetDefault("server.security_headers.corp", "same-origin")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")

	// Syslog input defaults (TLS on 6514)
	v.SetDefault("inputs.syslog.enabled", false)
	v.SetDefault("inputs.syslog.host", "127.0.0.1")
	v.SetDefault("inputs.syslog.port", 6514)
	v.SetDefault("inputs.syslog.tls.cert_file", "")
	v.SetDefault("inputs.syslog.tls.key_file", "")
	v.SetDefault("inputs.syslog.tls.min_version", "1.2")
	v.SetDefault("inputs.syslog.tls.cipher_suites", []string{})
	v.SetDefault("inputs.syslog.allow_list", []string{})
	v.SetDefault("inputs.syslog.idle_timeout", "5m")
	v.SetDefault("inputs.syslog.read_buffer_size", 65536)
	v.SetDefault("inputs.syslog.max_connections", 1000)
	v.SetDefault("inputs.syslog.verbose_logging", false)
	v.SetDefault("inputs.syslog.tls.auto_cert.enabled", true)
	v.SetDefault("inputs.syslog.tls.auto_cert.hosts", []string{})
	v.SetDefault("inputs.syslog.tls.auto_cert.valid_days", 365)
	v.SetDefault("inputs.syslog.tls.auto_cert.renew_before_days", 30)
	v.SetDefault("inputs.syslog.tls.auto_cert.output_dir", "./certs/syslog")
	v.SetDefault("inputs.syslog.tls.auto_cert.common_name", "Bibbl Syslog AutoCert")

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
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9444
	}
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
				if strings.TrimSpace(vv) == "" {
					continue
				}
				parts := strings.Split(vv, ",")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				ats[k] = parts
			case []any:
				var parts []string
				for _, x := range vv {
					if s, ok := x.(string); ok {
						parts = append(parts, strings.TrimSpace(s))
					}
				}
				if len(parts) > 0 {
					ats[k] = parts
				}
			}
		}
	case map[string]string:
		for k, v := range m {
			if v != "" {
				ats[k] = []string{v}
			}
		}
	}
	cfg.Server.AuthTokens = ats
	cfg.Server.ContentSecurityPolicy = v.GetString("server.content_security_policy")
	cfg.Server.SecurityHeaders.HSTS.Enabled = v.GetBool("server.security_headers.hsts.enabled")
	cfg.Server.SecurityHeaders.HSTS.MaxAge = v.GetInt("server.security_headers.hsts.max_age")
	cfg.Server.SecurityHeaders.HSTS.IncludeSubdomains = v.GetBool("server.security_headers.hsts.include_subdomains")
	cfg.Server.SecurityHeaders.HSTS.Preload = v.GetBool("server.security_headers.hsts.preload")
	cfg.Server.SecurityHeaders.PermissionsPolicy = v.GetString("server.security_headers.permissions_policy")
	cfg.Server.SecurityHeaders.COOP = v.GetString("server.security_headers.coop")
	cfg.Server.SecurityHeaders.COEP = v.GetString("server.security_headers.coep")
	cfg.Server.SecurityHeaders.CORP = v.GetString("server.security_headers.corp")
	cfg.Logging.Level = v.GetString("logging.level")
	cfg.Logging.Format = v.GetString("logging.format")

	// Inputs.Syslog
	cfg.Inputs.Syslog.Enabled = v.GetBool("inputs.syslog.enabled")
	cfg.Inputs.Syslog.Host = v.GetString("inputs.syslog.host")
	cfg.Inputs.Syslog.Port = v.GetInt("inputs.syslog.port")
	cfg.Inputs.Syslog.TLS.CertFile = v.GetString("inputs.syslog.tls.cert_file")
	cfg.Inputs.Syslog.TLS.KeyFile = v.GetString("inputs.syslog.tls.key_file")
	cfg.Inputs.Syslog.TLS.MinVersion = v.GetString("inputs.syslog.tls.min_version")
	cfg.Inputs.Syslog.TLS.CipherSuites = readStringSlice(v.Get("inputs.syslog.tls.cipher_suites"))
	cfg.Inputs.Syslog.AllowList = readStringSlice(v.Get("inputs.syslog.allow_list"))
	cfg.Inputs.Syslog.IdleTimeout = v.GetDuration("inputs.syslog.idle_timeout")
	cfg.Inputs.Syslog.ReadBufferSize = v.GetInt("inputs.syslog.read_buffer_size")
	cfg.Inputs.Syslog.MaxConnections = v.GetInt("inputs.syslog.max_connections")
	cfg.Inputs.Syslog.VerboseLogging = v.GetBool("inputs.syslog.verbose_logging")
	cfg.Inputs.Syslog.TLS.AutoCert.Enabled = v.GetBool("inputs.syslog.tls.auto_cert.enabled")
	cfg.Inputs.Syslog.TLS.AutoCert.Hosts = readStringSlice(v.Get("inputs.syslog.tls.auto_cert.hosts"))
	cfg.Inputs.Syslog.TLS.AutoCert.ValidDays = v.GetInt("inputs.syslog.tls.auto_cert.valid_days")
	cfg.Inputs.Syslog.TLS.AutoCert.RenewBeforeDays = v.GetInt("inputs.syslog.tls.auto_cert.renew_before_days")
	cfg.Inputs.Syslog.TLS.AutoCert.OutputDir = v.GetString("inputs.syslog.tls.auto_cert.output_dir")
	cfg.Inputs.Syslog.TLS.AutoCert.CommonName = v.GetString("inputs.syslog.tls.auto_cert.common_name")

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
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, "server.port must be 1-65535")
	}
	if c.Server.MaxRequestBytes <= 0 || c.Server.MaxRequestBytes > 100*1024*1024 {
		errors = append(errors, "server.max_request_bytes out of range (1 .. 104857600)")
	}
	if c.Server.SecurityHeaders.HSTS.MaxAge < 0 {
		errors = append(errors, "server.security_headers.hsts.max_age must be >= 0")
	}
	switch c.Server.TLS.MinVersion {
	case "", "1.2", "1.3":
	default:
		errors = append(errors, "server.tls.min_version must be 1.2 or 1.3")
	}
	if c.Server.TLS.ClientAuth != "" && c.Server.TLS.ClientAuth != "require" && c.Server.TLS.ClientAuth != "verify" {
		errors = append(errors, "server.tls.client_auth must be empty, 'require' or 'verify'")
	}
	switch strings.ToLower(c.Logging.Level) {
	case "debug", "info", "warn", "error":
	default:
		errors = append(errors, "logging.level must be debug|info|warn|error")
	}
	if c.Logging.Format != "text" && c.Logging.Format != "json" {
		errors = append(errors, "logging.format must be text|json")
	}
	if c.Inputs.Syslog.Port < 0 || c.Inputs.Syslog.Port > 65535 {
		errors = append(errors, "inputs.syslog.port invalid")
	}
	if t := c.Server.TLS.ClientAuth; t != "" && c.Server.TLS.ClientCAFile == "" {
		errors = append(errors, "server.tls.client_ca_file required when client_auth set")
	}
	if ac := c.Inputs.Syslog.TLS.AutoCert; ac.Enabled {
		if ac.ValidDays <= 0 {
			errors = append(errors, "inputs.syslog.tls.auto_cert.valid_days must be > 0")
		}
		if ac.RenewBeforeDays < 0 {
			errors = append(errors, "inputs.syslog.tls.auto_cert.renew_before_days must be >= 0")
		}
		if ac.RenewBeforeDays >= ac.ValidDays {
			errors = append(errors, "inputs.syslog.tls.auto_cert.renew_before_days must be less than valid_days")
		}
	}
	// warnings (do not block startup)
	if c.Server.AuthToken == "" {
		warnings = append(warnings, "server.auth_token empty - API unprotected")
	}
	if len(c.Server.AuthTokens) == 0 {
		warnings = append(warnings, "server.auth_tokens empty - RBAC disabled")
	}
	return
}

func readStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return copyStrings(v)
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	default:
		return nil
	}
}

func copyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
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
