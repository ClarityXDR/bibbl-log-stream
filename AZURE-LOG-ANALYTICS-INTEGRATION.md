# Azure Log Analytics Workspace Integration

## Overview

Bibbl Log Stream supports sending logs to Azure Log Analytics Workspace (including Microsoft Sentinel) using the HTTP Data Collector API with automatic severity-based routing.

## Features

### 🎯 Severity-Based Routing

- **Critical Alerts** → Separate table for immediate response
- **High Priority** → Dedicated table for urgent investigation
- **Medium Priority** → Standard monitoring table
- **Low/Info** → General logs table (optional)

### 🚀 Performance Optimizations

- **Intelligent Batching**: Groups up to 500 events or 1MB per batch
- **Automatic Flushing**: Configurable intervals (default: 10 seconds)
- **Concurrent Workers**: Multiple parallel connections (default: 2)
- **Exponential Backoff**: Automatic retry with increasing delays
- **Connection Pooling**: Reuses HTTP connections for efficiency

### 🔐 Security & Compliance

- **HMAC-SHA256 Signing**: RFC 1123 compliant authentication
- **TLS 1.2+**: Encrypted data in transit
- **Resource Tagging**: Optional Azure Resource ID for context
- **Audit Trail**: Full OpenTelemetry tracing support

## Configuration

### Workspace Credentials

Find your credentials in Azure Portal:

1. Navigate to **Log Analytics workspace**
2. Click **Agents** in the left menu
3. Copy **Workspace ID** and **Primary Key**

### Destination Setup

#### Option 1: Via UI (Recommended)

1. Navigate to **Destinations** page
2. Click **Add Destination**
3. Select type: `azure_loganalytics`
4. Configure:

   ```json
   {
     "workspaceID": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
     "sharedKey": "base64-encoded-key==",
     "logType": "SecurityAlerts",
     "resourceGroup": "your-rg-name",
     "batchMaxEvents": 500,
     "batchMaxBytes": 1048576,
     "flushIntervalSec": 10,
     "concurrency": 2,
     "maxRetries": 3,
     "retryDelaySec": 2
   }
   ```

#### Option 2: Via Config File

```yaml
destinations:
  - name: "Sentinel Critical Alerts"
    type: "azure_loganalytics"
    config:
      workspaceID: "${AZURE_WORKSPACE_ID}"
      sharedKey: "${AZURE_SHARED_KEY}"
      logType: "CriticalSecurityAlerts"
      resourceGroup: "prod-security-rg"
      batchMaxEvents: 250  # Smaller batches for critical alerts
      flushIntervalSec: 5   # Faster flushing
```

### Table Naming

- Specify the base table name (e.g., `SecurityAlerts`)
- Azure automatically appends `_CL` suffix → `SecurityAlerts_CL`
- Maximum 100 characters
- Only letters, numbers, and underscores
- No special characters or spaces

## Severity-Based Routing Examples

### Example 1: Three-Tier Routing

Create three routes for different alert severities:

**Critical Alerts Route:**

```json
{
  "name": "Critical Alerts to Sentinel",
  "filter": "_raw.includes('severity') && JSON.parse(_raw).severity === 'Critical'",
  "pipelineID": "versa-parser-pipeline-id",
  "destination": "sentinel-critical-dest-id",
  "final": true
}
```

**High Priority Route:**

```json
{
  "name": "High Priority to Sentinel",
  "filter": "_raw.includes('severity') && JSON.parse(_raw).severity === 'High'",
  "pipelineID": "paloalto-parser-pipeline-id",
  "destination": "sentinel-high-dest-id",
  "final": true
}
```

**Medium Priority Route:**

```json
{
  "name": "Medium Priority to Sentinel",
  "filter": "_raw.includes('severity') && ['Medium','Warning'].includes(JSON.parse(_raw).severity)",
  "pipelineID": "default-pipeline-id",
  "destination": "sentinel-medium-dest-id",
  "final": true
}
```

### Example 2: Parser-Specific Routing

Route parsed Versa and Palo Alto logs based on their severity fields:

**Versa Critical:**

```javascript
// Filter expression
event.severity === "Critical" && event.logType === "flowIdLog"
```

**Palo Alto High:**

```javascript
// Filter expression  
event.threat_severity === "high" || event.THREAT_ID
```

## Table Creation in Azure

Tables are created automatically on first ingestion:

1. **First Event**: Azure creates table with schema based on fields
2. **Schema Evolution**: New fields added automatically
3. **Data Types**: Inferred from JSON values (string_s, number_d, boolean_b, datetime_t)
4. **Query Ready**: Available in Log Analytics ~2-5 minutes after ingestion

### Query Your Data

```kql
// Query critical alerts from last hour
CriticalSecurityAlerts_CL
| where TimeGenerated > ago(1h)
| where severity_s == "Critical"
| project TimeGenerated, source_s, message_s, threat_id_s
| order by TimeGenerated desc

// Count alerts by severity
SecurityAlerts_CL
| summarize Count=count() by severity_s
| render piechart

// Palo Alto threats over time
PaloAltoThreats_CL  
| where TimeGenerated > ago(24h)
| summarize Count=count() by bin(TimeGenerated, 1h), threat_severity_s
| render timechart
```

