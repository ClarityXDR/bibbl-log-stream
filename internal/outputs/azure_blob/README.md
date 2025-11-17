# Azure Blob Storage Output

Enterprise-grade Azure Blob Storage output destination for Bibbl Log Stream with Cribl parity and Microsoft compliance.

## Features

### Core Capabilities
- **Multiple Write Modes**: Append blob (streaming) and block blob (batch)
- **Flexible Authentication**: SAS token, Azure AD Service Principal, and Managed Identity
- **Path Templates**: Dynamic blob paths with date/time and source variables
- **Compression**: Optional gzip or zstd compression
- **Multiple Formats**: JSON, JSONL, CSV, and raw output

### Enterprise Features
- **Encryption at Rest**: Built-in support with customer-managed keys (CMK/KMS)
- **Lifecycle Management**: Configurable retention and transition policies
- **Dead Letter Queue**: Automatic handling of failed events
- **Event Replay**: Replay capability for error recovery
- **Local Buffer Failover**: Automatic local buffering when Azure is unavailable
- **Auto-Retry**: Exponential backoff for transient errors
- **Regional Support**: Multi-region deployment with private endpoint support

## Configuration

### Basic Configuration

```yaml
outputs:
  azure_blob:
    storage_account: "mystorageaccount"
    container: "logs"
    auth_type: "sas"  # or "azure_ad" or "managed_identity"
    write_mode: "append"  # or "block"
```

### Authentication Methods

#### 1. SAS Token (Simplest)
```yaml
auth_type: "sas"
sas_token: "sp=racwdl&st=2024-01-01T00:00:00Z&se=2025-01-01T00:00:00Z&spr=https&sv=2021-06-08&sr=c&sig=xxxx"
```

#### 2. Azure AD Service Principal
```yaml
auth_type: "azure_ad"
tenant_id: "your-tenant-id"
client_id: "your-client-id"
client_secret: "your-client-secret"
```

#### 3. Managed Identity (Recommended for Azure VMs)
```yaml
auth_type: "managed_identity"
```

### Write Modes

#### Append Blob (Streaming)
Best for continuous log streaming with low latency.

```yaml
write_mode: "append"
path_template: "logs/{date}/{hour}/{source}.log"
compression_type: "gzip"
format: "jsonl"
```

#### Block Blob (Batch)
Best for high-volume ingestion with batch optimization.

```yaml
write_mode: "block"
max_batch_size: 1000       # events per batch
max_batch_bytes: 10485760  # 10MB
flush_interval: "30s"
compression_type: "gzip"
format: "jsonl"
```

### Path Templates

Dynamic path generation using template variables:

- `{date}`: Current date (YYYY-MM-DD)
- `{year}`: Current year (YYYY)
- `{month}`: Current month (MM)
- `{day}`: Current day (DD)
- `{hour}`: Current hour (HH)
- `{minute}`: Current minute (MM)
- `{source}`: Event source name

**Examples:**
```yaml
# Daily logs by source
path_template: "logs/{date}/{source}.log"

# Hourly partitioning
path_template: "logs/{year}/{month}/{day}/{hour}/{source}.log"

# Source-based organization
path_template: "{source}/{date}/events.log"
```

### Encryption

#### Server-Side Encryption (Default)
Azure Storage automatically encrypts all data at rest.

#### Customer-Managed Keys (CMK)
```yaml
encryption_enabled: true
customer_managed_key: "https://myvault.vault.azure.net/keys/mykey/version"
```

### Lifecycle Management

```yaml
lifecycle_policy:
  enabled: true
  processed_retention_days: 90    # Keep processed events for 90 days
  error_retention_days: 180       # Keep error events for 180 days
  failed_retention_days: 365      # Keep failed events for 1 year
  transition_to_cool_days: 30     # Move to cool tier after 30 days
  transition_to_archive_days: 90  # Move to archive tier after 90 days
```

### Resilience and Failover

#### Local Buffer
Automatically buffer data locally when Azure is unavailable:

```yaml
local_buffer_path: "/var/lib/bibbl/buffer/azure_blob.dat"
local_buffer_size: 1073741824  # 1GB
```

The local buffer will:
- Store events when Azure is down
- Automatically replay when connection is restored
- Prevent data loss during outages

#### Retry Configuration
```yaml
retry_attempts: 3      # Maximum retry attempts
retry_backoff: "1s"    # Initial backoff duration (exponential)
```

