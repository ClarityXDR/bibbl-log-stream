# Multi-stage build for Bibbl Log Stream
# 1) Build React web UI (internal/web)
# 2) Build Go binary embedding the built static assets
# 3) Create minimal runtime image

ARG GO_VERSION=1.22
ARG NODE_VERSION=20

# --- Web UI build ---
FROM node:${NODE_VERSION}-alpine AS web
WORKDIR /src

# Only copy dependency manifests first to leverage Docker cache
COPY internal/web/package*.json internal/web/
WORKDIR /src/internal/web
RUN npm ci || npm install
COPY internal/web/ /src/internal/web/
RUN npm run build

# --- Go build ---
FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor/ ./vendor/
# If vendor exists, Go will honor -mod=vendor; download step is skipped
RUN if [ -d vendor ]; then echo "Using vendored modules"; else go mod download; fi
COPY . ./
# Bring in built web assets produced in the previous stage
COPY --from=web /src/internal/web/static /src/internal/web/static

ARG VERSION=0.1.0
ARG COMMIT=dev
ARG DATE
ENV CGO_ENABLED=0
RUN go build -mod=vendor \
    -ldflags "-w -s -X 'bibbl/internal/version.Version=${VERSION}' -X 'bibbl/internal/version.Commit=${COMMIT}' -X 'bibbl/internal/version.Date=${DATE}'" \
    -o /out/bibbl-stream ./cmd/bibbl

# --- Runtime ---
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /
COPY --from=builder /out/bibbl-stream /bibbl-stream

# Bind on all interfaces by default inside container
ENV BIBBL_SERVER_HOST=0.0.0.0
# Default port
EXPOSE 9444

ENTRYPOINT ["/bibbl-stream"]