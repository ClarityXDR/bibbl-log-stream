package config

import "testing"

func TestValidateSplit(t *testing.T) {
    cfg := &Config{}
    cfg.Server.Port = 9444
    cfg.Server.MaxRequestBytes = 1024
    cfg.Logging.Level = "info"
    cfg.Logging.Format = "text"
    errs, warns := cfg.Validate()
    if len(errs) != 0 { t.Fatalf("expected no errors got %v", errs) }
    if len(warns) == 0 { t.Fatalf("expected warnings for missing auth") }
}