#### Dead Letter Queue
Handle persistently failing events:

```yaml
dead_letter_enabled: true
dead_letter_path: "failed/{date}/{source}-failed.log"
```

### Private Endpoint Support

For secure, private connectivity to Azure Storage:

```yaml
use_private_endpoint: true
private_endpoint_url: "https://mystorageaccount.privatelink.blob.core.windows.net"
region: "eastus"
```

## Complete Configuration Example

```yaml
outputs:
  azure_blob:
    # Connection
    storage_account: "mystorageaccount"
    container: "logs"
    region: "eastus"
    
    # Authentication
    auth_type: "managed_identity"
    
    # Write settings
    write_mode: "block"
    path_template: "logs/{date}/{hour}/{source}.log"
    max_batch_size: 1000
    max_batch_bytes: 10485760
    flush_interval: "30s"
    compression_type: "gzip"
    format: "jsonl"
    
    # Encryption
    encryption_enabled: true
    customer_managed_key: "https://myvault.vault.azure.net/keys/logkey/v1"
    
    # Lifecycle
    lifecycle_policy:
      enabled: true
      processed_retention_days: 90
      error_retention_days: 180
      transition_to_cool_days: 30
      transition_to_archive_days: 90
    
    # Resilience
    retry_attempts: 3
    retry_backoff: "1s"
    local_buffer_path: "/var/lib/bibbl/buffer/azure_blob.dat"
    local_buffer_size: 1073741824
    dead_letter_enabled: true
    dead_letter_path: "failed/{date}/{source}-failed.log"
    
    # Network
    use_private_endpoint: true
    private_endpoint_url: "https://mystorageaccount.privatelink.blob.core.windows.net"
```

## Azure Setup

### 1. Create Storage Account

```bash
# Create resource group
az group create --name bibbl-logs --location eastus

# Create storage account
az storage account create \
  --name mystorageaccount \
  --resource-group bibbl-logs \
  --location eastus \
  --sku Standard_LRS \
  --encryption-services blob \
  --min-tls-version TLS1_2
```

### 2. Create Container

```bash
az storage container create \
  --name logs \
  --account-name mystorageaccount \
  --auth-mode login
```

### 3. Configure Authentication

#### Option A: Generate SAS Token
```bash
az storage container generate-sas \
  --account-name mystorageaccount \
  --name logs \
  --permissions racwdl \
  --expiry 2025-12-31T23:59:59Z \
  --https-only \
  --output tsv
```

#### Option B: Create Service Principal
```bash
# Create service principal
az ad sp create-for-rbac \
  --name bibbl-log-stream \
  --role "Storage Blob Data Contributor" \
  --scopes /subscriptions/{subscription-id}/resourceGroups/bibbl-logs/providers/Microsoft.Storage/storageAccounts/mystorageaccount

# Note the tenant_id, client_id (appId), and client_secret (password)
```

#### Option C: Enable Managed Identity
```bash
# On Azure VM or Container Instances
az vm identity assign --name myvm --resource-group bibbl-logs

# Grant storage access
az role assignment create \
  --assignee {principal-id} \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/{subscription-id}/resourceGroups/bibbl-logs/providers/Microsoft.Storage/storageAccounts/mystorageaccount
```

### 4. Configure Firewall (Optional)

```bash
# Allow specific IP
az storage account network-rule add \
  --account-name mystorageaccount \
  --ip-address 203.0.113.10

# Or enable private endpoint
az network private-endpoint create \
  --name bibbl-storage-pe \
  --resource-group bibbl-logs \
  --vnet-name my-vnet \
  --subnet my-subnet \
  --private-connection-resource-id /subscriptions/{subscription-id}/resourceGroups/bibbl-logs/providers/Microsoft.Storage/storageAccounts/mystorageaccount \
  --connection-name bibbl-storage-connection \
  --group-id blob
```

### 5. Enable Customer-Managed Keys (Optional)

```bash
# Create Key Vault
az keyvault create \
  --name myvault \
  --resource-group bibbl-logs \
  --location eastus

# Create key
az keyvault key create \
  --vault-name myvault \
  --name logkey \
  --protection software

# Enable CMK on storage account
az storage account update \
  --name mystorageaccount \
  --resource-group bibbl-logs \
  --encryption-key-source Microsoft.Keyvault \
  --encryption-key-vault https://myvault.vault.azure.net \
  --encryption-key-name logkey
```

