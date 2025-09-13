# Health Check Fix Documentation

## Problem Solved

The bibbl-log-stream container was hanging during startup because the Docker health checks were failing. The health check configuration in docker-compose.yml was using `wget` to test the health endpoint, but `wget` was not available in the Alpine runtime image.

## Root Cause

1. **Missing dependency**: The health check used `wget` which wasn't installed in the Alpine base image
2. **External tool dependency**: Health checks relied on external tools rather than the application itself
3. **Network/port confusion**: Some inconsistency in expected ports between different docker-compose configurations

## Solution Implemented

### 1. Built-in Health Check Command

Added a `-health` flag to the main bibbl binary in `cmd/bibbl/main.go`:

- **Self-contained**: No external dependencies required
- **Smart protocol detection**: Tries HTTPS first (common case), falls back to HTTP
- **TLS-friendly**: Uses `InsecureSkipVerify` for self-signed certificates
- **Proper exit codes**: Returns 0 for success, 1 for failure
- **Timeout handling**: 10-second timeout prevents hanging

### 2. Updated Docker Configuration

#### Dockerfile:
```dockerfile
# Health check using built-in command
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/bibbl-stream", "-health"]
```

#### docker-compose.yml and docker-compose.simple.yml:
```yaml
healthcheck:
  test: ["/bibbl-stream", "-health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 30s
```

## Testing Results

### Local Testing ✅
```bash
# Start server
BIBBL_SERVER_HOST=127.0.0.1 BIBBL_SERVER_PORT=8080 ./bibbl-stream

# Test health check (separate process)
BIBBL_SERVER_HOST=127.0.0.1 BIBBL_SERVER_PORT=8080 ./bibbl-stream -health
# Output: "2025/09/13 01:16:37 Health check https passed"
# Exit code: 0
```

### Health Check Function Features

```go
func performHealthCheck(cfg *config.Config) int {
    addr := cfg.HTTPAddr()
    
    // HTTP client with TLS support
    client := &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                InsecureSkipVerify: true, // For self-signed certs
            },
        },
    }
    
    // Try HTTPS first, then HTTP
    schemes := []string{"https", "http"}
    
    for _, scheme := range schemes {
        url := fmt.Sprintf("%s://%s/api/v1/health", scheme, addr)
        
        resp, err := client.Get(url)
        if err != nil {
            log.Printf("Health check %s failed: %v", scheme, err)
            continue
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            log.Printf("Health check %s failed: HTTP %d", scheme, resp.StatusCode)
            continue
        }
        
        log.Printf("Health check %s passed", scheme)
        return 0
    }
    
    log.Printf("Health check failed on all schemes")
    return 1
}
```

## Benefits

1. **Zero dependencies**: No need for wget, curl, or other external tools
2. **Container-friendly**: Works in minimal Alpine images
3. **TLS-aware**: Handles both HTTP and HTTPS configurations automatically
4. **Robust**: Proper error handling and timeout management
5. **Consistent**: Same health check logic across all deployment methods
6. **Debuggable**: Clear logging of health check attempts and results

## Migration Guide

### For Existing Deployments

If you're using the old wget-based health checks, simply update your docker-compose.yml:

**Before:**
```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "--no-check-certificate", "https://localhost:9444/api/v1/health"]
```

**After:**
```yaml
healthcheck:
  test: ["/bibbl-stream", "-health"]
```

### Manual Testing

To test the health check manually in a running container:

```bash
# Enter the container
docker exec -it bibbl-stream /bin/sh

# Run health check
/bibbl-stream -health

# Should output something like:
# 2025/09/13 01:16:37 Health check https passed
# Exit code: 0
```

## Files Changed

- `cmd/bibbl/main.go`: Added `-health` flag and `performHealthCheck()` function
- `Dockerfile`: Added `HEALTHCHECK` instruction, fixed npm install command
- `docker-compose.yml`: Updated health check to use built-in command
- `docker-compose.simple.yml`: Updated health check to use built-in command

## Expected Behavior

With these changes, the container should:

1. **Start normally**: Server starts and listens on configured port
2. **Pass health checks**: Built-in health check succeeds after startup period
3. **Go healthy**: Docker marks container as healthy within 30-60 seconds
4. **Stay healthy**: Periodic health checks continue to pass

The startup hanging issue should be completely resolved.