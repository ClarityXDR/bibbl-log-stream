package azure_blob

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// Adapter provides integration with the pipeline engine
type Adapter struct {
	outputs map[string]*Output
	logger  *zap.Logger
}

// NewAdapter creates a new adapter
func NewAdapter(logger *zap.Logger) *Adapter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Adapter{
		outputs: make(map[string]*Output),
		logger:  logger,
	}
}

// CreateOutput creates and starts an output from configuration
func (a *Adapter) CreateOutput(id string, config map[string]interface{}) error {
	// Convert map to Config struct
	cfg, err := configFromMap(config)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create output
	output, err := NewOutput(cfg, a.logger.With(zap.String("output_id", id)))
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}

	// Start output
	if err := output.Start(); err != nil {
		return fmt.Errorf("failed to start output: %w", err)
	}

	a.outputs[id] = output
	return nil
}

// GetOutput returns an output by ID
func (a *Adapter) GetOutput(id string) (*Output, bool) {
	output, ok := a.outputs[id]
	return output, ok
}

// DeleteOutput stops and removes an output
func (a *Adapter) DeleteOutput(id string) error {
	output, ok := a.outputs[id]
	if !ok {
		return fmt.Errorf("output not found: %s", id)
	}

	if err := output.Stop(); err != nil {
		return fmt.Errorf("failed to stop output: %w", err)
	}

	delete(a.outputs, id)
	return nil
}

// WriteEvent writes an event to a specific output
func (a *Adapter) WriteEvent(id string, event *Event) error {
	output, ok := a.outputs[id]
	if !ok {
		return fmt.Errorf("output not found: %s", id)
	}

	return output.Write(event)
}

// GetMetrics returns metrics for an output
func (a *Adapter) GetMetrics(id string) (Metrics, error) {
	output, ok := a.outputs[id]
	if !ok {
		return Metrics{}, fmt.Errorf("output not found: %s", id)
	}

	return output.GetMetrics(), nil
}

// configFromMap converts a map[string]interface{} to Config
func configFromMap(m map[string]interface{}) (*Config, error) {
	// Marshal to JSON then unmarshal to Config
	// This handles type conversions properly
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// ValidateConfig validates a configuration without creating the output
func ValidateConfig(config map[string]interface{}) error {
	cfg, err := configFromMap(config)
	if err != nil {
		return err
	}
	return cfg.Validate()
}

// ConfigDefaults returns a map with default configuration values
func ConfigDefaults() map[string]interface{} {
	return map[string]interface{}{
		"write_mode":       "append",
		"path_template":    "logs/{date}/{hour}/{source}.log",
		"max_batch_size":   1000,
		"max_batch_bytes":  10485760, // 10MB
		"flush_interval":   "30s",
		"compression_type": "none",
		"format":           "jsonl",
		"retry_attempts":   3,
		"retry_backoff":    "1s",
		"encryption_enabled": false,
		"dead_letter_enabled": false,
		"use_private_endpoint": false,
		"lifecycle_policy": map[string]interface{}{
			"enabled":                    false,
			"processed_retention_days":   90,
			"error_retention_days":       180,
			"failed_retention_days":      365,
			"transition_to_cool_days":    30,
			"transition_to_archive_days": 90,
		},
	}
}

// ConfigExample returns an example configuration
func ConfigExample() map[string]interface{} {
	return map[string]interface{}{
		"storage_account": "mystorageaccount",
		"container":       "logs",
		"auth_type":       "managed_identity",
		"write_mode":      "block",
		"path_template":   "logs/{date}/{hour}/{source}.log",
		"max_batch_size":  1000,
		"max_batch_bytes": 10485760,
		"flush_interval":  "30s",
		"compression_type": "gzip",
		"format":          "jsonl",
		"encryption_enabled": true,
		"lifecycle_policy": map[string]interface{}{
			"enabled":                    true,
			"processed_retention_days":   90,
			"error_retention_days":       180,
			"transition_to_cool_days":    30,
			"transition_to_archive_days": 90,
		},
		"retry_attempts": 3,
		"retry_backoff":  "1s",
		"local_buffer_path": "/var/lib/bibbl/buffer/azure_blob.dat",
		"local_buffer_size": 1073741824,
		"dead_letter_enabled": true,
		"dead_letter_path": "failed/{date}/{source}-failed.log",
		"use_private_endpoint": false,
		"region": "eastus",
	}
}