## Performance Tuning

### Append Blob Mode
- Best for: Real-time log streaming
- Latency: < 100ms per event
- Throughput: Up to 1000 events/sec per blob
- Use when: Low latency is critical

### Block Blob Mode
- Best for: High-volume batch ingestion
- Latency: Configurable (flush_interval)
- Throughput: Up to 100,000 events/sec
- Use when: Maximum throughput is needed

### Recommended Settings by Scenario

#### Real-Time Monitoring
```yaml
write_mode: "append"
compression_type: "none"  # Lower latency
format: "jsonl"
```

#### High-Volume Ingestion
```yaml
write_mode: "block"
max_batch_size: 5000
max_batch_bytes: 52428800  # 50MB
flush_interval: "60s"
compression_type: "gzip"   # Better compression ratio
```

#### Cost-Optimized Archive
```yaml
write_mode: "block"
compression_type: "gzip"
lifecycle_policy:
  transition_to_cool_days: 7
  transition_to_archive_days: 30
```

## Monitoring

The output exposes the following metrics:

- `events_sent`: Total events successfully written
- `events_failed`: Total events that failed
- `bytes_sent`: Total bytes written to Azure
- `retry_attempts`: Number of retry attempts
- `local_buffer_writes`: Events written to local buffer
- `dead_letter_writes`: Events sent to dead letter queue
- `last_error`: Most recent error message
- `last_error_time`: Timestamp of last error

Access metrics via the `/metrics` endpoint in Prometheus format.

## Troubleshooting

### Connection Issues

**Problem**: Cannot connect to storage account
```
Error: failed to create Azure client: authentication failed
```

**Solution**: Verify authentication credentials and network connectivity
```bash
# Test connection
az storage blob list --account-name mystorageaccount --container-name logs --auth-mode login

# Check firewall rules
az storage account show --name mystorageaccount --query "networkRuleSet"
```

### Authentication Failures

**Problem**: 403 Forbidden errors
```
Error: failed to append block: status code 403
```

**Solution**: Verify RBAC permissions
```bash
# Check role assignments
az role assignment list --scope /subscriptions/{subscription-id}/resourceGroups/bibbl-logs/providers/Microsoft.Storage/storageAccounts/mystorageaccount
```

### Performance Issues

**Problem**: High latency or low throughput

**Solution**: 
1. Switch to block blob mode for batching
2. Increase `max_batch_size` and `max_batch_bytes`
3. Use compression to reduce network transfer
4. Enable private endpoint for lower latency

### Buffer Full

**Problem**: Local buffer fills up during extended outages
```
Error: local buffer full (size: 1073741824, max: 1073741824)
```

**Solution**:
1. Increase `local_buffer_size`
2. Check Azure connectivity and resolve issues
3. Monitor dead letter queue for persistent failures

## Compliance

This output meets the following compliance requirements:

- ✅ **Encryption at Rest**: All data encrypted using Azure Storage Service Encryption
- ✅ **Customer-Managed Keys**: Support for CMK/KMS
- ✅ **Data Residency**: Regional deployment support
- ✅ **Audit Logging**: All operations logged with metrics
- ✅ **Network Isolation**: Private endpoint support
- ✅ **Access Control**: RBAC via Azure AD
- ✅ **Data Retention**: Configurable lifecycle policies
- ✅ **Disaster Recovery**: Local buffer failover

## API Integration

Use the output programmatically:

```go
import "bibbl/internal/outputs/azure_blob"

// Create configuration
config := &azure_blob.Config{
    StorageAccount: "mystorageaccount",
    Container:      "logs",
    AuthType:       azure_blob.AuthTypeManagedIdentity,
    WriteMode:      azure_blob.WriteModeBlock,
}

// Create output
output, err := azure_blob.NewOutput(config, logger)
if err != nil {
    log.Fatal(err)
}

// Start output
if err := output.Start(); err != nil {
    log.Fatal(err)
}

// Write events
event := &azure_blob.Event{
    Timestamp: time.Now(),
    Source:    "myapp",
    Data: map[string]interface{}{
        "level": "info",
        "message": "Log event",
    },
}

if err := output.Write(event); err != nil {
    log.Error("Failed to write event", err)
}

// Stop output
output.Stop()
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/ClarityXDR/bibbl-log-stream/issues
- Documentation: https://github.com/ClarityXDR/bibbl-log-stream/blob/main/vision.md
