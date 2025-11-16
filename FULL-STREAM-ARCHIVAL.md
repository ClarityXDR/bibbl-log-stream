# Full-Stream Archival & Parallel Routing

## Overview

Bibbl Log Stream supports **parallel routing** - simultaneously sending the same events to multiple destinations with different processing pipelines. This is critical for:

- **Compliance**: Archive all raw logs to S3/ADLS for regulatory retention
- **Analytics**: Send parsed/enriched data to Sentinel for SIEM analysis
- **Hot/Cold Tiers**: Separate critical alerts (hot) from full archive (cold)
- **Multi-tenancy**: Route to different storage accounts by customer/region

## Architecture

```
                         ┌──────────────────────────────────┐
                         │  Syslog Input (Versa/Palo Alto) │
                         └──────────────┬───────────────────┘
                                        │
                                        ▼
                         ┌──────────────────────────────────┐
                         │  Parser Pipeline                 │
                         │  (Extract fields, normalize)     │
                         └──────────────┬───────────────────┘
                                        │
                         ┌──────────────┴───────────────────┐
                         │                                   │
                         ▼                                   ▼
        ┌────────────────────────────┐      ┌────────────────────────────┐
        │  Severity-Based Routes     │      │  Full-Stream Archive       │
        │  (Filter by severity)      │      │  (No filter = all events)  │
        └────────────┬───────────────┘      └────────────┬───────────────┘
                     │                                    │
          ┌──────────┼──────────┐                        │
          ▼          ▼           ▼                        ▼
     ┌────────┐ ┌────────┐ ┌────────┐           ┌──────────────┐
     │Critical│ │  High  │ │ Medium │           │ S3 / ADLS    │
     │Sentinel│ │Sentinel│ │Sentinel│           │ (Compressed) │
     └────────┘ └────────┘ └────────┘           └──────────────┘
```

## Pre-Configured Destinations

Bibbl automatically creates these destinations on startup:

### Severity-Based (Azure Log Analytics)

1. **Sentinel Critical Alerts** - Fast flushing (5sec), small batches (250 events)
2. **Sentinel High Priority** - Balanced (10sec, 500 events)
3. **Sentinel Medium Alerts** - Cost-optimized (20sec, 1000 events)

### Full-Stream Archive (S3/ADLS)

4. **S3 Versa Full Stream** - All Versa SD-WAN logs → S3
5. **ADLS Versa Full Stream** - All Versa SD-WAN logs → Azure Data Lake
6. **S3 Palo Alto Full Stream** - All Palo Alto NGFW logs → S3
7. **ADLS Palo Alto Full Stream** - All Palo Alto NGFW logs → Azure Data Lake

## Configuration

### Option 1: Use S3 for Archive

Perfect for multi-cloud or AWS-native environments.

**Step 1: Configure S3 Destination**

Navigate to **Destinations** → **S3 Versa Full Stream**:

```yaml
Bucket: security-logs-prod
Region: us-east-1
Prefix: versa/raw/
Path Template: versa/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/versa-${mm}-${ss}.jsonl.gz
Compression: gzip
Batch Max Events: 5000
Batch Max Bytes: 10485760  # 10MB
Flush Interval: 60 seconds
Concurrency: 3
```

**Step 2: AWS Credentials**

