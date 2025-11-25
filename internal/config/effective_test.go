package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestMarshalEffectiveRedactsSecrets(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.AuthToken = "super-secret"
	cfg.Server.AuthTokens = map[string][]string{
		"token-value": []string{"admin"},
	}
	cfg.Inputs.AkamaiDS2.ClientToken = "ct"
	cfg.Inputs.AkamaiDS2.ClientSecret = "cs"
	cfg.Inputs.AkamaiDS2.AccessToken = "at"

	out, err := cfg.MarshalEffective("json")
	if err != nil {
		t.Fatalf("MarshalEffective json: %v", err)
	}
	payload := string(out)
	normalized := strings.NewReplacer("\\u003c", "<", "\\u003e", ">").Replace(payload)
	for _, leak := range []string{"super-secret", "token-value", "ct", "cs", "at"} {
		if strings.Contains(normalized, fmt.Sprintf("\"%s\"", leak)) {
			t.Fatalf("expected %q to be redacted in %s", leak, payload)
		}
	}
	if !strings.Contains(normalized, redactedPlaceholder) {
		t.Fatalf("expected placeholder to appear: %s", payload)
	}
	if !strings.Contains(normalized, "<redacted:1 tokens>") {
		t.Fatalf("expected token summary placeholder: %s", payload)
	}

	if _, err := cfg.MarshalEffective("yaml"); err != nil {
		t.Fatalf("MarshalEffective yaml: %v", err)
	}

	if _, err := cfg.MarshalEffective("invalid"); err == nil {
		t.Fatalf("expected unsupported format error")
	}
}
