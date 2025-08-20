# Bibbl Log Stream

Single-binary, cross-platform log pipeline written in Go with an embedded React/TypeScript Web UI.

Quickstart:
- Build web UI: make web
- Build Windows exe: make windows
- Run server locally: go run ./cmd/bibbl
- Open http://localhost:8080

Config: copy config.example.yaml to config.yaml and adjust.

Environment overrides: any config key can be set with env vars prefixed by BIBBL_. Example:
	BIBBL_SERVER_PORT=9555 ./bibbl-stream

Version info: run `./bibbl-stream -version` to print build metadata (version, commit, date).

Health endpoints:
	/api/v1/health  (JSON ok)
	/ready          (readiness probe)
	/live           (liveness probe)

Metrics: /metrics (Prometheus format)

Security headers: basic hardened defaults (CSP, no sniff, frame deny) are applied by the server.

See vision.md for requirements and roadmap.