Use IAM instance role (recommended) or environment variables:

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1
```

**Step 3: Create Route**

Navigate to **Routes** → **Add Route**:

```yaml
Name: Versa Full Archive → S3
Pipeline: Versa SD-WAN Parser
Filter: true  # No filter = all events
Destination: S3 Versa Full Stream
Enabled: ✓
Final: false  # Allow other routes to also process
```

### Option 2: Use Azure Data Lake for Archive

Perfect for Azure-native environments or when using Sentinel.

**Step 1: Configure ADLS Destination**

Navigate to **Destinations** → **ADLS Versa Full Stream**:

```yaml
Storage Account: securitylogsprod
Filesystem: security-logs
Directory: versa/raw/
Path Template: versa/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/versa-${mm}-${ss}.jsonl.gz
Format: jsonl
Compression: gzip
Batch Max Events: 5000
Batch Max Bytes: 10485760  # 10MB
Flush Interval: 60 seconds
Concurrency: 3
Max Open Files: 2
```

**Step 2: Azure Credentials**

Use Managed Identity (recommended) or connection string:

```bash
export AZURE_STORAGE_ACCOUNT=securitylogsprod
export AZURE_STORAGE_KEY=...
# OR
export AZURE_STORAGE_CONNECTION_STRING=DefaultEndpointsProtocol=https;...
```

**Step 3: Create Route**

Navigate to **Routes** → **Add Route**:

```yaml
Name: Versa Full Archive → ADLS
Pipeline: Versa SD-WAN Parser
Filter: true  # No filter = all events
Destination: ADLS Versa Full Stream
Enabled: ✓
Final: false  # Allow other routes to also process
```

## Complete Setup Example (Dual Routing)

This example shows how to send **Critical alerts to Sentinel** AND **all events to S3**.

### Step 1: Configure Destinations (Already Pre-Created)

You only need to fill in credentials:

1. **Sentinel Critical Alerts**
   - Workspace ID: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
   - Shared Key: `[your-base64-key]`
   - Log Type: `CriticalSecurityAlerts`

2. **S3 Versa Full Stream**
   - Bucket: `security-logs-prod`
   - AWS credentials configured

### Step 2: Create Two Routes

**Route 1: Critical Alerts → Sentinel**

```yaml
Name: Versa Critical → Sentinel
Pipeline: Versa SD-WAN Parser
Filter: event.severity === "Critical" || event.severity === "critical"
Destination: Sentinel Critical Alerts
Enabled: ✓
Final: false  # Important! Don't stop routing here
```

**Route 2: All Events → S3**

```yaml
Name: Versa Full Archive → S3
Pipeline: Versa SD-WAN Parser
Filter: true  # Matches everything
Destination: S3 Versa Full Stream
Enabled: ✓
Final: false  # Or true if this is the last route
```

### Step 3: Verify Data Flow

**Check Sentinel (5-10 minutes delay)**

```kql
CriticalSecurityAlerts_CL
| where TimeGenerated > ago(1h)
| summarize Count=count() by severity_s, source_ip_s
| order by Count desc
```

**Check S3 (immediate)**

```bash
aws s3 ls s3://security-logs-prod/versa/raw/year=2024/month=11/day=16/hour=14/
# Should see: versa-30-45.jsonl.gz, versa-31-45.jsonl.gz, etc.

