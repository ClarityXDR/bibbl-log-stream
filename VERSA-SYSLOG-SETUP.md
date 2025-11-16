# Versa SD-WAN Syslog-over-TLS Setup Guide

This guide provides push-button instructions for configuring Versa SD-WAN appliances to send secure Syslog to Bibbl.

## Prerequisites

- Bibbl container running on `asm.clarityxdr.com` (192.168.0.217)
- Versa Director access (GUI or CLI)
- Network connectivity from Versa appliances to port 6514/tcp

## Step 1: Extract Bibbl TLS Certificate

From your Docker host:

```powershell
docker cp bibbl:/certs/syslog/bibbl-syslog-ca.pem ./bibbl-syslog-ca.pem
```

This PEM file contains the self-signed CA certificate Versa needs to trust.

## Step 2: Upload Certificate to Versa Director

### Via GUI (Recommended)

1. Log into **Versa Director**
2. Navigate to **Configuration → Devices → Certificates**
3. Click **Add Certificate**
4. **Name**: `Bibbl-Syslog-CA`
5. **Type**: CA Certificate
6. **Certificate**: Paste contents of `bibbl-syslog-ca.pem` or upload file
7. Click **Save**

### Via CLI

```bash
# SSH to Versa Director
ssh admin@director.example.com

# Upload certificate (replace path)
director> configure
director(config)> security certificates
director(config-certificates)> import ca-certificate Bibbl-Syslog-CA file /tmp/bibbl-syslog-ca.pem
director(config-certificates)> commit
```

## Step 3: Configure Remote Syslog Server on Versa Director

### Via GUI

1. **Configuration → Devices → Device Group (or specific device)**
2. **System → Logging → Remote Servers**
3. Click **Add Remote Server**
4. Configure:
   - **Name**: `Bibbl-ASM`
   - **Server Address**: `asm.clarityxdr.com` (or `192.168.0.217`)
   - **Port**: `6514`
   - **Protocol**: `TLS`
   - **Transport**: `TCP`
   - **TLS Profile**: Select or create profile referencing `Bibbl-Syslog-CA`
   - **Facility**: `local0` (or desired)
   - **Severity**: `informational` (or higher for production)
5. **Enable**: Check
6. Click **Save** and **Commit Changes**

### Via CLI

```bash
director(config)> logging remote-server Bibbl-ASM
director(config-remote-server-Bibbl-ASM)> host asm.clarityxdr.com
director(config-remote-server-Bibbl-ASM)> port 6514
director(config-remote-server-Bibbl-ASM)> protocol tls
director(config-remote-server-Bibbl-ASM)> transport tcp
director(config-remote-server-Bibbl-ASM)> tls-profile Bibbl-TLS
director(config-remote-server-Bibbl-ASM)> facility local0
director(config-remote-server-Bibbl-ASM)> severity informational
director(config-remote-server-Bibbl-ASM)> enable
director(config-remote-server-Bibbl-ASM)> commit
```

## Step 4: Create TLS Profile (if not exists)

### Via GUI

1. **Configuration → Security → TLS Profiles**
2. Click **Add TLS Profile**
3. **Name**: `Bibbl-TLS`
4. **Minimum TLS Version**: `TLS 1.2`
5. **Trusted CA Certificates**: Select `Bibbl-Syslog-CA`
6. **Verify Server Certificate**: Enable
7. **Cipher Suites**: Leave default (compatible with Bibbl)
8. Click **Save**

### Via CLI

```bash
director(config)> security tls-profile Bibbl-TLS
director(config-tls-profile)> min-version tlsv1.2
director(config-tls-profile)> trusted-ca-list Bibbl-Syslog-CA
director(config-tls-profile)> verify-server-certificate
director(config-tls-profile)> commit
```

## Step 5: Verify Configuration

### On Versa Appliance (CLI)

```bash
# Check remote server config
show logging remote-server Bibbl-ASM

# Test connectivity
ping asm.clarityxdr.com

# Verify TLS connection (may need openssl on appliance)
openssl s_client -connect asm.clarityxdr.com:6514 -CAfile /path/to/bibbl-syslog-ca.pem
```

### On Bibbl (Docker logs)

```powershell
docker logs -f bibbl
```

Look for:

- `syslog listener started on 0.0.0.0:6514 (TLS=true)`
- Connection attempts from Versa IPs (if `verbose_logging: true`)
- Ingested messages in the Web UI: `https://asm.clarityxdr.com:9444`

## Troubleshooting

### Certificate Validation Errors

**Symptom**: Versa logs show "certificate verify failed"

**Solutions**:

1. Verify `bibbl-syslog-ca.pem` uploaded correctly
2. Ensure Versa TLS profile references the correct CA
3. Check SAN/CN matches in certificate:

   ```bash
   openssl x509 -in bibbl-syslog-ca.pem -text -noout | grep -A1 "Subject Alternative Name"
   ```

4. Regenerate certificate with correct hostname if needed

### Connection Refused

**Symptom**: Versa cannot connect to port 6514

**Solutions**:

1. Verify Bibbl container is running: `docker ps | grep bibbl`
2. Check port mapping: `docker port bibbl`
3. Test from Windows host: `Test-NetConnection -ComputerName 192.168.0.217 -Port 6514`
4. Verify firewall rules (Windows Firewall, network ACLs)

### No Logs Appearing in Bibbl

**Symptom**: Connection succeeds but no messages in Bibbl UI

**Solutions**:

1. Enable verbose logging in `config.docker.yaml`: `verbose_logging: true`
2. Restart container: `docker restart bibbl`
3. Check Versa severity level (must be >= configured minimum)
4. Verify Versa appliance is generating logs: `show logging`
5. Check Bibbl pipeline routes in UI

### IP Allow-List Blocking

**Symptom**: Connections dropped immediately

**Solution**: Add Versa appliance IPs to `config.docker.yaml`:

```yaml
inputs:
  syslog:
    allow_list:
      - "10.0.0.0/8"
      - "192.168.1.50"
```

Restart container after changes.

## Production Recommendations

1. **Replace Self-Signed Certificate**: Use a proper PKI-signed cert for production
2. **Enable IP Allow-List**: Restrict to known Versa appliance subnets
3. **Tune Connection Limits**: Adjust `max_connections` based on appliance count
4. **Disable Verbose Logging**: Set `verbose_logging: false` after validation
5. **Monitor Certificate Expiry**: Bibbl auto-renews 45 days before expiry
6. **Use DNS**: Configure `asm.clarityxdr.com` in internal DNS for HA/mobility

## Certificate Renewal

Bibbl automatically renews self-signed certificates when:

- Current certificate expires within `renew_before_days` (default: 45)
- New hostnames added to `auto_cert.hosts`

To force immediate renewal:

```powershell
docker exec bibbl rm /certs/syslog/bibbl-syslog.crt /certs/syslog/bibbl-syslog.key
docker restart bibbl
```

Extract new certificate and re-upload to Versa Director.

## Reference: Versa Syslog Message Format

Versa appliances send RFC 5424-formatted messages:

```
<pri>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID [STRUCTURED-DATA] MSG
```

Bibbl automatically parses this format. No additional configuration needed.

## Support

- Bibbl logs: `docker logs bibbl`
- Web UI: `https://asm.clarityxdr.com:9444`
- Health check: `curl -k https://asm.clarityxdr.com:9444/api/v1/health`
- Versa documentation: [https://docs.versa-networks.com/](https://docs.versa-networks.com/)
