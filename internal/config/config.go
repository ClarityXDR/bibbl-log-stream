package config

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type TLSConfig struct {
	CertFile     string         `mapstructure:"cert_file" json:"cert_file" yaml:"cert_file"`
	KeyFile      string         `mapstructure:"key_file" json:"key_file" yaml:"key_file"`
	MinVersion   string         `mapstructure:"min_version" json:"min_version" yaml:"min_version"` // e.g. "1.2", "1.3"
	CipherSuites []string       `mapstructure:"cipher_suites" json:"cipher_suites" yaml:"cipher_suites"`
	ClientCAFile string         `mapstructure:"client_ca_file" json:"client_ca_file" yaml:"client_ca_file"`
	ClientAuth   string         `mapstructure:"client_auth" json:"client_auth" yaml:"client_auth"`
	AutoCert     AutoCertConfig `mapstructure:"auto_cert" json:"auto_cert" yaml:"auto_cert"`
}

type SecretsConfig struct {
	Vault VaultConfig `mapstructure:"vault" json:"vault" yaml:"vault"`
}

type VaultConfig struct {
	Enabled        bool           `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Address        string         `mapstructure:"address" json:"address" yaml:"address"`
	Namespace      string         `mapstructure:"namespace" json:"namespace" yaml:"namespace"`
	MountPath      string         `mapstructure:"mount_path" json:"mount_path" yaml:"mount_path"`
	KVVersion      int            `mapstructure:"kv_version" json:"kv_version" yaml:"kv_version"`
	Token          string         `mapstructure:"token" json:"token" yaml:"token"`
	TokenFile      string         `mapstructure:"token_file" json:"token_file" yaml:"token_file"`
	CacheTTL       time.Duration  `mapstructure:"cache_ttl" json:"cache_ttl" yaml:"cache_ttl"`
	RequestTimeout time.Duration  `mapstructure:"request_timeout" json:"request_timeout" yaml:"request_timeout"`
	TLSSkipVerify  bool           `mapstructure:"tls_skip_verify" json:"tls_skip_verify" yaml:"tls_skip_verify"`
	TLS            VaultTLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
}

type VaultTLSConfig struct {
	CAFile   string `mapstructure:"ca_file" json:"ca_file" yaml:"ca_file"`
	CertFile string `mapstructure:"cert_file" json:"cert_file" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" json:"key_file" yaml:"key_file"`
}

type OutputsConfig struct {
	AzureLogAnalytics AzureLogAnalyticsOutputConfig `mapstructure:"azure_log_analytics" json:"azure_log_analytics" yaml:"azure_log_analytics"`
}

type AzureLogAnalyticsOutputConfig struct {
	Enabled          bool        `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	WorkspaceID      string      `mapstructure:"workspace_id" json:"workspace_id" yaml:"workspace_id"`
	SharedKey        string      `mapstructure:"shared_key" json:"shared_key" yaml:"shared_key"`
	LogType          string      `mapstructure:"log_type" json:"log_type" yaml:"log_type"`
	ResourceGroup    string      `mapstructure:"resource_group" json:"resource_group" yaml:"resource_group"`
	ResourceID       string      `mapstructure:"resource_id" json:"resource_id" yaml:"resource_id"`
	BatchMaxEvents   int         `mapstructure:"batch_max_events" json:"batch_max_events" yaml:"batch_max_events"`
	BatchMaxBytes    int         `mapstructure:"batch_max_bytes" json:"batch_max_bytes" yaml:"batch_max_bytes"`
	FlushIntervalSec int         `mapstructure:"flush_interval_sec" json:"flush_interval_sec" yaml:"flush_interval_sec"`
	Concurrency      int         `mapstructure:"concurrency" json:"concurrency" yaml:"concurrency"`
	MaxRetries       int         `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	RetryDelaySec    int         `mapstructure:"retry_delay_sec" json:"retry_delay_sec" yaml:"retry_delay_sec"`
	Spill            SpillConfig `mapstructure:"spill" json:"spill" yaml:"spill"`
}

type SpillConfig struct {
	Enabled     bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Directory   string `mapstructure:"directory" json:"directory" yaml:"directory"`
	MaxBytes    int64  `mapstructure:"max_bytes" json:"max_bytes" yaml:"max_bytes"`
	SegmentSize int64  `mapstructure:"segment_size" json:"segment_size" yaml:"segment_size"`
	Encrypt     bool   `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"`
	KeyEnv      string `mapstructure:"key_env" json:"key_env" yaml:"key_env"`
}

