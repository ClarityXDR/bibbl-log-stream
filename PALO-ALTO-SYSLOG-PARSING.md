# Palo Alto Networks NGFW Syslog Parsing

This document describes the Palo Alto Networks Next-Generation Firewall syslog parsing capabilities in Bibbl Log Stream.

## Overview

The Palo Alto CSV Parser (`PaloAltoCsvParser`) extracts structured fields from Palo Alto Networks firewall syslog messages in CSV (Comma-Separated Values) format. Unlike the Versa parser which handles Key-Value Pairs, Palo Alto uses a positional CSV format where field positions vary by log type.

### Features

- **Comprehensive Log Type Support**: Parses all major Palo Alto log types
- **CSV Format Handling**: Properly handles quoted fields with embedded commas
- **Type Conversion**: Automatically converts numeric fields (ports, bytes, counts)
- **Legal Compliance**: Always preserves original `_raw` field
- **High Performance**: ~35,000-43,000 events/second throughput
- **Lenient Parsing**: Tolerant of malformed logs (configurable)

### Performance Characteristics

- **Throughput**: 35,000-43,000 events/second
- **Memory**: ~33 KB per event
- **Allocations**: 113-119 allocations per parse

## Supported Log Types

### Traffic & Security

| Log Type | Description | Subtypes |
|----------|-------------|----------|
| **TRAFFIC** | Network traffic flows | start, end, drop, deny |
| **THREAT** | Security threats detected | vulnerability, spyware, virus, wildfire, url, data |
| **DECRYPTION** | SSL/TLS decryption | forward, decrypt, no-decrypt |
| **TUNNEL** | Tunnel inspection logs | — |
| **SCTP** | SCTP protocol logs | — |

### Authentication & Identity

| Log Type | Description | Subtypes |
|----------|-------------|----------|
| **AUTHENTICATION** | User authentication events | — |
| **USERID** | User-ID mapping events | login, logout, failed |
| **HIP-MATCH** | Host Information Profile | — |
| **GLOBALPROTECT** | GlobalProtect VPN | gateway-auth, tunnel-up, tunnel-down |

### Administration & System

| Log Type | Description | Subtypes |
|----------|-------------|----------|
| **CONFIG** | Configuration changes | — |
| **SYSTEM** | System events | general, ha, vpn, nat, dos, url-filtering |
| **CORRELATION** | Correlated security events | — |
| **AUDIT** | Administrative audit logs | — |

### Specialized

| Log Type | Description | Subtypes |
|----------|-------------|----------|
| **GTP** | GPRS Tunneling Protocol | — |

## Log Format

### CSV Structure

Palo Alto syslog messages use comma-separated values with **positional fields**:

```
FUTURE_USE, Receive Time, Serial Number, Type, Subtype, FUTURE_USE, Generated Time, [Type-Specific Fields...]
```

**Key Characteristics**:

- Fields are **position-dependent** (not named)
- Different log types have different field schemas
- Uses `FUTURE_USE` as placeholder for reserved fields
- Quoted fields can contain embedded commas
- Empty fields represented as consecutive commas (`,,`)

### Example Logs

#### TRAFFIC Log (Network Flow)

```
,2024/01/15 10:30:45,007951000012345,TRAFFIC,end,,2024/01/15 10:30:44,192.168.1.100,10.0.0.50,0.0.0.0,0.0.0.0,Allow-Web,alice@corp.com,,web-browsing,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Log-Forwarding,,123456,1,54321,443,0,0,0x80000000,tcp,allow,2048,1024,1024,100,2024/01/15 10:30:35,10,any,,,0,0x0,US,GB
```

**Parsed Fields**:

```json
{
  "type": "TRAFFIC",
  "subtype": "end",
  "src": "192.168.1.100",
  "dst": "10.0.0.50",
  "sport": 54321,
  "dport": 443,
  "proto": "tcp",
  "action": "allow",
  "rule": "Allow-Web",
  "app": "web-browsing",
  "srcuser": "alice@corp.com",
  "bytes": 2048,
  "packets": 100,
  "srcloc": "US",
  "dstloc": "GB"
}
```