# Download and inspect
aws s3 cp s3://security-logs-prod/versa/raw/year=2024/.../versa-30-45.jsonl.gz - | gunzip | jq .
```

## Path Templates

Both S3 and ADLS support dynamic path templates with time-based variables:

### Available Variables

- `${yyyy}` - 4-digit year (2024)
- `${MM}` - 2-digit month (01-12)
- `${dd}` - 2-digit day (01-31)
- `${HH}` - 2-digit hour (00-23)
- `${mm}` - 2-digit minute (00-59)
- `${ss}` - 2-digit second (00-59)

### Example Patterns

**Hourly Partitioning (Recommended)**

```
versa/raw/year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/versa-${mm}-${ss}.jsonl.gz
```

- Easy to query specific time ranges
- Balances file count vs file size
- Works well with Athena/Synapse partitioning

**Daily Partitioning (High Volume)**

```
versa/raw/year=${yyyy}/month=${MM}/day=${dd}/versa-${HH}-${mm}.jsonl.gz
```

- Fewer top-level directories
- Larger files (better compression)

**Flat Structure (Simple)**

```
versa/raw/versa-${yyyy}${MM}${dd}-${HH}${mm}${ss}.jsonl.gz
```

- No directories
- Harder to query by date
- Good for short retention

**Custom Prefix by Source Type**

```
logs/${source_type}/year=${yyyy}/month=${MM}/day=${dd}/data.jsonl.gz
```

- Separate by log type
- Requires route-level customization

## Performance Tuning

### High-Volume Sources (>50K events/sec)

```yaml
batchMaxEvents: 10000
batchMaxBytes: 20971520  # 20MB
flushIntervalSec: 120    # 2 minutes
concurrency: 5
```

**Result**: Larger files, fewer API calls, lower cost, higher latency.

### Low-Latency Requirements (<1 minute)

```yaml
batchMaxEvents: 1000
batchMaxBytes: 1048576   # 1MB
flushIntervalSec: 30
concurrency: 3
```

**Result**: Smaller files, more API calls, faster availability.

### Cost-Optimized (Long-term Archive)

```yaml
batchMaxEvents: 50000
batchMaxBytes: 104857600 # 100MB
flushIntervalSec: 300    # 5 minutes
concurrency: 2
compression: gzip        # Must enable!
```

**Result**: Maximum compression, minimum API calls, lowest cost.

## Compression

**Always enable compression** for archival destinations:

```yaml
compression: gzip
```

**Compression Ratios**:

- JSON logs: 80-90% reduction (10x smaller)
- Syslog: 70-85% reduction (5-7x smaller)
- Binary/encrypted: Minimal reduction

**Example**:

- 10GB/day uncompressed → 1-2GB/day compressed
- S3 Standard: $0.023/GB/month → $0.02-0.05/month vs $0.23/month

## Cost Comparison

### S3 Standard (US-East-1)

**Storage**: $0.023/GB/month  
**Requests**: $0.005 per 1000 PUTs

**Example** (1M events/day, 2KB avg, 90% compression):

- Data: 2GB/day → 200MB/day compressed = 6GB/month
- Storage: 6GB × $0.023 = **$0.14/month**
- Requests: 1440 batches/day × 30 = 43,200 PUTs = **$0.22/month**
- **Total: $0.36/month**

### Azure Data Lake Gen2

**Storage**: $0.018/GB/month (cool tier)  
**Requests**: $0.10 per 10,000 writes

**Example** (same 1M events/day):

- Data: 6GB/month compressed
- Storage: 6GB × $0.018 = **$0.11/month**
- Requests: 43,200 writes = **$0.43/month**
- **Total: $0.54/month**

### Sentinel Ingestion (For Comparison)

**Data Ingestion**: $2.76/GB  
**No compression discount**

**Example** (1M events/day):

- Data: 60GB/month uncompressed
- Ingestion: 60GB × $2.76 = **$165.60/month**
- **Total: $165.60/month** (plus $0.15/GB retention)

**Savings**: S3/ADLS archive is **300-400x cheaper** than Sentinel ingestion!

## Monitoring

### Check Destination Status

```bash
curl -k https://localhost:8443/api/v1/destinations | jq '.[] | select(.name | contains("Full Stream"))'
```

### View Metrics

```bash
# S3 metrics
curl -k https://localhost:8443/metrics | grep s3_

# ADLS metrics
curl -k https://localhost:8443/metrics | grep adls_
```

### Monitor File Creation

**S3**:

```bash
# List latest files
aws s3 ls s3://security-logs-prod/versa/raw/year=2024/month=11/day=16/hour=14/ --recursive

# Count files in last hour
aws s3 ls s3://security-logs-prod/versa/raw/ --recursive | grep "$(date -u +%Y/%m/%d/%H)" | wc -l
```

**ADLS**:

```bash
# List latest files
az storage fs file list --account-name securitylogsprod --file-system security-logs --path versa/raw/

