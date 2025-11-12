package azure_blob

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"go.uber.org/zap"
)

// Output represents an Azure Blob Storage output destination
type Output struct {
	config       *Config
	client       *azblob.Client
	containerClient *container.Client
	logger       *zap.Logger
	
	// Batch processing (for block mode)
	batchMu      sync.Mutex
	batch        [][]byte
	batchSize    int
	batchBytes   int64
	flushTimer   *time.Timer
	
	// Local buffer for failover
	localBuffer  *LocalBuffer
	
	// Metrics
	mu           sync.RWMutex
	metrics      Metrics
	
	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	started      bool
}

// Metrics holds operational metrics
type Metrics struct {
	EventsSent        int64
	EventsFailed      int64
	BytesSent         int64
	RetryAttempts     int64
	LocalBufferWrites int64
	DeadLetterWrites  int64
	LastError         string
	LastErrorTime     time.Time
}

// Event represents a log event to be written
type Event struct {
	Timestamp time.Time
	Source    string
	Data      map[string]interface{}
	Raw       []byte
}

// NewOutput creates a new Azure Blob Storage output
func NewOutput(config *Config, logger *zap.Logger) (*Output, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	o := &Output{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize Azure client
	if err := o.initAzureClient(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize Azure client: %w", err)
	}

	// Initialize local buffer if configured
	if config.LocalBufferPath != "" {
		localBuffer, err := NewLocalBuffer(config.LocalBufferPath, config.LocalBufferSize)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize local buffer: %w", err)
		}
		o.localBuffer = localBuffer
	}

	return o, nil
}

// initAzureClient initializes the Azure Blob Storage client
func (o *Output) initAzureClient() error {
	var credential azcore.TokenCredential
	var err error

	switch o.config.AuthType {
	case AuthTypeSAS:
		// Use SAS token authentication
		accountURL := fmt.Sprintf("https://%s.blob.core.windows.net/?%s", o.config.StorageAccount, o.config.SASToken)
		if o.config.UsePrivateEndpoint && o.config.PrivateEndpointURL != "" {
			accountURL = fmt.Sprintf("%s/?%s", o.config.PrivateEndpointURL, o.config.SASToken)
		}
		o.client, err = azblob.NewClientWithNoCredential(accountURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create SAS client: %w", err)
		}

	case AuthTypeAzureAD:
		// Use Azure AD Service Principal
		credential, err = azidentity.NewClientSecretCredential(
			o.config.TenantID,
			o.config.ClientID,
			o.config.ClientSecret,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to create Azure AD credential: %w", err)
		}

	case AuthTypeManagedIdentity:
		// Use Managed Identity
		credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return fmt.Errorf("failed to create managed identity credential: %w", err)
		}

	default:
		return fmt.Errorf("unsupported auth type: %s", o.config.AuthType)
	}

	// Create client with credential (if not SAS)
	if credential != nil {
		accountURL := fmt.Sprintf("https://%s.blob.core.windows.net/", o.config.StorageAccount)
		if o.config.UsePrivateEndpoint && o.config.PrivateEndpointURL != "" {
			accountURL = o.config.PrivateEndpointURL
		}
		o.client, err = azblob.NewClient(accountURL, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}
	}

	// Get container client
	o.containerClient = o.client.ServiceClient().NewContainerClient(o.config.Container)

	// Ensure container exists
	if err := o.ensureContainer(); err != nil {
		return fmt.Errorf("failed to ensure container: %w", err)
	}

	return nil
}

// ensureContainer creates the container if it doesn't exist
func (o *Output) ensureContainer() error {
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	_, err := o.containerClient.Create(ctx, nil)
	if err != nil {
		// Check if container already exists
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 409 {
			// Container already exists, this is fine
			return nil
		}
		return fmt.Errorf("failed to create container: %w", err)
	}

	o.logger.Info("container created", zap.String("container", o.config.Container))
	return nil
}

// Start starts the output
func (o *Output) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return fmt.Errorf("output already started")
	}

	// Start flush timer for block mode
	if o.config.WriteMode == WriteModeBlock {
		flushInterval, err := o.config.FlushIntervalDuration()
		if err != nil {
			return fmt.Errorf("invalid flush interval: %w", err)
		}
		o.flushTimer = time.AfterFunc(flushInterval, o.flushBatch)
	}

	// Start local buffer recovery if enabled
	if o.localBuffer != nil {
		o.wg.Add(1)
		go o.recoverLocalBuffer()
	}

	o.started = true
	o.logger.Info("Azure Blob output started",
		zap.String("storage_account", o.config.StorageAccount),
		zap.String("container", o.config.Container),
		zap.String("write_mode", string(o.config.WriteMode)))

	return nil
}