#### THREAT Log (Security Alert)

```
,2024/01/15 11:00:00,007951000012345,THREAT,url,,2024/01/15 11:00:00,192.168.1.200,8.8.8.8,0.0.0.0,0.0.0.0,Block-Malware,bob@corp.com,,ssl,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Forward,,234567,1,55000,443,0,0,0x80000000,tcp,alert,http://malicious.example.com/payload,999888777(9999),hacking,high,client-to-server,111111,0x0,US,US
```

**Parsed Fields**:

```json
{
  "type": "THREAT",
  "subtype": "url",
  "src": "192.168.1.200",
  "dst": "8.8.8.8",
  "sport": 55000,
  "dport": 443,
  "action": "alert",
  "rule": "Block-Malware",
  "app": "ssl",
  "srcuser": "bob@corp.com",
  "misc": "http://malicious.example.com/payload",
  "threatid": "999888777(9999)",
  "category": "hacking",
  "severity": "high",
  "direction": "client-to-server"
}
```

#### SYSTEM Log (System Event)

```
,2024/01/15 13:00:00,007951000012345,SYSTEM,general,,2024/01/15 13:00:00,vsys1,general,,,, general,informational,System startup completed,54321,0x0
```

**Parsed Fields**:

```json
{
  "type": "SYSTEM",
  "subtype": "general",
  "vsys": "vsys1",
  "eventid": "general",
  "module": "general",
  "severity": "informational",
  "opaque": "System startup completed",
  "seqno": 54321
}
```

#### CONFIG Log (Configuration Change)

```
,2024/01/15 12:00:00,007951000012345,CONFIG,,,2024/01/15 12:00:00,PA-VM,set,admin-user,Web,Succeeded,/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system,<old-config>,<new-config>,12345,0x0
```

**Parsed Fields**:

```json
{
  "type": "CONFIG",
  "host": "PA-VM",
  "cmd": "set",
  "admin": "admin-user",
  "client": "Web",
  "config_result": "Succeeded",
  "path": "/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system"
}
```

## Common Fields

### Present in All Log Types

| Field | Type | Description |
|-------|------|-------------|
| `receive_time` | string | Time log received by management plane |
| `serial` | string | Firewall serial number |
| `type` | string | Log type (TRAFFIC, THREAT, SYSTEM, etc.) |
| `subtype` | string | Log subtype (varies by type) |
| `time_generated` | string | Time log generated on dataplane |
| `@timestamp` | string | ISO 8601 timestamp (parsed from receive_time) |
| `_raw` | string | Original syslog message (preserved) |
| `_parser` | string | Always "paloalto_csv" |
| `_parsed_at` | string | ISO 8601 timestamp of parse time |
| `paloalto_log_type` | string | Duplicate of `type` for convenience |

### TRAFFIC & THREAT Logs

| Field | Type | Description |
|-------|------|-------------|
| `src` | string | Source IP address |
| `dst` | string | Destination IP address |
| `natsrc` | string | NAT source IP |
| `natdst` | string | NAT destination IP |
| `sport` | int64 | Source port |
| `dport` | int64 | Destination port |
| `natsport` | int64 | NAT source port |
| `natdport` | int64 | NAT destination port |
| `proto` | string | IP protocol (tcp, udp, icmp) |
| `action` | string | Action taken (allow, deny, drop, alert, reset-*) |
| `rule` | string | Security rule name |
| `app` | string | Application name |
| `vsys` | string | Virtual system name |
| `from` | string | Source zone |
| `to` | string | Destination zone |
| `inbound_if` | string | Inbound interface |
| `outbound_if` | string | Outbound interface |
| `srcuser` | string | Source user |
| `dstuser` | string | Destination user |
| `sessionid` | int64 | Session ID |
| `logset` | string | Log forwarding profile |
| `seqno` | int64 | Sequence number |
| `actionflags` | string | Action flags (hex) |
| `srcloc` | string | Source country code |
| `dstloc` | string | Destination country code |

