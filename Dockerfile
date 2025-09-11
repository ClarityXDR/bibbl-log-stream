# Dockerfile for Bibbl Log Stream
# Build Go application with pre-built web UI

FROM golang:1.22-bullseye AS go-builder

WORKDIR /app

# Copy go mod files and vendor directory for offline build
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy source code
COPY . .

# Ensure web assets are present (should be pre-built)
RUN if [ ! -d "internal/web/static" ]; then echo "ERROR: Web assets not built. Run 'make web' first or use 'make docker' instead of 'docker build' directly."; exit 1; fi

# Generate embedded assets and build using vendor directory (no network required)
RUN go generate ./...

# Build the application using vendor dependencies
ARG VERSION=0.1.0
ARG COMMIT=docker
ARG DATE
RUN if [ -z "$DATE" ]; then DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ); fi && \
    LDFLAGS="-w -s -X 'bibbl/internal/version.Version=${VERSION}' -X 'bibbl/internal/version.Commit=${COMMIT}' -X 'bibbl/internal/version.Date=${DATE}'" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="$LDFLAGS" -o bibbl-stream cmd/bibbl/main.go

# Final runtime image using scratch for maximum security and minimal size
FROM scratch

# Copy CA certificates from builder stage
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data from builder stage
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app

# Copy binary and config from builder
COPY --from=go-builder /app/bibbl-stream /app/bibbl-stream
COPY --from=go-builder /app/config.example.yaml /app/config.example.yaml

# Expose ports
EXPOSE 9444 6514

# Set default environment variables for Docker deployment
ENV BIBBL_SERVER_HOST=0.0.0.0
ENV BIBBL_SERVER_PORT=9444

# No health check for scratch image - will be handled externally
# Health check would be: curl -k -f https://localhost:9444/api/v1/health

# Default command
CMD ["/app/bibbl-stream"]