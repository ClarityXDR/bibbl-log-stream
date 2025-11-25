package selfcheck

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibbl/internal/config"
)

// Dependencies surfaces optional clients required for checks.
type Dependencies struct {
	Vault interface{ HealthCheck(context.Context) error }
}

// Run executes startup dependency validation.
func Run(ctx context.Context, cfg *config.Config, deps Dependencies) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	if cfg.Secrets.Vault.Enabled {
		if deps.Vault == nil {
			return fmt.Errorf("vault enabled but no client available for health check")
		}
		if err := deps.Vault.HealthCheck(ctx); err != nil {
			return fmt.Errorf("vault health check failed: %w", err)
		}
	}
	if cfg.Outputs.AzureLogAnalytics.Enabled {
		if err := checkAzureEndpoint(ctx, cfg.Outputs.AzureLogAnalytics.WorkspaceID); err != nil {
			return err
		}
		if cfg.Outputs.AzureLogAnalytics.Spill.Enabled {
			if err := ensureWritableDir(cfg.Outputs.AzureLogAnalytics.Spill.Directory); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkAzureEndpoint(ctx context.Context, workspaceID string) error {
	ws := strings.TrimSpace(workspaceID)
	if ws == "" {
		return fmt.Errorf("outputs.azure_log_analytics.workspace_id required when enabled")
	}
	host := fmt.Sprintf("%s.ods.opinsights.azure.com:443", ws)
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("azure log analytics connectivity (%s) failed: %w", host, err)
	}
	_ = conn.Close()
	return nil
}

func ensureWritableDir(dir string) error {
	path := strings.TrimSpace(dir)
	if path == "" {
		return fmt.Errorf("spill directory not configured")
	}
	if err := os.MkdirAll(path, 0o750); err != nil {
		return fmt.Errorf("create spill directory %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(path, ".probe-*")
	if err != nil {
		return fmt.Errorf("write probe file in %s: %w", path, err)
	}
	tmp.Close()
	os.Remove(tmp.Name())
	// ensure we can resolve absolute path for metrics/logging
	_, err = filepath.Abs(path)
	return err
}