### TRAFFIC-Specific

| Field | Type | Description |
|-------|------|-------------|
| `bytes` | int64 | Total bytes transferred |
| `bytes_sent` | int64 | Bytes sent |
| `bytes_received` | int64 | Bytes received |
| `packets` | int64 | Total packets |
| `pkts_sent` | int64 | Packets sent |
| `pkts_received` | int64 | Packets received |
| `start` | string | Session start time |
| `elapsed` | int64 | Session duration (seconds) |
| `category` | string | URL category |
| `session_end_reason` | string | Why session ended (aged-out, tcp-fin, etc.) |
| `repeatcnt` | int64 | Repeat count |
| `flags` | string | TCP flags (hex) |
| `tunnel_id` | string | Tunnel/IMSI identifier |
| `monitortag` | string | Monitor tag/IMEI |
| `parent_session_id` | string | Parent session ID |
| `tunnel` | string | Tunnel type |
| `assoc_id` | int64 | SCTP association ID |
| `chunks` | int64 | SCTP chunks total |
| `chunks_sent` | int64 | SCTP chunks sent |
| `chunks_received` | int64 | SCTP chunks received |
| `rule_uuid` | string | Rule UUID |
| `http2_connection` | string | HTTP/2 connection ID |

### TRAFFIC Extended Fields (SD-WAN, Device-ID, Kubernetes)

| Field | Type | Description |
|-------|------|-------------|
| `link_change_count` | int64 | SD-WAN link changes |
| `policy_id` | string | Policy ID |
| `link_switches` | string | SD-WAN link switches |
| `sdwan_cluster` | string | SD-WAN cluster |
| `sdwan_device_type` | string | SD-WAN device type |
| `sdwan_cluster_type` | string | SD-WAN cluster type |
| `sdwan_site` | string | SD-WAN site name |
| `dynusergroup_name` | string | Dynamic user group |
| `xff_ip` | string | X-Forwarded-For IP |
| `src_category` | string | Source device category |
| `src_profile` | string | Source device profile |
| `src_model` | string | Source device model |
| `src_vendor` | string | Source device vendor |
| `src_osfamily` | string | Source OS family |
| `src_osversion` | string | Source OS version |
| `src_host` | string | Source hostname |
| `src_mac` | string | Source MAC address |
| `dst_category` | string | Destination device category |
| `dst_profile` | string | Destination device profile |
| `dst_model` | string | Destination device model |
| `dst_vendor` | string | Destination device vendor |
| `dst_osfamily` | string | Destination OS family |
| `dst_osversion` | string | Destination OS version |
| `dst_host` | string | Destination hostname |
| `dst_mac` | string | Destination MAC address |
| `container_id` | string | Container ID |
| `pod_namespace` | string | Kubernetes pod namespace |
| `pod_name` | string | Kubernetes pod name |
| `src_edl` | string | Source external dynamic list |
| `dst_edl` | string | Destination external dynamic list |
| `hostid` | string | Host ID |
| `src_dag` | string | Source dynamic address group |
| `dst_dag` | string | Destination dynamic address group |
| `session_owner` | string | Session owner |
| `nssai_sst` | string | 5G network slice SST |
| `nssai_sd` | string | 5G network slice SD |

### TRAFFIC Application Metadata (App-ID)

| Field | Type | Description |
|-------|------|-------------|
| `subcategory_of_app` | string | Application subcategory |
| `category_of_app` | string | Application category |
| `technology_of_app` | string | Application technology |
| `risk_of_app` | int64 | Application risk level (1-5) |
| `characteristic_of_app` | string | Application characteristics |
| `container_of_app` | string | Containerized app indicator |
| `tunneled_app` | string | Tunneled application |
| `is_saas_of_app` | int64 | Is SaaS application (0/1) |
| `sanctioned_state_of_app` | int64 | Sanctioned state (0/1) |
| `offloaded` | int64 | Offloaded indicator |
| `flow_type` | string | Flow type |
| `cluster_name` | string | Cluster name |