## Performance Tuning

### For High-Volume Sources (>10K events/sec)

```json
{
  "batchMaxEvents": 1000,
  "batchMaxBytes": 5242880,
  "flushIntervalSec": 5,
  "concurrency": 5,
  "maxRetries": 3
}
```

### For Critical Alerts (Low Latency)

```json
{
  "batchMaxEvents": 100,
  "batchMaxBytes": 524288,
  "flushIntervalSec": 2,
  "concurrency": 3,
  "maxRetries": 5
}
```

### For Cost Optimization

```json
{
  "batchMaxEvents": 1000,
  "batchMaxBytes": 10485760,
  "flushIntervalSec": 30,
  "concurrency": 2,
  "maxRetries": 3
}
```

## Monitoring & Troubleshooting

### Check Destination Status

```bash
curl -k https://localhost:8443/api/v1/destinations
```

### View Metrics

```bash
# Prometheus metrics
curl -k https://localhost:8443/metrics | grep azure_loganalytics
```

### Common Issues

**Issue**: `403 Forbidden - InvalidAuthorization`

- **Solution**: Verify Workspace ID and Shared Key are correct
- Check key is base64-encoded (should end with `==`)

**Issue**: `400 Bad Request - InvalidLogType`  

- **Solution**: Log type must be alphanumeric + underscore only
- Remove `-CL` suffix (Azure adds it automatically)

**Issue**: Slow ingestion

- **Solution**: Increase `concurrency` and reduce `flushIntervalSec`
- Check Azure region latency

**Issue**: Data not appearing

- **Solution**: Wait 2-5 minutes for indexing
- Check table name has `_CL` suffix in queries
- Verify firewall allows outbound to `*.ods.opinsights.azure.com`

## Cost Optimization

### Data Ingestion Costs (Microsoft Sentinel)

- First 10 GB/day: Free tier (per workspace)
- Additional data: $2.46/GB (Pay-As-You-Go)
- Commitment Tiers: 100GB-5000GB/day with 15-50% discount

### Optimization Strategies

1. **Filter Before Sending**: Only send necessary severity levels
2. **Field Selection**: Remove verbose fields in pipeline
3. **Sampling**: Route only 10% of low-severity events
4. **Aggregation**: Pre-aggregate metrics before sending
5. **Table Separation**: Use cheaper workspaces for low-priority logs

### Example: Cost-Aware Routing

```javascript
// Only send Critical/High to expensive Sentinel workspace
if (event.severity === "Critical" || event.severity === "High") {
  route.destination = "sentinel-premium";
} else if (event.severity === "Medium") {
  route.destination = "loganalytics-standard";  // Cheaper
} else {
  route.destination = "local-storage";  // Free
}
```

## Advanced Features

### Custom Timestamp Field

Control the TimeGenerated field:

```json
{
  "time-generated-field": "event_timestamp"
}
```

Your event JSON:

```json
{
  "event_timestamp": "2025-11-16T10:30:45Z",
  "message": "Login successful"
}
```

### Azure Resource Context

Tag logs with Azure Resource ID for resource-context queries:

```json
{
  "resourceID": "/subscriptions/{sub-id}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{vm}"
}
```

Enables queries like:

```kql
SecurityAlerts_CL
| where _ResourceId == "/subscriptions/.../virtualMachines/webserver01"
```

## Comparison with Cribl

| Feature | Bibbl (This Implementation) | Cribl Stream |
|---------|----------------------------|--------------|
| Batching | ✅ Configurable | ✅ Configurable |
| Compression | ❌ Not yet | ✅ GZIP |
| Retry Logic | ✅ Exponential backoff | ✅ Exponential backoff |
| Multi-workspace | ✅ Multiple destinations | ✅ Multiple destinations |
| Table Name Control | ✅ Full control | ✅ Full control |
| Resource Tagging | ✅ Azure Resource ID | ✅ Azure Resource ID |
| Custom Timestamp | ✅ Supported | ✅ Supported |
| License Cost | ✅ Free & Open Source | ❌ $600+/month |

## Best Practices

1. **✅ Use Descriptive Table Names**: `PaloAltoFirewall_Critical_CL` not `logs_CL`
2. **✅ Separate by Severity**: Critical alerts in their own table
3. **✅ Enable Tracing**: Helps troubleshoot ingestion issues
4. **✅ Monitor Metrics**: Track batch sizes and retry counts
5. **✅ Test with Sample Data**: Validate routing before production
6. **✅ Document Your Schema**: Keep field mappings documented
7. **✅ Rotate Keys Regularly**: Use Azure Key Vault for secrets
8. **✅ Set Retention Policies**: Balance cost vs compliance requirements

## Support

For issues or questions:

- Check logs: `./bibbl-stream --log-level=debug`
- View traces in OpenTelemetry backend
- File issue on GitHub
- Consult Azure Monitor documentation

---

**Next Steps:**

1. ✅ Install/upgrade Bibbl Log Stream
2. ✅ Configure Azure Log Analytics destinations
3. ✅ Create severity-based routes
4. ✅ Test with sample data
5. ✅ Monitor ingestion in Azure Portal
6. ✅ Set up alerting rules in Sentinel