// Stop stops the output
func (o *Output) Stop() error {
	o.mu.Lock()
	if !o.started {
		o.mu.Unlock()
		return nil
	}
	o.started = false
	o.mu.Unlock()

	// Stop flush timer
	if o.flushTimer != nil {
		o.flushTimer.Stop()
	}

	// Flush any remaining batch
	if o.config.WriteMode == WriteModeBlock {
		o.flushBatch()
	}

	// Cancel context and wait for goroutines
	o.cancel()
	o.wg.Wait()

	// Close local buffer
	if o.localBuffer != nil {
		return o.localBuffer.Close()
	}

	o.logger.Info("Azure Blob output stopped")
	return nil
}

// Write writes an event to Azure Blob Storage
func (o *Output) Write(event *Event) error {
	o.mu.RLock()
	if !o.started {
		o.mu.RUnlock()
		return fmt.Errorf("output not started")
	}
	o.mu.RUnlock()

	// Format event
	data, err := o.formatEvent(event)
	if err != nil {
		return fmt.Errorf("failed to format event: %w", err)
	}

	// Choose write path based on mode
	switch o.config.WriteMode {
	case WriteModeAppend:
		return o.writeAppend(event, data)
	case WriteModeBlock:
		return o.writeBatch(data)
	default:
		return fmt.Errorf("unsupported write mode: %s", o.config.WriteMode)
	}
}

// formatEvent formats the event according to the configured format
func (o *Output) formatEvent(event *Event) ([]byte, error) {
	switch o.config.Format {
	case "json":
		return json.Marshal(event.Data)
	case "jsonl":
		data, err := json.Marshal(event.Data)
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case "raw":
		return event.Raw, nil
	case "csv":
		// Simple CSV implementation (would need enhancement for production)
		var buf bytes.Buffer
		for k, v := range event.Data {
			fmt.Fprintf(&buf, "%s=%v,", k, v)
		}
		data := buf.Bytes()
		if len(data) > 0 {
			data = data[:len(data)-1] // Remove trailing comma
		}
		return append(data, '\n'), nil
	default:
		return event.Raw, nil
	}
}

// writeAppend writes data using append blob (streaming)
func (o *Output) writeAppend(event *Event, data []byte) error {
	// Generate blob path
	blobPath, err := o.generateBlobPath(event)
	if err != nil {
		return fmt.Errorf("failed to generate blob path: %w", err)
	}

	// Compress if needed
	if o.config.CompressionType == "gzip" {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(data); err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		if err := gz.Close(); err != nil {
			return fmt.Errorf("failed to close compressor: %w", err)
		}
		data = buf.Bytes()
		blobPath += ".gz"
	}

	// Write with retry
	return o.writeWithRetry(func() error {
		return o.appendToBlob(blobPath, data)
	})
}

// appendToBlob appends data to an append blob
func (o *Output) appendToBlob(blobPath string, data []byte) error {
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	appendBlobClient := o.containerClient.NewAppendBlobClient(blobPath)

	// Check if blob exists, create if not
	_, err := appendBlobClient.GetProperties(ctx, nil)
	if err != nil {
		// Blob doesn't exist, create it
		_, err = appendBlobClient.Create(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create append blob: %w", err)
		}
	}

	// Append data
	_, err = appendBlobClient.AppendBlock(ctx, streaming.NopCloser(bytes.NewReader(data)), nil)
	if err != nil {
		return fmt.Errorf("failed to append block: %w", err)
	}

	o.mu.Lock()
	o.metrics.EventsSent++
	o.metrics.BytesSent += int64(len(data))
	o.mu.Unlock()

	return nil
}

// writeBatch adds data to the batch
func (o *Output) writeBatch(data []byte) error {
	o.batchMu.Lock()
	defer o.batchMu.Unlock()

	o.batch = append(o.batch, data)
	o.batchSize++
	o.batchBytes += int64(len(data))

	// Check if batch is full
	if o.batchSize >= o.config.MaxBatchSize || o.batchBytes >= o.config.MaxBatchBytes {
		go o.flushBatch()
	}

	return nil
}

// flushBatch flushes the current batch to Azure
func (o *Output) flushBatch() {
	o.batchMu.Lock()
	if len(o.batch) == 0 {
		o.batchMu.Unlock()
		return
	}

	batch := o.batch
	o.batch = nil
	o.batchSize = 0
	o.batchBytes = 0
	o.batchMu.Unlock()

	// Combine batch data
	var buf bytes.Buffer
	for _, data := range batch {
		buf.Write(data)
	}
	data := buf.Bytes()

	// Generate blob path
	blobPath := o.generateBatchBlobPath()

	// Compress if needed
	if o.config.CompressionType == "gzip" {
		var cbuf bytes.Buffer
		gz := gzip.NewWriter(&cbuf)
		if _, err := gz.Write(data); err != nil {
			o.recordError(fmt.Errorf("failed to compress batch: %w", err))
			return
		}
		if err := gz.Close(); err != nil {
			o.recordError(fmt.Errorf("failed to close compressor: %w", err))
			return
		}
		data = cbuf.Bytes()
		blobPath += ".gz"
	}

	// Write with retry
	err := o.writeWithRetry(func() error {
		return o.uploadBlockBlob(blobPath, data)
	})

	if err != nil {
		o.recordError(err)
		// Try local buffer
		if o.localBuffer != nil {
			if err := o.localBuffer.Write(data); err != nil {
				o.logger.Error("failed to write to local buffer", zap.Error(err))
			} else {
				o.mu.Lock()
				o.metrics.LocalBufferWrites++
				o.mu.Unlock()
			}
		}
	}

	// Reset timer
	if o.flushTimer != nil {
		flushInterval, _ := o.config.FlushIntervalDuration()
		o.flushTimer.Reset(flushInterval)
	}
}