### THREAT-Specific

| Field | Type | Description |
|-------|------|-------------|
| `misc` | string | Threat/Filename/URL |
| `threatid` | string | Threat ID |
| `category` | string | Threat category |
| `severity` | string | Threat severity (informational, low, medium, high, critical) |
| `direction` | string | Traffic direction (client-to-server, server-to-client) |
| `contenttype` | string | Content MIME type |
| `pcap_id` | string | Packet capture ID |
| `filedigest` | string | File hash (MD5/SHA256) |
| `cloud` | string | Cloud service |
| `url_idx` | int64 | URL index |
| `user_agent` | string | HTTP user agent |
| `filetype` | string | File type |
| `xff` | string | X-Forwarded-For header |
| `referer` | string | HTTP referer |
| `sender` | string | Email sender |
| `subject` | string | Email subject |
| `recipient` | string | Email recipient |
| `reportid` | string | WildFire report ID |
| `http_method` | string | HTTP method (GET, POST, etc.) |
| `thr_category` | string | Threat category |
| `contentver` | string | Content version |
| `sig_flags` | string | Signature flags |

### CONFIG Log

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | Hostname |
| `cmd` | string | Command executed (set, delete, edit, etc.) |
| `admin` | string | Administrator username |
| `client` | string | Client type (Web, CLI, API) |
| `config_result` | string | Result (Succeeded, Failed) |
| `path` | string | Configuration path |
| `before_change_detail` | string | Configuration before change |
| `after_change_detail` | string | Configuration after change |

### SYSTEM Log

| Field | Type | Description |
|-------|------|-------------|
| `vsys` | string | Virtual system |
| `eventid` | string | Event ID |
| `object` | string | Object name |
| `module` | string | Module (general, management, auth, ha, upgrade, chassis) |
| `severity` | string | Severity (informational, low, medium, high, critical) |
| `opaque` | string | Detailed description (up to 512 bytes) |

### AUTHENTICATION Log

| Field | Type | Description |
|-------|------|-------------|
| `vsys` | string | Virtual system |
| `ip` | string | Client IP address |
| `user` | string | Username |
| `normalize_user` | string | Normalized username |
| `object` | string | Authentication object |
| `authpolicy` | string | Authentication policy |
| `repeatcnt` | int64 | Repeat count |
| `authid` | string | Authentication ID |
| `vendor` | string | Vendor (radius, ldap, saml, etc.) |
| `serverprofile` | string | Server profile name |
| `description` | string | Event description |
| `clienttype` | string | Client type |
| `event` | string | Event type |
| `factorno` | int64 | MFA factor number |
| `auth_protocol` | string | Authentication protocol |

### USERID Log

| Field | Type | Description |
|-------|------|-------------|
| `vsys` | string | Virtual system |
| `ip` | string | User IP address |
| `user` | string | Username |
| `datasourcename` | string | Data source name |
| `eventid` | string | Event ID |
| `timeout` | int64 | Timeout value |
| `beginport` | int64 | Port range begin |
| `endport` | int64 | Port range end |
| `datasource` | string | Data source |
| `datasourcetype` | string | Data source type |
| `factortype` | string | Factor type |
| `factorcompletiontime` | string | Factor completion time |

### HIP-MATCH Log

| Field | Type | Description |
|-------|------|-------------|
| `srcuser` | string | Source user |
| `vsys` | string | Virtual system |
| `machinename` | string | Machine name |
| `os` | string | Operating system |
| `src` | string | Source IP |
| `matchname` | string | HIP profile match name |
| `matchtype` | string | Match type |
| `ipv6` | string | IPv6 address |
| `hostid` | string | Host ID |

### GLOBALPROTECT Log

