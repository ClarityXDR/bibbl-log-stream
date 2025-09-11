# Makefile for Bibbl Log Stream

VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "")
LDFLAGS := -w -s -X 'bibbl/internal/version.Version=$(VERSION)' -X 'bibbl/internal/version.Commit=$(COMMIT)' -X 'bibbl/internal/version.Date=$(DATE)'

all: windows linux linux-arm web

windows: web
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bibbl-stream.exe cmd/bibbl/main.go

linux: web
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bibbl-stream cmd/bibbl/main.go

linux-arm: web
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bibbl-stream-arm64 cmd/bibbl/main.go

web:
	cd internal/web && npm install && npm run build
	go generate ./...

vendor:
	go mod vendor

clean:
	rm -f bibbl-stream bibbl-stream.exe bibbl-stream-arm64
	rm -rf internal/web/static/*
	rm -rf vendor/

test:
	go test ./...

# Race detection (requires CGO/gcc in environment)
race:
	set CGO_ENABLED=1 && go test -race ./...

# Docker targets
docker: web vendor
	docker build -t bibbl-stream:latest .

docker-compose:
	make web vendor
	docker compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f bibbl-stream

.PHONY: all windows linux linux-arm web vendor clean test race docker docker-compose docker-up docker-down docker-logs
