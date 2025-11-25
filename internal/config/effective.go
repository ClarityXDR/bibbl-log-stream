package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const redactedPlaceholder = "<redacted>"

// MarshalEffective returns the effective configuration rendered in the requested format
// after redacting sensitive fields.
func (c *Config) MarshalEffective(format string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("nil config")
	}
	sanitized := c.redactedClone()
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "yaml", "yml":
		return yaml.Marshal(&sanitized)
	case "json":
		return json.MarshalIndent(&sanitized, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func (c *Config) redactedClone() Config {
	if c == nil {
		return Config{}
	}
	clone := *c
	if clone.Server.AuthToken != "" {
		clone.Server.AuthToken = redactedPlaceholder
	}
	if len(c.Server.AuthTokens) > 0 {
		clone.Server.AuthTokens = map[string][]string{
			fmt.Sprintf("<redacted:%d tokens>", len(c.Server.AuthTokens)): []string{"roles hidden"},
		}
	}
	if clone.Inputs.AkamaiDS2.ClientToken != "" {
		clone.Inputs.AkamaiDS2.ClientToken = redactedPlaceholder
	}
	if clone.Inputs.AkamaiDS2.ClientSecret != "" {
		clone.Inputs.AkamaiDS2.ClientSecret = redactedPlaceholder
	}
	if clone.Inputs.AkamaiDS2.AccessToken != "" {
		clone.Inputs.AkamaiDS2.AccessToken = redactedPlaceholder
	}
	if clone.Outputs.AzureLogAnalytics.SharedKey != "" {
		clone.Outputs.AzureLogAnalytics.SharedKey = redactedPlaceholder
	}
	if clone.Secrets.Vault.Token != "" {
		clone.Secrets.Vault.Token = redactedPlaceholder
	}
	return clone
}
