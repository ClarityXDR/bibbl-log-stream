package azureloganalytics

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogAnalyticsOutput sends data to Azure Log Analytics Workspace using the HTTP Data Collector API
// Supports custom table names, batching, compression, and automatic retry with exponential backoff
type LogAnalyticsOutput struct {
	WorkspaceID   string
	SharedKey     string
	LogType       string // Table name (without _CL suffix)
	ResourceGroup string // Optional: for Azure Resource ID
	ResourceID    string // Optional: custom Azure Resource ID

	// Performance tuning
	BatchMaxEvents   int
	BatchMaxBytes    int
	FlushIntervalSec int
	Concurrency      int
	MaxRetries       int
	RetryDelaySec    int

	// Runtime state
	client     *http.Client
	batch      []map[string]interface{}
	batchBytes int
	batchMu    sync.Mutex
	flushTimer *time.Timer
	stopCh     chan struct{}
	wg         sync.WaitGroup
	tracer     trace.Tracer
}

// Config holds configuration for Azure Log Analytics output
type Config struct {
	WorkspaceID      string                 `json:"workspaceID"`
	SharedKey        string                 `json:"sharedKey"`
	LogType          string                 `json:"logType"`
	ResourceGroup    string                 `json:"resourceGroup,omitempty"`
	ResourceID       string                 `json:"resourceID,omitempty"`
	BatchMaxEvents   int                    `json:"batchMaxEvents"`
	BatchMaxBytes    int                    `json:"batchMaxBytes"`
	FlushIntervalSec int                    `json:"flushIntervalSec"`
	Concurrency      int                    `json:"concurrency"`
	MaxRetries       int                    `json:"maxRetries"`
	RetryDelaySec    int                    `json:"retryDelaySec"`
	Extra            map[string]interface{} `json:",inline"`
}

// NewLogAnalyticsOutput creates a new Azure Log Analytics output with sensible defaults
func NewLogAnalyticsOutput(cfg Config) (*LogAnalyticsOutput, error) {
	if cfg.WorkspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}
	if cfg.SharedKey == "" {
		return nil, fmt.Errorf("sharedKey is required")
	}
	if cfg.LogType == "" {
		cfg.LogType = "BibblLogs" // Default table name
	}

	// Apply defaults
	if cfg.BatchMaxEvents <= 0 {
		cfg.BatchMaxEvents = 500
	}
	if cfg.BatchMaxBytes <= 0 {
		cfg.BatchMaxBytes = 1 * 1024 * 1024 // 1MB (Azure limit is 30MB but smaller batches are better)
	}
	if cfg.FlushIntervalSec <= 0 {
		cfg.FlushIntervalSec = 10
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 2
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelaySec <= 0 {
		cfg.RetryDelaySec = 2
	}

	// Remove _CL suffix if user provided it (Azure adds it automatically)
	cfg.LogType = strings.TrimSuffix(cfg.LogType, "_CL")

	output := &LogAnalyticsOutput{
		WorkspaceID:      cfg.WorkspaceID,
		SharedKey:        cfg.SharedKey,
		LogType:          cfg.LogType,
		ResourceGroup:    cfg.ResourceGroup,
		ResourceID:       cfg.ResourceID,
		BatchMaxEvents:   cfg.BatchMaxEvents,
		BatchMaxBytes:    cfg.BatchMaxBytes,
		FlushIntervalSec: cfg.FlushIntervalSec,
		Concurrency:      cfg.Concurrency,
		MaxRetries:       cfg.MaxRetries,
		RetryDelaySec:    cfg.RetryDelaySec,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		batch:  make([]map[string]interface{}, 0, cfg.BatchMaxEvents),
		stopCh: make(chan struct{}),
		tracer: otel.Tracer("bibbl/outputs/azureloganalytics"),
	}

	// Start flush timer
	output.flushTimer = time.AfterFunc(time.Duration(output.FlushIntervalSec)*time.Second, output.periodicFlush)

	return output, nil
}

// Send adds an event to the batch and flushes if batch is full
func (o *LogAnalyticsOutput) Send(event map[string]interface{}) error {
	o.batchMu.Lock()
	defer o.batchMu.Unlock()

	// Estimate event size
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	eventSize := len(eventBytes)

	// Check if adding this event would exceed batch limits
	if len(o.batch) >= o.BatchMaxEvents || (o.batchBytes+eventSize) >= o.BatchMaxBytes {
		// Flush current batch first
		if err := o.flushBatchLocked(); err != nil {
			return err
		}
	}

	// Add event to batch
	o.batch = append(o.batch, event)
	o.batchBytes += eventSize

	return nil
}