| Field | Type | Description |
|-------|------|-------------|
| `vsys` | string | Virtual system |
| `eventid` | string | Event ID |
| `stage` | string | Connection stage |
| `auth_method` | string | Authentication method |
| `tunnel_type` | string | Tunnel type (ipsec, ssl) |
| `srcuser` | string | Username |
| `srcregion` | string | Source region/country |
| `machinename` | string | Client machine name |
| `public_ip` | string | Client public IPv4 |
| `public_ipv6` | string | Client public IPv6 |
| `private_ip` | string | Client private IPv4 |
| `private_ipv6` | string | Client private IPv6 |
| `hostid` | string | Host ID |
| `serialnumber` | string | Client serial number |
| `client_ver` | string | Client version |
| `client_os` | string | Client OS |
| `client_os_ver` | string | Client OS version |
| `reason` | string | Event reason |
| `error` | string | Error message |
| `selection_type` | string | Gateway selection type |
| `response_time` | int64 | Response time (ms) |
| `priority` | int64 | Gateway priority |
| `attempted_gateways` | string | Attempted gateways |
| `gateway` | string | Connected gateway |

### DECRYPTION Log

| Field | Type | Description |
|-------|------|-------------|
| `src`, `dst`, `rule`, `app`, etc. | — | Same as TRAFFIC |
| `policy_name` | string | Decryption policy name |
| `decrypt_mirror` | string | Decrypt mirror |
| `ssl_version` | string | SSL/TLS version |
| `ssl_cipher_suite` | string | Cipher suite |
| `elliptic_curve` | string | Elliptic curve |
| `error_index` | string | Error index |
| `root_status` | string | Root certificate status |
| `chain_status` | string | Certificate chain status |
| `proxy_type` | string | Proxy type |
| `cert_serial` | string | Certificate serial |
| `fingerprint` | string | Certificate fingerprint |
| `cert_start_time` | string | Certificate start time |
| `cert_end_time` | string | Certificate end time |
| `cert_version` | string | Certificate version |
| `cert_size` | int64 | Certificate size |
| `cn_length` | int64 | Common name length |
| `issuer_cn_length` | int64 | Issuer CN length |
| `root_cn_length` | int64 | Root CN length |

### GTP Log

| Field | Type | Description |
|-------|------|-------------|
| `src`, `dst`, `rule`, etc. | — | Basic network fields |
| `event_type` | string | GTP event type |
| `msisdn` | string | Mobile subscriber number |
| `apn` | string | Access Point Name |
| `rat` | string | Radio Access Technology |
| `msg_type` | string | GTP message type |
| `end_ip_addr` | string | End IP address |
| `teid1` | string | Tunnel endpoint ID 1 |
| `teid2` | string | Tunnel endpoint ID 2 |
| `gtp_interface` | string | GTP interface |
| `cause_code` | int64 | Cause code |
| `mcc` | string | Mobile Country Code |
| `mnc` | string | Mobile Network Code |
| `area_code` | int64 | Area code |
| `cell_id` | int64 | Cell ID |
| `event_code` | int64 | Event code |
| `imsi` | string | IMSI |
| `imei` | string | IMEI |

### CORRELATION Log

| Field | Type | Description |
|-------|------|-------------|
| `vsys` | string | Virtual system |
| `category` | string | Event category |
| `severity` | string | Severity |
| `eventid` | string | Event ID |
| `object_name` | string | Object name |
| `object_id` | string | Object ID |
| `evidence` | string | Evidence |

### AUDIT Log

| Field | Type | Description |
|-------|------|-------------|
| `before_change_detail` | string | Configuration before |
| `after_change_detail` | string | Configuration after |
| `audit_comment` | string | Audit comment |

### Device Group Hierarchy (All Logs)

| Field | Type | Description |
|-------|------|-------------|
| `dg_hier_level_1` | int64 | Device group level 1 |
| `dg_hier_level_2` | int64 | Device group level 2 |
| `dg_hier_level_3` | int64 | Device group level 3 |
| `dg_hier_level_4` | int64 | Device group level 4 |
| `vsys_name` | string | Virtual system name |
| `device_name` | string | Device name |

