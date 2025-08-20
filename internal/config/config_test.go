package config

import (
	"os"
	"testing"
)

func TestEnvOverrides(t *testing.T) {
	os.Setenv("BIBBL_SERVER_PORT", "9555")
	defer os.Unsetenv("BIBBL_SERVER_PORT")
	cfg := Load()
	if cfg.Server.Port != 9555 {
		// 0 indicates fallback not applied
		if cfg.Server.Port == 0 { t.Fatalf("expected env var to set port to 9555, got 0") }
		if cfg.Server.Port != 9555 { t.Fatalf("expected 9555 got %d", cfg.Server.Port) }
	}
}
