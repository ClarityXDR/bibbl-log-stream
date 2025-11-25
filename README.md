# Bibbl Log Stream

Single-binary, cross-platform log pipeline written in Go with an embedded React/TypeScript Web UI.

Quickstart:

- Build web UI: `make web`
- Build Windows exe: `make windows`
- Run server locally: `go run ./cmd/bibbl`
- Open [http://localhost:8080](http://localhost:8080)

Config: copy config.example.yaml to config.yaml and adjust.

Environment overrides: any config key can be set with env vars prefixed by `BIBBL_`. Example:

    BIBBL_SERVER_PORT=9555 ./bibbl-stream

Version info: run `./bibbl-stream -version` to print build metadata (version, commit, date).

Health endpoints:

- `/api/v1/health` (JSON ok)
- `/ready` (readiness probe)
- `/live` (liveness probe)

Metrics: /metrics (Prometheus format)

Security headers: basic hardened defaults (CSP, no sniff, frame deny) are applied by the server.

Syslog TLS auto-cert: when `inputs.syslog.tls.auto_cert.enabled` is true (default), Bibbl automatically generates and renews a self-signed certificate for the Syslog-over-TLS listener. The PEM files are stored under `./certs/syslog/` (share the `.crt`/`.pem` with Versa SD-WAN firewalls, keep the `.key` private). Add additional SANs via `inputs.syslog.tls.auto_cert.hosts` or the `BIBBL_SYSLOG_TLS_EXTRA_HOSTS` environment variable.

## Pipeline filters & routing

- Pipelines can declare structured filters via the API or UI. The Pipelines modal now includes a filter builder where you can pick any field, choose `include` or `exclude`, and supply comma-separated values—no need to hand-edit `filter:` strings.
- Under the hood every filter row is persisted as a `filter:` transform (for example, `filter:severity=critical|high|med`). Existing handcrafted functions are parsed and shown back in the builder so you can tweak them visually.
- API clients may send filters explicitly:

                {
                    "name": "Sentinel High",
                    "description": "Only high value events",
                    "functions": ["Parse CEF"],
                    "filters": [
                        { "field": "severity", "values": ["critical", "high", "med"], "mode": "include" },
                        { "field": "vendorRisk", "values": ["critical"], "mode": "exclude" }
                    ]
                }

- Filter drop counts and totals are exported via `bibbl_pipeline_events_processed_total{status="filtered"}` and mirrored in `/api/v1/pipelines/stats`, which now includes processed counts plus drop percentages so operators can verify suppression rates.

See vision.md for requirements and roadmap.
