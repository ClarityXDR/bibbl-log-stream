# Azure Blob Storage Output - Enterprise Deployment Guide

This guide provides step-by-step instructions for deploying Bibbl Log Stream with Azure Blob Storage output in production environments.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Azure Infrastructure Setup](#azure-infrastructure-setup)
3. [Authentication Configuration](#authentication-configuration)
4. [Bibbl Configuration](#bibbl-configuration)
5. [Security Hardening](#security-hardening)
6. [High Availability Setup](#high-availability-setup)
7. [Monitoring and Alerting](#monitoring-and-alerting)
8. [Troubleshooting](#troubleshooting)
9. [Performance Optimization](#performance-optimization)

## Prerequisites

### Required
- Azure subscription with appropriate permissions
- Azure CLI installed (`az --version`)
- Bibbl Log Stream binary (Windows or Linux)
- Network connectivity to Azure

### Recommended
- Azure Key Vault for secrets management
- Azure Monitor for centralized monitoring
- Private networking (VNet, Private Endpoints)

## Azure Infrastructure Setup

### 1. Create Resource Group

```bash
# Set variables
RESOURCE_GROUP="bibbl-logs-prod"
LOCATION="eastus"
STORAGE_ACCOUNT="bibbllogsprod"  # Must be globally unique
CONTAINER_NAME="logs"

# Create resource group
az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION \
  --tags "Environment=Production" "Application=BibblLogStream"
```

### 2. Create Storage Account

```bash
# Create Storage Account V2 with encryption
az storage account create \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Standard_LRS \
  --kind StorageV2 \
  --encryption-services blob \
  --https-only true \
  --min-tls-version TLS1_2 \
  --allow-blob-public-access false \
  --tags "Environment=Production" "ManagedBy=Bibbl"

# Enable blob versioning
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT \
  --enable-versioning true

# Enable soft delete
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT \
  --enable-delete-retention true \
  --delete-retention-days 30
```

### 3. Create Containers

```bash
# Get storage account key (temporary for container creation)
ACCOUNT_KEY=$(az storage account keys list \
  --account-name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --query "[0].value" -o tsv)

# Create containers
az storage container create \
  --name $CONTAINER_NAME \
  --account-name $STORAGE_ACCOUNT \
  --account-key $ACCOUNT_KEY \
  --auth-mode key

# Create dead letter container
az storage container create \
  --name "failed-events" \
  --account-name $STORAGE_ACCOUNT \
  --account-key $ACCOUNT_KEY \
  --auth-mode key

# Create recovered events container
az storage container create \
  --name "recovered" \
  --account-name $STORAGE_ACCOUNT \
  --account-key $ACCOUNT_KEY \
  --auth-mode key
```

### 4. Configure Lifecycle Management

```bash
# Create lifecycle policy JSON
cat > lifecycle-policy.json <<EOF
{
  "rules": [
    {
      "enabled": true,
      "name": "move-to-cool",
      "type": "Lifecycle",
      "definition": {
        "actions": {
          "baseBlob": {
            "tierToCool": {
              "daysAfterModificationGreaterThan": 30
            },
            "tierToArchive": {
              "daysAfterModificationGreaterThan": 90
            },
            "delete": {
              "daysAfterModificationGreaterThan": 365
            }
          }
        },
        "filters": {
          "blobTypes": ["blockBlob", "appendBlob"],
          "prefixMatch": ["logs/"]
        }
      }
    },
    {
      "enabled": true,
      "name": "delete-failed-events",
      "type": "Lifecycle",
      "definition": {
        "actions": {
          "baseBlob": {
            "delete": {
              "daysAfterModificationGreaterThan": 180
            }
          }
        },
        "filters": {
          "blobTypes": ["blockBlob"],
          "prefixMatch": ["failed/"]
        }
      }
    }
  ]
}
EOF

# Apply lifecycle policy
az storage account management-policy create \
  --account-name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --policy @lifecycle-policy.json
```

## Authentication Configuration

### Option 1: Managed Identity (Recommended for Azure VMs)

#### A. Enable System-Assigned Managed Identity

```bash
# For Azure VM
VM_NAME="bibbl-vm"
az vm identity assign \
  --name $VM_NAME \
  --resource-group $RESOURCE_GROUP

# Get principal ID
PRINCIPAL_ID=$(az vm identity show \
  --name $VM_NAME \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

# Grant Storage Blob Data Contributor role
az role assignment create \
  --assignee $PRINCIPAL_ID \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT
```

#### B. Configuration
```yaml
outputs:
  azure_blob:
    storage_account: "bibbllogsprod"
    container: "logs"
    auth_type: "managed_identity"
    write_mode: "block"
```

### Option 2: Service Principal (Recommended for non-Azure environments)

```bash
# Create service principal
SP_NAME="bibbl-log-stream-sp"
SP_OUTPUT=$(az ad sp create-for-rbac \
  --name $SP_NAME \
  --role "Storage Blob Data Contributor" \
  --scopes /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT)

# Extract credentials
TENANT_ID=$(echo $SP_OUTPUT | jq -r '.tenant')
CLIENT_ID=$(echo $SP_OUTPUT | jq -r '.appId')
CLIENT_SECRET=$(echo $SP_OUTPUT | jq -r '.password')

# Store in Azure Key Vault (recommended)
KEYVAULT_NAME="bibbl-kv-prod"
az keyvault create \
  --name $KEYVAULT_NAME \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

az keyvault secret set --vault-name $KEYVAULT_NAME --name "azure-tenant-id" --value "$TENANT_ID"
az keyvault secret set --vault-name $KEYVAULT_NAME --name "azure-client-id" --value "$CLIENT_ID"
az keyvault secret set --vault-name $KEYVAULT_NAME --name "azure-client-secret" --value "$CLIENT_SECRET"
```

#### Configuration
```yaml
outputs:
  azure_blob:
    storage_account: "bibbllogsprod"
    container: "logs"
    auth_type: "azure_ad"
    tenant_id: "${AZURE_TENANT_ID}"      # From Key Vault or env var
    client_id: "${AZURE_CLIENT_ID}"
    client_secret: "${AZURE_CLIENT_SECRET}"
    write_mode: "block"
```

### Option 3: SAS Token (Simplest, but less secure)

```bash
# Generate SAS token (valid for 1 year)
END_DATE=$(date -u -d "1 year" '+%Y-%m-%dT%H:%M:%SZ')
SAS_TOKEN=$(az storage container generate-sas \
  --account-name $STORAGE_ACCOUNT \
  --name $CONTAINER_NAME \
  --permissions racwdl \
  --expiry $END_DATE \
  --https-only \
  --output tsv)

echo "SAS Token: $SAS_TOKEN"
```

#### Configuration
```yaml
outputs:
  azure_blob:
    storage_account: "bibbllogsprod"
    container: "logs"
    auth_type: "sas"
    sas_token: "${AZURE_SAS_TOKEN}"  # From environment variable
    write_mode: "block"
```

## Bibbl Configuration

### Production Configuration Template

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 9444
  tls:
    cert_file: "/etc/bibbl/tls/server.crt"
    key_file: "/etc/bibbl/tls/server.key"
    min_version: "1.2"

logging:
  level: "info"
  format: "json"

outputs:
  azure_blob_prod:
    # Connection
    storage_account: "bibbllogsprod"
    container: "logs"
    region: "eastus"
    
    # Authentication (Managed Identity)
    auth_type: "managed_identity"
    
    # Write settings
    write_mode: "block"
    path_template: "logs/{year}/{month}/{day}/{hour}/{source}.log.gz"
    max_batch_size: 5000
    max_batch_bytes: 52428800  # 50MB
    flush_interval: "60s"
    compression_type: "gzip"
    format: "jsonl"
    
    # Encryption
    encryption_enabled: true
    customer_managed_key: "https://bibbl-kv-prod.vault.azure.net/keys/logkey/current"
    
    # Lifecycle
    lifecycle_policy:
      enabled: true
      processed_retention_days: 90
      error_retention_days: 180
      failed_retention_days: 365
      transition_to_cool_days: 30
      transition_to_archive_days: 90
    
    # Resilience
    retry_attempts: 3
    retry_backoff: "1s"
    local_buffer_path: "/var/lib/bibbl/buffer/azure_blob.dat"
    local_buffer_size: 2147483648  # 2GB
    dead_letter_enabled: true
    dead_letter_path: "failed/{date}/{source}-failed.log"
```

## Security Hardening

### 1. Enable Customer-Managed Keys (CMK)

```bash
# Create Key Vault
az keyvault create \
  --name bibbl-kv-prod \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --enable-purge-protection true \
  --enable-soft-delete true

# Create encryption key
az keyvault key create \
  --vault-name bibbl-kv-prod \
  --name logkey \
  --protection software \
  --kty RSA \
  --size 2048

# Grant storage account access to Key Vault
STORAGE_PRINCIPAL=$(az storage account show \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --query identity.principalId -o tsv)

az keyvault set-policy \
  --name bibbl-kv-prod \
  --object-id $STORAGE_PRINCIPAL \
  --key-permissions get unwrapKey wrapKey

# Enable CMK on storage account
KEY_URI=$(az keyvault key show \
  --vault-name bibbl-kv-prod \
  --name logkey \
  --query key.kid -o tsv)

az storage account update \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --encryption-key-source Microsoft.Keyvault \
  --encryption-key-vault $KEY_URI
```

### 2. Configure Network Security

```bash
# Disable public access
az storage account update \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --default-action Deny

# Allow specific IPs (if needed)
az storage account network-rule add \
  --account-name $STORAGE_ACCOUNT \
  --ip-address 203.0.113.10

# Or create Private Endpoint
VNET_NAME="bibbl-vnet"
SUBNET_NAME="storage-subnet"

az network private-endpoint create \
  --name bibbl-storage-pe \
  --resource-group $RESOURCE_GROUP \
  --vnet-name $VNET_NAME \
  --subnet $SUBNET_NAME \
  --private-connection-resource-id /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --group-id blob \
  --connection-name bibbl-storage-connection
```

### 3. Enable Diagnostic Logging

```bash
# Create Log Analytics workspace
WORKSPACE_NAME="bibbl-logs-analytics"
az monitor log-analytics workspace create \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --location $LOCATION

# Enable diagnostic settings
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --query id -o tsv)

az monitor diagnostic-settings create \
  --name bibbl-storage-diagnostics \
  --resource /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --workspace $WORKSPACE_ID \
  --logs '[{"category": "StorageRead", "enabled": true}, {"category": "StorageWrite", "enabled": true}]' \
  --metrics '[{"category": "Transaction", "enabled": true}]'
```

## High Availability Setup

### 1. Multi-Region Deployment

```bash
# Create secondary storage account in different region
SECONDARY_LOCATION="westus"
SECONDARY_STORAGE="bibbllogsprodwest"

az storage account create \
  --name $SECONDARY_STORAGE \
  --resource-group $RESOURCE_GROUP \
  --location $SECONDARY_LOCATION \
  --sku Standard_LRS \
  --kind StorageV2 \
  --encryption-services blob \
  --https-only true

# Configure geo-redundancy (optional)
az storage account update \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_GRS  # Or Standard_GZRS for zone redundancy
```

### 2. Load Balancing Configuration

Deploy multiple Bibbl instances behind a load balancer, each with identical configuration pointing to the same storage account or different regional accounts.

### 3. Failover Configuration

```yaml
# Primary configuration
outputs:
  azure_blob_primary:
    storage_account: "bibbllogsprod"
    container: "logs"
    region: "eastus"
    # ... other settings

  # Failover configuration
  azure_blob_secondary:
    storage_account: "bibbllogsprodwest"
    container: "logs"
    region: "westus"
    # ... same settings as primary
```

## Monitoring and Alerting

### 1. Azure Monitor Metrics

```bash
# Create alert for high error rate
az monitor metrics alert create \
  --name bibbl-storage-errors \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --condition "avg Transactions where ResponseType = ClientThrottlingError > 10" \
  --window-size 5m \
  --evaluation-frequency 1m \
  --action-group-name bibbl-alerts

# Create alert for low availability
az monitor metrics alert create \
  --name bibbl-storage-availability \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --condition "avg Availability < 99" \
  --window-size 5m \
  --evaluation-frequency 1m \
  --action-group-name bibbl-alerts
```

### 2. Prometheus Metrics

Bibbl exposes Prometheus metrics at `/metrics`:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'bibbl'
    static_configs:
      - targets: ['bibbl-server:9444']
    metrics_path: '/metrics'
```

Key metrics to monitor:
- `bibbl_azure_blob_events_sent_total`
- `bibbl_azure_blob_events_failed_total`
- `bibbl_azure_blob_bytes_sent_total`
- `bibbl_azure_blob_retry_attempts_total`
- `bibbl_azure_blob_local_buffer_writes_total`

### 3. Grafana Dashboard

Import the pre-built Grafana dashboard (if available) or create custom panels for:
- Ingestion rate (events/sec)
- Error rate
- Latency percentiles
- Buffer utilization
- Dead letter queue size

## Troubleshooting

### Connection Issues

**Symptom:** Cannot connect to storage account

```bash
# Test connectivity
az storage blob list \
  --account-name $STORAGE_ACCOUNT \
  --container-name $CONTAINER_NAME \
  --auth-mode login

# Check firewall rules
az storage account show \
  --name $STORAGE_ACCOUNT \
  --query networkRuleSet

# Verify DNS resolution (for private endpoints)
nslookup $STORAGE_ACCOUNT.blob.core.windows.net
```

### Authentication Failures

**Symptom:** 403 Forbidden errors

```bash
# Verify RBAC assignments
az role assignment list \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT

# Test managed identity
curl -H Metadata:true "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://storage.azure.com/"
```

### Performance Issues

```bash
# Check storage account metrics
az monitor metrics list \
  --resource /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --metric Transactions \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z \
  --interval PT1H

# Check throttling
az monitor metrics list \
  --resource /subscriptions/$(az account show --query id -o tsv)/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT \
  --metric ClientThrottlingError \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z
```

## Performance Optimization

### 1. Batch Size Tuning

For high-volume scenarios:
```yaml
write_mode: "block"
max_batch_size: 10000
max_batch_bytes: 104857600  # 100MB
flush_interval: "120s"
```

### 2. Compression

Enable compression for better throughput:
```yaml
compression_type: "gzip"  # Reduces network transfer by ~80%
```

### 3. Path Template Optimization

Use hourly or daily partitions:
```yaml
# Daily: Better for archival
path_template: "logs/{year}/{month}/{day}/{source}.log"

# Hourly: Better for recent data access
path_template: "logs/{year}/{month}/{day}/{hour}/{source}.log"
```

### 4. Regional Deployment

Deploy Bibbl in the same region as the storage account to minimize latency.

### 5. SKU Selection

- **Standard_LRS**: Lowest cost, single-region
- **Standard_GRS**: Geo-redundant, higher availability
- **Premium_LRS**: SSD-based, ultra-low latency (for append blobs)

## Cost Optimization

### 1. Use Lifecycle Policies

Automatically transition data to cool/archive tiers:
```yaml
lifecycle_policy:
  transition_to_cool_days: 30     # $0.01/GB vs $0.018/GB
  transition_to_archive_days: 90  # $0.002/GB
```

### 2. Enable Compression

Reduces storage costs by ~80%:
```yaml
compression_type: "gzip"
```

### 3. Optimize Batch Size

Larger batches reduce transaction costs:
```yaml
max_batch_size: 10000
max_batch_bytes: 104857600  # 100MB
```

### 4. Reserved Capacity

Purchase Azure Storage reserved capacity for 1-3 year commitments (up to 38% discount).

## Maintenance

### Regular Tasks

1. **Monthly**: Review lifecycle policies and adjust retention
2. **Quarterly**: Rotate SAS tokens (if used)
3. **Yearly**: Review and renew CMK keys
4. **Ongoing**: Monitor metrics and adjust batch sizes

### Backup and Recovery

```bash
# Enable blob versioning (already done in setup)
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT \
  --enable-versioning true

# Enable soft delete
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT \
  --enable-delete-retention true \
  --delete-retention-days 30
```

## Support and Resources

- **Bibbl Documentation**: https://github.com/ClarityXDR/bibbl-log-stream
- **Azure Storage Documentation**: https://docs.microsoft.com/azure/storage/
- **Azure Support**: https://azure.microsoft.com/support/
