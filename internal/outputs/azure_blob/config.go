package azure_blob

import (
	"fmt"
	"time"
)

// AuthType specifies the authentication method for Azure Storage
type AuthType string

const (
	AuthTypeSAS             AuthType = "sas"             // SAS token
	AuthTypeAzureAD         AuthType = "azure_ad"        // Azure AD Service Principal
	AuthTypeManagedIdentity AuthType = "managed_identity" // Managed Identity
)

// WriteMode specifies how data is written to blobs
type WriteMode string

const (
	WriteModeAppend WriteMode = "append" // Append blob (streaming)
	WriteModeBlock  WriteMode = "block"  // Block blob (batch)
)

// Config holds the configuration for Azure Blob Storage output
type Config struct {
	// Connection settings
	StorageAccount string   `json:"storage_account" yaml:"storage_account"` // Storage account name
	Container      string   `json:"container" yaml:"container"`             // Container name
	AuthType       AuthType `json:"auth_type" yaml:"auth_type"`             // Authentication type

	// Authentication - SAS Token
	SASToken string `json:"sas_token,omitempty" yaml:"sas_token,omitempty"` // SAS token (if auth_type=sas)

	// Authentication - Azure AD
	TenantID     string `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty"`         // Azure AD tenant ID
	ClientID     string `json:"client_id,omitempty" yaml:"client_id,omitempty"`         // Service principal client ID
	ClientSecret string `json:"client_secret,omitempty" yaml:"client_secret,omitempty"` // Service principal secret

	// Write settings
	WriteMode       WriteMode `json:"write_mode" yaml:"write_mode"`                 // append or block
	PathTemplate    string    `json:"path_template" yaml:"path_template"`           // Path template with variables
	MaxBatchSize    int       `json:"max_batch_size" yaml:"max_batch_size"`         // Max events per batch (block mode)
	MaxBatchBytes   int64     `json:"max_batch_bytes" yaml:"max_batch_bytes"`       // Max bytes per batch (block mode)
	FlushInterval   string    `json:"flush_interval" yaml:"flush_interval"`         // Flush interval (block mode)
	CompressionType string    `json:"compression_type" yaml:"compression_type"`     // none, gzip, zstd
	Format          string    `json:"format" yaml:"format"`                         // json, jsonl, csv, raw

	// Encryption
	EncryptionEnabled bool   `json:"encryption_enabled" yaml:"encryption_enabled"` // Enable encryption at rest
	CustomerManagedKey string `json:"customer_managed_key,omitempty" yaml:"customer_managed_key,omitempty"` // CMK/KMS key URL

	// Lifecycle management
	LifecyclePolicy LifecyclePolicy `json:"lifecycle_policy" yaml:"lifecycle_policy"`

	// Resilience
	RetryAttempts     int    `json:"retry_attempts" yaml:"retry_attempts"`           // Max retry attempts for transient errors
	RetryBackoff      string `json:"retry_backoff" yaml:"retry_backoff"`             // Backoff duration between retries
	LocalBufferPath   string `json:"local_buffer_path" yaml:"local_buffer_path"`     // Local buffer path for failover
	LocalBufferSize   int64  `json:"local_buffer_size" yaml:"local_buffer_size"`     // Max local buffer size
	DeadLetterEnabled bool   `json:"dead_letter_enabled" yaml:"dead_letter_enabled"` // Enable dead letter queue
	DeadLetterPath    string `json:"dead_letter_path" yaml:"dead_letter_path"`       // Dead letter path template

	// Network
	UsePrivateEndpoint bool   `json:"use_private_endpoint" yaml:"use_private_endpoint"` // Use private endpoint
	PrivateEndpointURL string `json:"private_endpoint_url,omitempty" yaml:"private_endpoint_url,omitempty"` // Private endpoint URL
	Region             string `json:"region" yaml:"region"`                             // Azure region
}

// LifecyclePolicy defines retention and transition rules
type LifecyclePolicy struct {
	Enabled                bool   `json:"enabled" yaml:"enabled"`
	ProcessedRetentionDays int    `json:"processed_retention_days" yaml:"processed_retention_days"` // Days to keep processed events
	ErrorRetentionDays     int    `json:"error_retention_days" yaml:"error_retention_days"`         // Days to keep error events
	FailedRetentionDays    int    `json:"failed_retention_days" yaml:"failed_retention_days"`       // Days to keep failed events
	TransitionToCoolDays   int    `json:"transition_to_cool_days" yaml:"transition_to_cool_days"`   // Days before moving to cool tier
	TransitionToArchiveDays int   `json:"transition_to_archive_days" yaml:"transition_to_archive_days"` // Days before archiving
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.StorageAccount == "" {
		return fmt.Errorf("storage_account is required")
	}
	if c.Container == "" {
		return fmt.Errorf("container is required")
	}

	// Validate auth type
	switch c.AuthType {
	case AuthTypeSAS:
		if c.SASToken == "" {
			return fmt.Errorf("sas_token is required when auth_type is 'sas'")
		}
	case AuthTypeAzureAD:
		if c.TenantID == "" || c.ClientID == "" || c.ClientSecret == "" {
			return fmt.Errorf("tenant_id, client_id, and client_secret are required when auth_type is 'azure_ad'")
		}
	case AuthTypeManagedIdentity:
		// No additional validation needed
	default:
		return fmt.Errorf("invalid auth_type: %s (must be sas, azure_ad, or managed_identity)", c.AuthType)
	}

	// Validate write mode
	switch c.WriteMode {
	case WriteModeAppend, WriteModeBlock:
	default:
		return fmt.Errorf("invalid write_mode: %s (must be append or block)", c.WriteMode)
	}

	// Validate path template
	if c.PathTemplate == "" {
		c.PathTemplate = "logs/{date}/{hour}/{source}.log"
	}

	// Validate batch settings for block mode
	if c.WriteMode == WriteModeBlock {
		if c.MaxBatchSize <= 0 {
			c.MaxBatchSize = 1000
		}
		if c.MaxBatchBytes <= 0 {
			c.MaxBatchBytes = 10 * 1024 * 1024 // 10MB
		}
		if c.FlushInterval == "" {
			c.FlushInterval = "30s"
		}
	}

	// Validate compression
	switch c.CompressionType {
	case "", "none", "gzip", "zstd":
	default:
		return fmt.Errorf("invalid compression_type: %s (must be none, gzip, or zstd)", c.CompressionType)
	}

	// Validate format
	switch c.Format {
	case "", "json", "jsonl", "csv", "raw":
		if c.Format == "" {
			c.Format = "jsonl"
		}
	default:
		return fmt.Errorf("invalid format: %s (must be json, jsonl, csv, or raw)", c.Format)
	}

	// Validate retry settings
	if c.RetryAttempts == 0 {
		c.RetryAttempts = 3
	}
	if c.RetryBackoff == "" {
		c.RetryBackoff = "1s"
	}

	return nil
}

// FlushIntervalDuration returns the flush interval as a time.Duration
func (c *Config) FlushIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(c.FlushInterval)
}

// RetryBackoffDuration returns the retry backoff as a time.Duration
func (c *Config) RetryBackoffDuration() (time.Duration, error) {
	return time.ParseDuration(c.RetryBackoff)
}
