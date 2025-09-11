# Dockerfile for Bibbl Log Stream
# Build Go application with pre-built web UI

FROM golang:1.22-alpine AS go-builder

# Install ca-certificates for TLS certificate verification and update them
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
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
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1001 -S bibbl && \
    adduser -u 1001 -S bibbl -G bibbl

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