# Or use Azure Storage Explorer GUI
```

## Querying Archived Data

### AWS Athena (S3)

**Step 1: Create External Table**

```sql
CREATE EXTERNAL TABLE versa_archive (
  timestamp STRING,
  severity STRING,
  event_type STRING,
  source_ip STRING,
  dest_ip STRING,
  threat_name STRING,
  raw_message STRING
)
PARTITIONED BY (
  year STRING,
  month STRING,
  day STRING,
  hour STRING
)
ROW FORMAT SERDE 'org.openx.data.jsonserde.JsonSerDe'
LOCATION 's3://security-logs-prod/versa/raw/'
TBLPROPERTIES ('has_encrypted_data'='false');
```

**Step 2: Add Partitions**

```sql
MSCK REPAIR TABLE versa_archive;
-- Or manually:
ALTER TABLE versa_archive ADD PARTITION (year='2024', month='11', day='16', hour='14')
LOCATION 's3://security-logs-prod/versa/raw/year=2024/month=11/day=16/hour=14/';
```

**Step 3: Query**

```sql
SELECT severity, COUNT(*) as event_count, COUNT(DISTINCT source_ip) as unique_sources
FROM versa_archive
WHERE year='2024' AND month='11' AND day='16' AND hour='14'
  AND severity IN ('Critical', 'High')
GROUP BY severity
ORDER BY event_count DESC;
```

### Azure Synapse (ADLS)

**Step 1: Create External Table**

```sql
CREATE EXTERNAL FILE FORMAT jsonl_gzip
WITH (
    FORMAT_TYPE = DELIMITEDTEXT,
    FORMAT_OPTIONS (FIELD_TERMINATOR = '\n'),
    DATA_COMPRESSION = 'org.apache.hadoop.io.compress.GzipCodec'
);

CREATE EXTERNAL TABLE versa_archive
WITH (
    LOCATION = 'versa/raw/',
    DATA_SOURCE = security_logs_storage,
    FILE_FORMAT = jsonl_gzip
)
AS
SELECT
    JSON_VALUE(line, '$.timestamp') AS timestamp,
    JSON_VALUE(line, '$.severity') AS severity,
    JSON_VALUE(line, '$.event_type') AS event_type,
    JSON_VALUE(line, '$.source_ip') AS source_ip
FROM OPENROWSET(
    BULK 'versa/raw/year=2024/month=11/day=16/**/*.jsonl.gz',
    DATA_SOURCE = 'security_logs_storage',
    FORMAT = 'CSV',
    FIELDTERMINATOR = '\n'
) WITH (line NVARCHAR(MAX)) AS rows;
```

**Step 2: Query**

```sql
SELECT severity, COUNT(*) as event_count
FROM versa_archive
WHERE timestamp >= '2024-11-16T00:00:00Z'
  AND severity IN ('Critical', 'High')
GROUP BY severity;
```

## Troubleshooting

### Events Not Appearing in S3/ADLS

**Check 1: Destination enabled?**

```bash
curl -k https://localhost:8443/api/v1/destinations | jq '.[] | select(.name | contains("Full Stream")) | {name, enabled, status}'
```

**Check 2: Route enabled?**

```bash
curl -k https://localhost:8443/api/v1/routes | jq '.[] | select(.destination | contains("Full")) | {name, enabled, filter}'
```

**Check 3: Credentials configured?**

```bash
# S3
aws s3 ls s3://security-logs-prod/ || echo "Credentials invalid"