## Configuration

### Basic Usage (Standalone Parser)

```go
import "bibbl/pkg/filters"

parser := filters.NewPaloAltoCSVParser()

event := map[string]interface{}{
    "_raw": ",2024/01/15 10:30:45,007951000012345,TRAFFIC,end,...",
}

err := parser.Parse(event)
if err != nil {
    // Handle error
}

// Access parsed fields
logType := event["type"]
srcIP := event["src"]
action := event["action"]
```

### Parser Options

```go
parser := filters.NewPaloAltoCSVParser()

// Preserve original _raw field (default: true, ALWAYS for legal compliance)
parser.PreserveRaw = true

// Strict mode: return errors on parse failures (default: false, lenient)
parser.StrictMode = false
```

### Pipeline Configuration

Add to your pipeline configuration:

```yaml
pipelines:
  - name: palo-alto-traffic
    description: "Parse Palo Alto NGFW syslog"
    inputs:
      - type: syslog
        port: 5514
        protocol: tcp
    
    processors:
      - type: custom
        function: paloalto_csv_parser
        config:
          preserve_raw: true
          strict_mode: false
    
    outputs:
      - type: sentinel
        workspace_id: "${SENTINEL_WORKSPACE_ID}"
        shared_key: "${SENTINEL_SHARED_KEY}"
        log_type: "PaloAltoNetworks"
```

### Route Configuration (Web UI)

1. **Create Route**:
   - Navigate to **Routes** → **Add Route**
   - Name: `palo-alto-ngfw-syslog`
   - Description: `Palo Alto Networks NGFW Syslog Parser`

2. **Configure Source**:
   - Input: `Syslog TCP (port 5514)` or `HTTP`
   - Expected format: CSV

3. **Add Parser**:
   - Processor: **Palo Alto CSV Parser**
   - Preserve Raw: `✓ Enabled` (required for legal compliance)
   - Strict Mode: `☐ Disabled` (lenient by default)

4. **Configure Destination**:
   - Output: `Microsoft Sentinel`
   - Log Type: `PaloAltoNetworks`

## Integration with Azure Sentinel

### Custom Log Table

Logs are ingested into a custom table in Azure Sentinel:

```
PaloAltoNetworks_CL
```

### Common KQL Queries

#### Traffic Overview

```kql
PaloAltoNetworks_CL
| where type_s == "TRAFFIC"
| summarize 
    TotalBytes = sum(bytes_d),
    TotalPackets = sum(packets_d),
    SessionCount = count()
    by bin(TimeGenerated, 5m), action_s
| render timechart
```

#### Top Blocked Threats

```kql
PaloAltoNetworks_CL
| where type_s == "THREAT" and action_s in ("alert", "block", "drop")
| summarize Count = count() by threatid_s, category_s, severity_s
| top 20 by Count desc
| project ThreatID = threatid_s, Category = category_s, Severity = severity_s, Count
```

#### Authentication Failures

```kql
PaloAltoNetworks_CL
| where type_s == "AUTHENTICATION" and event_s has "fail"
| summarize FailureCount = count() by user_s, ip_s, vendor_s
| where FailureCount > 5
| order by FailureCount desc
| project User = user_s, SourceIP = ip_s, Vendor = vendor_s, FailureCount
```

#### URL Filtering Blocks

```kql
PaloAltoNetworks_CL
| where type_s == "THREAT" and subtype_s == "url" and action_s == "block"
| extend URL = misc_s
| summarize BlockCount = count() by URL, category_s, srcuser_s
| top 50 by BlockCount desc
| project URL, Category = category_s, User = srcuser_s, BlockCount
```

#### GlobalProtect VPN Connections

```kql
PaloAltoNetworks_CL
| where type_s == "GLOBALPROTECT"
| summarize arg_max(TimeGenerated, *) by srcuser_s, machinename_s
| where eventid_s has "connect"
| project 
    TimeGenerated,
    User = srcuser_s,
    Machine = machinename_s,
    PublicIP = public_ip_s,
    PrivateIP = private_ip_s,
    Gateway = gateway_s,
    ClientVersion = client_ver_s
```

