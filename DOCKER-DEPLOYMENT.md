# Bibbl Log Stream - Portainer Deployment Guide

This guide covers deploying Bibbl Log Stream using Portainer with the assigned configuration:
- **IP Address**: 192.168.0.218
- **Domain**: bibbl.clarityxdr.com

## Prerequisites

1. Portainer installed and running
2. Docker Engine on the target host
3. Network configuration allowing access to 192.168.0.218
4. DNS resolution for bibbl.clarityxdr.com pointing to 192.168.0.218

## Deployment Methods

### Method 1: Portainer Stacks (Recommended)

1. **Access Portainer**
   - Navigate to your Portainer instance
   - Go to "Stacks" in the left sidebar

2. **Create New Stack**
   - Click "Add stack"
   - Name: `bibbl-log-stream`
   - Build method: `Git Repository` or `Upload`

3. **Stack Configuration**

   **Option A: Git Repository**
   ```
   Repository URL: https://github.com/ClarityXDR/bibbl-log-stream
   Reference: refs/heads/main
   Compose path: docker-compose.yml
   ```

   **Option B: Upload docker-compose.yml**
   - Copy the contents of `docker-compose.yml` into the web editor

4. **Environment Variables**
   Add these environment variables in Portainer:
   ```
   BIBBL_VERSION=0.1.0
   BIBBL_COMMIT=production
   BIBBL_BUILD_DATE=2024-01-01T00:00:00Z
   ```

5. **Deploy**
   - Click "Deploy the stack"
   - Monitor deployment in the logs

### Method 2: Portainer App Templates

1. **Create Custom Template**
   - Go to "App Templates" → "Custom Templates"
   - Add new template with the docker-compose.yml content

2. **Deploy from Template**
   - Select the custom template
   - Configure environment variables
   - Deploy

### Method 3: Manual Container Deployment

1. **Build Image** (if not using pre-built)
   ```bash
   git clone https://github.com/ClarityXDR/bibbl-log-stream
   cd bibbl-log-stream
   docker build -t bibbl-stream:latest .
   ```

2. **Create Container in Portainer**
   - Go to "Containers" → "Add container"
   - Configure as shown in the docker-compose.yml

## Network Configuration

### Custom Network Setup
The deployment requires a custom network with the specific IP range:

1. **Create Network in Portainer**
   - Go to "Networks" → "Add network"
   - Name: `bibbl-network`
   - Driver: `bridge`
   - Subnet: `192.168.0.0/24`
   - Gateway: `192.168.0.1`

2. **Assign Static IP**
   - The container will be assigned 192.168.0.218 automatically via docker-compose

## SSL/TLS Configuration

### Auto-Generated Certificates (Default)
- Bibbl Stream will automatically generate self-signed certificates
- Certificates will include `bibbl.clarityxdr.com` in the SAN

### Custom Certificates (Recommended for Production)
1. **Create certificates directory**
   ```bash
   mkdir -p ./certs
   ```

2. **Add your certificates**
   ```
   ./certs/server.crt    # Server certificate
   ./certs/server.key    # Server private key
   ./certs/ca.crt        # CA certificate (optional)
   ```

3. **Update environment variables**
   ```
   BIBBL_SERVER_TLS_CERT_FILE=/app/certs/server.crt
   BIBBL_SERVER_TLS_KEY_FILE=/app/certs/server.key
   ```

## Configuration Options

### Environment Variables
Key environment variables you can set in Portainer:

```bash
# Server
BIBBL_SERVER_HOST=0.0.0.0
BIBBL_SERVER_PORT=9444

# TLS
BIBBL_SERVER_TLS_MIN_VERSION=1.2

# Logging
BIBBL_LOGGING_LEVEL=info
BIBBL_LOGGING_FORMAT=json

# Syslog Input
BIBBL_INPUTS_SYSLOG_ENABLED=true
BIBBL_INPUTS_SYSLOG_PORT=6514

# Azure/Sentinel (if needed)
BIBBL_OUTPUTS_SENTINEL_WORKSPACE_ID=your-workspace-id
BIBBL_OUTPUTS_SENTINEL_SHARED_KEY=your-shared-key
```

### Volume Mounts
Configure persistent storage in Portainer:

- **Data**: `/app/data` → `bibbl-data` volume
- **Logs**: `/app/logs` → `bibbl-logs` volume  
- **Config**: `./config:/app/config:ro` (optional)
- **Certs**: `./certs:/app/certs:ro` (optional)

## Access and Verification

### Web Interface
1. **HTTPS Access**
   ```
   https://bibbl.clarityxdr.com:9444
   https://192.168.0.218:9444
   ```

2. **Health Check**
   ```
   https://bibbl.clarityxdr.com:9444/health
   ```

### Syslog Testing
Test syslog input:
```bash
# Using openssl for TLS syslog
echo '<134>1 2024-01-01T12:00:00Z test-host bibbl-test - - Test message' | \
  openssl s_client -connect bibbl.clarityxdr.com:6514 -quiet
```

## Troubleshooting

### Common Issues

1. **Container won't start**
   - Check logs in Portainer: Containers → bibbl-stream → Logs
   - Verify network configuration
   - Check port conflicts

2. **Network connectivity issues**
   - Verify DNS resolution: `nslookup bibbl.clarityxdr.com`
   - Check firewall rules for ports 9444 and 6514
   - Verify custom network configuration

3. **TLS certificate issues**
   - Check container logs for certificate generation messages
   - Verify certificate files if using custom certs
   - Test with `curl -k` to bypass certificate validation temporarily

### Logs and Monitoring
- **Container logs**: Available in Portainer → Containers → bibbl-stream → Logs
- **Application logs**: Stored in `/app/logs` volume
- **Health status**: Monitor via health check endpoint

## Security Considerations

1. **Firewall Configuration**
   - Only allow necessary ports (9444, 6514)
   - Restrict access to management interfaces

2. **TLS Settings**
   - Use TLS 1.2 minimum
   - Consider client certificate authentication for enhanced security

3. **Secrets Management**
   - Use Portainer secrets for sensitive configuration
   - Avoid hardcoding credentials in environment variables

## Maintenance

### Updates
1. Pull latest image: `docker pull bibbl-stream:latest`
2. Recreate stack in Portainer
3. Monitor health check after update

### Backup
- Backup persistent volumes (`bibbl-data`, `bibbl-logs`)
- Export stack configuration from Portainer
- Backup custom certificates and configuration files