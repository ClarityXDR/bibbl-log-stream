# Severity-Based Routing Setup Guide

This guide shows you how to set up automated routing of security alerts to Azure Log Analytics (Sentinel) based on severity levels.

## Overview

The system comes pre-configured with:

- **Parsers**: Versa SD-WAN KVP and Palo Alto NGFW CSV parsers
- **Destinations**: Three Azure Log Analytics destinations for Critical, High, and Medium alerts
- **Example Routes**: Filter expressions ready to use (just need to be enabled)

## Quick Setup (5 Minutes)

### 1. Configure Destinations

Navigate to **Destinations** page and configure the three pre-created Azure Log Analytics destinations:

#### Sentinel Critical Alerts

```yaml
Workspace ID: <your-workspace-guid>
Shared Key: <your-primary-key>
Log Type: CriticalSecurityAlerts
Batch Size: 250 events
Flush Interval: 5 seconds
```

#### Sentinel High Priority

```yaml
Workspace ID: <same-workspace-guid>
Shared Key: <same-primary-key>
Log Type: HighPriorityAlerts
Batch Size: 500 events
Flush Interval: 10 seconds
```

#### Sentinel Medium Alerts

```yaml
Workspace ID: <same-workspace-guid>
Shared Key: <same-primary-key>
Log Type: MediumPriorityAlerts
Batch Size: 1000 events
Flush Interval: 20 seconds
```

**Where to find these values:**

1. Go to Azure Portal → Log Analytics Workspace
2. Click **Agents** (left menu)
3. Copy **Workspace ID** and **Primary Key**

### 2. Create Routes

Navigate to **Routes** page and create these routes:

#### Route 1: Versa Critical Alerts

```yaml
Name: Versa Critical → Sentinel
Pipeline: Versa SD-WAN Parser
Filter: event.severity === "Critical" || event.severity === "critical"
Destination: Sentinel Critical Alerts
Enabled: ✓
```

#### Route 2: Versa High Alerts

```yaml
Name: Versa High → Sentinel
Pipeline: Versa SD-WAN Parser
Filter: event.severity === "High" || event.severity === "high"
Destination: Sentinel High Priority
Enabled: ✓
```

#### Route 3: Versa Medium Alerts

```yaml
Name: Versa Medium → Sentinel
Pipeline: Versa SD-WAN Parser
Filter: event.severity === "Medium" || event.severity === "medium" || event.severity === "Warning"
Destination: Sentinel Medium Alerts
Enabled: ✓
```

#### Route 4: Palo Alto Critical Alerts

```yaml
Name: Palo Alto Critical → Sentinel
Pipeline: Palo Alto NGFW Parser
Filter: event.threat_severity === "critical" || event.severity_level_d > 8
Destination: Sentinel Critical Alerts
Enabled: ✓
```

#### Route 5: Palo Alto High Alerts

```yaml
Name: Palo Alto High → Sentinel
Pipeline: Palo Alto NGFW Parser
Filter: event.threat_severity === "high" || (event.severity_level_d >= 6 && event.severity_level_d <= 8)
Destination: Sentinel High Priority
Enabled: ✓
```

#### Route 6: Palo Alto Medium Alerts

```yaml
Name: Palo Alto Medium → Sentinel
Pipeline: Palo Alto NGFW Parser
Filter: event.threat_severity === "medium" || (event.severity_level_d >= 3 && event.severity_level_d <= 5)
Destination: Sentinel Medium Alerts
Enabled: ✓
```

### 3. Verify Ingestion

After 5-10 minutes, query your Azure Log Analytics workspace:

```kql
// Check Critical Alerts
CriticalSecurityAlerts_CL
| where TimeGenerated > ago(1h)
| summarize Count=count() by source_ip_s, severity_s
| order by Count desc

// Check High Priority
HighPriorityAlerts_CL
| where TimeGenerated > ago(1h)
| summarize Count=count() by source_type_s, threat_name_s
| top 10 by Count

// Check Medium Alerts
MediumPriorityAlerts_CL
| where TimeGenerated > ago(1h)
| summarize Count=count() by bin(TimeGenerated, 5m)
| render timechart
```

## Severity Field Mapping

### Versa SD-WAN Events