type TelemetryConfig struct {
	OTLP OTLPConfig `mapstructure:"otlp" json:"otlp" yaml:"otlp"`
}

type OTLPConfig struct {
	Endpoint    string            `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Headers     map[string]string `mapstructure:"headers" json:"headers" yaml:"headers"`
	Insecure    bool              `mapstructure:"insecure" json:"insecure" yaml:"insecure"`
	Compression string            `mapstructure:"compression" json:"compression" yaml:"compression"`
	Timeout     time.Duration     `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	SampleRatio float64           `mapstructure:"sample_ratio" json:"sample_ratio" yaml:"sample_ratio"`
}

type AutoCertConfig struct {
	Enabled         bool     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Hosts           []string `mapstructure:"hosts" json:"hosts" yaml:"hosts"`
	ValidDays       int      `mapstructure:"valid_days" json:"valid_days" yaml:"valid_days"`
	RenewBeforeDays int      `mapstructure:"renew_before_days" json:"renew_before_days" yaml:"renew_before_days"`
	OutputDir       string   `mapstructure:"output_dir" json:"output_dir" yaml:"output_dir"`
	CommonName      string   `mapstructure:"common_name" json:"common_name" yaml:"common_name"`
}

type Config struct {
	Server struct {
		Host                  string              `mapstructure:"host" json:"host" yaml:"host"`
		Port                  int                 `mapstructure:"port" json:"port" yaml:"port"`
		ReadTimeout           time.Duration       `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
		WriteTimeout          time.Duration       `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
		TLS                   TLSConfig           `mapstructure:"tls" json:"tls" yaml:"tls"`
		MaxRequestBytes       int                 `mapstructure:"max_request_bytes" json:"max_request_bytes" yaml:"max_request_bytes"`
		RateLimitPerMin       int                 `mapstructure:"rate_limit_per_min" json:"rate_limit_per_min" yaml:"rate_limit_per_min"`
		ContentSecurityPolicy string              `mapstructure:"content_security_policy" json:"content_security_policy" yaml:"content_security_policy"`
		AuthToken             string              `mapstructure:"auth_token" json:"auth_token" yaml:"auth_token"`
		AuthTokens            map[string][]string `mapstructure:"auth_tokens" json:"auth_tokens" yaml:"auth_tokens"`
		SecurityHeaders       struct {
			HSTS struct {
				Enabled           bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
				MaxAge            int  `mapstructure:"max_age" json:"max_age" yaml:"max_age"`
				IncludeSubdomains bool `mapstructure:"include_subdomains" json:"include_subdomains" yaml:"include_subdomains"`
				Preload           bool `mapstructure:"preload" json:"preload" yaml:"preload"`
			} `mapstructure:"hsts" json:"hsts" yaml:"hsts"`
			PermissionsPolicy string `mapstructure:"permissions_policy" json:"permissions_policy" yaml:"permissions_policy"`
			COOP              string `mapstructure:"coop" json:"coop" yaml:"coop"`
			COEP              string `mapstructure:"coep" json:"coep" yaml:"coep"`
			CORP              string `mapstructure:"corp" json:"corp" yaml:"corp"`
		} `mapstructure:"security_headers" json:"security_headers" yaml:"security_headers"`
	} `mapstructure:"server" json:"server" yaml:"server"`
	Logging struct {
		Level  string `mapstructure:"level" json:"level" yaml:"level"`
		Format string `mapstructure:"format" json:"format" yaml:"format"`
	} `mapstructure:"logging" json:"logging" yaml:"logging"`
	Outputs   OutputsConfig   `mapstructure:"outputs" json:"outputs" yaml:"outputs"`
	Secrets   SecretsConfig   `mapstructure:"secrets" json:"secrets" yaml:"secrets"`
	Telemetry TelemetryConfig `mapstructure:"telemetry" json:"telemetry" yaml:"telemetry"`
	Inputs    struct {
		Syslog struct {
			Enabled        bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
			Host           string        `mapstructure:"host" json:"host" yaml:"host"`
			Port           int           `mapstructure:"port" json:"port" yaml:"port"`
			TLS            TLSConfig     `mapstructure:"tls" json:"tls" yaml:"tls"`
			AllowList      []string      `mapstructure:"allow_list" json:"allow_list" yaml:"allow_list"`
			IdleTimeout    time.Duration `mapstructure:"idle_timeout" json:"idle_timeout" yaml:"idle_timeout"`
			ReadBufferSize int           `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`
			MaxConnections int           `mapstructure:"max_connections" json:"max_connections" yaml:"max_connections"`
			VerboseLogging bool          `mapstructure:"verbose_logging" json:"verbose_logging" yaml:"verbose_logging"`
		} `mapstructure:"syslog" json:"syslog" yaml:"syslog"`
		AkamaiDS2 struct {
			Enabled         bool        `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
			Host            string      `mapstructure:"host" json:"host" yaml:"host"`
			ClientToken     string      `mapstructure:"client_token" json:"client_token" yaml:"client_token"`
			ClientSecret    string      `mapstructure:"client_secret" json:"client_secret" yaml:"client_secret"`
			AccessToken     string      `mapstructure:"access_token" json:"access_token" yaml:"access_token"`
			IntervalSeconds int         `mapstructure:"interval_seconds" json:"interval_seconds" yaml:"interval_seconds"`
			Streams         interface{} `mapstructure:"streams" json:"streams" yaml:"streams"`
		} `mapstructure:"akamai_ds2" json:"akamai_ds2" yaml:"akamai_ds2"`
	} `mapstructure:"inputs" json:"inputs" yaml:"inputs"`
}

func Load() *Config {
	cfg, err := loadConfig("")
	if err != nil {
		panic(err)
	}
	return cfg
}

// LoadFile loads configuration from a specific file path, returning an error if it cannot be read.
func LoadFile(path string) (*Config, error) {
	return loadConfig(strings.TrimSpace(path))
}

func loadConfig(configPath string) (*Config, error) {
	v := viper.New()
	if configPath == "" {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.SetConfigType("yaml")
	} else {
		v.SetConfigFile(configPath)
	}
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

	// Secrets / Vault defaults
	v.SetDefault("secrets.vault.enabled", false)
	v.SetDefault("secrets.vault.address", "http://vault:8200")
	v.SetDefault("secrets.vault.mount_path", "secret")
	v.SetDefault("secrets.vault.kv_version", 2)
	v.SetDefault("secrets.vault.cache_ttl", "5m")
	v.SetDefault("secrets.vault.request_timeout", "10s")
	v.SetDefault("secrets.vault.tls_skip_verify", true)
	v.SetDefault("secrets.vault.token", "")
	v.SetDefault("secrets.vault.token_file", "")
	v.SetDefault("secrets.vault.namespace", "")
	v.SetDefault("secrets.vault.tls.ca_file", "")
	v.SetDefault("secrets.vault.tls.cert_file", "")
	v.SetDefault("secrets.vault.tls.key_file", "")

	// Telemetry defaults
	v.SetDefault("telemetry.otlp.endpoint", "")
	v.SetDefault("telemetry.otlp.insecure", false)
	v.SetDefault("telemetry.otlp.compression", "gzip")
	v.SetDefault("telemetry.otlp.timeout", "10s")
	v.SetDefault("telemetry.otlp.sample_ratio", 1.0)
	v.SetDefault("telemetry.otlp.headers", map[string]any{})

	// Output defaults
	v.SetDefault("outputs.azure_log_analytics.enabled", false)
	v.SetDefault("outputs.azure_log_analytics.workspace_id", "")
	v.SetDefault("outputs.azure_log_analytics.shared_key", "")
	v.SetDefault("outputs.azure_log_analytics.log_type", "BibblLogs")
	v.SetDefault("outputs.azure_log_analytics.batch_max_events", 500)
	v.SetDefault("outputs.azure_log_analytics.batch_max_bytes", 1024*1024)
	v.SetDefault("outputs.azure_log_analytics.flush_interval_sec", 10)
	v.SetDefault("outputs.azure_log_analytics.concurrency", 2)
	v.SetDefault("outputs.azure_log_analytics.max_retries", 3)
	v.SetDefault("outputs.azure_log_analytics.retry_delay_sec", 2)
	v.SetDefault("outputs.azure_log_analytics.resource_group", "")
	v.SetDefault("outputs.azure_log_analytics.resource_id", "")
	v.SetDefault("outputs.azure_log_analytics.spill.enabled", true)
	v.SetDefault("outputs.azure_log_analytics.spill.directory", "./data/spill/azure")
	v.SetDefault("outputs.azure_log_analytics.spill.max_bytes", int64(10*1024*1024*1024))
	v.SetDefault("outputs.azure_log_analytics.spill.segment_size", int64(1*1024*1024))
	v.SetDefault("outputs.azure_log_analytics.spill.encrypt", false)
	v.SetDefault("outputs.azure_log_analytics.spill.key_env", "BIBBL_SPILL_KEY")

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

	if configPath != "" {
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	} else {
		_ = v.ReadInConfig()
	}

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

	// Outputs
	cfg.Outputs.AzureLogAnalytics.Enabled = v.GetBool("outputs.azure_log_analytics.enabled")
	cfg.Outputs.AzureLogAnalytics.WorkspaceID = v.GetString("outputs.azure_log_analytics.workspace_id")
	cfg.Outputs.AzureLogAnalytics.SharedKey = v.GetString("outputs.azure_log_analytics.shared_key")
	cfg.Outputs.AzureLogAnalytics.LogType = v.GetString("outputs.azure_log_analytics.log_type")
	cfg.Outputs.AzureLogAnalytics.ResourceGroup = v.GetString("outputs.azure_log_analytics.resource_group")
	cfg.Outputs.AzureLogAnalytics.ResourceID = v.GetString("outputs.azure_log_analytics.resource_id")
	cfg.Outputs.AzureLogAnalytics.BatchMaxEvents = v.GetInt("outputs.azure_log_analytics.batch_max_events")
	cfg.Outputs.AzureLogAnalytics.BatchMaxBytes = v.GetInt("outputs.azure_log_analytics.batch_max_bytes")
	cfg.Outputs.AzureLogAnalytics.FlushIntervalSec = v.GetInt("outputs.azure_log_analytics.flush_interval_sec")
	cfg.Outputs.AzureLogAnalytics.Concurrency = v.GetInt("outputs.azure_log_analytics.concurrency")
	cfg.Outputs.AzureLogAnalytics.MaxRetries = v.GetInt("outputs.azure_log_analytics.max_retries")
	cfg.Outputs.AzureLogAnalytics.RetryDelaySec = v.GetInt("outputs.azure_log_analytics.retry_delay_sec")
	cfg.Outputs.AzureLogAnalytics.Spill.Enabled = v.GetBool("outputs.azure_log_analytics.spill.enabled")
	cfg.Outputs.AzureLogAnalytics.Spill.Directory = v.GetString("outputs.azure_log_analytics.spill.directory")
	cfg.Outputs.AzureLogAnalytics.Spill.MaxBytes = v.GetInt64("outputs.azure_log_analytics.spill.max_bytes")
	cfg.Outputs.AzureLogAnalytics.Spill.SegmentSize = v.GetInt64("outputs.azure_log_analytics.spill.segment_size")
	cfg.Outputs.AzureLogAnalytics.Spill.Encrypt = v.GetBool("outputs.azure_log_analytics.spill.encrypt")
	cfg.Outputs.AzureLogAnalytics.Spill.KeyEnv = v.GetString("outputs.azure_log_analytics.spill.key_env")

	// Secrets
	cfg.Secrets.Vault.Enabled = v.GetBool("secrets.vault.enabled")
	cfg.Secrets.Vault.Address = v.GetString("secrets.vault.address")
	cfg.Secrets.Vault.Namespace = v.GetString("secrets.vault.namespace")
	cfg.Secrets.Vault.MountPath = v.GetString("secrets.vault.mount_path")
	cfg.Secrets.Vault.KVVersion = v.GetInt("secrets.vault.kv_version")
	cfg.Secrets.Vault.Token = v.GetString("secrets.vault.token")
	cfg.Secrets.Vault.TokenFile = v.GetString("secrets.vault.token_file")
	cfg.Secrets.Vault.CacheTTL = v.GetDuration("secrets.vault.cache_ttl")
	cfg.Secrets.Vault.RequestTimeout = v.GetDuration("secrets.vault.request_timeout")
	cfg.Secrets.Vault.TLSSkipVerify = v.GetBool("secrets.vault.tls_skip_verify")
	cfg.Secrets.Vault.TLS.CAFile = v.GetString("secrets.vault.tls.ca_file")
	cfg.Secrets.Vault.TLS.CertFile = v.GetString("secrets.vault.tls.cert_file")
	cfg.Secrets.Vault.TLS.KeyFile = v.GetString("secrets.vault.tls.key_file")

	// Telemetry
	cfg.Telemetry.OTLP.Endpoint = v.GetString("telemetry.otlp.endpoint")
	cfg.Telemetry.OTLP.Headers = readStringMap(v.Get("telemetry.otlp.headers"))
	cfg.Telemetry.OTLP.Insecure = v.GetBool("telemetry.otlp.insecure")
	cfg.Telemetry.OTLP.Compression = v.GetString("telemetry.otlp.compression")
	cfg.Telemetry.OTLP.Timeout = v.GetDuration("telemetry.otlp.timeout")
	cfg.Telemetry.OTLP.SampleRatio = v.GetFloat64("telemetry.otlp.sample_ratio")
	if cfg.Secrets.Vault.CacheTTL <= 0 {
		cfg.Secrets.Vault.CacheTTL = 5 * time.Minute
	}
	if cfg.Secrets.Vault.RequestTimeout <= 0 {
		cfg.Secrets.Vault.RequestTimeout = 10 * time.Second
	}
	if cfg.Outputs.AzureLogAnalytics.Spill.MaxBytes <= 0 {
		cfg.Outputs.AzureLogAnalytics.Spill.MaxBytes = 10 * 1024 * 1024 * 1024
	}
	if cfg.Outputs.AzureLogAnalytics.Spill.SegmentSize <= 0 {
		cfg.Outputs.AzureLogAnalytics.Spill.SegmentSize = 1 * 1024 * 1024
	}
	if cfg.Telemetry.OTLP.SampleRatio <= 0 {
		cfg.Telemetry.OTLP.SampleRatio = 1
	}

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
	return cfg, nil
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
	if c.Outputs.AzureLogAnalytics.Enabled {
		if strings.TrimSpace(c.Outputs.AzureLogAnalytics.WorkspaceID) == "" {
			errors = append(errors, "outputs.azure_log_analytics.workspace_id required when enabled")
		}
		if strings.TrimSpace(c.Outputs.AzureLogAnalytics.SharedKey) == "" {
			warnings = append(warnings, "outputs.azure_log_analytics.shared_key empty - expected vault reference or override")
		}
		if c.Outputs.AzureLogAnalytics.Spill.Enabled {
			if strings.TrimSpace(c.Outputs.AzureLogAnalytics.Spill.Directory) == "" {
				errors = append(errors, "outputs.azure_log_analytics.spill.directory required when spill enabled")
			}
			if c.Outputs.AzureLogAnalytics.Spill.MaxBytes <= 0 {
				errors = append(errors, "outputs.azure_log_analytics.spill.max_bytes must be > 0")
			}
			if c.Outputs.AzureLogAnalytics.Spill.SegmentSize <= 0 {
				errors = append(errors, "outputs.azure_log_analytics.spill.segment_size must be > 0")
			}
			if c.Outputs.AzureLogAnalytics.Spill.SegmentSize > c.Outputs.AzureLogAnalytics.Spill.MaxBytes {
				errors = append(errors, "outputs.azure_log_analytics.spill.segment_size cannot exceed max_bytes")
			}
		}
	}
	if c.Secrets.Vault.Enabled {
		if strings.TrimSpace(c.Secrets.Vault.Address) == "" {
			errors = append(errors, "secrets.vault.address required when enabled")
		}
		if strings.TrimSpace(c.Secrets.Vault.Token) == "" && strings.TrimSpace(c.Secrets.Vault.TokenFile) == "" {
			errors = append(errors, "secrets.vault.token or token_file required when vault enabled")
		}
		if c.Secrets.Vault.KVVersion != 1 && c.Secrets.Vault.KVVersion != 2 {
			errors = append(errors, "secrets.vault.kv_version must be 1 or 2")
		}
		if c.Secrets.Vault.CacheTTL <= 0 {
			warnings = append(warnings, "secrets.vault.cache_ttl not set - falling back to 5m")
		}
		if c.Secrets.Vault.RequestTimeout <= 0 {
			warnings = append(warnings, "secrets.vault.request_timeout not set - falling back to 10s")
		}
	}
	if c.Telemetry.OTLP.SampleRatio < 0 || c.Telemetry.OTLP.SampleRatio > 1 {
		errors = append(errors, "telemetry.otlp.sample_ratio must be between 0 and 1")
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

func readStringMap(value interface{}) map[string]string {
	switch v := value.(type) {
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			out[k] = val
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				out[k] = s
			}
		}
		return out
	default:
		return map[string]string{}
	}
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
