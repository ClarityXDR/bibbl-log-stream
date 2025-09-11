# Build Instructions

This document describes how to build Bibbl Log Stream in different environments.

## Prerequisites

- Go 1.22+
- Node.js 18+ (for web UI)
- Docker (for containerized builds)

## Building the Application

### Quick Start

```bash
# Build everything (Windows, Linux, web UI)
make all

# Build for specific platform
make windows  # Creates bibbl-stream.exe
make linux    # Creates bibbl-stream
```

### Building for Docker

**Option 1: Using Makefile (Recommended)**
```bash
# This builds web assets and vendor dependencies, then builds Docker image
make docker
```

**Option 2: Using docker-compose with Makefile**
```bash
# This ensures web assets are built before docker-compose build
make docker-compose
```

**Option 3: Manual steps**
```bash
# 1. Build web assets first
make web

# 2. Create vendor dependencies
make vendor

# 3. Then build with Docker
docker build -t bibbl-stream:latest .
# or
docker compose build
```

## Common Issues

### Docker Build Failure: "Web assets not built"

**Error:**
```
ERROR: Web assets not built. Run 'make web' first or use 'make docker' instead of 'docker build' directly.
```

**Solution:**
The Dockerfile requires web assets to be pre-built. Use one of these approaches:

1. **Use the Makefile:** `make docker` (recommended)
2. **Use docker-compose via Makefile:** `make docker-compose`
3. **Build web assets first:** `make web` then `docker build`

### Missing Node.js Dependencies

If you get Node.js related errors:
```bash
cd internal/web
npm install
cd ../..
make web
```

## Build Targets

- `make all` - Build all platforms and web UI
- `make windows` - Windows executable (bibbl-stream.exe)
- `make linux` - Linux executable (bibbl-stream)
- `make linux-arm` - Linux ARM64 executable (bibbl-stream-arm64)
- `make web` - Build React web UI (creates internal/web/static/)
- `make vendor` - Create Go module vendor directory
- `make docker` - Build Docker image with all dependencies
- `make docker-compose` - Build via docker-compose with dependencies
- `make clean` - Remove all build artifacts
- `make test` - Run Go tests

## Development Workflow

1. **First time setup:**
   ```bash
   git clone <repo>
   cd bibbl-log-stream
   make web    # Build web UI
   make vendor # Cache Go dependencies
   ```

2. **Regular development:**
   ```bash
   # After making changes to Go code
   make linux  # or make windows

   # After making changes to web UI
   make web
   make linux
   ```

3. **Docker development:**
   ```bash
   # After any changes
   make docker
   ```

## Production Deployment

For production deployments, use the Docker images:

```bash
# Build production image
make docker

# Deploy with docker-compose
docker compose up -d
```

The application will be available at:
- Web UI: https://localhost:9444
- Syslog TLS: port 6514