The Versa parser extracts these severity-related fields:

- `event.severity` (string): "Critical", "High", "Medium", "Low", "Info"
- `event.level` (string): Alias for severity in some log formats

**Example parsed event:**

```json
{
  "_raw": "<134>1 2024-01-15T10:30:45Z versa-sd-wan ...",
  "severity": "Critical",
  "event_type": "Security",
  "threat_name": "Malware Detected",
  "source_ip": "10.1.2.3"
}
```

### Palo Alto NGFW Events

The Palo Alto parser extracts these severity-related fields:

- `event.threat_severity` (string): "critical", "high", "medium", "low", "informational"
- `event.severity_level_d` (integer): Numeric severity (0-10)
- `event.log_subtype` (string): "alert", "wildfire", etc.

**Example parsed event:**

```json
{
  "_raw": "1,2024/01/15 10:30:45,001234567890...",
  "threat_severity": "critical",
  "severity_level_d": 9,
  "threat_name": "Zeus Trojan C2 Traffic",
  "category": "command-and-control"
}
```

## Advanced Routing Patterns

### Route by Threat Category

```javascript
// Route specific threat types to different tables
event.category === "command-and-control" || 
event.category === "malware" || 
event.threat_name.includes("ransomware")
```

### Route by Source Network

```javascript
// Separate internal vs external threats
event.source_ip.startsWith("10.") || 
event.source_ip.startsWith("172.16.") || 
event.source_ip.startsWith("192.168.")
```

### Combine Severity + Category

```javascript
// Critical C2 traffic only
(event.severity === "Critical" || event.threat_severity === "critical") &&
(event.category === "command-and-control" || event.threat_name.includes("C2"))
```

### Time-Based Routing

```javascript
// Route high-severity events during business hours differently
var hour = new Date(event.timestamp || event.generated_time).getHours();
(event.severity === "High" || event.threat_severity === "high") && 
(hour >= 8 && hour <= 18)
```

## Performance Optimization

### Critical Alerts (Fastest Response)

- **Batch Size**: 250 events (small batches)
- **Flush Interval**: 5 seconds (aggressive)
- **Concurrency**: 3 workers
- **Expected Latency**: 5-15 seconds
- **Use Case**: C2 traffic, malware, data exfiltration

### High Priority (Balanced)

- **Batch Size**: 500 events
- **Flush Interval**: 10 seconds
- **Concurrency**: 2 workers
- **Expected Latency**: 10-30 seconds
- **Use Case**: Suspicious activity, policy violations, unusual auth

### Medium Alerts (Cost-Optimized)

- **Batch Size**: 1000 events
- **Flush Interval**: 20 seconds
- **Concurrency**: 2 workers
- **Expected Latency**: 20-60 seconds
- **Use Case**: Informational, low-priority warnings, auditing

## Monitoring & Alerting

### Create Azure Monitor Alerts

#### Critical Alert Ingestion Failure

```kql
AzureDiagnostics
| where Category == "OperationalLogs"
| where OperationName == "Data collection"
| where ResultType == "Failed"
| where Resource contains "CriticalSecurityAlerts"
| summarize FailureCount=count() by bin(TimeGenerated, 5m)
| where FailureCount > 5
```

#### High Volume of Critical Events

```kql
CriticalSecurityAlerts_CL
| where TimeGenerated > ago(15m)
| summarize Count=count()
| where Count > 1000
```

#### Missing Expected Events

```kql
let expected_sources = dynamic(["Versa-SD-WAN", "Palo-Alto-FW"]);
CriticalSecurityAlerts_CL
| where TimeGenerated > ago(1h)
| summarize Sources=make_set(source_type_s)
| extend Missing = set_difference(expected_sources, Sources)
| where array_length(Missing) > 0
```

## Cost Management

### Estimated Monthly Costs

Based on 1M events/day with the recommended split (5% Critical, 25% High, 70% Medium):

- **Critical**: 50K events/day × 2KB avg = 100MB/day = 3GB/month → ~$6/month
- **High**: 250K events/day × 2KB avg = 500MB/day = 15GB/month → ~$30/month
- **Medium**: 700K events/day × 2KB avg = 1.4GB/day = 42GB/month → ~$84/month

