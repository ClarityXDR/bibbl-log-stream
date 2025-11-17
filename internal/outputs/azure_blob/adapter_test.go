package azure_blob

import (
	"testing"

	"go.uber.org/zap"
)

func TestConfigFromMap(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			input: map[string]interface{}{
				"storage_account": "mystorageaccount",
				"container":       "logs",
				"auth_type":       "sas",
				"sas_token":       "token",
				"write_mode":      "append",
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			input: map[string]interface{}{
				"storage_account": "mystorageaccount",
				"auth_type":       "sas",
				"sas_token":       "token",
				"write_mode":      "append",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := configFromMap(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("configFromMap() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Validate the config
			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid SAS config",
			config: map[string]interface{}{
				"storage_account": "mystorageaccount",
				"container":       "logs",
				"auth_type":       "sas",
				"sas_token":       "token",
				"write_mode":      "append",
			},
			wantErr: false,
		},
		{
			name: "invalid - missing storage account",
			config: map[string]interface{}{
				"container":  "logs",
				"auth_type":  "sas",
				"sas_token":  "token",
				"write_mode": "append",
			},
			wantErr: true,
		},
		{
			name: "invalid - wrong auth type",
			config: map[string]interface{}{
				"storage_account": "mystorageaccount",
				"container":       "logs",
				"auth_type":       "invalid",
				"write_mode":      "append",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdapterConfigDefaults(t *testing.T) {
	defaults := ConfigDefaults()

	// Check that essential defaults are present
	requiredDefaults := []string{
		"write_mode",
		"path_template",
		"max_batch_size",
		"max_batch_bytes",
		"flush_interval",
		"compression_type",
		"format",
		"retry_attempts",
		"retry_backoff",
	}

	for _, key := range requiredDefaults {
		if _, ok := defaults[key]; !ok {
			t.Errorf("ConfigDefaults() missing required key: %s", key)
		}
	}
}

func TestConfigExample(t *testing.T) {
	example := ConfigExample()

	// Validate the example config
	err := ValidateConfig(example)
	if err != nil {
		t.Errorf("ConfigExample() should be valid, got error: %v", err)
	}

	// Check that it has all required fields
	requiredFields := []string{
		"storage_account",
		"container",
		"auth_type",
		"write_mode",
	}

	for _, key := range requiredFields {
		if _, ok := example[key]; !ok {
			t.Errorf("ConfigExample() missing required field: %s", key)
		}
	}
}

func TestAdapter(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewAdapter(logger)

	if adapter == nil {
		t.Fatal("NewAdapter() should not return nil")
	}

	if adapter.outputs == nil {
		t.Error("Adapter.outputs should be initialized")
	}
}

func TestAdapterGetOutputNotFound(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewAdapter(logger)

	_, ok := adapter.GetOutput("nonexistent")
	if ok {
		t.Error("GetOutput() should return false for nonexistent output")
	}
}

func TestAdapterDeleteOutputNotFound(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewAdapter(logger)

	err := adapter.DeleteOutput("nonexistent")
	if err == nil {
		t.Error("DeleteOutput() should return error for nonexistent output")
	}
}

func TestAdapterWriteEventNotFound(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewAdapter(logger)

	event := &Event{
		Source: "test",
		Data:   map[string]interface{}{"message": "test"},
	}

	err := adapter.WriteEvent("nonexistent", event)
	if err == nil {
		t.Error("WriteEvent() should return error for nonexistent output")
	}
}

func TestAdapterGetMetricsNotFound(t *testing.T) {
	logger := zap.NewNop()
	adapter := NewAdapter(logger)

	_, err := adapter.GetMetrics("nonexistent")
	if err == nil {
		t.Error("GetMetrics() should return error for nonexistent output")
	}
}