#### Configuration Changes

```kql
PaloAltoNetworks_CL
| where type_s == "CONFIG"
| project 
    TimeGenerated,
    Admin = admin_s,
    Command = cmd_s,
    Client = client_s,
    Result = config_result_s,
    Path = path_s
| order by TimeGenerated desc
```

#### Top Applications by Bandwidth

```kql
PaloAltoNetworks_CL
| where type_s == "TRAFFIC" and action_s == "allow"
| summarize TotalBytes = sum(bytes_d), SessionCount = count() by app_s
| extend TotalMB = TotalBytes / 1048576
| top 20 by TotalMB desc
| project Application = app_s, TotalMB, SessionCount
| render barchart
```

#### Geolocation Traffic Analysis

```kql
PaloAltoNetworks_CL
| where type_s == "TRAFFIC"
| where isnotempty(srcloc_s) and isnotempty(dstloc_s)
| summarize BytesTransferred = sum(bytes_d), Sessions = count() 
    by srcloc_s, dstloc_s, action_s
| extend BytesMB = BytesTransferred / 1048576
| top 100 by BytesMB desc
| project 
    SourceCountry = srcloc_s,
    DestCountry = dstloc_s,
    Action = action_s,
    BytesMB,
    Sessions
```

#### Anomalous Port Activity

```kql
let baselinePorts = PaloAltoNetworks_CL
| where type_s == "TRAFFIC" and TimeGenerated between (ago(7d) .. ago(1d))
| summarize BaselineSessions = count() by dport_d
| where BaselineSessions > 10;
PaloAltoNetworks_CL
| where type_s == "TRAFFIC" and TimeGenerated > ago(1h)
| summarize RecentSessions = count() by dport_d
| join kind=leftanti baselinePorts on $left.dport_d == $right.dport_d
| where RecentSessions > 5
| order by RecentSessions desc
| project Port = dport_d, Sessions = RecentSessions
```

#### Device-ID Inventory

```kql
PaloAltoNetworks_CL
| where type_s == "TRAFFIC" and isnotempty(src_vendor_s)
| summarize arg_max(TimeGenerated, *) by src_s
| project 
    IPAddress = src_s,
    Vendor = src_vendor_s,
    Model = src_model_s,
    OSFamily = src_osfamily_s,
    OSVersion = src_osversion_s,
    Hostname = src_host_s,
    MAC = src_mac_s,
    LastSeen = TimeGenerated
| order by LastSeen desc
```

## Type Conversion

The parser automatically converts fields to appropriate Go types:

| Field Pattern | Converted To | Example |
|---------------|--------------|---------|
| `*port`, `*id`, `seqno`, `repeatcnt` | `int64` | `sport: 443` |
| `bytes*`, `packets*`, `chunks*` | `int64` | `bytes: 2048` |
| `*_count`, `*_level_*`, `factorno` | `int64` | `repeatcnt: 5` |
| `elapsed`, `timeout`, `response_time` | `int64` | `elapsed: 120` |
| `risk_of_app`, `is_saas_of_app`, `offloaded` | `int64` | `risk: 4` |
| All other fields | `string` | `action: "allow"` |

## Error Handling

### Lenient Mode (Default)

- Missing fields: Ignored (field not added to result)
- Malformed CSV: Partial parse, no error
- Type conversion failures: Field stored as string
- Empty values: Field set to empty string or nil

### Strict Mode

```go
parser.StrictMode = true
```

- Missing required fields: Returns error
- CSV parse failures: Returns error
- Insufficient fields: Returns error
- Continues parsing type-specific fields even if some fail

### Common Errors