// uploadBlockBlob uploads data as a block blob
func (o *Output) uploadBlockBlob(blobPath string, data []byte) error {
	ctx, cancel := context.WithTimeout(o.ctx, 60*time.Second)
	defer cancel()

	blockBlobClient := o.containerClient.NewBlockBlobClient(blobPath)

	_, err := blockBlobClient.UploadBuffer(ctx, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upload block blob: %w", err)
	}

	o.mu.Lock()
	o.metrics.EventsSent += int64(len(o.batch))
	o.metrics.BytesSent += int64(len(data))
	o.mu.Unlock()

	return nil
}

// writeWithRetry writes data with retry logic
func (o *Output) writeWithRetry(fn func() error) error {
	backoff, _ := o.config.RetryBackoffDuration()
	
	var lastErr error
	for attempt := 0; attempt <= o.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			o.mu.Lock()
			o.metrics.RetryAttempts++
			o.mu.Unlock()
			time.Sleep(backoff * time.Duration(attempt))
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is transient
		if !o.isTransientError(err) {
			break
		}

		o.logger.Warn("transient error, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	o.recordError(lastErr)
	return lastErr
}

// isTransientError checks if an error is transient and should be retried
func (o *Output) isTransientError(err error) bool {
	if err == nil {
		return false
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		// Retry on 5xx errors and specific 4xx errors
		switch respErr.StatusCode {
		case 408, 429, 500, 502, 503, 504:
			return true
		}
	}

	// Check for network errors
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "temporary")
}

// generateBlobPath generates a blob path from the template
func (o *Output) generateBlobPath(event *Event) (string, error) {
	path := o.config.PathTemplate

	// Replace template variables
	replacer := strings.NewReplacer(
		"{date}", event.Timestamp.Format("2006-01-02"),
		"{year}", event.Timestamp.Format("2006"),
		"{month}", event.Timestamp.Format("01"),
		"{day}", event.Timestamp.Format("02"),
		"{hour}", event.Timestamp.Format("15"),
		"{minute}", event.Timestamp.Format("04"),
		"{source}", event.Source,
	)

	return replacer.Replace(path), nil
}

// generateBatchBlobPath generates a blob path for batch uploads
func (o *Output) generateBatchBlobPath() string {
	now := time.Now()
	path := o.config.PathTemplate

	replacer := strings.NewReplacer(
		"{date}", now.Format("2006-01-02"),
		"{year}", now.Format("2006"),
		"{month}", now.Format("01"),
		"{day}", now.Format("02"),
		"{hour}", now.Format("15"),
		"{minute}", now.Format("04"),
		"{source}", "batch",
	)

	basePath := replacer.Replace(path)
	
	// Add timestamp to make unique
	return fmt.Sprintf("%s-%d", basePath, now.UnixNano())
}

// recoverLocalBuffer recovers data from local buffer
func (o *Output) recoverLocalBuffer() {
	defer o.wg.Done()

	if o.localBuffer == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.tryRecoverLocalBuffer()
		}
	}
}

// tryRecoverLocalBuffer attempts to recover data from local buffer
func (o *Output) tryRecoverLocalBuffer() {
	if o.localBuffer == nil {
		return
	}

	// Try to read and upload buffered data
	for {
		data, err := o.localBuffer.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			o.logger.Error("failed to read from local buffer", zap.Error(err))
			break
		}

		// Generate blob path for recovered data
		blobPath := fmt.Sprintf("recovered/%s-%d", time.Now().Format("2006-01-02"), time.Now().UnixNano())

		// Try to upload
		err = o.writeWithRetry(func() error {
			return o.uploadBlockBlob(blobPath, data)
		})

		if err != nil {
			o.logger.Error("failed to recover data from local buffer", zap.Error(err))
			// Put it back in the buffer
			if err := o.localBuffer.Write(data); err != nil {
				o.logger.Error("failed to write back to local buffer", zap.Error(err))
			}
			break
		}

		o.logger.Info("recovered data from local buffer", zap.String("blob", blobPath))
	}
}

// recordError records an error in metrics
func (o *Output) recordError(err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.metrics.EventsFailed++
	o.metrics.LastError = err.Error()
	o.metrics.LastErrorTime = time.Now()

	o.logger.Error("write error", zap.Error(err))
}

// GetMetrics returns the current metrics
func (o *Output) GetMetrics() Metrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.metrics
}
