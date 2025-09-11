# Dockerfile for Bibbl Log Stream
# Build Go application with pre-built web UI

FROM golang:1.22-bullseye AS go-builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies with relaxed TLS verification for build environment issues
ENV GOPROXY=direct
ENV GOSUMDB=off
RUN go mod download

# Copy source code
COPY . .

# Ensure web assets are present (should be pre-built)
RUN if [ ! -d "internal/web/static" ]; then echo "ERROR: Web assets not built. Run 'make web' first."; exit 1; fi

# Generate embedded assets and build
RUN go generate ./...

# Build the application
ARG VERSION=0.1.0
ARG COMMIT=docker
ARG DATE
RUN if [ -z "$DATE" ]; then DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ); fi && \
    LDFLAGS="-w -s -X 'bibbl/internal/version.Version=${VERSION}' -X 'bibbl/internal/version.Commit=${COMMIT}' -X 'bibbl/internal/version.Date=${DATE}'" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bibbl-stream cmd/bibbl/main.go

# Final runtime image
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates tzdata wget && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 bibbl && \
    useradd -u 1001 -g bibbl -s /bin/false bibbl

# Create directories
RUN mkdir -p /app/data /app/logs /app/certs && \
    chown -R bibbl:bibbl /app

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/bibbl-stream .
COPY --from=go-builder /app/config.example.yaml ./config.example.yaml

# Set ownership
RUN chown bibbl:bibbl bibbl-stream config.example.yaml

USER bibbl

# Expose ports
EXPOSE 9444 6514

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9444/api/v1/health || exit 1

# Default command
CMD ["./bibbl-stream"]