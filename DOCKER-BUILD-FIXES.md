# Docker Build Fixes

This document describes the fixes made to resolve Docker build issues.

## Problem

The Docker build was failing due to network connectivity issues when trying to install packages and download Go dependencies.

## Root Cause

1. Network connectivity issues in build environment prevented downloading packages from Debian repositories
2. Go module downloads were failing due to DNS/network resolution issues

## Solution

1. **Vendor Dependencies**: Used `go mod vendor` to create a local copy of all dependencies, eliminating need for network access during Docker build
2. **Scratch Base Image**: Switched from Debian to `scratch` base image to eliminate package installation requirements
3. **Multi-stage Build**: Used Go builder stage to compile the binary, then copy only the necessary files to the final image
4. **Environment Variables**: Configured proper environment variables for Docker deployment

## Changes Made

### Dockerfile

- Switched to `scratch` base image for security and minimal size
- Added vendor directory copying to avoid network dependencies during build
- Used `-mod=vendor` flag for Go build to use local dependencies
- Added environment variables for proper host binding in containers
- Copied CA certificates and timezone data from builder stage

### Makefile

- Added `vendor` target to create local dependency cache
- Updated `docker` target to depend on both `web` and `vendor`
- Added vendor cleanup to `clean` target

### Build Process

1. `make web` - Builds the React web UI
2. `go mod vendor` - Creates local dependency cache
3. `docker build` - Builds container with no network dependencies

## Usage

### Quick Start

```bash
# Build Docker image
make docker

# Run container
docker run -d -p 9444:9444 -e BIBBL_SERVER_HOST=0.0.0.0 bibbl-stream:latest

# Test health endpoint
curl -k https://localhost:9444/api/v1/health
```

### Environment Variables

- `BIBBL_SERVER_HOST=0.0.0.0` - Required for Docker container networking
- `BIBBL_SERVER_PORT=9444` - Server port (default)

### Health Check

The container exposes HTTPS on port 9444. Health check endpoint:

```bash
curl -k https://localhost:9444/api/v1/health
```

## Security Notes

- Uses scratch base image for minimal attack surface
- No external packages installed at runtime
- Self-signed certificates generated automatically
- CA certificates included for outbound HTTPS connections

## Benefits

1. **Network Independent**: Builds work in restricted network environments
2. **Secure**: Minimal base image with no unnecessary packages
3. **Fast**: Smaller image size and faster startup
4. **Reproducible**: Vendor dependencies ensure consistent builds