**Total**: ~$120/month for 1M events/day (vs. ~$200/month for single-table ingestion)

### Cost Reduction Strategies

1. **Pre-filtering**: Filter out low-value events before parsing
2. **Field reduction**: Remove unnecessary fields from Medium alerts
3. **Aggregation**: Aggregate similar medium-severity events
4. **Retention tiers**: Use different retention periods per severity
5. **Basic logs**: Consider Basic Logs tier for Medium alerts (80% cheaper)

## Troubleshooting

### Events Not Appearing in Azure

**Check 1: Verify destination configuration**

```bash
curl http://localhost:8080/api/destinations | jq '.[] | select(.name | contains("Sentinel"))'
```

**Check 2: Verify routes are enabled**

```bash
curl http://localhost:8080/api/routes | jq '.[] | select(.enabled == true)'
```

**Check 3: Check pipeline metrics**

```bash
curl http://localhost:8080/metrics | grep bibbl_pipeline
```

**Check 4: Test connection to Azure**

```bash
# In bibbl-stream console
POST https://<workspace-id>.ods.opinsights.azure.com/api/logs?api-version=2016-04-01
Headers:
  Authorization: SharedKey <workspace-id>:<signature>
  Log-Type: TestConnection
  x-ms-date: <RFC1123-date>
Body: [{"test":"connection","timestamp":"2024-01-15T10:30:00Z"}]
```

### High Latency for Critical Alerts

**Solution 1: Reduce batch size**

```yaml
batchMaxEvents: 100  # Down from 250
flushIntervalSec: 3  # Down from 5
```

**Solution 2: Increase concurrency**

```yaml
concurrency: 5  # Up from 3
```

**Solution 3: Dedicated pipeline**
Create a separate pipeline for critical alerts with no other routes.

### Events Going to Wrong Tables

**Common Issue**: Severity field not parsed correctly

**Fix**: Check parser output in Transform Workbench:

1. Go to **Transform** page
2. Select parser pipeline
3. Paste sample log
4. Click **Test**
5. Verify `severity` field exists and has expected value

**Versa Example:**

```
Input: <134>1 2024-01-15T10:30:45Z versa severity=Critical event=Malware
Output should contain: { "severity": "Critical", ... }
```

**Palo Alto Example:**

```
Input: 1,2024/01/15,001234,THREAT,file,2304,...,critical,...
Output should contain: { "threat_severity": "critical", "severity_level_d": 9, ... }
```

## Best Practices

1. **Start with one source**: Get Versa or Palo Alto working end-to-end before adding routes
2. **Test filters**: Use Transform Workbench to test filter expressions before creating routes
3. **Monitor costs**: Set up Azure Cost Management alerts for unexpected increases
4. **Review quarterly**: Adjust severity thresholds based on actual threat landscape
5. **Document custom routes**: Add comments in route names explaining business logic
6. **Backup configurations**: Export pipeline/route configs monthly via UI

## Comparison with Cribl Stream

| Feature | Bibbl Stream | Cribl Stream |
|---------|--------------|--------------|
| Severity routing | ✅ Native | ✅ Native |
| Table name control | ✅ Full control | ✅ Full control |
| Batching config | ✅ Per-destination | ✅ Per-route |
| Cost optimization | ✅ 3-tier profiles | ⚠️ Manual tuning |
| Filter language | ✅ JavaScript | ✅ Cribl Expr |
| Performance | 🚀 Go native | ⚠️ Node.js overhead |
| Single binary | ✅ Yes | ❌ Full install |
| Cross-platform | ✅ Windows + Linux | ⚠️ Mainly Linux |

## Next Steps

1. ✅ Set up destinations (5 min)
2. ✅ Create routes (10 min)
3. ✅ Send test data (5 min)
4. ✅ Verify in Azure (5 min)
5. 📊 Create Sentinel workbooks
6. 🔔 Set up alerting rules
7. 📈 Monitor performance & costs

**Need Help?** Check the detailed documentation:

- `AZURE-LOG-ANALYTICS-INTEGRATION.md` - Complete Azure integration guide
- `VERSA-SYSLOG-PARSING.md` - Versa parser documentation
- `PALO-ALTO-SYSLOG-PARSING.md` - Palo Alto parser documentation
