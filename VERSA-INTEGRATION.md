# Versa SD-WAN Integration Guide

## Overview

Bibbl Log Stream can receive syslog events from Versa SD-WAN Director over secure TLS connections. This guide explains how to export the required certificates and configure Versa Director to send logs to Bibbl.

## Understanding Versa's TLS Requirements

According to [Versa's official documentation](https://docs.versa-networks.com/Management_and_Orchestration/Versa_Analytics/Configuration/Configure_Log_Collectors_and_Log_Exporter_Rules), when configuring a remote syslog collector with TLS:

- **Certificate Format**: Privacy Enhanced Mail (PEM) format only
- **Certificate Chains**: NOT supported - single certificate files only
- **TLS Mode**: One-way TLS (server authentication) or mutual TLS
- **Required Files** (for one-way TLS):
  - `ca-cert-path`: CA certificate in PEM format to trust the remote syslog server

## Certificate Export API

Bibbl provides three API endpoints for certificate management:

### 1. List Available Certificates

```bash
GET /api/v1/syslog/certs
```

Returns metadata about all available syslog TLS certificates:

```json
{
  "certificates": [
    {
      "name": "bibbl-syslog.crt",
      "path": "certs/syslog/bibbl-syslog.crt",
      "description": "Server certificate (PEM format)",
      "forVersa": true,
      "size": 1233,
      "exists": true
    },
    {
      "name": "bibbl-syslog-ca.pem",
      "path": "certs/syslog/bibbl-syslog-ca.pem",
      "description": "CA bundle (PEM format) - For Versa Director",
      "forVersa": true,
      "size": 1233,
      "exists": true
    }
  ],
  "versaGuide": {
    "title": "Versa SD-WAN Integration Guide",
    "steps": [...],
    "required_files": ["bibbl-syslog-ca.pem"],
    "documentation": "https://docs.versa-networks.com/...",
    "tls_mode": "One-way TLS (server authentication only)",
    "notes": [
      "Versa requires certificates in PEM format",
      "Versa does NOT support certificate chains",
      "The private key should never be uploaded to Versa"
    ]
  }
}
```

### 2. Download Individual Certificate

```bash
GET /api/v1/syslog/certs/download?name=bibbl-syslog-ca.pem
```

Downloads a single certificate file. Valid names:
- `bibbl-syslog.crt` - Server certificate
- `bibbl-syslog-ca.pem` - CA bundle (recommended for Versa)
- `bibbl-syslog.key` - Private key (keep secure, do NOT upload to Versa)

### 3. Download Complete Bundle

```bash
GET /api/v1/syslog/certs/bundle
```

Downloads a ZIP file (`bibbl-versa-certs.zip`) containing:
- `bibbl-syslog.crt` - Server certificate
- `bibbl-syslog-ca.pem` - CA bundle
- `README.txt` - Configuration instructions

**Note**: The private key is intentionally excluded from the bundle for security.

## Versa Director Configuration Steps

### Step 1: Download Certificates

```bash
# Download the complete bundle
curl -k -o bibbl-versa-certs.zip https://your-bibbl-server:8443/api/v1/syslog/certs/bundle

# Or download just the CA certificate
curl -k -o bibbl-syslog-ca.pem https://your-bibbl-server:8443/api/v1/syslog/certs/download?name=bibbl-syslog-ca.pem
```

### Step 2: Configure Remote Collector in Versa Director

1. **Navigate to Analytics Configuration**
   - In Versa Director GUI, select the **Analytics** tab
   - Go to: **Administration > Configurations > Log Collector Exporter**

2. **Select the Remote Collector Tab**
   - Click the **Remote Collector** tab
   - Click **+ Add** to create a new remote collector

3. **Configure Basic Settings**
   ```
   Name:                 bibbl-tls-collector
   Description:          Bibbl Log Stream via TLS
   Destination Address:  <Bibbl server IP or hostname>
   Destination Port:     6514
   Type:                 TCP
   Template:             <Select your syslog template>
   ```

4. **Enable TLS and Upload Certificate**
   - Check the **TLS** checkbox to enable TLS transport
   - **CA Certificate**: Upload `bibbl-syslog-ca.pem`
   - **Client Certificate**: Leave empty (for one-way TLS)
   - **Private Key**: Leave empty (for one-way TLS)

5. **Click Save Changes**

### Step 3: Create Remote Template (if needed)

1. Select the **Remote Template** tab
2. Click **+ Add**
3. Configure:
   ```
   Name:        bibbl-syslog-template
   Description: Syslog format for Bibbl
   Type:        Syslog
   Format:      KVP (or your preferred format)
   ```

### Step 4: Create Remote Profile

1. Select the **Remote Profile** tab
2. Click **+ Add**
3. Configure:
   ```
   Name:             bibbl-profile
   Description:      Bibbl Log Stream collector
   Collector Group:  Select the collector created in Step 2
   ```

### Step 5: Create Exporter Rules

1. Select the **Exporter Rules** tab
2. Click **+ Add**
3. Configure:
   ```
   Name:                     bibbl-all-logs
   Local Collector:          <Select your local collector>
   Log Types:                Select desired log types (e.g., idp-log, urlf-log, etc.)
   Remote Collector Profile: bibbl-profile
   ```

4. Click **Save Changes**

### Step 6: Verify Configuration

1. Check Versa Analytics logs:
   ```bash
   sudo tail -f /var/log/syslog
   ```

2. Verify connection from Bibbl side:
   ```bash
   # Check if Versa is connecting
   docker logs bibbl-stream | grep syslog
   
   # Test connection manually
   openssl s_client -connect your-bibbl-server:6514
   ```

## Certificate Files Explained

### bibbl-syslog.crt
- **Format**: PEM (Privacy Enhanced Mail)
- **Type**: X.509 certificate
- **Purpose**: Server's public certificate for TLS syslog
- **Versa Usage**: Can be used as CA cert (self-signed)
- **Security**: Safe to share publicly

### bibbl-syslog-ca.pem
- **Format**: PEM
- **Type**: CA bundle (same as .crt for self-signed)
- **Purpose**: Explicitly named as CA bundle for clarity
- **Versa Usage**: Upload to "CA Certificate" field in Versa Director
- **Security**: Safe to share publicly

### bibbl-syslog.key
- **Format**: PEM
- **Type**: RSA private key
- **Purpose**: Server's private key for TLS
- **Versa Usage**: **NEVER upload to Versa** (server-side only)
- **Security**: **KEEP PRIVATE** - compromise allows impersonation

## TLS Modes

### One-Way TLS (Current Implementation)
- **Client**: Versa Director (validates Bibbl's certificate)
- **Server**: Bibbl (presents certificate to client)
- **Required Files**: Only `bibbl-syslog-ca.pem` uploaded to Versa
- **Security**: Prevents eavesdropping, authenticates server

### Mutual TLS (Optional Future Enhancement)
- **Client**: Versa Director (presents its own client certificate)
- **Server**: Bibbl (validates client certificate)
- **Required Files**: 
  - Bibbl uploads Versa's CA to trust Versa's client cert
  - Versa uploads Bibbl's CA and provides its own client cert/key
- **Security**: Bidirectional authentication

## Troubleshooting

### Connection Refused
```bash
# Verify Bibbl is listening
docker exec bibbl-stream netstat -tuln | grep 6514

# Check firewall
sudo firewall-cmd --list-ports  # Or equivalent for your firewall
```

### Certificate Validation Errors
```bash
# Verify certificate validity
openssl x509 -in bibbl-syslog-ca.pem -text -noout

# Test TLS handshake
openssl s_client -connect your-bibbl-server:6514 -CAfile bibbl-syslog-ca.pem
```

### No Logs Appearing in Bibbl
1. Check Versa exporter rules are active
2. Verify log types are selected in exporter rule
3. Check Bibbl API: `GET /api/v1/sources` - verify syslog source is active
4. Check Bibbl logs: `docker logs bibbl-stream`

### Versa Shows "All Brokers Down"
- This message is for Kafka collectors, not syslog TLS
- Check the remote collector connection status separately

## Environment Variables

Bibbl supports the following environment variables for certificate configuration:

```bash
# Override certificate directory
BIBBL_SYSLOG_CERT_DIR=/custom/path/to/certs

# Add additional hostnames to certificate
BIBBL_TLS_EXTRA_HOSTS=bibbl.example.com,192.168.1.100
```

## Security Best Practices

1. **Never Upload Private Keys**: The `.key` file should remain on the Bibbl server only
2. **Use Strong Passwords**: If API authentication is enabled
3. **Firewall Rules**: Restrict port 6514 to known Versa IP addresses
4. **Certificate Rotation**: Bibbl auto-rotates certificates before expiry
5. **Monitor Logs**: Watch for authentication failures or unusual connections

## API Examples

### Python
```python
import requests

# Download certificate list
response = requests.get('https://bibbl-server:8443/api/v1/syslog/certs', verify=False)
certs = response.json()

# Download bundle
bundle = requests.get('https://bibbl-server:8443/api/v1/syslog/certs/bundle', verify=False)
with open('bibbl-versa-certs.zip', 'wb') as f:
    f.write(bundle.content)
```

### PowerShell
```powershell
# Download bundle
Invoke-WebRequest -Uri "https://bibbl-server:8443/api/v1/syslog/certs/bundle" `
    -OutFile "bibbl-versa-certs.zip" `
    -SkipCertificateCheck

# Extract
Expand-Archive -Path "bibbl-versa-certs.zip" -DestinationPath "./versa-certs"
```

### cURL
```bash
# List certificates
curl -k https://bibbl-server:8443/api/v1/syslog/certs | jq

# Download CA certificate
curl -k -o bibbl-syslog-ca.pem \
  'https://bibbl-server:8443/api/v1/syslog/certs/download?name=bibbl-syslog-ca.pem'

# Download bundle
curl -k -o bibbl-versa-certs.zip \
  https://bibbl-server:8443/api/v1/syslog/certs/bundle
```

## References

- [Versa Analytics Configuration Guide](https://docs.versa-networks.com/Management_and_Orchestration/Versa_Analytics/Configuration/Configure_Log_Collectors_and_Log_Exporter_Rules)
- [Versa SD-WAN Documentation](https://docs.versa-networks.com/Secure_SD-WAN)
- [RFC 5424 - Syslog Protocol](https://tools.ietf.org/html/rfc5424)
- [RFC 5425 - TLS Transport Mapping for Syslog](https://tools.ietf.org/html/rfc5425)

## Support

For issues or questions:
- GitHub: https://github.com/ClarityXDR/bibbl-log-stream/issues
- Documentation: See README.md in the project root
- Versa Support: Contact your Versa Networks support representative