# ADLS
az storage account show --name securitylogsprod || echo "Credentials invalid"
```

**Check 4: Pipeline metrics**

```bash
curl -k https://localhost:8443/metrics | grep -E 'bibbl_route_events_sent|bibbl_destination_events'
```

### Files Too Small or Too Large

**Too Small** (< 1MB compressed):

- Increase `batchMaxEvents` or `batchMaxBytes`
- Increase `flushIntervalSec`
- Result: Fewer API calls, lower cost

**Too Large** (> 100MB compressed):

- Decrease `batchMaxEvents` or `batchMaxBytes`
- Decrease `flushIntervalSec`
- Result: Faster queries, better parallelism

**Optimal Size**: 10-50MB compressed (100-500MB uncompressed)

### High S3/ADLS Costs

**Solution 1: Increase batch sizes**

```yaml
batchMaxEvents: 10000  # Up from 5000
flushIntervalSec: 120  # Up from 60
```

**Solution 2: Use lifecycle policies**

S3:

```json
{
  "Rules": [{
    "Status": "Enabled",
    "Transitions": [{
      "Days": 30,
      "StorageClass": "GLACIER"
    }]
  }]
}
```

ADLS:

```json
{
  "rules": [{
    "name": "move-to-cool",
    "enabled": true,
    "type": "Lifecycle",
    "definition": {
      "actions": {
        "baseBlob": {
          "tierToCool": { "daysAfterModificationGreaterThan": 30 }
        }
      }
    }
  }]
}
```

**Solution 3: Enable compression** (if not already)

```yaml
compression: gzip  # 5-10x reduction!
```

## Best Practices

### 1. Always Use Parallel Routing

✅ **Good**: Critical alerts → Sentinel, All events → S3  
❌ **Bad**: Only critical alerts saved (data loss!)

### 2. Set `Final: false` for Archive Routes

```yaml
Name: Versa Full Archive → S3
Final: false  # Allow other routes to process
```

This ensures severity-based routes still fire.

### 3. Use Separate Destinations per Source Type

```
S3 Versa Full Stream → versa/raw/...
S3 Palo Alto Full Stream → paloalto/raw/...
```

Don't mix source types in the same bucket prefix.

### 4. Enable Compression for Archive

```yaml
compression: gzip
```

Always. No exceptions. Saves 80-90% storage cost.

### 5. Partition by Hour for Most Use Cases

```
year=${yyyy}/month=${MM}/day=${dd}/hour=${HH}/...
```

Daily partitions too large, minute partitions too granular.

### 6. Monitor Batch Sizes

```bash
curl -k https://localhost:8443/metrics | grep batch_size
```

Aim for 10-50MB per file.

### 7. Use Lifecycle Policies

Move old data to cold storage after 30-90 days.

### 8. Test Query Performance

Run sample Athena/Synapse queries before going to production.

### 9. Document Retention Requirements

```yaml
# S3 Lifecycle Policy
Retention: 7 years (regulatory)
Transition to Glacier: 90 days
Delete after: 2557 days
```

### 10. Backup Destination Configs

Export configs monthly via UI.

## Multi-Cloud Strategy

### Hybrid: Sentinel + S3 + ADLS

**Use Case**: Azure-native SIEM, multi-cloud data lake

```
Versa SD-WAN Logs
├─ Critical Alerts → Sentinel (Azure Log Analytics)
├─ All Events → S3 (AWS, primary archive)
└─ All Events → ADLS (Azure, backup/compliance)
```

**Benefits**:

- Real-time SIEM in Azure
- Primary archive in AWS (cheaper)
- Backup archive in Azure (geo-redundancy)

**Cost** (1M events/day):

- Sentinel ingestion (5% of events): ~$8/month
- S3 archive (all events): ~$0.36/month
- ADLS archive (all events): ~$0.54/month
- **Total: ~$9/month** vs $165/month for Sentinel-only

## Next Steps

1. ✅ Configure S3 or ADLS credentials
2. ✅ Enable full-stream archive destinations
3. ✅ Create routes with `Final: false`
4. ✅ Verify data appears in S3/ADLS
5. ✅ Set up lifecycle policies
6. ✅ Create Athena/Synapse tables
7. ✅ Document retention policies
8. ✅ Monitor costs monthly

**Need Help?**

- S3 Documentation: `README.md` (S3 Output section)
- ADLS Documentation: `README.md` (Azure Data Lake section)
- Sentinel Setup: `SEVERITY-ROUTING-SETUP.md`
- Azure Integration: `AZURE-LOG-ANALYTICS-INTEGRATION.md`
