package vault

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"bibbl/internal/config"

	vaultapi "github.com/hashicorp/vault/api"
)

// Client wraps a Hashicorp Vault client with simple caching and placeholder resolution.
type Client struct {
	cfg   config.VaultConfig
	api   *vaultapi.Client
	cache map[string]cachedSecret
	mu    sync.RWMutex
}

type cachedSecret struct {
	data    map[string]interface{}
	expires time.Time
}

// NewClient initializes a Vault client using the provided configuration.
func NewClient(cfg config.VaultConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	conf := vaultapi.DefaultConfig()
	if cfg.Address != "" {
		conf.Address = cfg.Address
	}
	if cfg.RequestTimeout > 0 {
		conf.Timeout = cfg.RequestTimeout
	}
	tlsConf := &vaultapi.TLSConfig{
		CACert:     cfg.TLS.CAFile,
		ClientCert: cfg.TLS.CertFile,
		ClientKey:  cfg.TLS.KeyFile,
		Insecure:   cfg.TLSSkipVerify,
	}
	if err := conf.ConfigureTLS(tlsConf); err != nil {
		return nil, fmt.Errorf("configure vault tls: %w", err)
	}
	apiClient, err := vaultapi.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}
	if cfg.Namespace != "" {
		apiClient.SetNamespace(cfg.Namespace)
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" && cfg.TokenFile != "" {
		data, err := os.ReadFile(cfg.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("read vault token file: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" {
		return nil, fmt.Errorf("vault token required when vault enabled")
	}
	apiClient.SetToken(token)

	return &Client{
		cfg:   cfg,
		api:   apiClient,
		cache: make(map[string]cachedSecret),
	}, nil
}

// Resolve satisfies the secrets.Resolver interface.
func (c *Client) Resolve(ctx context.Context, ref string) (string, error) {
	if c == nil {
		return ref, nil
	}
	secretPath, field, err := parseRef(ref)
	if err != nil {
		return "", err
	}
	data, err := c.readPath(ctx, secretPath)
	if err != nil {
		return "", err
	}
	val, ok := data[field]
	if !ok {
		return "", fmt.Errorf("vault field %s missing at %s", field, secretPath)
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// HealthCheck validates connectivity to Vault.
func (c *Client) HealthCheck(ctx context.Context) error {
	if c == nil {
		return nil
	}
	_, err := c.api.Sys().HealthWithContext(ctx)
	return err
}

func (c *Client) readPath(ctx context.Context, rawPath string) (map[string]interface{}, error) {
	full := c.fullPath(rawPath)
	now := time.Now()
	c.mu.RLock()
	if cached, ok := c.cache[full]; ok && now.Before(cached.expires) {
		dataCopy := make(map[string]interface{}, len(cached.data))
		for k, v := range cached.data {
			dataCopy[k] = v
		}
		c.mu.RUnlock()
		return dataCopy, nil
	}
	c.mu.RUnlock()

	secret, err := c.api.Logical().ReadWithContext(ctx, full)
	if err != nil {
		return nil, fmt.Errorf("vault read %s: %w", full, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("vault secret %s not found", full)
	}
	data := secret.Data
	if c.cfg.KVVersion == 2 {
		if nested, ok := data["data"].(map[string]interface{}); ok {
			data = nested
		}
	}
	c.mu.Lock()
	c.cache[full] = cachedSecret{data: data, expires: now.Add(c.cfg.CacheTTL)}
	c.mu.Unlock()

	dataCopy := make(map[string]interface{}, len(data))
	for k, v := range data {
		dataCopy[k] = v
	}
	return dataCopy, nil
}

func (c *Client) fullPath(p string) string {
	trimmedPath := strings.TrimLeft(p, "/")
	if trimmedPath == "" {
		return strings.Trim(c.cfg.MountPath, "/")
	}
	mount := strings.Trim(c.cfg.MountPath, "/")
	if mount == "" {
		return trimmedPath
	}
	if strings.HasPrefix(trimmedPath, mount) {
		return trimmedPath
	}
	if c.cfg.KVVersion == 2 && !strings.Contains(trimmedPath, "/data/") {
		return path.Join(mount, "data", trimmedPath)
	}
	return path.Join(mount, trimmedPath)
}

func parseRef(ref string) (string, string, error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return "", "", fmt.Errorf("empty vault reference")
	}
	if !strings.HasPrefix(raw, "vault://") {
		return "", "", fmt.Errorf("invalid vault reference %s", raw)
	}
	withoutScheme := strings.TrimPrefix(raw, "vault://")
	pathPart := withoutScheme
	field := "value"
	if idx := strings.Index(withoutScheme, "#"); idx >= 0 {
		pathPart = withoutScheme[:idx]
		fieldCandidate := withoutScheme[idx+1:]
		if fieldCandidate != "" {
			field = fieldCandidate
		}
	}
	pathPart = strings.TrimLeft(pathPart, "/")
	if pathPart == "" {
		return "", "", fmt.Errorf("vault reference %s missing path", ref)
	}
	return pathPart, field, nil
}