| Error | Cause | Resolution |
|-------|-------|------------|
| `no _raw field in event` | Event missing `_raw` key | Ensure syslog input adds `_raw` |
| `_raw field is not a string` | `_raw` is wrong type | Check input pipeline |
| `insufficient fields: got N, need at least 10` | Truncated CSV | Check syslog forwarding config |
| `failed to parse CSV` (strict) | Malformed CSV | Enable lenient mode or fix source |

## Performance Optimization

### Throughput Characteristics

- **TRAFFIC logs**: ~35,000 events/second
- **THREAT logs**: ~43,000 events/second
- **Memory per event**: ~33 KB
- **Allocations per event**: 113-119

### Optimization Tips

1. **Batch Processing**: Process multiple events in parallel
2. **Field Selection**: Only extract needed fields (modify parser if needed)
3. **Pre-filtering**: Filter by log type before parsing if possible
4. **Buffer Sizing**: Use adequate buffer sizes in syslog input

### Benchmarking

```bash
go test ./pkg/filters -bench=BenchmarkPaloAlto -benchmem
```

Expected output:

```
BenchmarkPaloAltoCsvParser_Parse-16        51721    28830 ns/op    33184 B/op    119 allocs/op
BenchmarkPaloAltoCsvParser_ParseThreat-16  45766    23250 ns/op    33056 B/op    113 allocs/op
```

## Troubleshooting

### Logs Not Parsing

**Symptoms**: Events have `_raw` but no parsed fields

**Causes**:

- Wrong parser (using Versa parser for Palo Alto logs)
- CSV format mismatch (custom format on firewall)
- Insufficient fields in CSV

**Solutions**:

1. Verify log format on firewall (should be CSV, not CEF)
2. Check first few fields match expected format
3. Enable lenient mode
4. Check firewall syslog server profile uses default CSV format

### Missing Fields

**Symptoms**: Some fields not appearing in parsed output

**Causes**:

- Empty values in CSV
- Field position mismatch (custom format)
- Field not in this log type's schema

**Solutions**:

1. Check raw log for field value
2. Verify log type is correct
3. Check if field applies to this log type
4. Enable debug logging to see field positions

### Type Conversion Issues

**Symptoms**: Numeric fields appearing as strings

**Causes**:

- Non-numeric values in numeric fields
- Parser doesn't recognize field as numeric

**Solutions**:

1. Check raw field value
2. Verify field name matches expected pattern
3. Add field to type conversion logic if needed

### Performance Issues

**Symptoms**: Low throughput, high CPU usage

**Causes**:

- Large quoted fields with many commas
- Very long log messages
- Inefficient downstream processing

**Solutions**:

1. Profile with `go test -cpuprofile`
2. Increase parser concurrency
3. Use field filtering
4. Batch output writes

## Legal & Compliance

### Data Retention

The parser **always preserves the `_raw` field** regardless of the `PreserveRaw` setting. This is required for:

- **Legal hold**: Original evidence preservation
- **Forensic analysis**: Bit-for-bit audit trail
- **Compliance**: SOC 2, ISO 27001, PCI-DSS requirements
- **Debugging**: Original message for troubleshooting

### Field Redaction

If PII redaction is required:

1. Apply PII redaction **after** parsing
2. Use separate redaction processor
3. Keep `_raw` in secure archival storage
4. Redact parsed fields only

Example pipeline:

```yaml
processors:
  - type: paloalto_csv_parser
  - type: pii_redactor
    fields: ["srcuser", "dstuser", "user", "email"]
  - type: conditional_output
    conditions:
      - field: "_raw"
        action: route_to_archive
```

## See Also

- [Versa SD-WAN Syslog Parsing](./VERSA-SYSLOG-PARSING.md) - KVP format parser
- [Palo Alto Networks Documentation](https://docs.paloaltonetworks.com/)
- [Azure Sentinel Integration](./AZURE-INTEGRATION.md)
- [Pipeline Configuration Guide](./PIPELINE-CONFIG.md)

## Support

For issues or questions:

- Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- Review test cases in `pkg/filters/paloalto_csv_parser_test.go`
- Enable debug logging for detailed parse information
