package azure_blob

import (
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SAS config",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeSAS,
				SASToken:       "sp=racwdl&st=2024-01-01T00:00:00Z&se=2025-01-01T00:00:00Z&spr=https&sv=2021-06-08&sr=c&sig=xxxx",
				WriteMode:      WriteModeAppend,
			},
			wantErr: false,
		},
		{
			name: "valid Azure AD config",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeAzureAD,
				TenantID:       "tenant-id",
				ClientID:       "client-id",
				ClientSecret:   "client-secret",
				WriteMode:      WriteModeBlock,
			},
			wantErr: false,
		},
		{
			name: "valid Managed Identity config",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeManagedIdentity,
				WriteMode:      WriteModeAppend,
			},
			wantErr: false,
		},
		{
			name: "missing storage account",
			config: Config{
				Container: "logs",
				AuthType:  AuthTypeSAS,
				SASToken:  "token",
				WriteMode: WriteModeAppend,
			},
			wantErr: true,
			errMsg:  "storage_account is required",
		},
		{
			name: "missing container",
			config: Config{
				StorageAccount: "mystorageaccount",
				AuthType:       AuthTypeSAS,
				SASToken:       "token",
				WriteMode:      WriteModeAppend,
			},
			wantErr: true,
			errMsg:  "container is required",
		},
		{
			name: "SAS without token",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeSAS,
				WriteMode:      WriteModeAppend,
			},
			wantErr: true,
			errMsg:  "sas_token is required",
		},
		{
			name: "Azure AD without credentials",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeAzureAD,
				WriteMode:      WriteModeAppend,
			},
			wantErr: true,
			errMsg:  "tenant_id, client_id, and client_secret are required",
		},
		{
			name: "invalid auth type",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       "invalid",
				WriteMode:      WriteModeAppend,
			},
			wantErr: true,
			errMsg:  "invalid auth_type",
		},
		{
			name: "invalid write mode",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeSAS,
				SASToken:       "token",
				WriteMode:      "invalid",
			},
			wantErr: true,
			errMsg:  "invalid write_mode",
		},
		{
			name: "invalid compression type",
			config: Config{
				StorageAccount:  "mystorageaccount",
				Container:       "logs",
				AuthType:        AuthTypeSAS,
				SASToken:        "token",
				WriteMode:       WriteModeAppend,
				CompressionType: "bzip2",
			},
			wantErr: true,
			errMsg:  "invalid compression_type",
		},
		{
			name: "invalid format",
			config: Config{
				StorageAccount: "mystorageaccount",
				Container:      "logs",
				AuthType:       AuthTypeSAS,
				SASToken:       "token",
				WriteMode:      WriteModeAppend,
				Format:         "xml",
			},
			wantErr: true,
			errMsg:  "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	config := Config{
		StorageAccount: "mystorageaccount",
		Container:      "logs",
		AuthType:       AuthTypeSAS,
		SASToken:       "token",
		WriteMode:      WriteModeBlock,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Check defaults
	if config.PathTemplate == "" {
		t.Error("PathTemplate should have default value")
	}
	if config.MaxBatchSize == 0 {
		t.Error("MaxBatchSize should have default value")
	}
	if config.MaxBatchBytes == 0 {
		t.Error("MaxBatchBytes should have default value")
	}
	if config.FlushInterval == "" {
		t.Error("FlushInterval should have default value")
	}
	if config.Format == "" {
		t.Error("Format should have default value")
	}
	if config.RetryAttempts == 0 {
		t.Error("RetryAttempts should have default value")
	}
	if config.RetryBackoff == "" {
		t.Error("RetryBackoff should have default value")
	}
}

func TestFlushIntervalDuration(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		want     time.Duration
		wantErr  bool
	}{
		{
			name:     "valid duration",
			interval: "30s",
			want:     30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "valid duration minutes",
			interval: "5m",
			want:     5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			interval: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{FlushInterval: tt.interval}
			got, err := config.FlushIntervalDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("FlushIntervalDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("FlushIntervalDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryBackoffDuration(t *testing.T) {
	tests := []struct {
		name    string
		backoff string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "valid duration",
			backoff: "1s",
			want:    1 * time.Second,
			wantErr: false,
		},
		{
			name:    "valid duration milliseconds",
			backoff: "500ms",
			want:    500 * time.Millisecond,
			wantErr: false,
		},
		{
			name:    "invalid duration",
			backoff: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{RetryBackoff: tt.backoff}
			got, err := config.RetryBackoffDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryBackoffDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RetryBackoffDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