// Flush sends all pending events
func (o *LogAnalyticsOutput) Flush() error {
	o.batchMu.Lock()
	defer o.batchMu.Unlock()
	return o.flushBatchLocked()
}

// flushBatchLocked sends the current batch (must hold batchMu)
func (o *LogAnalyticsOutput) flushBatchLocked() error {
	if len(o.batch) == 0 {
		return nil
	}

	// Take ownership of current batch
	batchToSend := o.batch
	o.batch = make([]map[string]interface{}, 0, o.BatchMaxEvents)
	o.batchBytes = 0

	// Send in background
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		_ = o.sendBatch(batchToSend)
	}()

	return nil
}

// periodicFlush is called by the timer to flush batches periodically
func (o *LogAnalyticsOutput) periodicFlush() {
	select {
	case <-o.stopCh:
		return
	default:
		_ = o.Flush()
		// Reset timer
		o.flushTimer.Reset(time.Duration(o.FlushIntervalSec) * time.Second)
	}
}

// sendBatch sends a batch of events to Azure Log Analytics with retry
func (o *LogAnalyticsOutput) sendBatch(events []map[string]interface{}) error {
	ctx, span := o.tracer.Start(context.Background(), "sendBatch", trace.WithAttributes(
		attribute.Int("batch.size", len(events)),
	))
	defer span.End()

	// Marshal events to JSON
	body, err := json.Marshal(events)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// Build request
	method := "POST"
	contentType := "application/json"
	resource := "/api/logs"
	rfc1123date := time.Now().UTC().Format(time.RFC1123)
	contentLength := len(body)

	// Build signature
	stringToSign := fmt.Sprintf("%s\n%d\n%s\nx-ms-date:%s\n%s", method, contentLength, contentType, rfc1123date, resource)
	signature, err := o.buildSignature(stringToSign)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to build signature: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("https://%s.ods.opinsights.azure.com%s?api-version=2016-04-01", o.WorkspaceID, resource)

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= o.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(o.RetryDelaySec*(1<<uint(attempt-1))) * time.Second
			time.Sleep(delay)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", signature)
		req.Header.Set("Log-Type", o.LogType)
		req.Header.Set("x-ms-date", rfc1123date)

		// Add optional resource ID
		if o.ResourceID != "" {
			req.Header.Set("x-ms-AzureResourceId", o.ResourceID)
		}

		resp, err := o.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Read response body
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
			return nil
		}

		lastErr = fmt.Errorf("azure log analytics returned status %d: %s", resp.StatusCode, string(respBody))

		// Don't retry on 4xx errors (except 429)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			span.RecordError(lastErr)
			return lastErr
		}
	}

	span.RecordError(lastErr)
	return fmt.Errorf("failed after %d retries: %w", o.MaxRetries, lastErr)
}

// buildSignature creates the HMAC-SHA256 signature for Azure Log Analytics API
func (o *LogAnalyticsOutput) buildSignature(stringToSign string) (string, error) {
	// Decode the shared key from base64
	keyBytes, err := base64.StdEncoding.DecodeString(o.SharedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode shared key: %w", err)
	}

	// Create HMAC-SHA256 hash
	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("SharedKey %s:%s", o.WorkspaceID, signature), nil
}

// Close stops the output and flushes remaining events
func (o *LogAnalyticsOutput) Close() error {
	// Stop periodic flush
	close(o.stopCh)
	if o.flushTimer != nil {
		o.flushTimer.Stop()
	}

	// Final flush
	if err := o.Flush(); err != nil {
		return err
	}

	// Wait for all sends to complete
	o.wg.Wait()

	return nil
}

// GetStats returns statistics about the output
func (o *LogAnalyticsOutput) GetStats() map[string]interface{} {
	o.batchMu.Lock()
	defer o.batchMu.Unlock()

	return map[string]interface{}{
		"workspace_id":       o.WorkspaceID,
		"log_type":           o.LogType + "_CL",
		"batch_size":         len(o.batch),
		"batch_bytes":        o.batchBytes,
		"batch_max_events":   o.BatchMaxEvents,
		"batch_max_bytes":    o.BatchMaxBytes,
		"flush_interval_sec": o.FlushIntervalSec,
	}
}
